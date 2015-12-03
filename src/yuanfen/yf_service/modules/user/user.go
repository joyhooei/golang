package user

import (
	"fmt"
	"strings"
	"time"
	"yf_pkg/cachedb"
	"yf_pkg/encrypt"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/award"
	"yuanfen/yf_service/cls/data_model/ban"
	"yuanfen/yf_service/cls/data_model/coin"
	"yuanfen/yf_service/cls/data_model/discovery"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/login"
	"yuanfen/yf_service/cls/data_model/relation"
	"yuanfen/yf_service/cls/data_model/relation/base"
	"yuanfen/yf_service/cls/data_model/tag"
	"yuanfen/yf_service/cls/data_model/topic"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/data_model/usercontrol"
	"yuanfen/yf_service/cls/message"
	"yuanfen/yf_service/cls/notify"
	"yuanfen/yf_service/cls/unread"
)

const (
	OTHER_MAX_PIC = 20
)

type UserModule struct {
	log     *log.MLogger
	mdb     *mysql.MysqlDB
	rdb     *redis.RedisPool
	cache   *redis.RedisPool
	cachedb *cachedb.CacheDB
	mode    string
}

func (sm *UserModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	sm.cachedb = env.ModuleEnv.(*cls.CustomEnv).CacheDB
	sm.mode = env.ModuleEnv.(*cls.CustomEnv).Mode
	coin.Init(env)
	award.Init(env)
	login.Init(env)
	ban.Init(env)
	user_overview.Init(env)
	usercontrol.Init(env)
	return
}

func (sm *UserModule) SetUPass(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	uname := req.GetParam("uname")
	var uid uint32
	err := sm.mdb.QueryRow("select uid from user_main where `username`=?", uname).Scan(&uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	upass := req.GetParam("upass")
	spass := encrypt.MD5Sum(upass)
	password := encrypt.MD5Sum(utils.ToString(uid) + spass)
	_, err = sm.mdb.Exec("update user_main set password=? where username=?", password, uname)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	result["res"] = uid
	return
}

/*
获取用户自己的信息 SecInfo

请求URL：s/user/Info

返回结果：

{
	"status": "ok",
	"res": {
 		"uid":21111,  //用户ID
		"phone": "15110001000", //绑定手机号 如果为空字符串则未绑定
		"avatarstatus":1,//头像状态，0为未审核，1为通过，2为拒绝,3为未上传
		"goldcoin":3331,//钻石余额
		"gender":1,	//性别,1为男性，2为女性, 0为未填写性别
		"age":20,	//年龄
		"nickname":"翅膀的鱼",//昵称
		"avatar":"photo/20141015/1109218597.png", //头像URL
		"star":7, 	//星座
		"height":170,//身高
		"job":"医生", 	//职业
		"edu":5,    //学历
		"interest":"足球,下棋",//兴趣爱好
		"tag":"标签",//性格标签
		"workarea":"清河新城", //工作地点
		"workunit":"XX公司", //工作单位
		"school":"人民大学", //学校
		"homeprovince":"清河新城", //家乡省
		"homecity":"清河新城", //家乡市
		"trade":"IT互联网",//从事行业
		"province":"湖南",//用户所在省
		"city":"湖南",//用户所在省
		"regtime"://注册时间 日期时间类型
		"certify_video":0 //视频认证 0 未认证，1 已认证
		"certify_phone":1 //手机认证 0 未认证，1 已认证
		"require":{//择偶要求
			"province":"北京市"//要求对方所在省 默认值为空 不限制为"不限"
			"city":"海淀区"		//要求对方所在市 默认值为空 不限制为"不限"
			"minage":25,		//年龄最小为 为0表示无最小限制  -1为默认值
			"maxage":25,		//年龄最大 为999表示无限制  -1为默认值
			"minheight":170,	//身高最低 0为不限制  -1为默认值
			"maxheight":0,		//身高最高 999为不限制 -1为默认值
			"minedu":1,			//学历最低 0为不限制  -1为默认值
			"needtag":"事业型"	//感兴趣的类型 列表
			"aboutme":"",//交友寄语
			"hardrequire":1//是否硬性要求 0非硬性 1 是硬性 2 不全是硬性 -1为默认值
		}
		"dynamicpic":{//最近动态图片
			"n":10,//动态数量
			"pics":["http://111.jpg","http://222.jpg"] //图片列表
		}
		"photolist":[ //照片列表
		    	{
	            "albumid": "857",  //图片ID
	            "albumname":"",  //名称
	            "pic":"photo/20141030/1041299985.jpg",  //图片URL
	            "picdesc":"",//图片描述
				"status":1	//状态，0为未审核，1为正常，2为冻结
	           },{...}
	           ]
		"dynamic_img":"xxx" // 动态背景图片

		"avatarlevel":1//头像等级-1未通过 0差 3默认 6好 9 优秀
		"following":111//我关注的
		"followed":21//关注我的
		"mustcomplete":"avatar,height,homeprovince,homecity,workarea,workunit,school,job,trade",//必填项列表
		"choosecomplete":"aboutme,interest,tag",//选填项列表
		"protect":{
			"canfind":1,	//允许附近的人找到我 1表示 不允许 0表示允许 0为默认值
			"chatremind":1,	//私聊提醒 1表示 不允许 0表示允许 0为默认值
			"stranger":1,	//陌生人新消息提醒 1表示 不允许 0表示允许 0为默认值
			"praise":1,		//被点赞提醒 1表示 不允许 0表示允许 0为默认值
			"commit":1,		//被评论提醒 1表示 不允许 0表示允许 0为默认值
			"msgnotring":1,  //消息无提示音,0为有提示音,1为关闭提示音(默认值)
			"msgnotshake":1, //消息无震动,0为有震动(默认值),1为关闭震动
			"nightring":1,   //晚上是否响铃震动,0半夜不响铃(默认值),1为半夜仍响铃
		}
	}
	 "video_info": {  // 视屏认证信息
		"isdo": true, // 是否已经执行认证
		"status": 0,  // 状态，0待处理. 1,通过,-1.已拒绝,-2.放弃
		"tm": "2015-08-17T15:01:07+08:00"
      }
	  "markMeNum":10   // 标记我的用户数量
}
*/
func (sm *UserModule) SecInfo(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	rmap, err := usercontrol.GetUserInfo(req.Uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["uid"], _ = utils.ToUint32(rmap["uid"])                //用户ID
	res["phone"] = rmap["phone"]                               //手机号
	res["avatarstatus"], _ = utils.ToInt(rmap["avatarstatus"]) //头像状态，0为未审核，1为通过，2为拒绝,3为未上传
	res["goldcoin"], _ = utils.ToInt(rmap["goldcoin"])         //金币余额
	igender, _ := utils.ToInt(rmap["gender"])
	res["gender"] = igender                    //性别
	res["earn"], _ = utils.ToInt(rmap["earn"]) //男性打工总收入

	if t, err := utils.ToTime(rmap["birthday"]); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "birthday 类型错误")
	} else {
		res["birthday"] = t
	}

	res["age"] = usercontrol.StringAge(rmap["birthday"])

	res["nickname"] = rmap["nickname"]             //昵称
	res["province"] = rmap["province"]             //省份
	res["city"] = rmap["city"]                     //城市
	res["avatar"] = rmap["avatar"]                 //头像URL
	res["height"], _ = utils.ToInt(rmap["height"]) //身高
	res["job"] = rmap["job"]                       //职业

	res["workarea"] = rmap["workarea"]         //工作地点
	res["workunit"] = rmap["workunit"]         //工作单位
	res["school"] = rmap["school"]             //学校
	res["homeprovince"] = rmap["homeprovince"] //家乡省
	res["homecity"] = rmap["homecity"]         //家乡市
	res["trade"] = rmap["trade"]               //从事行业
	res["edu"], _ = utils.ToInt(rmap["edu"])   //学历

	res["star"], _ = utils.ToInt(rmap["star"]) //星座  计算列
	// res["aboutme"] = rmap["aboutme"]           //个人签名

	res["interest"] = rmap["interest"] //爱好

	require := make(map[string]interface{})
	require["province"] = rmap["requireprovince"]
	require["city"] = rmap["requirecity"]
	require["minage"], _ = utils.ToInt(rmap["minage"])
	require["maxage"], _ = utils.ToInt(rmap["maxage"])
	require["minheight"], _ = utils.ToInt(rmap["minheight"])
	require["maxheight"], _ = utils.ToInt(rmap["maxheight"])
	require["minedu"], _ = utils.ToInt(rmap["minedu"])
	require["needtag"] = rmap["needtag"] //感兴趣的类型
	require["hardrequire"], _ = utils.ToInt(rmap["hardrequire"])

	require["aboutme"] = rmap["aboutme"]
	res["require"] = require

	res["tag"] = rmap["tag"]                                 //标签
	res["avatarlevel"], _ = utils.ToInt(rmap["avatarlevel"]) //头像级别

	res["following"], _ = relation.FollowingNum(req.Uid)      //我关注的数量
	res["followed"], _ = relation.FollowedNum(req.Uid)        //关注我的数量
	if t, err := utils.ToTime(rmap["reg_time"]); err != nil { //注册时间
		return service.NewError(service.ERR_INVALID_PARAM, "regtime 类型错误")
	} else {
		res["regtime"] = t
	}

	res["certify_video"], _ = utils.ToInt(rmap["certify_video"])
	res["certify_phone"], _ = utils.ToInt(rmap["phonestat"])

	sqlr, _, err := usercontrol.GetUserPicture(req.Uid, 0, 20)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	photolist := make([]interface{}, 0, 0)
	for _, v := range sqlr {
		item := make(map[string]interface{})
		if v["pic"] == rmap["avatar"] {
			item["albumid"], _ = utils.ToInt(v["albumid"]) //图片ID
			item["pic"] = v["pic"]                         //图片URL
			item["status"] = v["status"]                   //图片状态
			photolist = append(photolist, item)
		}
	}
	for _, v := range sqlr {
		item := make(map[string]interface{})
		if v["pic"] != rmap["avatar"] {
			item["albumid"], _ = utils.ToInt(v["albumid"]) //图片ID
			item["pic"] = v["pic"]                         //图片URL
			item["status"] = v["status"]                   //图片状态
			photolist = append(photolist, item)
		}
	}
	res["photolist"] = photolist
	if dyn, pics, err := user_overview.GetUserLastDynamicPic(req.Uid, true); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	} else {
		res["dynamicpic"] = map[string]interface{}{"n": dyn, "pics": pics}
	}

	// 获取视频认证状态数据
	if video_info, e := usercontrol.GetVideoStatus(req.Uid); e == nil {
		res["video_info"] = video_info
	}
	res["mustcomplete"] = common.USERCOMPLETE_LIST_MUST
	res["choosecomplete"] = common.USERCOMPLETE_LIST_CHOOSE

	res["dynamic_img"] = rmap["dynamic_img"] //动态背景图

	if info, err := usercontrol.GetUserProtect(req.Uid); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	} else {
		res["protect"] = info
	}

	// 获取标记我的用户数量
	markMeNum := 0
	if m, e := general.GetInterestedUids(req.Uid); e == nil {
		markMeNum = len(m)
	}
	res["markMeNum"] = markMeNum

	result["res"] = res
	//sm.log.Append(req.body, 1)
	return
}

/*
获取用户简略信息 SecUserSimple
请求URL：s/user/UserSimple
参数: {"uidlist":[11052,11053,11054]}
描述
返回结果：
{
	"status": "ok”,
	"msg": "success"
    "code": 0,
	"res“:{
			"list":[
				{
				"uid":1001,
				"isvip":1, // VIP状态 1为VIP
    			"grade":1,//等级
				"gender":1,	//性别
				"age":20,	//年龄
				"nickname":"翅膀的鱼",//昵称
				"province":"31", 省份
				"city":22, //城市
				"avatar":"photo/20141015/1109218597.png", //头像URL
				"height":170,//身高
				"aboutme":"是个怎样的人",//个人签名
				"stat":"封停状态",//用户是否被封 0为正常 5为封号
				},
					{ … }
   			]
}
*/
func (sm *UserModule) SecUserSimple(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	// sm.log.Append(fmt.Sprintf("Secinfo %v", req.Uid), log.NOTICE)
	uidlist := make([]uint32, 0, 0)
	if v, ok := req.Body["uidlist"]; !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "uidlist没找到")
	} else {
		var li []interface{}
		switch value := v.(type) {
		case []interface{}:
			li = value
		default:
			return service.NewError(service.ERR_INVALID_PARAM, "uidlist没找到")
		}
		for _, v := range li {
			if v2 := utils.GetComonFloat64ToUint32(v, 0); v2 != 0 {
				uidlist = append(uidlist, v2)
			}
		}
	}
	rmap, err := user_overview.GetUserObjects(uidlist...)

	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	list := make([]map[string]interface{}, 0, 0)
	for k, v := range rmap {
		mp := make(map[string]interface{})
		mp["uid"] = k               //用户ID
		mp["isvip"] = v.Isvip       //VIP状态 1为VIP
		mp["grade"] = v.Grade       //等级
		mp["gender"] = v.Gender     //性别
		mp["age"] = v.Age           //年龄
		mp["nickname"] = v.Nickname //昵称
		mp["province"] = v.Province //省份
		mp["city"] = v.City         //城市
		mp["avatar"] = v.Avatar     //头像URL
		mp["height"] = v.Height     //身高
		mp["aboutme"] = v.Aboutme   //内心独白
		mp["stat"] = v.Stat         //内心独白
		list = append(list, mp)
	}
	res := make(map[string]interface{})
	res["list"] = list
	result["res"] = res
	return
}

/*
获取其他用户信息

请求URL：s/user/OtherInfo

参数:
	{
		"uid":10001,
		"notify":true 	//[opt]是否通知被查看者，默认不通知
	}

返回值code为 -101 时,为找不到此用户

返回结果：

{
	"status": "ok",
 	"res":{
 		"uid":1000111,//用户uid
		"gender":1,	//性别
		"age":20,	//年龄
		"nickname":"翅膀的鱼",//昵称
		"avatar":"photo/20141015/1109218597.png", //头像URL
		"star":7, 	//星座
		"height":170,//身高
		"job":"医生", 	//职业
		"edu":5,    //学历
		"interest ":"足球,下棋",//兴趣爱好
		"tag":"标签",//性格标签
		"workarea":"清河新城", //工作地点
		"workunit":"XX公司", //工作单位
		"school":"人民大学", //学校
		"homeprovince":"清河新城", //家乡省
		"homecity":"清河新城", //家乡市
		"trade":"IT业", //从事行业
		"province":"湖南",//用户所在省
		"city":"湖南",//用户所在省
		"regtime"://注册时间 日期时间类型
		"certify_video":0 //视频认证 0 未认证，1 已认证
		"certify_phone":1 //手机认证 0 未认证，1 已认证
		"require":{//择偶要求
			"province":"北京市"//要求对方所在省 默认值为空 不限制为"不限"
			"city":"海淀区"		//要求对方所在市 默认值为空 不限制为"不限"
			"minage":25,		//年龄最小为 为0表示无最小限制  -1为默认值
			"maxage":25,		//年龄最大 为999表示无限制  -1为默认值
			"minheight":170,	//身高最低 0为不限制  -1为默认值
			"maxheight":0,		//身高最高 999为不限制 -1为默认值
			"minedu":1,			//学历最低 0为不限制  -1为默认值
			"needtag":"事业型","体贴型"//感兴趣的类型
			"aboutme":"",//交友寄语
			"hardrequire":1//是否硬性要求 0非硬性 1 是硬性 2 不全是硬性 -1为默认值
		}
		"dynamicpic":{//最近动态图片
			"n":10,//动态数量
			"pics":["http://111.jpg","http://222.jpg"] //图片列表
		}
		"photolist":[  //照片列表
	    	{
	    		"albumid": "857",  //图片ID
				"albumname":"",  //名称
				"pic":"photo/20141030/1041299985.jpg",  //图片URL
				"picdesc":""//图片描述
			}.{...}
		]
		"dynamic_img":"xxx" // 动态背景图片

		"isfollow":1//是否关注 0-未标记，1-有好感，2-特别关注，3-不喜欢(拉黑)
		"online":1//是否在线 1为已经在线
		"lat":1.23232323//位置信息
		"lng":2.2323232//位置信息
		"gpsprovince":"湖南省"
		"gpscity":"长沙市"
		"isblack":0,//是否是黑名单
		"logintime":"三小时前"//登陆时间 字符串
		"stat":0,//是否被封号 0正常 5为已封号
		"issystemuser":0//是否系统用户,1:是系统用户,2:是普通用户
		"isfriend":1//是否是好友，1是好友，0不是好友
	}
}
*/
func (sm *UserModule) SecOtherInfo(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var uid uint32
	var notify bool
	if err := req.Parse("uid", &uid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("notify", &notify, false); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}

	rmap, err := usercontrol.GetOtherInfo(req.Uid, uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["uid"] = uid
	res["goldcoin"], _ = utils.ToInt(rmap["goldcoin"]) //金币余额
	igender, _ := utils.ToInt(rmap["gender"])
	res["gender"] = igender //性别

	if t, err := utils.ToTime(rmap["birthday"]); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "birthday 类型错误")
	} else {
		res["birthday"] = t
	}
	res["age"] = usercontrol.StringAge(rmap["birthday"])

	res["nickname"] = rmap["nickname"]             //昵称
	res["province"] = rmap["province"]             // 省份
	res["city"] = rmap["city"]                     //城市                  //地区
	res["avatar"] = rmap["avatar"]                 //头像URL
	res["height"], _ = utils.ToInt(rmap["height"]) //身高
	res["job"] = rmap["job"]                       //职业
	res["edu"], _ = utils.ToInt(rmap["edu"])       //学历

	res["workarea"] = rmap["workarea"]         //工作地点
	res["workunit"] = rmap["workunit"]         //工作单位
	res["school"] = rmap["school"]             //学校
	res["homeprovince"] = rmap["homeprovince"] //家乡省
	res["homecity"] = rmap["homecity"]         //家乡市
	res["trade"] = rmap["trade"]               //从事行业
	res["star"], _ = utils.ToInt(rmap["star"]) //星座
	res["aboutme"] = rmap["aboutme"]           //个人签名
	res["interest"] = rmap["interest"]         //爱好
	require := make(map[string]interface{})
	require["province"] = rmap["requireprovince"]
	require["city"] = rmap["requirecity"]
	require["minage"], _ = utils.ToInt(rmap["minage"])
	require["maxage"], _ = utils.ToInt(rmap["maxage"])
	require["minheight"], _ = utils.ToInt(rmap["minheight"])
	require["maxheight"], _ = utils.ToInt(rmap["maxheight"])
	require["minedu"], _ = utils.ToInt(rmap["minedu"])
	require["hardrequire"], _ = utils.ToInt(rmap["hardrequire"])
	require["needtag"] = rmap["needtag"]
	require["aboutme"] = rmap["aboutme"]
	res["require"] = require

	res["tag"] = rmap["tag"]                                  //标签
	if t, err := utils.ToTime(rmap["reg_time"]); err != nil { //注册时间
		return service.NewError(service.ERR_INVALID_PARAM, "regtime 类型错误")
	} else {
		res["regtime"] = t
	}
	res["logintime"] = "七天前"
	if tmlogin, err := user_overview.LoginTime(uid); err != nil {
		// return service.NewError(service.ERR_INTERNAL, err.Error())
	} else {
		res["logintime"] = utils.FormatPrevLogin(tmlogin)
	}
	res["certify_video"], _ = utils.ToInt(rmap["certify_video"])
	res["certify_phone"], _ = utils.ToInt(rmap["phonestat"])
	res["stat"], _ = utils.ToInt(rmap["stat"])

	if lat, lng, e := general.UserLocation(uid); e != nil {
		res["lat"] = 0
		res["lng"] = 0
		res["gpsprovince"] = ""
		res["gpscity"] = ""
	} else {
		res["lat"] = lat
		res["lng"] = lng
		res["gpsprovince"] = ""
		res["gpscity"] = ""
		prov, cit, _ := general.City(lat, lng)
		res["gpsprovince"] = prov
		res["gpscity"] = cit

	}

	res["isfollow"] = 0
	if r, err := relation.IsFollow(req.Uid, uid); err != nil {
		// return service.NewError(service.ERR_INTERNAL, err.Error())
	} else {
		res["isfollow"] = r
	}

	res["online"] = 0
	ov, err := user_overview.IsOnline(uid)
	if ov2, ok := ov[uid]; ok {
		if ov2 {
			res["online"] = 1
		}
	}

	res["isblack"] = 0
	if b, _ := base.IsInBlacklist(req.Uid, uid); b {
		res["isblack"] = 1
	}

	sqlr, _, err := usercontrol.GetUserPicture(uid, 0, OTHER_MAX_PIC)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}

	photolist := make([]interface{}, 0, 0)
	// secrct_photolist := make([]interface{}, 0, 0)
	for _, v := range sqlr {
		item := make(map[string]interface{})
		if v["pic"] == rmap["avatar"] {
			item["albumid"], _ = utils.ToInt(v["albumid"]) //图片ID
			item["pic"] = v["pic"]                         //图片URL
			item["status"] = v["status"]                   //图片状态
			photolist = append(photolist, item)
		}
	}
	for _, v := range sqlr {
		item := make(map[string]interface{})
		if v["pic"] != rmap["avatar"] {
			item["albumid"], _ = utils.ToInt(v["albumid"]) //图片ID
			item["pic"] = v["pic"]                         //图片URL
			item["status"] = v["status"]                   //图片状态
			photolist = append(photolist, item)
		}
	}
	res["photolist"] = photolist

	if dyn, pics, err := user_overview.GetUserLastDynamicPic(uid, false); err != nil {
		// return service.NewError(service.ERR_INTERNAL, err.Error())
		sm.log.AppendObj(err, "GetUserLastDynamicPic err ")
	} else {
		res["dynamicpic"] = map[string]interface{}{"n": dyn, "pics": pics}
	}

	res["dynamic_img"] = rmap["dynamic_img"] //动态背景图
	if uid < 1000000 {                       //是否系统用户
		res["issystemuser"] = 1
	} else {
		res["issystemuser"] = 0
	}
	res["isfriend"] = 0
	if r, err := base.IsFriend(req.Uid, uid); err == nil {
		if r {
			res["isfriend"] = 1
		}
	}

	result["res"] = res
	if notify {
		message.SendMessage(message.VISIT, message.Visit{req.Uid, uid}, result)
	}
	return
}

func (sm *UserModule) ClearCache(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	i, _ := utils.ToUint32(req.Body["uid"])
	user_overview.ClearUserObjects(i)
	return
}

/*
资料引导信息提交 SecGuideInfo

请求URL：s/user/GuideInfo

参数: 	{
			"gender":1,//性别
			"age":24, //年龄
			"nickname":"呵呵",//昵称
			"job":"医生", 	//职业
			"trade":"IT业", //从事行业
		}

返回值code为 -101 时,为找不到此用户

返回结果：
{
	"status": "ok”,
	"msg": "success"
	"code": 0,
 	"res":{
	}
}
*/
func (sm *UserModule) SecGuideInfo(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var gender, age int
	var nickname, job, trade string
	if e = req.Parse("gender", &gender, "age", &age, "nickname", &nickname, "job", &job, "trade", &trade); e != nil {
		return
	}

	err := usercontrol.SetUserGuide(req.Uid, gender, age, nickname, job, trade)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("%v", err.Error()))
	}
	user_overview.ClearUserObjects(req.Uid)
	usercontrol.UpdateNickChange(req.Uid)
	return
}

/*
设置我的个人信息 SecSetInfo

请求URL：s/user/SetInfo

参数: 	{
			"nickname":"翅膀的鱼",//[opt]昵称
			"avatar":"photo/20141015/1109218597.png", //[opt]头像URL
			"height":170,//[opt]身高
			"job":"医生", 	//[opt]职业
			"edu":5,    //[opt]学历
			"star":7, 	//[opt]星座
			"interest ":"足球,下棋",//[opt]爱好
			"tag":"标签",//[opt]性格标签
			"age":21,	//[opt]年龄
			"trade":"金融业",  //[opt]行业
			"school":"人民大学", //[opt]学校
			"homeprovince":"清河新城", //[opt]家乡省
			"homecity":"清河新城", //[opt]家乡市
		}

返回值code为 -101 时,为找不到此用户

返回结果：
{
	"status": "ok”,
	"msg": "success"
	"code": 0,
 	"res":{
	}
}
*/
func (sm *UserModule) SecSetInfo(req *service.HttpRequest, result map[string]interface{}) (e error) {
	//sqm := make(map[string]interface{})
	birthdaychange := false
	avatarstat := false
	clearcache := false
	info := make(map[string]interface{})
	var birthday time.Time
	var key string
	// keys := make([]string, 0, 0)
	// params := make([]interface{}, 0, 0)
	if v, ok := req.Body["height"]; ok {
		if i := utils.GetComonFloat64ToInt(v, 0); i == 0 {
			return service.NewError(service.ERR_INVALID_PARAM, "height 类型错误")
		} else {
			info["height"] = i
			clearcache = true
		}
	}
	if v, ok := req.Body["edu"]; ok {
		if i := utils.GetComonFloat64ToInt(v, -1); i == -1 {
			return service.NewError(service.ERR_INVALID_PARAM, "edu 类型错误")
		} else {
			info["edu"] = i
		}
	}
	if v, ok := req.Body["star"]; ok {
		if i := utils.GetComonFloat64ToInt(v, -1); i == -1 {
			return service.NewError(service.ERR_INVALID_PARAM, "star 类型错误")
		} else {
			info["star"] = i
		}
	}
	if v, ok := req.Body["age"]; ok {
		age, e := utils.ToInt(v)
		if e != nil {
			return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("%v", e.Error()))
		}
		birthday = utils.AgeToBirthday(age)
		info["birthday"] = birthday
		birthdaychange = true
	}
	if v, ok := req.Body["nickname"]; ok {
		info["nickname"] = utils.ToString(v)
		usercontrol.UpdateNickChange(req.Uid)
		key = "nickname"
		clearcache = true
	}
	if v, ok := req.Body["homeprovince"]; ok {
		info["homeprovince"] = utils.ToString(v)
		clearcache = true
	}
	if v, ok := req.Body["homecity"]; ok {
		info["homecity"] = utils.ToString(v)
		clearcache = true
	}
	if v, ok := req.Body["avatar"]; ok {
		url := utils.ToString(v)
		if ir, er := general.CheckImgByUrl(general.IMGCHECK_SEXY_AND_AD, url); er == nil && ir.Status != 0 {
			return service.NewError(service.ERR_INTERNAL, "图片审核失败", "图片审核失败")
		} else {
			sm.log.AppendObj(e, "set userinfo checkImg is error", req.Uid, url)
		}
		usercontrol.SetAvatar(req.Uid, url)
		usercontrol.UpdateNickChange(req.Uid)
		key = "avatar"
		avatarstat = true

	}
	if v, ok := req.Body["job"]; ok {
		info["job"] = utils.ToString(v)
	}
	if v, ok := req.Body["trade"]; ok {
		info["trade"] = utils.ToString(v)
	}
	if v, ok := req.Body["aboutme"]; ok {
		info["aboutme"] = utils.ToString(v)
		key = "aboutme"
		clearcache = true
	}
	if v, ok := req.Body["interest"]; ok {
		info["interest"] = utils.ToString(v)
	}

	if v, ok := req.Body["workarea"]; ok {
		info["workarea"] = utils.ToString(v)
	}
	if v, ok := req.Body["workunit"]; ok {
		info["workunit"] = utils.ToString(v)
	}
	if v, ok := req.Body["school"]; ok {
		name := utils.ToString(v)
		info["school"] = name
		if school, found := usercontrol.GetSchool(name); found {
			info["edu"] = school.Edu
		}
	}

	if v, ok := req.Body["tag"]; ok {
		info["tag"] = utils.ToString(v)
		tag.UseTag(req.Uid, common.TAG_TYPE_USER, utils.ToString(v))
		clearcache = true
	}
	if !avatarstat {
		err := usercontrol.SetUserDetailInfo(req.Uid, info)
		if err != nil {
			return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("%v", err.Error()))
		}

	}
	if birthdaychange {
		message.SendMessage(message.BIRTHDAY_CHANGE, message.Online{req.Uid}, result)
		discovery.UpdateDiscovery(req.Uid, "birthday", birthday)
	}
	if clearcache {
		user_overview.ClearUserObjects(req.Uid)
	}

	usercontrol.CheckVerify(req.Uid, key)

	usercontrol.StatUserInfo(req.Uid)
	//	err = certify.CheckHonesty(req.Uid, common.PRI_GET_INFO)
	//	if err != nil {
	//	return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("%v", err.Error()))
	//	}
	return
}

/*
设置我的交友需求 SecSetRequire

请求URL：s/user/SetRequire

参数:
{
	"province":"北京市"//[opt]要求对方所在省
	"city":"海淀区"	//[opt]要求对方所在市
	"minage":0,		//[opt]年龄最小为 为0表示无最小限制
	"maxage":25,	//[opt]年龄最大 为0表示无最小限制
	"minheight":170,//[opt]身高最低 0为不限制
	"maxheight":0,//[opt]身高最高 0为不限制
	"minedu":1,//[opt]学历最低 0为不限制
	"aboutme":"",//[opt]交友寄语
	"needtag":"进取型",//[opt]感兴趣的类型
	"hardrequire":1//[opt]是否硬性要求 0非硬性 1 是硬性 2 不全是硬性
}

返回值code为 -101 时,为找不到此用户

返回结果：
{
	"status": "ok”,
}
*/
func (sm *UserModule) SecSetRequire(req *service.HttpRequest, result map[string]interface{}) (e error) {
	info := make(map[string]interface{})
	var key string
	var minage int
	if err := req.Parse("minage", &minage); err == nil {
		info["minage"] = minage
	}

	var maxage int
	if err := req.Parse("maxage", &maxage); err == nil {
		info["maxage"] = maxage
	}

	var minheight int
	if err := req.Parse("minheight", &minheight); err == nil {
		info["minheight"] = minheight
	}

	var maxheight int
	if err := req.Parse("maxheight", &maxheight); err == nil {
		info["maxheight"] = maxheight
	}

	var minedu int
	if err := req.Parse("minedu", &minedu); err == nil {
		info["minedu"] = minedu
	}

	var hardrequire int
	if err := req.Parse("hardrequire", &hardrequire); err == nil {
		info["hardrequire"] = hardrequire
	}
	if v, ok := req.Body["province"]; ok {
		info["requireprovince"] = utils.ToString(v)
	}
	if v, ok := req.Body["city"]; ok {
		info["requirecity"] = utils.ToString(v)
	}
	if v, ok := req.Body["needtag"]; ok {
		info["needtag"] = utils.ToString(v)
	}
	if v, ok := req.Body["aboutme"]; ok {
		info["aboutme"] = utils.ToString(v)
		key = "aboutme"
	}
	e = usercontrol.SetUserDetailInfo(req.Uid, info)
	if key != "" {
		usercontrol.CheckVerify(req.Uid, key)
	}
	return
}

// SecAddPhoto 用于添加单张照片
//
// URI: s/user/AddPhoto
//
// 参数
// {
// 	"albumname":""//照片名称
// 	"pic":"http://......."//照片URL
// 	"picdesc":""//照片描述
// 	"type":"1"//[opt]照片类型 0为普通 1为私密 不填为普通
// }
//
// 返回值
// {
// 		"status":"ok"
// 		"res":
// 		{
// 			"albumids":[12222，13222]//照片ID列表
// 		}
// }
func (sm *UserModule) SecAddPhoto(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("albumname", "pic", "picdesc")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	var tp int
	if _, ok := req.EnsureBody("type"); ok {
		tp, _ = utils.ToInt(req.Body["type"])
	}

	albumname := utils.ToString(req.Body["albumname"])
	pic := utils.ToString(req.Body["pic"])
	picdesc := utils.ToString(req.Body["picdesc"])

	if ir, e := general.CheckImgByUrl(general.IMGCHECK_SEXY_AND_HUMAN, pic); e == nil && ir.Status != 0 {
		return service.NewError(service.ERR_INTERNAL, "图片审核失败", "图片审核失败")
	} else {
		sm.log.AppendObj(e, "set AddPhoto  checkImg is error", req.Uid, pic)
	}

	v, err := usercontrol.AddPic(req.Uid, albumname, pic, "", picdesc, tp)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("MYSQL insert error code %v", err.Error()))
	}
	usercontrol.CheckVerify(req.Uid, "avatar")
	usercontrol.NotifyAndDelInviteList(req.Uid, usercontrol.INVITE_KEY_PHOTO)
	res := make(map[string]interface{})
	// if i, err := sqlr.LastInsertId(); err == nil {
	res["albumid"] = v
	result["res"] = res
	return
}

/*
 SecAddPhoto2 用于添加多张照片

 URI: s/user/AddPhoto2

 参数
 {
 		"pics":["http://111.jpg","http://222.jpg"]//照片URL列表  不超过10个
 		"type":"1"//[opt]照片类型 0为普通 1为私密 不填为普通
 }

 返回值
  {
	 "res": {
		 "albumids": [
		 29069,
		 29070
		 ],
		 "invalid_pic": {
			 "code": 2013,
			 "error_pic": [
				 "http://image1.yuanfenba.net/uploads/oss/chat/201508/31/00470858095.jpg"
			 ],
			 "error_reason": "图片涉黄"
		 }
	 },
	 "status": "ok",
	 "tm": 1446604284
 }
*/
func (sm *UserModule) SecAddPhoto2(req *service.HttpRequest, result map[string]interface{}) (e error) {
	abs, ok := req.EnsureBody("pics")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	var tp int
	if _, ok := req.EnsureBody("type"); ok {
		tp, _ = utils.ToInt(req.Body["type"])
	}
	pics, err := utils.ToStringSlice(req.Body["pics"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("MYSQL insert error code %v", err.Error()))
	}
	// 添加黄发
	res := make(map[string]interface{})
	v, err := usercontrol.AddPics(req.Uid, tp, pics...)
	usercontrol.CheckVerify(req.Uid, "avatar")
	usercontrol.NotifyAndDelInviteList(req.Uid, usercontrol.INVITE_KEY_PHOTO)
	res["albumids"] = v

	// 异步执行图片检测
	go usercontrol.DoPhotosCheckImg(req.Uid, v, pics)
	// 保留结构
	res["invalid_pic"] = map[string]interface{}{"error_pic": []string{}, "error_reason": "图片涉黄", "code": service.ERR_INVALID_IMG}
	result["res"] = res
	return
}

/*
删除照片

URI: s/user/DelPhoto

参数
	{
		"albumid":"1111"//照片ID
	}

返回值
	{
			"status":"ok"
	}
*/
func (sm *UserModule) SecDelPhoto(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	albumid := utils.GetComonFloat64ToInt(req.Body["albumid"], 0)
	err := usercontrol.DelPic(req.Uid, albumid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	// err = certify.CheckHonesty(req.Uid, common.PRI_GET_PHOTOS)
	// if err != nil {
	// 	return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("%v", err.Error()))
	// }
	return
}

/*
获取用户收货地址

请求URL：s/user/GetAddr

返回结果：
	{
		"status": "ok",
		"res":{
			"addr":[
				{
					"addrid":11531, //地址ID
					"phone":"13111555677",//电话
					"province":"湖南",  //省份
					"city":"长沙",  //城市
					"username":"收货姓名",//收货姓名
					"address ":"XX路XX村XX号"//详细地址
				},{...}
			]
		}
	}
*/
func (sm *UserModule) SecGetAddr(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {

	v, err := usercontrol.GetAddr(req.Uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["addr"] = v
	result["res"] = res
	return
}

/*
设置用户收货地址

请求URL：s/user/SetAddr

参数:
	 {
		 "act":1,//设置方式1为增加2为更改 3为删除
		 "addrid":11531, //地址ID  若为增加 则不需要此字段
		 "phone":"13111555677",//电话
		 "province":"湖南",  //省份
		 "city":"长沙",  //城市
		 "username":"收货姓名",//收货姓名
		 "address ":"XX路XX村XX号"//详细地址
	}

返回值:
	{
			"status":"ok"
	}
*/
func (sm *UserModule) SecSetAddr(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("act")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	iact, err := utils.ToInt(req.Body["act"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "iact")
	}
	switch iact {
	case 1: //增加
		abs, ok := req.EnsureBody("phone", "province", "city", "address", "username")
		if !ok {
			return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
		}
		phone := utils.ToString(req.Body["phone"])
		province := utils.ToString(req.Body["province"])
		city := utils.ToString(req.Body["city"])
		address := utils.ToString(req.Body["address"])
		username := utils.ToString(req.Body["username"])

		res := make(map[string]interface{})
		addrid, err := usercontrol.AddAddr(req.Uid, phone, province, city, address, username)
		if err != nil {
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
		res["addrid"] = addrid
		result["res"] = res

	case 2: //修改
		abs, ok := req.EnsureBody("phone", "province", "city", "address", "username", "addrid")
		if !ok {
			return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
		}
		phone := utils.ToString(req.Body["phone"])
		province := utils.ToString(req.Body["province"])
		city := utils.ToString(req.Body["city"])
		address := utils.ToString(req.Body["address"])
		username := utils.ToString(req.Body["username"])
		addrid, err := utils.ToUint32(req.Body["addrid"])
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, "addrid")
		}
		if err = usercontrol.SetAddr(req.Uid, addrid, phone, province, city, address, username); err != nil {
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
	case 3: //删除
		if abs, ok := req.EnsureBody("addrid"); !ok {
			return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
		}
		addrid, err := utils.ToInt(req.Body["addrid"])
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, "addrid")
		}
		if err := usercontrol.DelAddr(req.Uid, addrid); err != nil {
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
	}
	return
}

/*
查询用户收支明细

请求URL：s/user/CoinLog

参数: {"cur":0,“ps”: 10}


返回结果：

{
	"status": "ok",
	"msg": "success"
    "code": 0,
	"res":{
		"list":[
			{
				"info": "充值", //信息
				"time":...,//时间
				"coin":5  //收支的金币 正数收入 负数支出
				"forid":1123//为谁花的钱 为0 为为自己花的 预留
				"type":1//收支类型 1充值 2赠送 3惊喜打工 4礼物
				},
				{ … }
   			],
       	"pages": { …分页结构}
		}
}
*/
func (sm *UserModule) SecCoinLog(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {

	cur := utils.GetComonFloat64ToInt(req.Body["cur"], 1)
	ps := utils.GetComonFloat64ToInt(req.Body["ps"], 10)
	list, total, err := usercontrol.GetCoinLog(req.Uid, cur, ps)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})

	rlist := make([]interface{}, 0, 0)
	for _, value := range list {
		vv := make(map[string]interface{})
		for k, v := range value {
			if k == "time" {
				if t, err := utils.ToTime(v); err != nil {
					vv[k] = utils.Now
				} else {
					vv[k] = t
				}
			} else {
				vv[k] = v
			}
		}
		rlist = append(rlist, vv)
	}
	res["list"] = rlist
	res["pages"] = utils.PageInfo(total, cur, ps)
	result["res"] = res
	return
}

/*
开始充值

充值类型：1 财付通 2 网银在线 3 银联，4支付宝,5手动,6手机充值卡,7支付宝wap,8微信,9银联语音

请求URL：s/user/PayBegin
	{
		"tp":1，//充值平台类型
		"productid":123322,//商品ID
		"extra":{//额外信息
			"cardMoney":100.00//充值卡金额 双精度浮点
			"sn":"xxxxxxxxxxxx"//卡号
			"password":"ssxxcxxxxxxxxxxxx"//卡密码
			"cardType":"1"//卡类型，0移动，1联通，2电信；如果为空，系统自动识别
		}
	}

返回结果：
	{
		"status": "ok",
		"res":{
			"order_no":"1233w422ss"  //订单号
			"content":{}//充值具体信息 充值接口返回
		}
	}
*/
func (sm *UserModule) SecPayBegin(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	// var tp, money, productid int
	abs, ok := req.EnsureBody("tp", "productid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	tp, err := utils.ToInt(req.Body["tp"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "tp")
	}
	// money, err := utils.ToInt(req.Body["money"])
	// if err != nil {
	// 	return service.NewError(service.ERR_INVALID_PARAM, "money")
	// }
	productid, err := utils.ToInt(req.Body["productid"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "productid")
	}
	// info := utils.ToString(req.Body["info"])
	var extra map[string]interface{}
	if content, ok := req.Body["extra"]; ok {
		switch value := content.(type) {
		case map[string]interface{}:
			extra = value
		}
	}
	order_no, content, err := usercontrol.PayBegin(req.Uid, tp, productid, req.IP(), extra)
	if err != nil {
		return service.NewError(service.ERR_PAY_INVALID, "充值错误", err.Error())
	}
	res := make(map[string]interface{})
	res["order_no"] = order_no
	res["tp"] = tp
	res["content"] = content
	result["res"] = res
	return
}

/*
充值提供给PHP的回调

请求URL：user/PayCallback

参数：
	{
		"tp":1，//充值平台类型
		"order_no":"1233w422ss"  //订单号
		"stat":1,  //1 充值成功 2 充值失败 （3 超时）
		"money":1000,  //金额 成功充值的钱数 以分为单位

	}
返回结果：
	{
	"status": "ok",
	}
*/
func (sm *UserModule) PayCallback(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	sm.log.Append(fmt.Sprintf("PayCallback %v", req.Body))
	abs, ok := req.EnsureBody("tp", "order_no", "stat", "money")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	tp, err := utils.ToInt(req.Body["tp"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "tp")
	}
	stat, err := utils.ToInt(req.Body["stat"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "stat")
	}
	money, err := utils.ToInt(req.Body["money"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "money")
	}
	order_no := utils.ToString(req.Body["order_no"])

	err = usercontrol.PayCallback(tp, order_no, stat, money)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["order_no"] = order_no
	result["res"] = res
	return
}

/*
客户端获取充值结果

请求URL：s/user/PayQuery
	{
		"tp":1//支付类型
		"order_no":"1233w422ss"  //订单号
	}
返回结果：
	{
		"status": "ok",
		"res":{
				"stat":0,  //0未返回 1 充值成功 2 充值失败 （3 超时）
				"money":10.00,  //金额 成功充值的钱数 双精度浮点
				"info":"充值VIP一个月"//充值描述
		}
	}
*/
func (sm *UserModule) SecPayQuery(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("tp", "order_no")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	tp, err := utils.ToInt(req.Body["tp"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "tp")
	}
	order_no := utils.ToString(req.Body["order_no"])

	stat, money, info, err := usercontrol.PayQuery(tp, req.Uid, order_no)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["stat"] = stat
	res["money"] = money
	res["info"] = info
	result["res"] = res
	return
}

// SecProduct 充值商品列表
// 请求URL：s/user/Product
// 参数
// {
// 	"type":1，//1金币，2VIP  ,0全部 ,3飞机游戏次数 4 身份认证
// 	"tp2":1 //[opt]附加参数 测试用 如果为1 则包含一金币商品 正式环境不包含此字段
// 	"os":0//[opt]运行平台 默认为安卓 1为IOS
// }
// 返回结果：
// {
// "status": "ok",
// "res":{
// 	"list":[
// 		{
// 			"id":1,  //商品ID
// 			"name":"一金币",  //名称
// 			"info":"充值一个金币"//充值描述
// 			"money":0.01//需要的钱 双精度浮点
// 			"img":"/aaa/bbb.hpg"//图片地址
// 			"recommend":1//是否推荐 1为推荐否则0
// 			"coincost":3000//金币购买消耗的金币数
// 			"iapid":"11122222"//IOS平台的产品ID
// 		}
// 		,{..}
// 	]
// }
// }
func (sm *UserModule) SecProduct(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("type")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	tp, err := utils.ToInt(req.Body["type"])
	tp2 := 0
	abs, ok = req.EnsureBody("tp2")
	if ok {
		tp2, _ = utils.ToInt(req.Body["tp2"])
	}
	os := 0
	abs, ok = req.EnsureBody("os")
	if ok {
		os, _ = utils.ToInt(req.Body["os"])
	}
	r, err := usercontrol.ProductList(tp, tp2, os)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	unread.UpdateReadTime(req.Uid, common.UNREAD_WALLET)
	res := make(map[string]interface{})
	res["list"] = r
	un := make(map[string]interface{})
	un[common.UNREAD_WALLET] = 0
	unread.GetUnreadNum(req.Uid, un)
	result[common.UNREAD_KEY] = un
	result["res"] = res
	return
}

/*
SecPayIosQuery IOS充值结果查询
请求URL：s/user/PayIosQuery
参数

{
	"receiptdata":""，//充值结果数据
	"ifsandbox":1 //[opt]附加参数 默认为正式版 如果为1 表示沙箱测试
	"transaction_id":"11122222"//[opt]订单号
}

返回结果：

{
	"status": "ok",
	"res":{
	}
	"balance":{}  //用户余额变动
}
*/
func (sm *UserModule) SecPayIosQuery(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var receiptdata, transaction_id string
	var ifsandbox int
	if err := req.Parse("receiptdata", &receiptdata); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.Parse("ifsandbox", &ifsandbox); err != nil {

	}

	if err := req.Parse("transaction_id", &transaction_id); err != nil {
		transaction_id = ""
	}
	r, err := usercontrol.PayIosQuery(req.Uid, ifsandbox, receiptdata, transaction_id)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	}
	for k, v := range r {
		result[k] = v
	}
	return
}

/*
举报用户

请求URL：s/user/Complain

参数:
	{
		"type":1,			//举报类型,1举报用户,2举报动态
		"id":123,			//类型为1时为要举报的用户ID，类型为2时为动态的ID
		"info":"举报内容"	//详细信息
	}
返回结果：
	{
		"status": "ok",
	}
*/
func (sm *UserModule) SecComplain(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	var info string
	var tp int
	if e = req.Parse("type", &tp, "id", &id, "info", &info); e != nil {
		return
	}
	if err := ban.UserComplain(id, req.Uid, info, tp); err != nil {
		return err
	}
	return
}

//投诉 LIST
func (sm *UserModule) SecGetComplainList(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("index", "count")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	index, err := utils.ToInt(req.Body["index"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	count, err := utils.ToInt(req.Body["count"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	if logs, err := ban.GetComplainList(index, count); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	} else {
		res["list"] = logs
	}
	result["res"] = res
	return
}

//过滤 LIST
func (sm *UserModule) SecGetFilterUser(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("index", "count")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	index, err := utils.ToInt(req.Body["index"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	count, err := utils.ToInt(req.Body["count"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	if logs, err := ban.GetFilterUser(index, count); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	} else {
		res["list"] = logs
	}
	result["res"] = res
	return
}

//设置用户位置
func (sm *UserModule) UserArea(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	// uid, err := utils.ToUint32(req.GetParam("uid"))
	// if err != nil {
	// 	return service.NewError(service.ERR_INVALID_PARAM, "uid")
	// }
	abs, ok := req.EnsureBody("lat", "lng")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	lat, err := utils.ToFloat64(req.Body["lat"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	lng, err := utils.ToFloat64(req.Body["lng"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	//发送消息设置
	message.SendMessage(message.LOCATION_CHANGE, message.LocationChange{req.Uid, lat, lng}, result)
	res := make(map[string]interface{})
	result["res"] = res
	return
}

//用户信息后台用
func (sm *UserModule) UserDetail(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	// abs, ok := req.EnsureBody("uid")
	// if !ok {
	// 	return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	// }
	uid, err := utils.ToUint32(req.GetParam("uid"))
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "uid")
	}

	rmap, err := usercontrol.GetOtherInfo(req.Uid, uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["uid"] = uid
	res["username"] = rmap["username"]
	res["isvip"], _ = utils.ToInt(rmap["isvip"])       // VIP状态 1为VIP
	res["goldcoin"], _ = utils.ToInt(rmap["goldcoin"]) //金币余额
	iexp := 0
	iexp, _ = utils.ToInt(rmap["exp"]) //经验值
	res["exp"] = iexp
	res["grade"] = usercontrol.ExpToGrade(iexp)    //等级
	res["luck"], _ = utils.ToInt(rmap["luck"])     //幸运值
	res["gender"], _ = utils.ToInt(rmap["gender"]) //性别
	res["age"] = usercontrol.StringAge(rmap["birthday"])
	res["nickname"] = rmap["nickname"]                     //昵称
	res["province"] = rmap["province"]                     // 省份
	res["city"] = rmap["city"]                             //城市
	res["area"] = rmap["area"]                             //地区
	res["avatar"] = rmap["avatar"]                         //头像URL
	res["height"], _ = utils.ToInt(rmap["height"])         //身高
	res["job"] = rmap["job"]                               //职业
	res["edu"], _ = utils.ToInt(rmap["edu"])               //学历
	res["income"], _ = utils.ToInt(rmap["income"])         //最高收入
	res["min_income"], _ = utils.ToInt(rmap["min_income"]) //最低收入
	res["marry"], _ = utils.ToInt(rmap["marry"])           //婚姻
	res["star"], _ = utils.ToInt(rmap["star"])             //星座 计算列
	res["aboutme"] = rmap["aboutme"]                       //个人签名
	res["interest"] = rmap["interest"]                     //爱好
	res["skill"] = rmap["interest"]                        //技能
	res["require"] = rmap["require"]                       //另一半要求
	res["looking"] = rmap["looking"]                       //对爱情的看法
	res["tag"] = rmap["tag"]                               //标签
	private, _ := utils.ToInt(rmap["private"])             //保密状态 保密的话 看不到联系方式
	res["private"] = private
	res["contact"] = ""
	if private == 0 {
		res["contact"] = rmap["contact"] //联系方式
	}
	res["localtag"] = rmap["localtag"] //附近主题

	if lat, lng, e := general.UserLocation(uid); e != nil {
		res["lat"] = 0
		res["lng"] = 0
	} else {
		res["lat"] = lat
		res["lng"] = lng
	}
	ov, err := user_overview.IsOnline(uid)
	if ov2, ok := ov[uid]; ok {
		if ov2 {
			res["online"] = 1
		}
	}
	sqlr, _, err := usercontrol.GetUserPicture(uid, 0, OTHER_MAX_PIC)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	photolist := make([]interface{}, 0, len(sqlr))
	for _, v := range sqlr {
		item := make(map[string]interface{})
		item["albumid"], _ = utils.ToInt(v["albumid"]) //图片ID
		item["albumname"] = v["albumname"]             //图片名称
		item["pic"] = v["pic"]                         //图片URL
		item["picdesc"] = v["picdesc"]                 //图片描述
		item["status"] = v["status"]                   //图片描述
		photolist = append(photolist, item)
	}
	res["photolist"] = photolist
	result["res"] = res
	return
}

/*
BUG反馈
请求URL：s/user/BugBack

参数:
	{
		"info":"建议内容"						//BUG内容
		"imgs":["http://1.jpg","http://2.jpg"]	//图片列表
	}
返回结果：
	{
		"status":"ok”
	}
*/
func (sm *UserModule) SecBugBack(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var info string
	var imgs []string
	if e = req.Parse("info", &info, "imgs", &imgs); e != nil {
		return
	}
	// strimg:=strings.sp imgs.
	strimg := strings.Join(imgs, ",")
	err := usercontrol.Feedback(req.Uid, info, strimg, 0)
	if err != nil {
		return err
	}
	msg := make(map[string]interface{})
	var but notify.But
	msg[common.FOLDER_KEY] = common.FOLDER_OTHER
	msg["type"] = common.MSG_TYPE_RICHTEXT
	msg["tip"] = "bug反馈成功"
	m := map[string]interface{}{"type": common.RICHTEXT_TYPE_TEXT, "text": "您提交的BUG我们已经收到，秋千程序猿们会仔细排查修改，感谢您的支持", "but": but}
	msg_arr := make([]map[string]interface{}, 0, 1)
	msg["msgs"] = append(msg_arr, m)
	general.SendMsg(common.UID_PROBLEM, req.Uid, msg, "")
	return
}

/*
意见建议
请求URL：s/user/Feedback

参数:
	{
		"info":"建议内容"//建议内容
		"imgs":["http://1.jpg","http://2.jpg"]	//图片列表
	}
返回结果：
	{
		"status":"ok”
	}
*/
func (sm *UserModule) SecFeedback(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var info string
	var imgs []string
	if e = req.Parse("info", &info, "imgs", &imgs); e != nil {
		return
	}
	// strimg:=strings.sp imgs.
	strimg := strings.Join(imgs, ",")
	err := usercontrol.Feedback(req.Uid, info, strimg, 1)
	if err != nil {
		return err
	}
	msg := make(map[string]interface{})
	var but notify.But
	msg[common.FOLDER_KEY] = common.FOLDER_OTHER
	msg["type"] = common.MSG_TYPE_RICHTEXT
	msg["tip"] = "意见反馈成功"
	m := map[string]interface{}{"type": common.RICHTEXT_TYPE_TEXT, "text": "您提交的意见我们已经收到，我们会认真阅读,秋千团队非常感谢您的支持", "but": but}
	msg_arr := make([]map[string]interface{}, 0, 1)
	msg["msgs"] = append(msg_arr, m)
	general.SendMsg(common.UID_PROBLEM, req.Uid, msg, "")
	return
}

//当日上线奖励列表
func (sm *UserModule) SecOnlineAwards(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {

	stat, list, err := award.OnlineAwards(req.Uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["stat"] = stat
	res["list"] = list
	result["res"] = res
	return
}

//领取当日奖励
func (sm *UserModule) SecReceiveOnlineAward(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	icoin, iplay, err := award.ReceiveOnlineAward(req.Uid)
	if err.Code != service.ERR_NOERR {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	}
	if iplay != 0 {
		//	un := make(map[string]interface{})
		//	un[common.UNREAD_PLANE_FREE] = 0
		//	unread.GetUnreadNum(req.Uid, un)
		//	result[common.UNREAD_KEY] = un
		not, e := notify.GetNotify(req.Uid, notify.NOTIFY_PLANE_NUM, nil, "系统消息", fmt.Sprintf("每日上线奖励 %v飞行点", iplay), req.Uid)
		if e == nil {
			result[notify.NOTIFY_KEY] = not
		}
	}
	if icoin != 0 {
		if c, _, err := coin.GetUserCoinInfo(req.Uid); err == nil {
			result[common.USER_BALANCE] = c
			result[common.USER_BALANCE_CHANGE] = fmt.Sprintf("每日上线奖励 %v金币", icoin)
		}
		not, e := notify.GetNotify(req.Uid, notify.NOTIFY_COIN, nil, "系统消息", fmt.Sprintf("每日上线奖励 %v金币", icoin), req.Uid)
		if e == nil {
			result[notify.NOTIFY_KEY] = not
		}
	}
	//stat.Append(req.Uid, stat.ACTION_ONLINE_AWARD, map[string]interface{}{})
	usercontrol.RefStar(req.Uid)
	return
}

func (sm *UserModule) SecCoinBuy(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("productid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	productid, err := utils.ToInt(req.Body["productid"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "productid 错误")
	}

	info, blance, e := usercontrol.CoinBuy(req.Uid, productid)
	if e.Code != service.ERR_NOERR {
		return e
	}
	result[common.USER_BALANCE] = blance
	result[common.USER_BALANCE_CHANGE] = info

	//un := make(map[string]interface{})
	//	un[common.UNREAD_PLANE_FREE] = 0
	//	unread.GetUnreadNum(req.Uid, un)
	//	result[common.UNREAD_KEY] = un
	not, err2 := notify.GetNotify(req.Uid, notify.NOTIFY_PLANE_NUM, nil, "系统消息", info, req.Uid)
	if err2 == nil {
		result[notify.NOTIFY_KEY] = not
	}
	return
}

func (sm *UserModule) AdminSetAvatarStat(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("list")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	ulist, err := utils.ToStringSlice(req.Body["list"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "list 错误")
	}
	if err := usercontrol.AdminSetAvatarStat(ulist...); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	}
	return
}

func (sm *UserModule) AdminTagInfo(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("tid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	tid, err := utils.ToUint32(req.Body["tid"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "list 错误")
	}
	res := make(map[string]interface{})
	if r, err := topic.GetTopics(tid); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	} else {
		res["info"] = r
	}

	result["res"] = res
	return
}

func (sm *UserModule) ClearUserCache(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("uid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	uid, err := utils.ToUint32(req.Body["uid"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "uid 错误")
	}
	if err := user_overview.ClearUserObjects(uid); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	}
	message.SendMessage(message.CLEAR_CACHE, message.ClearCache{uid}, result)
	return
}

func (sm *UserModule) GetTenUserFollow(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("uid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	uid, err := utils.ToUint32(req.Body["uid"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "uid 错误")
	}
	var count int
	if count, err = usercontrol.SendTenFollow(uid); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	}
	res := make(map[string]interface{})
	res["count"] = count
	result["res"] = res
	return
}

func (sm *UserModule) AdminTjInfo(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("uid", "date")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	uid, err := utils.ToUint32(req.Body["uid"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "uid 错误")
	}
	date := utils.ToString(req.Body["date"])
	// res := make(map[string]interface{})
	if r, err := usercontrol.AdminUidTongji(uid, date); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	} else {
		result["res"] = r
	}
	return
}

func (sm *UserModule) ManagerTJ(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("msgid", "from", "to", "type", "tag", "manager_id")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	from, err := utils.ToUint32(req.Body["from"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "from 错误")
	}
	to, err := utils.ToUint32(req.Body["to"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "to 错误")
	}
	msgid, err := utils.ToUint64(req.Body["msgid"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "msgid 错误")
	}
	manager_id, err := utils.ToUint32(req.Body["manager_id"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "manager_id 错误")
	}
	tp := utils.ToString(req.Body["type"])
	tag := utils.ToString(req.Body["tag"])
	if err := usercontrol.ManagerTJ(msgid, from, to, manager_id, tp, tag); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	}
	return
}

// SecGetQuestion 用于获取用户择友问答
//
// URI: s/user/GetQuestion
//
// 参数
// {
// 	"uid":"112233"//[opt]要查看的用户ID 不填写本字段表示查看自己的择友问答
// }
//
// 返回值
// {
// 		"status":"ok"
// 		"res":
// 		{
// 			"list":[
// 				{
// 				"id":10//问题ID
// 				"question":"问题1",//问题
// 				"answer":"回答一"//回答 为空字符串表示未回答
// 				},
// 				.........
// 			]
// 		}
// }
func (sm *UserModule) SecGetQuestion(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var uid uint32
	_, ok := req.EnsureBody("uid")
	if ok {
		if i, err := utils.ToUint32(req.Body["uid"]); err != nil {
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		} else {
			uid = i
		}
	} else {
		uid = req.Uid
	}
	res := make(map[string]interface{})

	if list, err := usercontrol.GetUidQuestion(uid); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	} else {
		res["list"] = list
	}

	result["res"] = res
	return
}

// SecSetQuestion 用于提交用户择友回答
//
// URI: s/user/SetQuestion
//
// 参数
// {
//		"id":11,		//问题ID
//		"answer":"回答1"//用户提交的回答
// }
//
// 返回值
// {
// 		"status":"ok"
// 		"res":
// 		{
// 		}
// }
func (sm *UserModule) SecSetQuestion(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("id", "answer")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	id, err := utils.ToUint32(req.Body["id"])
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), "from 错误")
	}
	answer := utils.ToString(req.Body["answer"])
	if err := usercontrol.SetQuestion(req.Uid, id, answer); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	}
	return
}

func (sm *UserModule) RefStar(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	if e := req.Parse("uid", &uid); e != nil {
		return e
	}
	e = usercontrol.RefStar(uid)
	return
}

/*
SearchSchool 返回搜索到的学校列表

URI: user/SearchSchool

参数
{
		"input":"吉林华"//用户输入的学校字符串
		"cur":1,
		"ps":10,
}

返回值
{
		"status":"ok"
		"res":
		{
			"schools":{
				"list":[
					{
						"school":"吉林华桥外国语学院",//学校名
						"owner":"吉林省教育厅",//办学单位
						"area":"长春市",//所在地
						"level":"本科",//级别
						"tip":"民办"//是否民办
					}
				],
				"pages":{...}
			}
		}
}
*/
func (sm *UserModule) SearchSchool(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	var input string
	if e := req.Parse("input", &input, "cur", &cur, "ps", &ps); e != nil {
		return e
	}
	res := make(map[string]interface{})
	if list, total, e := usercontrol.SearchSchool(input, cur, ps); e != nil {
		return e
	} else {
		item := make(map[string]interface{})
		item["list"] = list
		item["pages"] = utils.PageInfo(total, cur, ps)
		res["schools"] = item
	}
	result["res"] = res
	return
}

/*
设置动态背景图片

URL: /s/user/SetDynamicImg

参数：
	url: 背景图片地址

返回值：
	{
		"status": "ok",
		"tm": 1444705701
	}
*/
func (sm *UserModule) SecSetDynamicImg(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var url string
	if e = req.Parse("url", &url); e != nil {
		return
	}
	e = user_overview.SetUserDynamicImg(req.Uid, url)
	user_overview.ClearUserObjects(req.Uid)
	return
}

/*
设置用户隐私和防骚扰

URI: /s/user/SetProtect

参数:
	{
		"canfind":1,	//[opt]允许附近的人找到我 1表示 不允许 0表示允许 0为默认值
		"chatremind":1,	//[opt]私聊提醒 1表示 不允许 0表示允许 0为默认值
		"stranger":1,	//[opt]陌生人新消息提醒 1表示 不允许 0表示允许 0为默认值
		"praise":1,		//[opt]被点赞提醒 1表示 不允许 0表示允许 0为默认值
		"commit":1,		//[opt]被评论提醒 1表示 不允许 0表示允许 0为默认值
		"msgnotring":1,  //[opt]消息无提示音,0为有提示音,1为关闭提示音(默认值)
		"msgnotshake":1, //[opt]消息无震动,0为有震动(默认值),1为关闭震动
		"nightring":1,   //[opt]晚上是否响铃震动,0半夜不响铃(默认值),1为半夜仍响铃
	}

返回值：
	{
		"status": "ok"
	}
*/
func (sm *UserModule) SecSetProtect(req *service.HttpRequest, result map[string]interface{}) (e error) {

	sm.log.AppendObj(nil, "SetUserProtect: v pram", req.Body)
	info := make(map[string]interface{})
	var value int
	if err := req.Parse("canfind", &value); err == nil {
		sm.log.AppendObj(nil, "SetUserProtect: canfind ", value)
		info["canfind"] = value
	}
	if err := req.Parse("chatremind", &value); err == nil {
		sm.log.AppendObj(nil, "SetUserProtect: chatremind", value)
		info["chatremind"] = value
	}
	if err := req.Parse("stranger", &value); err == nil {
		sm.log.AppendObj(nil, "SetUserProtect: strrangeer", value)
		info["stranger"] = value
	}
	if err := req.Parse("praise", &value); err == nil {
		sm.log.AppendObj(nil, "SetUserProtect: prasise", value)
		info["praise"] = value
	}
	if err := req.Parse("commit", &value); err == nil {
		sm.log.AppendObj(nil, "SetUserProtect: commit", value)
		info["commit"] = value
	}
	if err := req.Parse("msgnotring", &value); err == nil {
		sm.log.AppendObj(nil, "SetUserProtect: msgnotring", value)
		info["msgnotring"] = value
	}
	if err := req.Parse("msgnotshake", &value); err == nil {
		sm.log.AppendObj(nil, "SetUserProtect: msgnotshake", value)
		info["msgnotshake"] = value
	}
	if err := req.Parse("nightring", &value); err == nil {
		sm.log.AppendObj(nil, "SetUserProtect: nitgthring", value)
		info["nightring"] = value
	}
	sm.log.AppendObj(nil, "SetUserProtect: ", info)
	err := usercontrol.SetUserProtect(req.Uid, info)
	if err != nil {
		return err
	}
	general.ClearUserProtect(req.Uid)
	return
}

/*
获取用户隐私和防骚扰

URI: /s/user/GetProtect

参数:无

返回值：
	{
		"status": "ok"
		"res":{
			"canfind":1,	//允许附近的人找到我 1表示 不允许 0表示允许 0为默认值
			"chatremind":1,	//私聊提醒 1表示 不允许 0表示允许 0为默认值
			"stranger":1,	//陌生人新消息提醒 1表示 不允许 0表示允许 0为默认值
			"praise":1,		//被点赞提醒 1表示 不允许 0表示允许 0为默认值
			"commit":1,		//被评论提醒 1表示 不允许 0表示允许 0为默认值
			"msgnotring":1,  //消息无提示音,0为有提示音,1为关闭提示音(默认值)
			"msgnotshake":1, //消息无震动,0为有震动(默认值),1为关闭震动
			"nightring":1,   //晚上是否响铃震动,0半夜不响铃(默认值),1为半夜仍响铃
		}
	}
*/
func (sm *UserModule) SecGetProtect(req *service.HttpRequest, result map[string]interface{}) (e error) {
	info, err := usercontrol.GetUserProtect(req.Uid)
	if err != nil {
		return err
	}
	result["res"] = info
	return
}

/*
设置用户头像

URI:/s/user/SetUserAvatar

参数:
	pic : [string] 头像url

返回值：
	{
		"status": "ok"
		"tm":123123123
	}
*/
func (sm *UserModule) SecSetUserAvatar(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var pic string
	if e = req.Parse("pic", &pic); e != nil {
		return
	}

	if ir, e := general.CheckImgByUrl(general.IMGCHECK_SEXY_AND_AD, pic); e == nil && ir.Status != 0 {
		return service.NewError(service.ERR_INTERNAL, "图片审核失败", "图片审核失败")
	} else {
		sm.log.AppendObj(e, "SetUserAvatar checkImg is error", req.Uid, pic, ir)
	}
	e = usercontrol.SetAvatar(req.Uid, pic)
	if e != nil {
		return
	}
	usercontrol.UpdateNickChange(req.Uid)
	usercontrol.CheckVerify(req.Uid, "avatar")
	return
}

/*
获取昵称和头像的最近变更

URI:	/s/user/GetNickChangedList

参数:
	{
		"uidlist": [1001260,1001550,5001111],			//需要更新的UID列表
		"tm": 1446036438								//标识时间戳,此时间之后的变更都返回
		"newuidlist": [1001260,1001550,5001111],		//新增的uid列表
	}

返回值：
	{
		"status": "ok"
		"res":{
			"list":[  //变更后的结果列表 无变更的uid不会在这个列表里出现
				{
					"uid":1001260,
					"nickname":"小白",//昵称
					"avatar":"http://"//头像
				}
			]
		}
	}
*/
func (sm *UserModule) SecGetNickChangedList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tm int64
	if e = req.Parse("tm", &tm); e != nil {
		return
	}
	uidlist := make([]uint32, 0, 0)
	if v, ok := req.Body["uidlist"]; !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "uidlist没找到")
	} else {
		var li []interface{}
		switch value := v.(type) {
		case []interface{}:
			li = value
		default:
			return service.NewError(service.ERR_INVALID_PARAM, "uidlist没找到")
		}
		for _, v := range li {
			if v2 := utils.GetComonFloat64ToUint32(v, 0); v2 != 0 {
				uidlist = append(uidlist, v2)
			}
		}
	}
	newuidlist := make([]uint32, 0, 0)
	if v, ok := req.Body["newuidlist"]; !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "newuidlist没找到")
	} else {
		var li []interface{}
		switch value := v.(type) {
		case []interface{}:
			li = value
		default:
			return service.NewError(service.ERR_INVALID_PARAM, "newuidlist没找到")
		}
		for _, v := range li {
			if v2 := utils.GetComonFloat64ToUint32(v, 0); v2 != 0 {
				newuidlist = append(newuidlist, v2)
			}
		}
	}
	rids, e := usercontrol.GetNickChangedUsers(tm, uidlist)
	if e != nil {
		return
	}
	for _, v := range newuidlist {
		rids = append(rids, v)
	}
	rmap, err := user_overview.GetUserObjects(rids...)

	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	list := make([]map[string]interface{}, 0, 0)
	for k, v := range rmap {
		mp := make(map[string]interface{})
		mp["uid"] = k               //用户ID
		mp["nickname"] = v.Nickname //昵称
		mp["avatar"] = v.Avatar     //头像URL
		list = append(list, mp)
	}
	res := make(map[string]interface{})
	res["list"] = list
	result["res"] = res
	return
}
