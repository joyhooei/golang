package mall

import (
	"yf_pkg/cachedb"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/data_model/mall"
)

type MallModule struct {
	log     *log.MLogger
	mdb     *mysql.MysqlDB
	rdb     *redis.RedisPool
	cache   *redis.RedisPool
	cachedb *cachedb.CacheDB
}

func (sm *MallModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	sm.cachedb = env.ModuleEnv.(*cls.CustomEnv).CacheDB

	return
}

/*
SecBuy：购买商品

URI: s/mall/Buy

参数:
		{
			"id":1,	//商品ID
		}
返回值:

	{
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *MallModule) SecBuy(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	if err := req.Parse("id", &id); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	err := mall.Buy(req.Uid, id)
	if err != nil {
		return err
	}
	res := map[string]interface{}{}
	result["res"] = res
	return
}

/*
SecList：获取用户的商品购买记录

URI: s/mall/List

参数:
		{
			"cur":1,	//页码
			"ps":10,	//每页条数
		}
返回值:

	{
		"res": {
			"items": {
				"list": [
					{
						"id": 1,	//交易ID
						"title": "星巴克88元会员卡",
						"pic": "http://image1.yuanfenba.net/uploads/oss/photo/201507/01/17163176935.jpg",
						"url": "http://test.a.imswing.cn:10080/mall/detail?id=1",
						"tm": "2015-07-01T18:22:31+08:00"	//购买时间
					}
				],
				"pages": {
					"cur": 1,
					"total": 3,
					"ps": 2,
					"pn": 2
				}
			}
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *MallModule) SecList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	if err := req.Parse("cur", &cur, "ps", &ps); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, total, err := mall.List(req.Uid, cur, ps)
	if err != nil {
		return err
	}
	pages := utils.PageInfo(total, cur, ps)
	res := make(map[string]interface{})
	items := make(map[string]interface{})
	items["list"] = list
	items["pages"] = pages
	res["items"] = items
	result["res"] = res
	return
}
