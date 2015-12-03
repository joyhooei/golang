package user

import (
	"fmt"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/coin"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/data_model/usercontrol"
	"yuanfen/yf_service/cls/notify"
	// "yuanfen/yf_service/cls/unread"
)

// //兑换金币
// func (sm *UserModule) SecGiftExchange(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
// 	giftid := utils.GetComonFloat64ToInt(req.Body["gid"], 0)
// 	if giftid == 0 {
// 		return service.NewError(service.ERR_INVALID_PARAM, "礼物ID错误")
// 	}
// 	earn, err := usercontrol.GiftExchange(req.Uid, giftid)
// 	if err.Code != service.ERR_NOERR {
// 		return err
// 	}
// 	res := make(map[string]interface{})
// 	res["earn"] = earn //兑换成功获得的金币
// 	result["res"] = res
// 	if c, _, err := coin.GetUserCoinInfo(req.Uid); err == nil {
// 		result[common.USER_BALANCE] = c
// 		result[common.USER_BALANCE_CHANGE] = fmt.Sprintf("兑换礼物获得 %v金币", earn)
// 	}
// 	not, err2 := notify.GetNotify(req.Uid, notify.NOTIFY_COIN, nil, "系统消息", fmt.Sprintf("兑换礼物获得 %v金币", earn), req.Uid)
// 	if err2 == nil {
// 		result[notify.NOTIFY_KEY] = not
// 	} else {
// 		return service.NewError(service.ERR_INTERNAL, err2.Error(), "兑换礼物错误:"+err2.Error())
// 	}
// 	return
// }

/*
送给我的礼物记录
请求URL：s/user/GiftList
参数: {"cur":0,“ps”: 10} //分页页码

返回结果：
{
	"status": "ok"
	"res": {
		"list":[
			{
			"id":11114,  //记录ID 用于领取礼物
			"fromuser":{用户对象: uid,avatar, nickname, grade,gender}
			"gid":1111,  //物品ID
			"n": "玫瑰花", //名称
			"info": "一束玫瑰花", //描述
			"img":"/photo/1810339779.jpg",  //图片URL
			"time":,	//送礼时间
     		"stat":1  //送礼状态（0：待处理，1：已接受}
			},
			{ … }
   			]
			“pages”: { …分页结构}
	}
}
*/
func (sm *UserModule) SecGiftList(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	cur := utils.GetComonFloat64ToInt(req.Body["cur"], 1)
	ps := utils.GetComonFloat64ToInt(req.Body["ps"], 10)
	list, total, err := usercontrol.GiftList(req.Uid, cur, ps)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	rlist := make([]map[string]interface{}, 0, 0)
	for _, v1 := range list {
		vv := make(map[string]interface{})
		for k, v := range v1 {
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
		uid, e := utils.StringToUint32(v1["uid"])
		if e != nil {
			continue
		}
		obj, e := user_overview.GetUserObjects(uid)
		if e != nil {
			continue
		}
		r, ok := obj[uid]
		if !ok {
			continue
		}
		uitem := make(map[string]interface{})
		uitem["gender"] = r.Gender
		uitem["nickname"] = r.Nickname
		uitem["grade"] = r.Grade
		uitem["avatar"] = r.Avatar
		uitem["uid"] = uid
		delete(v1, "uid")
		vv["fromuser"] = uitem
		rlist = append(rlist, vv)
	}
	res := make(map[string]interface{})
	res["list"] = rlist
	res["pages"] = utils.PageInfo(total, cur, ps)
	result["res"] = res
	// unread.UpdateReadTime(req.Uid, common.UNREAD_GIFT)
	// un := make(map[string]interface{})
	// un[common.UNREAD_GIFT] = 0
	// unread.GetUnreadNum(req.Uid, un)
	// result[common.UNREAD_KEY] = un
	return
}

/*
领取礼物 ,(同时收礼答谢，兑换钻石) SecGiftReceive
请求URL：s/user/GiftReceive

参数:

{
	"tag":""//tag
	"to":17151
	"content":{
		"type":"thx_present",
		"gift_record_id":11114  //送礼记录ID
	}
}

返回结果：

{
	"status": "ok",
	"res":{
		"msgid":"32234",//消息ID 收礼答谢消息的ID
	}
}
*/
// func (sm *UserModule) SecGiftReceive(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
// 	var to uint32
// 	var tag string
// 	var content interface{}
// 	var giftid int
// 	// var msg string
// 	if err := req.Parse("content", &content); err != nil {
// 		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
// 	}
// 	if err := req.ParseOpt("to", &to, 0, "tag", &tag, ""); err != nil {
// 		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
// 	}
// 	switch value := content.(type) {
// 	case map[string]interface{}:
// 		if iid, ok := value["gift_record_id"]; ok {
// 			if i, err := utils.ToInt(iid); err != nil {
// 				return service.NewError(service.ERR_INVALID_PARAM, err.Error())
// 			} else {
// 				giftid = i
// 			}
// 		} else {
// 			return service.NewError(service.ERR_INVALID_PARAM, "gift_record_id 参数错误")
// 		}
// 	default:
// 		return service.NewError(service.ERR_INVALID_PARAM, "参数错误")
// 	}
// 	msgid, price, err := usercontrol.GiftReceive(req.Uid, giftid, tag)
// 	if err != nil {
// 		return service.NewError(service.ERR_INTERNAL, err.Error())
// 	}

// 	if c, _, err := coin.GetUserCoinInfo(req.Uid); err == nil {
// 		result[common.USER_BALANCE] = c
// 		result[common.USER_BALANCE_CHANGE] = fmt.Sprintf("收取礼物得到 %v钻石", price)
// 	}
// 	res := make(map[string]interface{})
// 	res["msgid"] = msgid
// 	result["res"] = res
// 	un := make(map[string]interface{})
// 	un[common.UNREAD_GIFT] = 0
// 	unread.GetUnreadNum(req.Uid, un)
// 	result[common.UNREAD_KEY] = un
// 	noti, err2 := notify.GetNotify(req.Uid, notify.NOTIFY_COIN, nil, "系统消息", fmt.Sprintf("收取礼物得到 %v钻石", price), req.Uid)
// 	if err2 == nil {
// 		result[notify.NOTIFY_KEY] = noti
// 	} else {
// 		return service.NewError(service.ERR_INTERNAL, err2.Error(), "兑换礼物错误:"+err2.Error())
// 	}
// 	return
// }

/*
拒绝礼物 SecGiftReject

请求URL：s/user/GiftReject

参数:

{
	"gift_record_id":11114  //送礼记录ID
}

返回结果：

{
	"status": "ok",
	"res":{
	}
}
*/
// func (sm *UserModule) SecGiftReject(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
// 	var giftid int
// 	if err := req.Parse("gift_record_id", &giftid); err != nil {
// 		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
// 	}
// 	_, err := usercontrol.GiftReject(req.Uid, giftid, "")
// 	if err != nil {
// 		return service.NewError(service.ERR_INTERNAL, err.Error())
// 	}
// 	return
// }

// //收取所有礼物
// func (sm *UserModule) SecGiftReceiveAll(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
// 	if err := usercontrol.GiftReceiveAll(req.Uid); err != nil {
// 		return service.NewError(service.ERR_INTERNAL, err.Error(), "请求错误")
// 	}
// 	res := make(map[string]interface{})
// 	result["res"] = res
// 	un := make(map[string]interface{})
// 	un[common.UNREAD_GIFT] = 0
// 	unread.GetUnreadNum(req.Uid, un)
// 	result[common.UNREAD_KEY] = un
// 	return
// }

/*
给对方送礼 SecGiftSend
请求URL：s/user/GiftSend
参数:

{
	"tag":""//tag
	"to":17151
	"content":{
		"type":"give_present",
		"gid":13552, //要送的礼物ID
	}
}
返回:

{
	"status": "ok",
	"res":{
		"result":1,				//送礼结果，1：成功，0：失败
		"gift_record_id":10005;	//[opt]返回送礼记录ID 。（如果送礼失败，无此字段）
		"msgid":111111，		//[opt]消息ID 用于即时通讯。（如果送礼失败，无此字段）
		"level":2				//[opt]返回能送礼的最大等级。（如果送礼失败，无此字段）
		"tip_msg":{    				//[opt]小提示（如果送礼失败，包含此结构）
			"type":"hint",			//消息类型
			"content": "xxxx",   			//消息内容,可包含换行符(表示换行)
			"msgid": 50001,   			    //发送失败的消息id
			"but":{  						//超链接以及点击效果，如没有此字段则无超链接以及点击特效
				"tip":string		// 按钮提示文字信息
				"cmd":string		// 按钮执行命令
				"def":true      	// 是否为默认事件按钮 消息中无作用
				"Data":{} 			// cmd命令执行所需参数
		     },
			"tm":"2015-08-17T15:01:07+08:00"

			},
		 "cost":120         //(result=1 才会有) 送礼成功，花费钻石数量

	}
	}
	送礼等级不够(赠送未解锁礼物)的情况下：
	{
		"res": {
			"result": 0,
			"tip_msg": {
				"content": "礼物尚未解锁，请先赠送前面的礼物",
				"type": "hint"
				"tm":"2015-08-17T15:01:07+08:00"
			}
		},
		"status": "ok",
		"tm": 1446890978
	}

}
*/
func (sm *UserModule) SecGiftSend(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var to uint32
	var tag string
	var content interface{}
	var gid int
	if err := req.Parse("content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("to", &to, 0, "tag", &tag, ""); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	switch value := content.(type) {
	case map[string]interface{}:
		if iid, ok := value["gid"]; ok {
			if i, err := utils.ToInt(iid); err != nil {
				return service.NewError(service.ERR_INVALID_PARAM, err.Error())
			} else {
				gid = i
			}

		} else {
			return service.NewError(service.ERR_INVALID_PARAM, "gid 参数错误")
		}
	default:
		return service.NewError(service.ERR_INVALID_PARAM, "参数错误")
	}
	if to < 1000000 {
		res := make(map[string]interface{})
		res["result"] = 0
		notify.AddTipMsg(res, "无法向目标发送送礼", "", "", nil)
		e.Code = service.ERR_NOERR
		result["res"] = res
		//usercontrol.PushInvalidMsg(req.Uid, to)
		//return service.NewError(service.ERR_INTERNAL, "无法向目标发送此类消息", "无法向目标发送此类消息")
		return
	}
	id, msgid, price, e := usercontrol.GiftSend(req.Uid, to, gid, tag)
	if e.Code != service.ERR_NOERR {
		if e.Code == service.ERR_NOT_ENOUGH_MONEY {
			res := make(map[string]interface{})
			res["result"] = 0
			notify.AddTipMsg(res, "你的余额不足，请充值。", "点击立即充值", notify.CMD_OPEN_ACCOUNT, nil)
			e.Code = service.ERR_NOERR
			result["res"] = res
			return
		} else if e.Code == service.ERR_PERMISSION_DENIED {
			res := make(map[string]interface{})
			res["result"] = 0
			notify.AddTipMsg(res, "礼物尚未解锁，请先赠送前面的礼物", "", "", nil)
			e.Code = service.ERR_NOERR
			result["res"] = res
			return
		}
		return e
	}
	usercontrol.StatGift(req.Uid)
	if c, _, err := coin.GetUserCoinInfo(req.Uid); err == nil {
		result[common.USER_BALANCE] = c
		result[common.USER_BALANCE_CHANGE] = fmt.Sprintf("赠送礼物花费 %v钻石", price)
	}
	rl, err := usercontrol.GiftLevel(req.Uid, to)
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	res := make(map[string]interface{})
	res["cost"] = price
	res["result"] = 1
	res["gift_record_id"] = id
	res["msgid"] = msgid
	res["level"] = rl
	result["res"] = res
	return
}

//我的未兑换礼物
// func (sm *UserModule) SecGiftInfo(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {

// 	v, err := usercontrol.GiftInfo(req.Uid)
// 	if err != nil {
// 		return service.NewError(service.ERR_INTERNAL, err.Error())
// 	}
// 	res := make(map[string]interface{})
// 	res["info"] = v
// 	result["res"] = res
// 	return
// }

/*
可购买的礼物列表

请求URL：user/GiftShop

无参数

{
	"status": "ok",
	"res": {
		"info":[
			{
			"gid":13552, 					//礼物ID
			"n": "玫瑰花", 					//名称
			"info": "一束玫瑰花", 			//描述
			"img":"/photo/1810339779.jpg",  //略缩图URL
			"price":90,						//购买者花费的钻石
			"earn":10,						//可获取钻石
			"level":1，						//需要礼物等级 0-6
			"res":"http://res1.zip"			//礼物资源图片包
			},{ … }
   		]
	}
}
*/
func (sm *UserModule) GiftShop(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {

	v, err := usercontrol.GiftShop()
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["info"] = v
	result["res"] = res
	return
}

/*
给对方能送礼的等级 SecGiftLevel
请求URL：s/user/GiftLevel
参数:

{
	"to":17151
}

返回:

{
	"status": "ok",
	"res":{
		"level":0// 返回能送礼的最大等级
	}
}
*/
func (sm *UserModule) SecGiftLevel(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var to uint32
	if err := req.Parse("to", &to); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	rl, err := usercontrol.GiftLevel(req.Uid, to)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	res["level"] = rl
	result["res"] = res
	return
}

//测试送礼消息
func (sm *UserModule) TestGiftMsg(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var touid, fromuid uint32
	if touid, e = utils.ToUint32(req.GetParam("touid")); e != nil {
		return e
	}
	if fromuid, e = utils.ToUint32(req.GetParam("fromuid")); e != nil {
		return e
	}
	mid, e := usercontrol.GiftTest(fromuid, touid, 25, "")
	if e != nil {
		return
	}
	res := make(map[string]interface{})
	res["mid"] = mid
	result["res"] = res
	return
}
