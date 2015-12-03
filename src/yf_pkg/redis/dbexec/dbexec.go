package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"time"
	"yf_pkg/redis"
	"yf_pkg/utils"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Printf("invalid args : %s [ip] [port] [db] [command] [args_file]\n", os.Args[0])
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
	f, err := os.OpenFile(os.Args[5], os.O_RDONLY, 0660)
	if err != nil {
		fmt.Printf("%s err read from %s : %s\n", os.Args[0], os.Args[5], err)
		return
	}
	r := bufio.NewReader(f)
	defer f.Close()
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		args := []interface{}{}
		for _, i := range strings.Split(line[0:len(line)-1], " ") {
			args = append(args, i)
		}
		fmt.Println(os.Args[4], args)
		reply, e := conn.Do(os.Args[4], args...)
		if e != nil {
			fmt.Println(args[0], e.Error())
			continue
		}
		fmt.Println(args[0], reply)
	}
}
