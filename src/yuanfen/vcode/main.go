package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
	"yf_pkg/log"
	"yf_pkg/redis"
	"yuanfen/vcode/model"
)

var GlobalConfig Config

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s [config]\n", os.Args[0])
		return
	}

	rand.Seed(time.Now().UnixNano())
	err := GlobalConfig.Load(os.Args[1])
	if err != nil {
		fmt.Println("Load Config error :", err.Error())
		return
	}
	cacheRds := redis.New(GlobalConfig.Redis.Cache.Master.String(), GlobalConfig.Redis.Cache.Slave.StringSlice(), GlobalConfig.Redis.Cache.MaxConn)
	mainLog, err := log.NewMLogger(GlobalConfig.Log.Dir+"/main", 10000, GlobalConfig.Log.Level)
	if err != nil {
		fmt.Println("Init log failed:", err.Error())
		return
	}
	model.Init(cacheRds, mainLog)

	http.HandleFunc("/pic", model.Pic)
	http.HandleFunc("/verify", model.Verify)
	s := &http.Server{
		Addr:         GlobalConfig.Address.String(true),
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}
	s.ListenAndServe()
}
