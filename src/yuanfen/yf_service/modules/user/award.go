package user

import (
	"fmt"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/coin"
	"yuanfen/yf_service/cls/data_model/usercontrol"
	"yuanfen/yf_service/cls/unread"
)

/*
SecAwardList 我的奖品列表

请求URL：s/user/AwardList

参数:

{
	"type":1,//类型 0为实体物品 1 为虚拟物品包括金币飞行次数
	"cur":1,
	"ps": 10
}

描述 pn:页码

返回结果:

{
	"status": "ok",
	"res":{
	"list":[
		{
			"id":11114,  //记录ID
			"name":"IPHONE手机"//奖品名称
			"type": 1, //物品类型 1：Y币 ，2：实物 ，3：充值卡  待扩展	"img":"/photo/1810339779.jpg",  //图片URL
			"price":600000,//价值
			"info":"土豪金"//描述
			"tm":,//中奖时间
			"from":1,//奖品来源 1 游戏 2： 挖宝 3：抽奖4虚拟奖品
     		"status":1    //状态 奖品状态：0 待领取 1. 已领取 2.等待充值（发货） 3：已完成
			"log_id":1233,//发货记录ID，用来查看发货记录
			"frominfo":"打飞机游戏获得"//奖品来源描述
		},
		{ … }
   	]
		"pages":{…}//分页结构
}
}
*/
func (sm *UserModule) SecAwardList(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	cur := utils.GetComonFloat64ToInt(req.Body["cur"], 1)
	ps := utils.GetComonFloat64ToInt(req.Body["ps"], 10)
	itype := 0
	_, ok := req.EnsureBody("type")
	if ok {
		itype, _ = utils.ToInt(req.Body["type"])
	}
	list, total, err := usercontrol.GetAwardLog(itype, req.Uid, cur, ps)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})

	res["list"] = list
	res["pages"] = utils.PageInfo(total, cur, ps)
	result["res"] = res
	return
}

/*
SecAwardItem 领取实物奖品

此请求会在用户填完地址 确认领奖后提交

请求URL：s/user/AwardItem

参数

{
	"id":13552，//领奖ID 对应 奖品记录ID，奖品列表里的id
	"addrid":112323  //地址ID 对应收货地址的ID
}

返回结果:

{
	"status": "ok",
	"res":{
		"info":"我们将在三天内发货,敬请等待"
	}
}
*/
func (sm *UserModule) SecAwardItem(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("id", "addrid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	id, err := utils.ToInt(req.Body["id"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "id")
	}
	addrid, err := utils.ToInt(req.Body["addrid"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "addrid")
	}
	err = usercontrol.AwardItem(req.Uid, id, addrid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	un := make(map[string]interface{})
	un[common.UNREAD_MYAWARD] = 0
	unread.GetUnreadNum(req.Uid, un)
	result[common.UNREAD_KEY] = un
	res := make(map[string]interface{})
	res["info"] = "我们将在三天内发货,敬请等待"
	result["res"] = res
	return
}

/*
SecAwardPhone 领取实物奖品

此请求会在用户填完地址 确认领奖后提交

请求URL：s/user/AwardPhone

参数

{
	"id":13552，//领奖ID 对应 奖品记录ID，奖品列表里的id
	"phone":"13122220000"//充值手机

}

返回结果:

{
	"status": "ok",
	"res":{
		"info":"我们将在三天内发货,敬请等待"
	}
}
*/
func (sm *UserModule) SecAwardPhone(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var id int
	var phone string
	if v, ok := req.Body["id"]; ok {
		iid, err := utils.ToInt(v)
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, err.Error())
		} else {
			id = iid
		}
	} else {
		return service.NewError(service.ERR_INVALID_PARAM, "参数错误")
	}
	if v, ok := req.Body["phone"]; ok {
		phone = utils.ToString(v)
	} else {
		return service.NewError(service.ERR_INVALID_PARAM, "参数错误")
	}

	err := usercontrol.AwardPhone(req.Uid, id, phone)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["info"] = "我们将在今天为您充值,敬请等待"
	result["res"] = res
	return
}

//领取金币奖品
func (sm *UserModule) SecAwardCoin(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var id int
	if v, ok := req.Body["id"]; ok {
		iid, err := utils.ToInt(v)
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, err.Error())
		} else {
			id = iid
		}
	} else {
		return service.NewError(service.ERR_INVALID_PARAM, "参数错误")
	}

	price, err := usercontrol.AwardCoin(req.Uid, id)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if c, _, err := coin.GetUserCoinInfo(req.Uid); err == nil {
		result[common.USER_BALANCE] = c
		result[common.USER_BALANCE_CHANGE] = fmt.Sprintf("兑换奖品获得 %v金币", price)
	}
	un := make(map[string]interface{})
	un[common.UNREAD_MYAWARD] = 0
	unread.GetUnreadNum(req.Uid, un)
	result[common.UNREAD_KEY] = un
	return
}

//领取虚拟奖品
func (sm *UserModule) SecAwardVirtual(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var id int
	if v, ok := req.Body["id"]; ok {
		iid, err := utils.ToInt(v)
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, err.Error())
		} else {
			id = iid
		}
	} else {
		return service.NewError(service.ERR_INVALID_PARAM, "参数错误")
	}

	rmap, err := usercontrol.AwardVirtual(req.Uid, id)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	for k, v := range rmap {
		result[k] = v
	}

	/*	un := make(map[string]interface{})
		un[common.UNREAD_PLANE_FREE] = 0
		un[common.UNREAD_MYAWARD] = 0
		unread.GetUnreadNum(req.Uid, un)
		result[common.UNREAD_KEY] = un
	*/
	return
}

func (sm *UserModule) SecAwardTrans(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var log_id int
	if v, ok := req.Body["log_id"]; ok {
		i, err := utils.ToInt(v)
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, err.Error())
		} else {
			log_id = i
		}
	} else {
		return service.NewError(service.ERR_INVALID_PARAM, "参数错误")
	}

	data, err := usercontrol.AwardTrans(req.Uid, log_id)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	for k, v := range data {
		res[k] = v
	}
	result["res"] = res
	return
}

/*
SecAwardInfo 查看奖品发货信息

请求URL：s/user/AwardInfo

{
	"id":13552,//奖品ID
}

返回值：

{
	"status": "ok",
	"res":{
		"type":1, //物品类型 1：Y币 ，2：实物 ，3：充值卡  待扩展
		"status":1,    //状态 奖品状态：0 待领取 1. 已领取 2.等待充值（发货） 3：已完成
	 	"charge_phone":"1382222222",//手机充值号手机充值卡有此字段，实物包含后面字段
		"address":"",//发货地址
		"address_phone":"13811111111",//收货手机号
		"address_name":"XXX",//收货人姓名
		"name":"中通快递",//物流公司
	 	"num":"1123232",//物流单号
		"tm"://发货时间
	}
}
*/
func (sm *UserModule) SecAwardInfo(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("id")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	id, err := utils.ToInt(req.Body["id"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "id")
	}
	data, err := usercontrol.AwardInfo(req.Uid, id)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	result["res"] = data
	return
}
