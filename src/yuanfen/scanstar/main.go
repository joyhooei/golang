package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
	// "yf_pkg/log"
	// "yf_pkg/service"
	// "yf_pkg/utils"
	"yuanfen/scanstar/cls"
	"yuanfen/scanstar/cls/common"
	"yuanfen/scanstar/cls/scanuser"
)

var customEnv cls.CustomEnv
var GlobalConfig common.Config

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s [config]\n", os.Args[0])
		return
	}
	err := GlobalConfig.Load(os.Args[1])
	if err != nil {
		fmt.Println("Load Config error :", err.Error())
		return
	}
	if err = customEnv.Init(&GlobalConfig); err != nil {
		fmt.Println(err.Error())
		return
	}
	rand.Seed(time.Now().UnixNano())
	fmt.Println("scanuser.Init")
	scanuser.Scan(customEnv.StatDB, customEnv.MainDB, customEnv.MainLog)
}
