package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
	"yf_pkg/mysql"
	"yf_pkg/service"
)

var db *mysql.MysqlDB
var GlobalConfig Config

func getEnv(module string) *service.Env {
	return service.NewEnv(map[string]interface{}{"db": db, "table": GlobalConfig.Table})
}
func isValidUser(r *http.Request) uint32 {
	return 1
}

//配置文件中的必要项
var keywords = map[string]bool{
	"address":   true,
	"log_dir":   true,
	"log_level": true,
	"main_rdb":  true,
	"main_wdb":  true,
}

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

	db, err = mysql.New(GlobalConfig.Mysql.Master, GlobalConfig.Mysql.Slave)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//为客户端提供的http服务
	var c service.Config = service.Config{GlobalConfig.Address.String(), GlobalConfig.Log.Dir, GlobalConfig.Log.Level, getEnv, isValidUser}
	server, err := service.New(&c)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if e := server.AddModule("filter", &FilterModule{}); e != nil {
		fmt.Println(e.Error())
		return
	}
	fmt.Println(server.StartService().Error())
}
