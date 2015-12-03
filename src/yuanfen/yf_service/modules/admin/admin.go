package admin

import (
	"errors"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/data_model/discovery"
)

// 一些不涉及到业务逻辑的对外接口
type AdminModule struct {
	log    *log.MLogger
	mdb    *mysql.MysqlDB
	rdb    *redis.RedisPool
	cache  *redis.RedisPool
	statDb *mysql.MysqlDB
	mode   string
}

func (co *AdminModule) Init(env *service.Env) (err error) {
	co.log = env.Log
	co.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	co.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	co.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	co.mode = env.ModuleEnv.(*cls.CustomEnv).Mode
	co.statDb = env.ModuleEnv.(*cls.CustomEnv).StatDB
	return
}

/*
SetRegIP修改IP注册限制数，如果IP不填，则使用请求发送者IP。

URI: admin/SetRegIP?ip=111.222.333.444&n=100

参数:
	ip: 要修改的IP
	n: 允许注册的次数
返回值:
	{
		"status": "ok",
		"tm": 1438489368
	}

*/
func (co *AdminModule) SetRegIP(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if co.mode != "test" {
		return errors.New("can only run in test mode")
	}
	ip := req.GetParam("ip")
	if ip == "" {
		ip = req.IP()
	}
	n, e := utils.ToInt(req.GetParam("n"))
	if e != nil {
		return e
	}
	co.cache.SetEx(redis_db.CACHE_REGIP, ip, 86400, n)
	result["ip"] = req.IP()
	result["n"] = n
	return
}

/*
ClearRecommend清空用户的已推荐列表

URI: Admin/ClearRecommend?uid=123

参数:
	uid: 123	//用户ID
返回值:
	{
		"status": "ok",
		"tm": 1438489368
	}

*/
func (co *AdminModule) ClearRecommend(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if co.mode != "test" {
		return errors.New("can only run in test mode")
	}
	uidStr := req.GetParam("uid")
	uid, e := utils.ToUint32(uidStr)
	if e != nil {
		return e
	}
	err := discovery.ClearRecommend(uid)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	result["res"] = res
	return
}
