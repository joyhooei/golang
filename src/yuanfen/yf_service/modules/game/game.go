package game

import (
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/data_model/service_game"
)

type GameModule struct {
	log   *log.MLogger
	mdb   *mysql.MysqlDB
	rdb   *redis.RedisPool
	cache *redis.RedisPool
}

func (gm *GameModule) Init(env *service.Env) (err error) {
	gm.log = env.Log
	gm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	gm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	gm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	return
}

/*
获取游戏列表

URL：game/GameList

返回值：
	{
		"res": [
		{
			"appid": "qiuqian_sanxiao",  // 游戏appid
			"class": "com.wofuns.mem1942.AppActivity", // 包名
			"img": "http://image2.yuanfenba.net/oss/other/game_icon_1.png", // 游戏图片
			"info": "类型:男女组队游戏",  // 游戏描述
			"name": "空中大作战",         // 游戏名称
			"pack": "com.wofuns.mem1942", // package
			"size": "18.46M",			// 大小
			"url": "http://yfb.oss-cn-beijing.aliyuncs.com/oss%2Fapk%2Fmem1942-07-29a.apk" // 下载地址
		}
		],
		"status": "ok",
		"tm": 1444719598
	}
*/
func (pm *GameModule) GameList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	game_list, err := service_game.GetGameDataList()
	if err != nil {
		return err
	}
	res := make([]map[string]interface{}, 0, len(game_list))
	for _, g := range game_list {
		info := make(map[string]interface{})
		info["appid"] = g.AppId
		info["class"] = g.Class
		info["img"] = g.Img
		info["info"] = g.Info
		info["name"] = g.Name
		info["pack"] = g.Pack
		info["size"] = g.Size
		info["url"] = g.Url
		res = append(res, info)
	}
	/*	unread_m := map[string]interface{}{"num": 0, "show": ""}
		result["unread"] = map[string]interface{}{common.UNREAD_GAME: unread_m}
		// 更新未读消息
		uidc, _ := req.Cookie("uid")
		if uidc != nil {
			uid, _ := utils.StringToUint32(uidc.Value)
			if uid > 0 {
				unread.UpdateReadTime(uid, common.UNREAD_GAME)
			}
		}
	*/
	result["res"] = res
	return
}

/*
进入游戏大厅

URL：/s/game/Entry

返回值：

	{
		"res": {
			"tag": "game_1" // 用户tag
		},
		"status": "ok",
		"tm": 1444705701
	}
*/
func (pm *GameModule) SecEntry(req *service.HttpRequest, result map[string]interface{}) (e error) {
	tag, e := service_game.GetUserGameTag(req.Uid, 300)
	if e != nil {
		return
	}
	res := map[string]interface{}{"tag": tag}
	result["res"] = res
	return
}

/*
退出游戏大厅

URL：/s/game/Exit

返回值：

	{
		"status": "ok",
		"tm": 1444705701
	}
*/
func (pm *GameModule) SecExit(req *service.HttpRequest, result map[string]interface{}) (e error) {
	// 更具uid获取该用户所有的tag，并选出第最个tag
	go service_game.ExitDeleteTag(req.Uid)
	return
}

/*
获取推荐游戏

URL：/game/Recommend

返回值：
	{
		"res": [  // 字段内容见http://120.131.64.91:8182/pkg/yuanfen/yf_service/modules/game/#GameModule.GameList
		{
			"appid": "qiuqian_sanxiao",
			"class": "com.wofuns.mem1942.AppActivity",
			"img": "http://image2.yuanfenba.net/oss/other/game_icon_1.png",
			"info": "类型:男女组队游戏",
			"isHot": 1,  // 是否为热门游戏(1 是，0 否)
			"isNew": 0,  // 是否为新游戏 （1 是，0 否）
			"name": "空中大作战",
			"pack": "com.wofuns.mem1942",
			"size": "18.46M",
			"url": "http://yfb.oss-cn-beijing.aliyuncs.com/oss%2Fapk%2Fmem1942-07-29a.apk"
		}
		],
		"status": "ok",
		"tm": 1444724537
	}
*/
func (pm *GameModule) Recommend(req *service.HttpRequest, result map[string]interface{}) (e error) {
	gl, e := service_game.GetGameDataList()
	if e != nil {
		return
	}
	res := make([]map[string]interface{}, 0, 5)
	for _, g := range gl {
		if g.IsHot == 1 || g.IsNew == 1 {
			info := make(map[string]interface{})
			info["appid"] = g.AppId
			info["class"] = g.Class
			info["img"] = g.Img
			info["info"] = g.Info
			info["name"] = g.Name
			info["pack"] = g.Pack
			info["size"] = g.Size
			info["url"] = g.Url
			info["isHot"] = g.IsHot
			info["isNew"] = g.IsNew
			res = append(res, info)
		}
	}
	result["res"] = res
	return
}
