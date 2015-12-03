package main

import (
	"dsp/api"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
	"yf_pkg/service"
	"yuanfen/push/hub/config"
)

var conf config.Config

func isValidUser(r *http.Request) uint32 {
	return 1
}

func getEnv(module string) *service.Env {
	return service.NewEnv(conf)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s [config]\n", os.Args[0])
		return
	}

	rand.Seed(time.Now().UnixNano())
	//读取配置文件
	err := conf.Load(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//侦听HTTP端口
	var c service.Config = service.Config{conf.Address.String(), conf.Log.Dir, conf.Log.Level, getEnv, isValidUser}
	server, err := service.New(&c, false, true)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//创建自定义module
	server.AddModule("dsp", &api.Receiver{})
	fmt.Println(server.StartService())
}
