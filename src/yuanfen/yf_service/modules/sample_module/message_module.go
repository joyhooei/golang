package sample_module

import (
	"fmt"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yuanfen/yf_service/cls"
)

type MessageModule struct {
	log *log.MLogger
	mdb *mysql.MysqlDB
	rdb *redis.RedisPool
}

func (sm *MessageModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	//message.RegisterCallback(message.SAMPLE, "sample_message", sm.Call)
	//message.RegisterNotification(message.SAMPLE, sm.Notify)
	return
}

func (sm *MessageModule) Hello(req *service.HttpRequest, res map[string]interface{}) (e error) {
	res["result"] = "World!"
	return
}
func (sm *MessageModule) Call(msgID int, data interface{}) (result interface{}) {
	req := data.(*service.HttpRequest)
	res := make(map[string]interface{})
	res["msgid"] = msgID
	res["body"] = req.Body
	return res
}
func (sm *MessageModule) Notify(msgID int, data interface{}) {
	req := data.(*service.HttpRequest)
	fmt.Println("msgid=", msgID, req.Body)
	return
}
