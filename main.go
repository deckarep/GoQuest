package main

import (
	"bytes"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"log"
	"math"
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

func main() {
	fmt.Println("Welcome to GoQuest")

	TestConnection()

	CreateEmptyDungeon()
	b := GetDungeon()

	paddedDungeon := PadDungeon(b)

	dungeonBoard := DungeonBytesToBoard(paddedDungeon)
	PrintBoard(dungeonBoard)
}

func TestConnection() {
	conn := pool.Get()
	x, err := redis.String(conn.Do("SET", "NAME", "RALPH"))
	if err != nil {
		log.Println("Perhaps Redis is offline or somethin'")
		log.Fatal(err)
	}
	fmt.Println("Success!")
	fmt.Println(x)
}

func CreateEmptyDungeon() {
	log.Println("CreateEmptyDungeon()")
	//middle of the grid

	conn := pool.Get()
	_, err := conn.Do("SETBIT", "DUNGEON", STARTINGROOMOFFSET, 1)

	if err != nil {
		log.Fatal(err)
	}
}

func GetDungeon() []byte {
	log.Println("GetDungeon()")

	conn := pool.Get()
	repl, err := redis.Bytes(conn.Do("GETRANGE", "DUNGEON", 0, BOARDSIZE))
	if err != nil {
		log.Fatal(err)
	}

	return repl
}

func PadDungeon(dungeon []byte) []byte {

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

	fmt.Println("[" + strings.Join(result, ",") + "]")
}
