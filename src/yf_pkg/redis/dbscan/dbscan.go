package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
	"yf_pkg/redis"
	"yf_pkg/utils"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Printf("invalid args : %s [ip] [port] [db] [command] [args]\n", os.Args[0])
		return
	}
	addr := fmt.Sprintf("%v:%v", os.Args[1], os.Args[2])
	db, e := utils.ToInt(os.Args[3])
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	rand.Seed(time.Now().UnixNano())
	redisdb := redis.New(addr, []string{addr}, 2)
	conn := redisdb.GetReadConnection(db)
	defer conn.Close()
	keys, e := redis.Strings(conn.Do("KEYS", "*"))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	args := []interface{}{""}
	for _, s := range os.Args[5:] {
		args = append(args, s)
	}
	for _, key := range keys {
		args[0] = key
		reply, e := conn.Do(os.Args[4], args...)
		if e != nil {
			fmt.Println(key, e.Error())
			return
		}
		fmt.Println(key, reply)
	}
}
