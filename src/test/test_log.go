package main

import (
	"code.serveyou.cn/pkg/log"
	"time"
)

func main(){
	l, err := log.New("my.log", 100)
	if err != nil {
		panic(err.Error())
	}
	l.Append("hello")
	l.Append("hello1")
	l.Append("hello2")
	l.Append("hello3")
	l.Close()
	time.Sleep(10*time.Second)
}
