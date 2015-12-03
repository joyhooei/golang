package game

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"yf_pkg/net/http"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/data_model/award"
	"yuanfen/yf_service/cls/data_model/coin"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/relation"
	"yuanfen/yf_service/cls/data_model/service_game"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

/*
游戏平台登录接口(通过授权码获取access token)

URL: /game/Login

参数:
	uid: [uint32]用户uid
	appid: [string] 游戏appid
	auth:[string]用户授权码
	tm:[int64]服务器当前时间戳（秒值）
	sign:数字签名
	注意：签名规则 uid+appid+auth+tm+私钥 拼接字符串，然后求md5值

返回值:
	{
		"res": {
			"token": "a86ef3b925008cc8ee122bd93b698c20",  // 授权token，游戏服务器与秋千服务器交互凭证
			"user": {    // 用户信息
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",  // 用户头像
				"diamond": 10,  // 钻石余额
				"gender": 1,	// 性别 1 男 2 女
				"nickname": "小气的豪猪", // 昵称
				"uid": 5000761           // 用户uid
			}
		},
		"status": "ok",
		"tm": 1442643079
	}
*/
func (pm *GameModule) Login(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var appid, auth, sign string
	var tm int64
	if e = req.Parse("uid", &uid, "appid", &appid, "auth", &auth, "tm", &tm, "sign", &sign); e != nil {
		return
	}
	is_ok, g, e := service_game.CheckAppValid(appid)
	if e != nil || !is_ok {
		return service.NewError(service.ERR_INTERNAL, "appid is valid")
	}
	// 检验参数
	para_s := utils.ToString(uid) + appid + auth + utils.ToString(tm) + g.Secret
	if sign != general.Md5(para_s) {
		//		return errors.New("签名验证失败")
	}
	a, e := service_game.GetGameAuth(uid, appid)
	if e != nil {
		return
	}
	if a.Code != auth {
		return service.NewError(service.ERR_INTERNAL, "auth is valid")
	}
	status, er := service_game.CheckGameAuthIsValid(a)
	if status == service_game.GAMEAUTH_STATUS_NOAUTH || status == service_game.GAMEAUTH_STATUS_CODE_TIMEOUT {
		return er
	}
	tx, e := pm.mdb.Begin()
	if e != nil {
		return
	}
	token := service_game.GetAuthCode(uid, appid)
	token_tm := utils.Now.AddDate(0, 0, 1)
	if status == service_game.GAMEAUTH_STATUS_TOKEN_TIMEOUT {
		if e = service_game.UpdateAuthToken(tx, uid, appid, token, token_tm); e != nil {
			tx.Rollback()
			return
		}
	}
	tx.Commit()
	user, e := user_overview.GetUserObject(uid)
	if e != nil {
		return
	}
	diamond, _, e := coin.GetUserCoinInfo(uid)
	if e != nil {
		return
	}
	res := make(map[string]interface{})
	res["token"] = token
	u := make(map[string]interface{})
	u["uid"] = user.Uid
	u["avatar"] = user.Avatar
	u["nickname"] = user.Nickname
	u["gender"] = user.Gender
	u["diamond"] = diamond
	res["user"] = u
	result["res"] = res
	return
}

/*
获取好友列表

URL: /game/Friend

参数:
	uid: [uint32]用户uid
	appid: [string]用户uid
	token:[string]用户授权码
	tm:[int64]服务器当前时间戳（秒值）
	sign:数字签名
	注意：签名验证规则 uid+token+tm+私钥 拼接字符串，然后求md5值

返回值:
	{
		"res": [
		{
			"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
			"gender": 1,
			"nickname": "小气的豪猪",
			"uid": 5000761
		}
		],
		"status": "ok",
		"tm": 1442644499
	}
*/
func (pm *GameModule) Friend(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var token, sign string
	var tm int64
	if e = req.Parse("uid", &uid, "token", &token, "tm", &tm, "sign", &sign); e != nil {
		return
	}
	_, g, er := service_game.CheckAuthToken(uid, token)
	if er.Code != service.ERR_NOERR {
		return er
	}

	// 检验参数
	para_s := utils.ToString(uid) + token + utils.ToString(tm) + g.Secret
	if sign != general.Md5(para_s) {
		//		return errors.New("签名验证失败")
	}
	m, _, e := relation.GetFriendUids(uid, 1, 10000)
	if e != nil {
		return
	}
	uids := make([]uint32, 0, len(m))
	for uid, _ := range m {
		uids = append(uids, uid)
	}
	ulist := make([]map[string]interface{}, 0, 10)
	if len(uids) > 0 {
		um, er := user_overview.GetUserObjects(uids...)
		if er != nil {
			return er
		}
		for uid, user := range um {
			u := make(map[string]interface{})
			u["uid"] = uid
			u["avatar"] = user.Avatar
			u["nickname"] = user.Nickname
			u["gender"] = user.Gender
			ulist = append(ulist, u)
		}
	}
	result["res"] = ulist
	return
}

/*
获取用户账户余额（钻石）

URL: /game/Amount

参数:
	uid: [uint32]用户uid
	token:[string]用户授权码
	tm:[int64]服务器当前时间戳（秒值）
	sign:数字签名
	注意：签名规则 uid+token+tm+私钥 拼接字符串，然后求md5值

返回值:
	{
		"res": {
			"diamond": 10
		},
		"status": "ok",
		"tm": 1442644928
	}
*/
func (pm *GameModule) Amount(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var token, sign string
	var tm int64
	if e = req.Parse("uid", &uid, "token", &token, "tm", &tm, "sign", &sign); e != nil {
		return
	}
	// 验证token
	_, g, er := service_game.CheckAuthToken(uid, token)
	if er.Code != service.ERR_NOERR {
		return er
	}
	// 检验参数
	para_s := utils.ToString(uid) + token + utils.ToString(tm) + g.Secret
	if sign != general.Md5(para_s) {
		//		return errors.New("签名验证失败")
	}
	diamond, _, e := coin.GetUserCoinInfo(uid)
	res := make(map[string]interface{})
	res["diamond"] = diamond
	result["res"] = res
	return
}

/*
钻石变更

URL: /game/AmountUpdate

参数:
	uid: [uint32]用户uid
	token:[string]用户授权码
	tm:[int64]服务器当前时间戳（秒值）
	sign:数字签名
	change:[int]钻石变化量，>0 表示增加 <0 表示减少
	info:[string]描述，如：购买道具
	注意：签名规则 uid+token+change+tm+私钥 拼接字符串，然后求md5值

返回值:
	{
		"res": {     // 返回当前账户变更后余额
			"diamond": 10
		},
		"status": "ok",
		"tm": 1442644928
	}
*/
func (pm *GameModule) AmountUpdate(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var token, sign, info string
	var tm int64
	var change int
	if e = req.Parse("uid", &uid, "token", &token, "tm", &tm, "sign", &sign, "change", &change, "info", &info); e != nil {
		return
	}
	// 验证token
	_, g, er := service_game.CheckAuthToken(uid, token)
	if er.Code != service.ERR_NOERR {
		return er
	}

	// 检验参数
	para_s := utils.ToString(uid) + token + utils.ToString(tm) + g.Secret
	if sign != general.Md5(para_s) {
		//		return errors.New("签名验证失败")
	}
	tx, e := pm.mdb.Begin()
	if e != nil {
		return
	}
	er = coin.UserCoinChange(tx, uid, 0, coin.COST_GAME_PLANE, 0, change, info)
	if er.Code != service.ERR_NOERR {
		tx.Rollback()
		return er
	}
	diamond, e := coin.GetUserCoin(tx, uid)
	if e != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
	res := make(map[string]interface{})
	res["diamond"] = diamond
	result["res"] = res
	return
}

/*
用户喊话

URL: /game/Msg

参数:
	uid: [uint32]用户uid
	token:[string]用户授权码
	tm:[int64]服务器当前时间戳（秒值）
	sign:数字签名
	content:[string]喊话内容
	注意：签名规则 uid+token+content+tm+私钥 拼接字符串，然后求md5值

返回值:
	{
		"status": "ok",
		"tm": 1442644928
	}
*/
func (pm *GameModule) Msg(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var token, sign, content string
	var tm int64
	if e = req.Parse("uid", &uid, "token", &token, "tm", &tm, "sign", &sign, "content", &content); e != nil {
		return
	}
	// 验证token
	_, g, er := service_game.CheckAuthToken(uid, token)
	if er.Code != service.ERR_NOERR {
		return er
	}

	// 检验参数
	para_s := utils.ToString(uid) + token + content + utils.ToString(tm) + g.Secret
	if sign != general.Md5(para_s) {
		//		return errors.New("签名验证失败")
	}
	// 执行发送消息

	return
}

/*
获取游戏奖品列表

URL: /game/Awards

参数:
	appid:[string]游戏appid
	tm:[int64]服务器当前时间戳（秒值）
	sign:数字签名
	注意：签名规则 appid+tm+私钥 拼接字符串，然后求md5值

返回值:
	{
		"res": [
		{
			"id": 18,     // 奖品id
			"img": "http://image1.yuanfenba.net/oss/other/award_iphone6.jpg",  // 奖品图片
			"name": "iphone6", 	// 奖品名称
			"price": 6800   	// 奖品价值
		}
		],
		"status": "ok",
		"tm": 1442646522
	}
*/
func (pm *GameModule) Awards(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var appid, sign string
	var tm int64
	if e = req.Parse("appid", &appid, "sign", &sign, "tm", &tm); e != nil {
		return
	}
	is_ok, g, e := service_game.CheckAppValid(appid)
	if e != nil {
		return
	}
	if !is_ok || g.AppId == "" {
		return errors.New("appid is wrong")
	}
	para_s := appid + utils.ToString(tm) + g.Secret
	// 检验参数
	if sign != general.Md5(para_s) {
		//	return service.NewError(service.ERR_INTERNAL, "签名验证失败")
	}
	awards, e := service_game.GetGameAwardConf(pm.mdb, appid)
	if e != nil {
		return
	}
	if len(awards) <= 0 {
		return
	}
	am, e := award.GetAwardMap()
	if e != nil {
		return
	}

	l := make([]map[string]interface{}, 0, 1)
	for _, a := range awards {
		if ac, ok := am[a.AwardId]; ok {
			item := make(map[string]interface{})
			item["name"] = ac.Name
			item["id"] = a.AwardId
			item["price"] = ac.Price
			item["img"] = ac.Img
			l = append(l, item)
		}
	}
	result["res"] = l
	return
}

/*
用户中奖

URL: /game/AllotAwards

参数:
	appid:"xxx",     // appid
	awardList:"[{uid:xxx,awardid:xxx,count:1,info:xx},{...}]"   // 中奖用户列表 , json 格式
	tm:[int64]服务器当前时间戳（秒值）
	sign:数字签名
	注意：签名规则 awardList+appid+tm+私钥 然后求md5值
	注释：
		awardList: // 中奖用户列表
			uid: 中奖用户
			awardid: 奖品编号id
			count: 数量
			info: 中奖描述（玩xx游戏获得小组第一名）

返回值:
	{
		"status": "ok",
		"tm": 1442646522
	}
*/
func (pm *GameModule) AllotAwards(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var appid, sign, awardList string
	var tm int64
	if e = req.Parse("appid", &appid, "sign", &sign, "tm", &tm, "awardList", &awardList); e != nil {
		return
	}
	is_ok, g, e := service_game.CheckAppValid(appid)
	if e != nil {
		return
	}
	if !is_ok || g.AppId == "" {
		return errors.New("appid is wrong")
	}
	para_s := awardList + appid + utils.ToString(tm) + g.Secret
	// 检验参数
	if sign != general.Md5(para_s) {
		//	return service.NewError(service.ERR_INTERNAL, "签名验证失败")
	}
	fmt.Println("sign: " + sign)
	fmt.Println("get sign: " + general.Md5(para_s))
	awardList_arr := make([]service_game.GameAward, 0, 10)
	if e = json.Unmarshal([]byte(awardList), &awardList_arr); e != nil {
		pm.log.AppendObj(e, "parse awardList is error:")
		return
	}
	tx, e := pm.mdb.Begin()
	if e != nil {
		return
	}
	// 生成中奖信息
	for _, a := range awardList_arr {
		//uid uint32, award_id int, from int, from_ext int, f_uid int, flag int, cnum int
		lastId, er := award.AwardAdd(tx, a.Uid, int(a.AwardId), 1, appid, 0, 0, 0, a.Info)
		if er.Code != service.ERR_NOERR || lastId <= 0 {
			pm.log.AppendObj(nil, "GenAwardRecord插入中奖记录失败", er.Desc, "award_uid:  ", a)
			tx.Rollback()
			return er
		}
	}
	tx.Commit()
	// 异步发送中奖推送
	go service_game.DoMoreAwardPush(awardList_arr)
	return
}

func (pm *GameModule) TestAllotAwards(req *service.HttpRequest, result map[string]interface{}) (e error) {
	awardList := []service_game.GameAward{service_game.GameAward{Uid: 5000761, AwardId: 1, Count: 1, Info: "xxxxx"}, service_game.GameAward{Uid: 5000761, AwardId: 1, Count: 1, Info: "玩游戏"}}

	si, _ := json.Marshal(awardList)
	data := make(map[string]interface{})
	data["appid"] = "qiuqian_sanxiao"
	data["tm"] = 1231231231
	data["awardList"] = string(si)

	parm := string(si) + "qiuqian_sanxiao" + utils.ToString(1231231231) + "qiuqian_game_2015_win"

	data["sign"] = fmt.Sprintf("%x", md5.Sum([]byte(parm)))
	bb, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Printf("-test:sing: --%x \n", md5.Sum([]byte(parm)))
	// 向游戏发送接口
	//HttpSend(host string, path string, params map[string]string, cookies map[string]string, data []byte)
	res, err := http.HttpSend("120.131.64.91:8181", "/game/AllotAwards", nil, nil, bb)
	if err != nil {
		return err
	}
	fmt.Println("---aa--", string(res))
	return
}
