package unread

import (
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	ur "yuanfen/yf_service/cls/unread"
)

type UnreadModule struct {
	log   *log.MLogger
	mdb   *mysql.MysqlDB
	rdb   *redis.RedisPool
	cache *redis.RedisPool
}

func (sm *UnreadModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	ur.Init(sm.rdb)
	return
}

func (sm *UnreadModule) SecGetAllUnreadNum(req *service.HttpRequest, result map[string]interface{}) (e error) {
	values, e := ur.GetAllUnreadNum(req.Uid)
	if e != nil {
		return
	}
	res := make(map[string]interface{})
	res[common.UNREAD_KEY] = values
	result["res"] = res
	return
}
func (sm *UnreadModule) SecGetUnreadNum(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var keys []string
	if err := req.Parse("keys", &keys); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	values := make(map[string]interface{})
	for _, key := range keys {
		values[key] = nil
	}
	ur.GetUnreadNum(req.Uid, values)
	res := make(map[string]interface{})
	res[common.UNREAD_KEY] = values
	result["res"] = res
	return
}

func (sm *UnreadModule) SecUpdateReadTime(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var keys []string
	if err := req.Parse("keys", &keys); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	for _, key := range keys {
		if err := ur.UpdateReadTime(req.Uid, key); err != nil {
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
	}
	return
}
