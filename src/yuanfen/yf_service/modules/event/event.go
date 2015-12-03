package event

import (
	"yf_pkg/cachedb"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/data_model/event"
)

type EventModule struct {
	log     *log.MLogger
	mdb     *mysql.MysqlDB
	rdb     *redis.RedisPool
	cache   *redis.RedisPool
	cachedb *cachedb.CacheDB
}

func (sm *EventModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	sm.cachedb = env.ModuleEnv.(*cls.CustomEnv).CacheDB

	return
}

func (sm *EventModule) SecDetail(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	if err := req.Parse("id", &id); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	detail, err := event.Detail(id)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	result["res"] = detail
	return
}
func (sm *EventModule) SecList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	if err := req.Parse("cur", &cur, "ps", &ps); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, total, err := event.List(cur, ps)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	pages := utils.PageInfo(total, cur, ps)
	res := make(map[string]interface{})
	events := make(map[string]interface{})
	events["list"] = list
	events["pages"] = pages
	res["events"] = events
	result["res"] = res
	return
}

func (sm *EventModule) SecFocusList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tp string
	if err := req.Parse("type", &tp); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, err := event.FocusList(tp)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["focus"] = list
	result["res"] = res
	return
}
