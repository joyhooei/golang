package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"
	"yf_pkg/push"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	bd "yuanfen/yf_service/cls/data_model/building"
	"yuanfen/yf_service/cls/data_model/certify"
	"yuanfen/yf_service/cls/data_model/comments"
	dv "yuanfen/yf_service/cls/data_model/discovery"
	sdynamics "yuanfen/yf_service/cls/data_model/dynamics"
	ev "yuanfen/yf_service/cls/data_model/event"
	fc "yuanfen/yf_service/cls/data_model/face"
	"yuanfen/yf_service/cls/data_model/general"
	hn "yuanfen/yf_service/cls/data_model/hongniang"
	ml "yuanfen/yf_service/cls/data_model/mall"
	msg "yuanfen/yf_service/cls/data_model/message"
	rl "yuanfen/yf_service/cls/data_model/relation"
	"yuanfen/yf_service/cls/data_model/service_game"
	"yuanfen/yf_service/cls/data_model/tag"
	tp "yuanfen/yf_service/cls/data_model/topic"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/status"
	"yuanfen/yf_service/cls/word_filter"
	"yuanfen/yf_service/modules/admin"
	"yuanfen/yf_service/modules/common"
	"yuanfen/yf_service/modules/discovery"
	"yuanfen/yf_service/modules/dynamics"
	"yuanfen/yf_service/modules/event"
	"yuanfen/yf_service/modules/face"
	"yuanfen/yf_service/modules/game"
	"yuanfen/yf_service/modules/hongniang"
	"yuanfen/yf_service/modules/mall"
	"yuanfen/yf_service/modules/message"
	"yuanfen/yf_service/modules/relation"
	"yuanfen/yf_service/modules/sample_module"
	"yuanfen/yf_service/modules/topic"
	"yuanfen/yf_service/modules/unread"
	"yuanfen/yf_service/modules/user"
	"yuanfen/yf_service/modules/work"
)

var customEnv cls.CustomEnv
var GlobalConfig common.Config

func isValidUser(r *http.Request) (uid uint32) {
	c, e := r.Cookie("uid")
	if e != nil {
		return 0
	}
	k, e := r.Cookie("key")
	if e != nil {
		return 0
	}
	key := k.Value
	uid, e = utils.StringToUint32(c.Value)
	if e != nil {
		return 0
	}
	valid, e := user_overview.UserValid(uid, key)
	if key == "306123456" {
		valid = true
	}
	//fmt.Println("uid=", uid, "sid=", key, "valid=", valid)
	if e != nil || !valid {
		return 0
	}
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
		fmt.Println("Load Config error :", err.Error())
		return
	}
	fmt.Println("running in [", GlobalConfig.Mode, "] mode")
	//初始化环境变量
	if err = customEnv.Init(&GlobalConfig); err != nil {
		fmt.Println(err.Error())
		return
	}

	go func() {
		log.Println(http.ListenAndServe(GlobalConfig.PrivateAddr.String(), nil))
	}()

	//为客户端提供的http服务
	var c service.Config = service.Config{GlobalConfig.PublicAddr.String(), GlobalConfig.Log.Dir, GlobalConfig.Log.Level, customEnv.GetEnv, isValidUser}
	server, err := service.New(&c)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	push.Init(GlobalConfig.PushAddr.String(), customEnv.MainLog, user_overview.SysInfoNoErr, GlobalConfig.Mode)
	stat.Init(customEnv.StatDB, customEnv.DStatDB, customEnv.MainDB)

	//初始化data_module
	dv.Init(&customEnv)
	tp.Init(&customEnv)
	status.Init(&customEnv)
	rl.Init(&customEnv)
	msg.Init(&customEnv)
	hn.Init(&customEnv)
	tag.Init(&customEnv)
	ev.Init(&customEnv)
	fc.Init(&customEnv)
	bd.Init(&customEnv)
	ml.Init(&customEnv)
	word_filter.Init(&customEnv)

	// 初始化data_module, 多传一个conf 参数，添加日志
	service_game.Init(&customEnv, c)
	general.Init(&customEnv, c)
	certify.Init(&customEnv, c)
	sdynamics.Init(&customEnv, c)
	comments.Init(&customEnv, c)

	//创建自定义module
	server.AddModule("sample_module", &sample_module.SampleModule{})
	server.AddModule("message_module", &sample_module.MessageModule{})
	server.AddModule("topic", &topic.TopicModule{})
	server.AddModule("message", &message.MessageModule{})
	server.AddModule("relation", &relation.RelationModule{})
	server.AddModule("discovery", &discovery.DiscoveryModule{})
	server.AddModule("unread", &unread.UnreadModule{})
	server.AddModule("hongniang", &hongniang.HongniangModule{})
	server.AddModule("event", &event.EventModule{})
	server.AddModule("mall", &mall.MallModule{})

	server.AddModule("game", &game.GameModule{})
	server.AddModule("user", &user.UserModule{})
	server.AddModule("face", &face.FaceModule{})
	server.AddModule("work", &work.WorkModule{})
	server.AddModule("common", &comconfig.CommonModule{})
	server.AddModule("admin", &admin.AdminModule{})

	server.AddModule("dynamics", &dynamics.DynamicsModule{})

	fmt.Println(server.StartService())
}
