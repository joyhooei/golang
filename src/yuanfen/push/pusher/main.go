package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
	"yf_pkg/log"
	"yf_pkg/net"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/push/pusher/api"
	"yuanfen/push/pusher/common"
	"yuanfen/push/pusher/db"
	"yuanfen/push/pusher/notifier"
	"yuanfen/push/pusher/user"
)

var nlog, ulog *log.Logger
var conf common.Config

func isValidUser(r *http.Request) uint32 {
	c, e := r.Cookie("uid")
	if e != nil {
		ulog.Append(fmt.Sprintf("read cookie uid failed : %v", e.Error()), log.DEBUG)
		return 0
	}
	uid, e := utils.StringToUint32(c.Value)
	if e != nil {
		ulog.Append(fmt.Sprintf("parse cookie uid [%v] failed : %v", c.Value, e.Error()), log.DEBUG)
		return 0
	}
	return uid
}

func getEnv(module string) *service.Env {
	return service.NewEnv(conf)
}

func Accept(ip string, port int) {
	ulog.Append(fmt.Sprintf("Listen %v:%v ...", ip, port))
	ln, err := net.Listen(ip, port)
	if err != nil {
		fmt.Println(err.Error())
		ulog.Append(err.Error())
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err.Error())
			ulog.Append(err.Error())
			return
		}
		go user.AddUser(conn)
	}
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
		fmt.Println("Load config error : ", err.Error())
		return
	}
	fmt.Println("init db ...")
	db.Init(&conf)
	fmt.Println("success")
	//创建用户行为日志
	ulog, err = log.New2(conf.Log.Dir+"/push_user.log", 10000, conf.Log.Level)
	//ulog, err = log.New2("/dev/null", 10000, conf.Items["log_level"])
	if err != nil {
		fmt.Println("Init log error :", err.Error())
		return
	}
	//创建通知日志
	nlog, err = log.New2(conf.Log.Dir+"/notify.log", 10000, conf.Log.Level)
	//ulog, err = log.New2("/dev/null", 10000, conf.Items["log_level"])
	if err != nil {
		fmt.Println("Init log error :", err.Error())
		return
	}

	notifier.Init(nlog)
	user.SetLog(ulog)

	for _, addr := range conf.TcpAddr {
		go Accept(addr.Ip, addr.Port)
	}

	//侦听HTTP端口
	var c service.Config = service.Config{conf.HttpAddr.String(), conf.Log.Dir, conf.Log.Level, getEnv, isValidUser}
	server, err := service.New(&c, false)
	if err != nil {
		fmt.Println("Init Config error :", err.Error())
		return
	}
	//创建自定义module
	server.AddModule("push", &api.Receiver{})
	fmt.Println(server.StartService())
}
