package work

import (
	//"encoding/json"

	"fmt"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/data_model/dynamics"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

type WorkModule struct {
	log           *log.MLogger
	mdb           *mysql.MysqlDB
	rdb           *redis.RedisPool
	cache         *redis.RedisPool
	webServiceUrl string
}

func (sm *WorkModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	sm.webServiceUrl = env.ModuleEnv.(*cls.CustomEnv).WebServiceUrl

	fmt.Println("--webServiceUrl- -：", sm.webServiceUrl)
	return
}

/*
SecShareApp 获取分享链接

URI: s/work/ShareApp

参数
	{

			"type":"sharearticles",		//分享类型 目前有 sharearticles sharedynamic sharepuzzle
			"dataid"1011111,			//分享内容ID.type为sharearticles时为文章ID，type为sharedynamic时为动态ID，type为sharepuzzle时为拼图游戏ID
	}

返回值
	{
			"status":"ok"
			"res":
			{
				"info":"分享秋千",		//分享描述
				"url":"http://",		//短链接
				"imgurl":"",//图片
				"title":"邀请好友来秋千"//标题
			}
	}
*/
func (sm *WorkModule) SecShareApp(req *service.HttpRequest, result map[string]interface{}) (e error) {
	csharearticlesbase := sm.webServiceUrl + "/share/article?uid=%v&id=%v"
	csharedynamicbase := sm.webServiceUrl + "/share/dynamic?uid=%v&id=%v"
	csharepuzzlebase := sm.webServiceUrl + "/share/puzzle?uid=%v&id=%v"
	var tp string
	var dataid uint32
	if err := req.Parse("type", &tp, "dataid", &dataid); err != nil {
		return err
	}
	user, e := user_overview.GetUserObject(req.Uid)
	if e != nil {
		return
	}
	if user.Avatar == "" {
		user.Avatar = "http://image1.yuanfenba.net/oss/other/log144.png"
	}
	res := make(map[string]interface{})
	switch tp {
	case "sharearticles":
		res["info"] = "我在秋千看到了这边文章，很有意思!"
		u, err := utils.UrlToShort(fmt.Sprintf(csharearticlesbase, req.Uid, dataid))
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, err.Error())
		}
		res["url"] = u
		res["title"] = "我在秋千看到了这边文章，很有意思!"
		res["imgurl"] = general.GetImgSizeUrl(user.Avatar, 100)
	case "sharedynamic":
		res["info"] = "我在秋千发布了新评论,快来看看吧"
		u, err := utils.UrlToShort(fmt.Sprintf(csharedynamicbase, req.Uid, dataid))
		if err != nil {
			return err
		}
		res["url"] = u
		res["title"] = "我在秋千发布了新评论,快来看看吧"
		res["imgurl"] = general.GetImgSizeUrl(user.Avatar, 100)
	case "sharepuzzle":
		// 查询自己是否玩过这个游戏
		c, _ := dynamics.CheckIsJoinDynamicGame(dataid, req.Uid)
		var info string
		info = "这个拼图也太难吧，快来帮帮我吧！"
		if c > 0 {
			info = "这么难都拼成功了，炫耀一下"
		}
		u, err := utils.UrlToShort(fmt.Sprintf(csharepuzzlebase, req.Uid, dataid))
		if err != nil {
			return err
		}
		res["url"] = u
		res["title"] = info
		res["info"] = info
		res["imgurl"] = general.GetImgSizeUrl(user.Avatar, 100)
	default:
		return service.NewError(service.ERR_INVALID_PARAM, "未知的类型", "未知的类型")
	}
	result["res"] = res
	return
}

func (sm *WorkModule) Test(req *service.HttpRequest, result map[string]interface{}) (e error) {
	return
}
