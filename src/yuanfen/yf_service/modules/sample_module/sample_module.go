package sample_module

import (
	"time"
	"yf_pkg/cachedb"
	"yf_pkg/lbs/baidu"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/push"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/message"
	"yuanfen/yf_service/cls/unread"
)

type SampleModule struct {
	log     *log.MLogger
	mdb     *mysql.MysqlDB
	rdb     *redis.RedisPool
	cachedb *cachedb.CacheDB
}

func (sm *SampleModule) UnreadNum(uid uint32, key string, from time.Time) (uint32, string) {
	return uint32(from.Unix() % 100), "haha"
}

func (sm *SampleModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cachedb = env.ModuleEnv.(*cls.CustomEnv).CacheDB

	unread.Register("test", sm.UnreadNum)
	return
}

func (sm *SampleModule) Hello(req *service.HttpRequest, res map[string]interface{}) (e error) {
	message.SendMessage(message.RECOMMEND_CHANGE, message.RecommendChange{5001849}, map[string]interface{}{})
	_, e = user_overview.GetUserObject(1008603)
	if e != nil {
		return
	}
	res["result"] = " World!"
	if e = push.AddTag(1, "hello"); e != nil {
		return e
	}
	if e = push.AddTag(1, "world"); e != nil {
		return e
	}
	if tags, e := push.GetUserTags(1); e != nil {
		return e
	} else {
		res[common.UTAG_KEY] = tags
	}
	return
}
func (sm *SampleModule) GetCityByGPS(req *service.HttpRequest, res map[string]interface{}) (e error) {
	var lat, lng float64
	if err := req.Parse("lat", &lat, "lng", &lng); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	city, province, e := baidu.GetCityByGPS(utils.Coordinate{lat, lng})
	res["city"] = city
	res["province"] = province
	return
}

func (sm *SampleModule) GetCityByIP(req *service.HttpRequest, res map[string]interface{}) (e error) {
	var ip string
	if err := req.Parse("ip", &ip); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	city, province, e := baidu.GetCityByIP(ip)
	res["city"] = city
	res["province"] = province
	return
}

func (sm *SampleModule) Alert(req *service.HttpRequest, res map[string]interface{}) (e error) {
	var module, content string
	if err := req.Parse("module", &module, "content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	general.Alert(module, content)
	return
}
func (sm *SampleModule) CheckIdCard(req *service.HttpRequest, res map[string]interface{}) (e error) {
	var id, name string
	if err := req.Parse("id", &id, "name", &name); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	match, _, e := general.IsMatch(id, name)
	if e != nil {
		return e
	}
	res["isMatch"] = match
	return
}
func (sm *SampleModule) Check(req *service.HttpRequest, res map[string]interface{}) (e error) {
	//keepalive检查服务是否正常
	return
}
func (sm *SampleModule) GetGPSByCity(req *service.HttpRequest, res map[string]interface{}) (e error) {
	var city, province string
	if err := req.Parse("city", &city, "province", &province); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	lat, lng, e := general.GetGPSByCity(city, province)
	if e != nil {
		return e
	}
	res["lat"] = lat
	res["lng"] = lng
	return
}

func (sm *SampleModule) SearchPlace(req *service.HttpRequest, res map[string]interface{}) (e error) {
	var radius, cur, ps int
	var lat, lng float64
	var keywords []string
	if err := req.Parse("cur", &cur, "ps", &ps, "lat", &lat, "lng", &lng, "radius", &radius, "keywords", &keywords); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	places, total, e := baidu.SearchPlace(lat, lng, radius, cur, ps, true, keywords...)
	if e != nil {
		return e
	}
	pages := utils.PageInfo(total, cur, ps)
	res["places"] = places
	res["page"] = pages
	return
}

func (sm *SampleModule) SuggestionPlace(req *service.HttpRequest, res map[string]interface{}) (e error) {
	var lat, lng float64
	var keyword, region string
	if err := req.Parse("keyword", &keyword, "region", &region); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("lat", &lat, common.LAT_NO_VALUE, "lng", &lng, common.LNG_NO_VALUE); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	var suggestions interface{}
	if lat == common.LAT_NO_VALUE {
		suggestions, e = baidu.SuggestionPlace(region, keyword)
	} else {
		suggestions, e = baidu.SuggestionPlace(region, keyword, lat, lng)
	}
	if e != nil {
		return e
	}
	res["suggestion"] = suggestions
	return
}
