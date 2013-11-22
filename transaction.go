package main

import (
	"github.com/garyburd/redigo/redis"
	"log"
	"time"
)

const (
	SERVER = ":6379"
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

type Transaction struct {
	err     error
	success func(interface{})
	reply   interface{}
}

func NewTransaction() *Transaction {
	return &Transaction{}
}

func (t *Transaction) Do(cb func(conn redis.Conn)) *Transaction {
	c := pool.Get()
	defer c.Close()
	c.Send("MULTI")
	cb(c)
	reply, err := c.Do("EXEC")
	t.reply = reply
	t.err = err
	return t
}

func (t *Transaction) OnFail(cb func(err error)) *Transaction {
	if t.err != nil {
		cb(t.err)
	} else {
		t.success(t.reply)
	}
	return t
}

func (t *Transaction) OnSuccess(cb func(reply interface{})) *Transaction {
	t.success = cb
	return t
}

func main() {

	NewTransaction().Do(func(c redis.Conn) {
		c.Send("INCR", "Bobby")
		c.Send("DECR", "Bobby")
		c.Send("INCR", "Bobby")
		c.Send("DECR", "Bobby")
		c.Send("INCR", "Bobby")
		c.Send("DECR", "Bobby")
		c.Send("INCR", "Bobby")
		c.Send("DECR", "Bobby")
		c.Send("INCR", "Bobby")
		c.Send("DECR", "Bobby")
		c.Send("INCR", "Bobby")
		c.Send("DECR", "Bobby")
		c.Send("INCR", "Bobby")
		//c.Send("INCRs", "Bobby")
	}).OnSuccess(func(reply interface{}) {
		log.Println("Success!")

		log.Println(reply)

	}).OnFail(func(err error) {
		log.Println("Oh no, transaction failed, alert user.")
		log.Println(err)
	})
}
