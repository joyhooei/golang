package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Address struct {
	Ip   string "ip"
	Port int    "port"
}

type Addresses []Address

type MysqlType struct {
	Master string   "master"
	Slave  []string "slave"
}

type RedisType struct {
	Master  Address   "master"
	Slave   Addresses "slave"
	MaxConn int       "max_conn"
}

type Config struct {
	PrivateAddr Address "private_addr"
	PublicAddr  Address "public_addr"
	Log         struct {
		Dir   string "dir"
		Level string "level"
	} "log"
	PushAddr Address "push"
	Mysql    struct {
		Main MysqlType "main"
		Sort MysqlType "sort"
	} "mysql"
	Redis struct {
		Main  RedisType "main"
		Cache RedisType "cache"
	} "redis"
}

func (a *Address) String() string {
	return fmt.Sprintf("%s:%v", a.Ip, a.Port)
}

func (as Addresses) StringSlice() []string {
	ret := make([]string, len(as))
	for i, v := range as {
		ret[i] = v.String()
	}
	return ret
}

func main() {
	file, e := os.Open(os.Args[1])
	if e != nil {
		fmt.Println(e)
		return
	}
	info, e := file.Stat()
	if e != nil {
		fmt.Println(e)
		return
	}
	defer file.Close()
	data := make([]byte, info.Size())
	file.Read(data)

	var m Config

	err := yaml.Unmarshal([]byte(data), &m)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- m:\n%v\n\n", m)

	d, err := yaml.Marshal(&m)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("--- m dump:\n%s\n\n", string(d))
}
