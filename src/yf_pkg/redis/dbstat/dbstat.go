package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
	"yf_pkg/redis"
	"yf_pkg/utils"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Printf("invalid args : %s [ip] [port] [db]\n", os.Args[0])
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
	fmt.Printf("keys : %v", len(keys))
	base := len(keys) / 100000
	if base < 5 {
		base = 1
	}
	fmt.Printf(" base : %v\n", base)
	i := 0
	var totalSize int64 = 0
	for _, key := range keys {
		if i%base == 0 {
			fmt.Printf("%v/%v\r", i, len(keys))
			reply, e := redis.String(conn.Do("DEBUG", "OBJECT", key))
			if e != nil {
				fmt.Println(key, e.Error())
				continue
			}
			//		fmt.Println(key, reply)
			kv := strings.Split(reply, " ")
			size := strings.Split(kv[4], ":")
			s, e := utils.ToInt64(size[1])
			if e != nil {
				fmt.Println(e.Error())
				return
			}
			totalSize += s
		}
		i++
	}
	totalSize = totalSize * int64(base)
	GB := float32(totalSize) / 1024 / 1024 / 1024
	MB := float32(totalSize) / 1024 / 1024
	KB := float32(totalSize) / 1024
	switch {
	case KB < 1:
		fmt.Printf(" size : %vB\n", totalSize)
	case MB < 1:
		fmt.Printf(" size : %.3fKB\n", KB)
	case GB < 1:
		fmt.Printf(" size : %.3fMB\n", MB)
	default:
		fmt.Printf(" size : %.3fGB\n", GB)
	}

}
