package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/post_logs/common"
	"yuanfen/post_logs/post_module"
)

var customEnv common.CustomEnv
var GlobalConfig common.Config

func isValidUser(r *http.Request) (uid uint32) {
	c, e := r.Cookie("uid")
	if e != nil {
		fmt.Println("parse cookie [uid] error ", e.Error())
		return 0
	}
	k, e := r.Cookie("key")
	if e != nil {
		fmt.Println("parse cookie [key] error ", e.Error())
		return 0
	}
	key := k.Value
	uid, e = utils.StringToUint32(c.Value)
	if e != nil {
		return 0
	}
	fmt.Printf("uid=%v, key=%v\n", uid, key)
	return uid
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s [config]\n", os.Args[0])
		return
	}

	rand.Seed(time.Now().UnixNano())
	err := GlobalConfig.Load(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//初始化环境变量
	if err = customEnv.Init(&GlobalConfig); err != nil {
		fmt.Println(err.Error())
		return
	}

	//为客户端提供的http服务
	var c service.Config = service.Config{GlobalConfig.PublicAddr.String(), GlobalConfig.Log.Dir, GlobalConfig.Log.Level, customEnv.GetEnv, isValidUser}
	server, err := service.New(&c, false)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	//创建自定义module
	server.AddModule("post_module", &post_module.PostModule{})
	fmt.Println(server.StartService())
}
