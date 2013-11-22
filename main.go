package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/codegangsta/martini"
	"github.com/garyburd/redigo/redis"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	SERVER           = ":6379"
	BOARDSIZESQUARED = 25
	BOARDSIZE        = BOARDSIZESQUARED * BOARDSIZESQUARED
)

var (
	STARTINGROOMOFFSET = int(math.Ceil(BOARDSIZE / 2))
)

/*
	TODO:
		create a command system that allows you to fire off a command either
		within a transaction as a MULTI EXEC or singular as a Do()

		consider: dungeon tiles can be user gravatar, it is their home

		interesting: we can simulate virtual host by using the xip.io service and based on
		a subdomain like: dungeon.127.0.0.1.xip.io:3000 we can observe in the handler the
		subdomain part.

*/

var pool = &redis.Pool{
	MaxIdle:     3,
	IdleTimeout: 240 * time.Second,
	Dial: func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", SERVER)

		if err != nil {
			return nil, err
		}
		return c, err
	},
	TestOnBorrow: func(c redis.Conn, t time.Time) error {
		_, err := c.Do("PING")
		return err
	},
}

type State struct {
	Dungeon []string
}

func main() {

	TestRTran()

	return

	m := martini.Classic()

	/*
		REMEMBER: ULTIMATE GOAL
		ABSOLUTELY NO STATE IN WEBSERVER...should all be in REDIS!!!!!!!
	*/

	//Sanity check on connection
	TestConnection()
	initDungeon()

	/*
		//serve app state
		http.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, GetDungeonJSON())
		})
	*/
	m.Get("/debug", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, req.RemoteAddr)
		fmt.Fprintln(w, req.URL)
		fmt.Fprintln(w, req.Host)
	})

	m.Get("/state", func(w http.ResponseWriter, req *http.Request) string {
		w.Header().Set("Content-Type", "application/json")
		return GetDungeonJSON()
	})

	m.Get("/add/:room", func(params martini.Params, w http.ResponseWriter, req *http.Request) string {
		w.Header().Set("Content-Type", "application/json")

		//TODO:validate input that it is a number in proper range
		v := params["room"]
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Println("Could not get room number")
			return "\"{\"err\":\"Could not parse room number\"}\""
		}

		AddRoom(i)
		return GetDungeonJSON()
	})

	m.Get("/clearall", func(w http.ResponseWriter, req *http.Request) string {
		w.Header().Set("Content-Type", "application/json")

		ClearAllState()
		return GetDungeonJSON()
	})

	m.Run()
}

func initDungeon() {
	defer un(trace("initDungeon()"))
	//ClearAllState()
	//CreateEmptyDungeon()
	AddRoom(0)
	AddRoom(1)
	AddRoom(2)
}

func trace(s string) (string, time.Time) {
	log.Println("START:", s)
	return s, time.Now()
}

func un(s string, startTime time.Time) {
	endTime := time.Now()
	log.Println("  END:", s, "ElapsedTime in seconds:", endTime.Sub(startTime))
}

func TestConnection() {

	conn := pool.Get()
	defer conn.Close()

	x, err := redis.String(conn.Do("SET", "NAME", "RALPH"))
	if err != nil {
		log.Println("Perhaps Redis is offline or somethin'")
		log.Fatal(err)
	}
	fmt.Println("Success!")
	fmt.Println(x)
}

func TestRTran() {
	RTran(func(conn redis.Conn) {
		conn.Send("INCR", "Bobby")
		conn.Send("DECR", "Bobby")
		conn.Send("INCR", "Bobby")
		conn.Send("DECR", "Bobby")
		conn.Send("INCR", "Bobby")
		conn.Send("DECR", "Bobby")
		conn.Send("INCR", "Bobby")
		conn.Send("DECR", "Bobby")
		conn.Send("INCR", "Bobby")
		conn.Send("DECR", "Bobby")
		conn.Send("INCR", "Bobby")
		conn.Send("DECR", "Bobby")
		conn.Send("INCRs", "Bobby")
	})
}

//Trying to come up with a transaction context manager for Redigo
func RTran(cb func(conn redis.Conn)) {
	conn := pool.Get()
	defer conn.Close()
	conn.Send("MULTI")
	cb(conn)
	_, err := conn.Do("EXEC")
	if err != nil {
		log.Println("Transaction failed!")
		log.Fatal(err)
	}
}

func ClearAllState() {
	log.Println("ClearAllState()")

	conn := pool.Get()
	defer conn.Close()

	conn.Send("MULTI")
	conn.Send("DEL", "DUNGEON")
	_, err := conn.Do("EXEC")

	if err != nil {
		log.Fatal("Couldn't clear all state")
	}

}

func AddRoom(offset int) {
	log.Println("AddRoom()")
	//middle of the grid

	conn := pool.Get()

	_, err := conn.Do("SETBIT", "DUNGEON", offset, 1)

	if err != nil {
		log.Fatal(err)
	}
}

func CreateEmptyDungeon() {
	log.Println("CreateEmptyDungeon()")

	AddRoom(STARTINGROOMOFFSET)
}

func GetDungeon() []byte {
	log.Println("GetDungeon()")

	conn := pool.Get()
	defer conn.Close()

	repl, err := redis.Bytes(conn.Do("GETRANGE", "DUNGEON", 0, BOARDSIZE))
	if err != nil {
		log.Fatal(err)
	}

	return repl
}

func PadDungeon(dungeon []byte) []byte {

	//TODO: fix board size, somehow we end up with a board that is 632 bits it should be 625 at the most.
	dungeonSizeInBytes := int(math.Ceil(BOARDSIZE / 8.0))

	paddedDungeon := make([]byte, dungeonSizeInBytes)

	for i, b := range dungeon {
		paddedDungeon[i] = b
	}

	return paddedDungeon
}

func PadBits(n int) string {
	pad := ""

	for i := 0; i < n; i++ {
		pad += "0"
	}
	return pad
}

func DungeonBytesToBoard(bSlice []byte) string {

	var buf bytes.Buffer

	for _, b := range bSlice {

		bitString := strconv.FormatInt(int64(b), 2)
		buf.WriteString(PadBits(8-len(bitString)) + bitString)

	}

	return buf.String()
}

func PrintBoard(dungeon string) {
	result := strings.Split(dungeon, "")

	//truncate to max size of dungeon because of extra padding of bits remainder of byte at the end
	result = result[0:BOARDSIZE]

	fmt.Println("[" + strings.Join(result, ",") + "]")
}

func GetDungeonJSON() string {
	defer un(trace("GetDungeonJSON"))

	b := GetDungeon()
	pd := PadDungeon(b)
	dungeon := DungeonBytesToBoard(pd)
	result := strings.Split(dungeon, "")

	//truncate to max size of dungeon because of extra padding of bits remainder of byte at the end
	result = result[0:BOARDSIZE]

	s := State{Dungeon: result}
	b, err := json.Marshal(s)

	if err != nil {
		log.Fatal("Could not get Dungeon JSON")
	}
	return string(b)
}
