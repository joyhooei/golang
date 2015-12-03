package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
	// "yf_pkg/log"
	"yf_pkg/net"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/push/pusher/common"
	// "yuanfen/push/pusher/db"
	// "yuanfen/push/pusher/notifier"
	// "yuanfen/push/pusher/user"
	"yuanfen/manageragent/cls/manager"
	"yuanfen/manageragent/cls/mtcp"
	"yuanfen/manageragent/modules/control"
)

var conf common.Config

func Accept(ip string, port int) {
	// ulog.Append(fmt.Sprintf("Listen %v:%v ...", ip, port))
	ln, err := net.Listen(ip, port)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println("new connection...")
		go manager.AddManager(conn)
	}
}

func isValidUser(r *http.Request) (uid uint32) {
	c, e := r.Cookie("uid")
	if e != nil {
		return 0
	}
	// k, e := r.Cookie("key")
	// if e != nil {
	// 	return 0
	// }
	// key := k.Value
	uid, e = utils.StringToUint32(c.Value)
	if e != nil {
		return 0
	}
	return uid
}

func getEnv(module string) *service.Env {
	return service.NewEnv(conf)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s [config]\n", os.Args[0])
		return
	}
	mtcp.SendToManager = manager.SendMessage

	err := conf.Load(os.Args[1])
	if err != nil {
		fmt.Println("Load config error : ", err.Error())
		return
	}
	err = manager.Init(&conf)
	if err != nil {
		fmt.Println(err.Error())
	}
	err = mtcp.Init(&conf)

	rand.Seed(time.Now().UnixNano())

	// 侦听TCP端口
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
	server.AddModule("control", &control.Control{})
	fmt.Println(server.StartService())
}
