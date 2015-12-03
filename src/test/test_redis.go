package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

func newPool(server string, maxActive int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   maxActive,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
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
}

var (
	pool        *redis.Pool
	redisServer = flag.String("redis-server", "192.168.1.91:26379", "redis server address. Format ip:port.")
	maxActive   = flag.Int("max-connections", 0, "max connection to resdis.0 is default value which means no limit.")
)

func hmget(values interface{}) error {
	flag.Parse()
	pool = newPool(*redisServer, *maxActive)
	rc := pool.Get()
	_, err := rc.Do("hset", "baobao", "a", 1)
	_, err = rc.Do("hset", "baobao", "b", 2)
	vs, err := redis.Values(rc.Do("hmget", "baobao", "a", "b", "c"))
	if err != nil {
		fmt.Println(err.Error())
	}
	return redis.ScanSlice(vs, values)
}

func main() {
	ids := []uint64{}
	err := hmget(&ids)
	if err != nil {
		fmt.Println(err.Error())
	}

	for i, value := range ids {
		fmt.Printf("%v: %v\n", i, value)
	}
}
