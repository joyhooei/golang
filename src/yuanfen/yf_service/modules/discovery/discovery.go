package discovery

import (
	"yf_pkg/cachedb"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/discovery"
	"yuanfen/yf_service/cls/data_model/tag"
)

type DiscoveryModule struct {
	log     *log.MLogger
	mdb     *mysql.MysqlDB
	rdb     *redis.RedisPool
	cache   *redis.RedisPool
	cachedb *cachedb.CacheDB
}

func (sm *DiscoveryModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	sm.cachedb = env.ModuleEnv.(*cls.CustomEnv).CacheDB

	return
}

/*
SecAdjacent搜索附近的人

URI: s/discovery/Adjacent

参数:
		{
			"gender":1, //[opt]性别，不填或-1表示异性,0表示不限性别,1-男性，2-女性
			"lat":13.22,	//[opt]用户所在纬度，不填则由服务器确定位置
			"lng":13.22,	//[opt]用户所在经度，不填则由服务器确定位置
			"building":"50f7d1461e9309c472210b6c",	//[opt]筛选某栋建筑的用户，不填表示不限
			"cur":1,	//页码
			"ps":10,	//每页条数
			"refresh":false, //[opt]是否强制刷新，不填表示不刷新
		}
返回值:

{
	"res": {
		"buildings":[
			{
				"id":"xxfdf",
				"name":"五彩城",
				"address":"信息路2-1号国际创业园1号楼1-4F"
				"lat":40.052384,
				"lng":116.312202
			}
		],
		"users": {
			"list": [
			{
				"uid": 1008602,
				"nickname": "姐姐",
				"age": 22,
				"gender":1,
				"height": 170,
				"avatar": "http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg",
				"photos":[	//形象照列表
					{"albumid":123,"pic":"http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg"},
					{"albumid":123,"pic":"http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg"}
				],
				"building":"五彩城",	//工作地点
				"aboutme":"想找男朋友",	//交友寄语
				"online_timeout": "2015-07-31T16:40:58+08:00",
				"lat": 40.03950500488281,
				"lng": 117.35199737548828,
				"distence": 89948.61444039512,	//距离（米）
			}
			],
			"pages": {
				"cur": 1,
				"total": 1,
				"ps": 2,
				"pn": 1
			}
		}
	},
	"status": "ok",
	"tm": 1438489368
}

*/
func (sm *DiscoveryModule) SecAdjacent(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var gender, cur, ps int
	var lat, lng float64
	var building string
	var refresh bool
	if err := req.Parse("cur", &cur, "ps", &ps); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("building", &building, "", "gender", &gender, -1, "refresh", &refresh, false, "lat", &lat, common.LAT_NO_VALUE, "lng", &lng, common.LNG_NO_VALUE); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, pages, buildings, err := discovery.AdjacentUsers(gender, building, req.Uid, lat, lng, cur, ps, refresh)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	users := make(map[string]interface{})
	users["list"] = list
	users["pages"] = pages
	res["users"] = users
	res["buildings"] = buildings
	result["res"] = res
	return
}

/*
SecSearch根据一定的过滤条件搜索附近的用户，如果所有的可选项都不填则采用智能搜索策略。

URI: s/discovery/Search

参数:
		{
			"gender":1, //[opt]性别，不填或-1表示异性,0表示不限性别,1-男性，2-女性
			"province":"湖南",	//[opt]所在省份，不填表示用户所在省
			"min_age":0,	//[opt]年龄下限，不填表示不限
			"max_age":999,	//[opt]年龄上限，不填表示不限
			"min_height":0,	//[opt]身高下限，不填表示不限
			"max_height":999,	//[opt]身高上限，不填表示不限
			"lat":13.22,	//[opt]用户所在纬度，不填则由服务器确定位置
			"lng":13.22,	//[opt]用户所在经度，不填则由服务器确定位置
			"edu":3, //[opt]学历，不填或0表示不限
			"homeprovince":"陕西", //[opt]家乡，不填表示不限
			"cur":1,	//页码
			"ps":10,	//每页条数
			"refresh":false, //[opt]是否强制刷新，不填表示不刷新
		}
返回值:

{
	"res": {
		"users": {
			"list": [
			{
				"uid": 1008602,
				"nickname": "姐姐",
				"age": 22,
				"avatar": "http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg",
				"height": 170,
				"city": "海淀区",
				"province": "北京市",
				"job":"UI设计师",	//职业
				"aboutme":"想找男朋友",	//交友寄语
				"online_timeout": "2015-07-31T16:40:58+08:00",
			}
			],
			"pages": {
				"cur": 1,
				"total": 1,
				"ps": 2,
				"pn": 1
			}
		}
	},
	"status": "ok",
	"tm": 1438489368
}

*/
func (sm *DiscoveryModule) SecSearch(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var province, homeprovince string
	var minEdu, minAge, maxAge, minHeight, maxHeight int
	var gender, cur, ps int
	var lat, lng float64
	var refresh bool
	if err := req.ParseOpt("gender", &gender, -1, "refresh", &refresh, false, "min_age", &minAge, 0, "max_age", &maxAge, common.MAX_AGE, "min_height", &minHeight, 0, "max_height", &maxHeight, common.MAX_HEIGHT, "province", &province, "", "edu", &minEdu, 0, "homeprovince", &homeprovince, ""); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("lat", &lat, common.LAT_NO_VALUE, "lng", &lng, common.LNG_NO_VALUE, "cur", &cur, 1, "ps", &ps, 10); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, pages, err := discovery.Search(req.Uid, gender, minAge, maxAge, minHeight, maxHeight, province, minEdu, homeprovince, cur, ps, refresh)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	users := make(map[string]interface{})
	users["list"] = list
	users["pages"] = pages
	res["users"] = users
	result["res"] = res
	return
}

/*
SecSearchByUsername根据用户名、uid、昵称搜索。
先匹配用户名和uid，如果没找到，再匹配昵称。
都是精确搜索。

URI: s/discovery/SearchByUsername

参数:
		{
			"username":"jiatao",	//用户名、uid或昵称
			"cur":1,	//页码
			"ps":10,	//每页条数
		}
返回值:
与SecSearch接口相同

*/
func (sm *DiscoveryModule) SecSearchByUsername(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var username string
	var cur, ps int
	if err := req.Parse("username", &username); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("cur", &cur, 1, "ps", &ps, 10); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, err := discovery.SearchByUsername(username, cur, ps)
	if err != nil {
		return err
	}
	pages := utils.PageInfo(100, cur, ps)
	res := make(map[string]interface{})
	users := make(map[string]interface{})
	users["list"] = list
	users["pages"] = pages
	res["users"] = users
	result["res"] = res
	return
}

/*
SecSearchByUid根据uid获取用户信息。

URI: s/discovery/SearchByUid

参数:
		{
			"uid":123
		}
返回值:

{
	"res": {
		"uid": 1008602,
		"nickname": "姐姐",
		"age": 22,
		"avatar": "http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg",
		"height": 170,
		"grade": 1,
		"city": "海淀区",
		"province": "北京市",
		"star": 1,
		"online": 0,
		"income": 0,
		"online_timeout": "2015-07-31T16:40:58+08:00",
		"status": {
			"stype": "in_topic",
			"id": 370,
			"show": "在话题室",
			"extra": ""
		},
		"lat": 40.03950500488281,
		"lng": 117.35199737548828,
		"distence": 89948.61444039512,	//距离（米）
		"birthday": "1993-07-06T00:00:00+08:00",
		"avatarLevel": 0,
		"certify_phone": 0,
		"certify_video": 0,
		"certify_idcard": 0,
		"certify_level": 0,
		"score": 0,
		"score_distence": 0,
		"score_age": 0,
		"score_online": 0,
		"score_tag": 0,
		"score_head": 0
	},
	"status": "ok",
	"tm": 1438489368
}

*/
func (sm *DiscoveryModule) SecSearchByUid(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	if err := req.Parse("uid", &uid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	user, err := discovery.SearchByUid(uid)
	if err != nil {
		return err
	}
	result["res"] = user
	return
}

func (sm *DiscoveryModule) SecTags(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	var tType string
	if err := req.Parse("type", &tType); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("cur", &cur, 1, "ps", &ps, 10); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, total, err := tag.GetTags(req.Uid, tType, cur, ps)
	if err != nil {
		return err
	}
	pages := utils.PageInfo(total, cur, ps)
	res := make(map[string]interface{})
	tags := make(map[string]interface{})
	tags["list"] = list
	tags["pages"] = pages
	res["tags"] = tags
	result["res"] = res
	return
}

/*
SecRecommend推荐匹配用户，每天最多推荐discover.MAX_RECOMMEND_NUM个。
如果名额已经用完，则返回的users数组长度为0。

URI: s/discovery/Recommend

参数:
		{
			"lat":13.22,	//[opt]用户所在纬度，不填则由服务器确定位置
			"lng":13.22,	//[opt]用户所在经度，不填则由服务器确定位置
		}
返回值:

	{
		"res": {
			"users": [
			{
				"uid": 1008602,
				"nickname": "姐姐",
				"age": 22,
				"avatar": "http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg",
				"photos":[	//形象照列表
					{"albumid":123,"pic":"http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg"},
					{"albumid":123,"pic":"http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg"}
				],
				"height": 170,
				"job":"UI设计师",	//职业
				"workunit":"炬鑫网络",	//工作单位
				"follow":false, //是否已标记
				"reason":{	//推荐理由
					"type":1,	//推荐类型，0-工作地点相近，1-同乡，2-校友，3-符合择友要求，4-同行，5-感兴趣
					"text":"你们是同行"	//理由描述
				}
			}
			]
			"left":23,	//剩余的推荐名额
			"is_cache":false //是否是缓存数据
		},
		"status": "ok",
		"tm": 1438489368
	}

*/
func (sm *DiscoveryModule) SecRecommend(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var lat, lng float64
	if err := req.ParseOpt("lat", &lat, common.LAT_NO_VALUE, "lng", &lng, common.LNG_NO_VALUE); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	users, left, isCache, err := discovery.Recommend(req.Uid, lat, lng)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	res["users"] = users
	res["left"] = left
	res["is_cache"] = isCache
	result["res"] = res
	return
}

/*
SecRecommendStat推荐统计

URI: s/discovery/RecommendStat

参数:
		{
			"lat":13.22,	//[opt]用户所在纬度，不填则由服务器确定位置
			"lng":13.22,	//[opt]用户所在经度，不填则由服务器确定位置
		}
返回值:

	{
		"res": {
			"adj_users": 27,	//附近的人数
			"city": "昌平",		//所在市
			"province": "北京",	//所在省
			"trade": "技术信息",//所属行业
			"trade_users": 64	//同行人数
			"building": [{
				"info": {
					"id": "216ab52014052d4f02294f0c",
					"name": "泰华龙旗大厦",
					"address": "北京市昌平区黄平路19号",
					"lat": 40.07167,
					"lng": 116.353582,
					"distence": 0.20612744866452165
				},
				"users": 1
			}]
		},
		"status": "ok",
		"tm": 1438489368
	}

*/
func (sm *DiscoveryModule) SecRecommendStat(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var lat, lng float64
	if err := req.ParseOpt("lat", &lat, common.LAT_NO_VALUE, "lng", &lng, common.LNG_NO_VALUE); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	data, err := discovery.RecommendStat(req.Uid, lat, lng)
	if err != nil {
		return err
	}
	result["res"] = data
	return
}
