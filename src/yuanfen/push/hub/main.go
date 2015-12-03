package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
	"yf_pkg/log"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/push/hub/api"
	"yuanfen/push/hub/config"
	"yuanfen/push/hub/db"
	"yuanfen/push/hub/node_manager"
)

var ulog *log.MLogger
var conf config.Config

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
	err = db.InitMysql(conf.Mysql.Main.Master, conf.Mysql.Main.Slave)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = db.InitRedis(conf.Redis.Main.Master.String(), conf.Redis.Main.Slave.StringSlice(), conf.Redis.Main.MaxConn)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//创建用户行为日志
	ulog, err = log.NewMLogger(conf.Log.Dir+"/push_user", 10000, conf.Log.Level)
	//ulog, err = log.New2("/dev/null", 10000, conf.Items["log_level"])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//节点管理器初始化
	err = node_manager.Init(ulog)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	//侦听HTTP端口
	var c service.Config = service.Config{conf.Address.String(), conf.Log.Dir, conf.Log.Level, getEnv, isValidUser}
	server, err := service.New(&c, true)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//创建自定义module
	server.AddModule("push", &api.Receiver{})
	fmt.Println(server.StartService())
}
