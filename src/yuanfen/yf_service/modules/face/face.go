package face

import (
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/face"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/notify"
)

type FaceModule struct {
}

func (sm *FaceModule) Init(env *service.Env) (err error) {

	return
}

/*
发送小游戏消息

请求URL：s/face/SendGame

参数:

{
	"to":17151,//对方ID
	"content":{
		"game_id":1,		//小游戏ID 1为猜拳 2为骰子
		"game_param":{}		//游戏自定义参数 暂时为空
	}
}

返回结果：

{
	"status": "ok",
	"res":{
		"data": {
			"result":1,	 //小游戏的结果，根据游戏类型返回，目前的两款游戏只有"result"一个参数
			"resultimg":"http://"//小游戏结果图
 		},
		"msgid":111222	//PUSH的消息ID
	}
}
*/
func (sm *FaceModule) SecSendGame(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var to uint32
	var content interface{}
	var game_id uint32
	if err := req.Parse("content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("to", &to, 0); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	switch value := content.(type) {
	case map[string]interface{}:
		if iid, ok := value["game_id"]; ok {
			if i, err := utils.ToUint32(iid); err != nil {
				return service.NewError(service.ERR_INVALID_PARAM, err.Error())
			} else {
				game_id = i
			}

		} else {
			return service.NewError(service.ERR_INVALID_PARAM, "game_id 参数错误")
		}
	default:
		return service.NewError(service.ERR_INVALID_PARAM, "参数错误")
	}
	r, msgid, resultimg, e := face.GameSend(req.Uid, to, game_id)
	if e != nil {
		return e
	}
	res := make(map[string]interface{})
	data := make(map[string]interface{})
	data["resultimg"] = resultimg
	data["result"] = r
	res["data"] = data
	res["msgid"] = msgid
	result["res"] = res
	return
}

/*
表情信息更新

请求URL  face/FaceInfo

参数:

{
	"ver":1001  //客户端表情版本号 初始为0
}

返回结果：

{
	"status": "ok",
	"res":{
		"newver":1002,					//新的版本号 如果为0 表示没有新版表情
		"facelist":[					//表情列表 需要根据男女确定显示
			{							//每个对象为一个分组
				"name":"打招呼"			//分组名称
				"ico":"http://"		//分组图标(男性用户)
				"pic":"http://"		//分组图标（女性用户）
				"type":"bigface"		//表情类型 smallface bigface minigame gift 暂时有四种 smallface的列表需要从资源中另外填充
				"gender":0				//分组所属性别1男 2女 0通用
				"list":[
					{//bigface  对象
						"id":111,				//表情ID
						"name":"在不在",		//表情名称
						"ico":"http://...",	//表情图标url
						"pic":"http://...",		//表情效果图片
						"gender":0				//表情所属性别1男 2女 0通用
					},{...}
				]
			},
			{
				"name":"小游戏"//分组名称
				"ico":"http://"//分组图标
				"type":"minigame"
				"gender":0				//分组所属性别1男 2女 0通用
				"list":[
					{							//minigame  对象
						"id":111,				//表情ID
						"name":"在不在",		//表情名称
						"ico":"http://...",		//图标url
						"pic":"http://...",		//小游戏效果图片
						"res":["http://1,png","http://2.png"],	//小游戏停止图列表，对应结果0-n
						"gender":0				//表情所属性别1男 2女 0通用
					},{...}
				]
			},
			{
				"name":"送礼物"//分组名称
				"ico":"http://"//分组图标
				"type":"gift"
				"gender":1				//分组所属性别1男 2女 0通用
				"list":[]
			},
			{
				"name":"表情"		//分组名称
				"ico":"http://"	//分组图标
				"type":"smallface"
				"gender":0				//分组所属性别1男 2女 0通用
				"list":[]
			},{...}
		]
		"meninput":[  //男性快捷输入 已针对用户性别做好预处理 客户端可以直接用
			{
				"key":"你好".//快捷输入的key
				"list":[1111,1112,1113]//匹配到的表情id
			}
		]
		"mendefault":""//男性默认表情图片 如为空字符串 表示没有默认表情图片
		"womeninput":[  //女性快捷输入 已针对用户性别做好预处理 客户端可以直接用
			{
				"key":"你好".//快捷输入的key
				"list":[1111,1112,1113]//匹配到的表情id
			}
		]
		"womendefault":""//女性默认表情图片 如为空字符串 表示没有默认表情图片
	}
}
*/
func (sm *FaceModule) FaceInfo(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var ver uint32
	if err := req.Parse("ver", &ver); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	r, e := face.GetFace(ver)
	if e != nil {
		return e
	}
	result["res"] = r
	return
}

func (sm *FaceModule) SendTest(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var touid, fromuid uint32
	if touid, e = utils.ToUint32(req.GetParam("touid")); e != nil {
		return e
	}
	if fromuid, e = utils.ToUint32(req.GetParam("fromuid")); e != nil {
		return e
	}

	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_RICHTEXT
	content["folder"] = common.FOLDER_OTHER
	content["tip"] = "登陆奖励金币"
	msgs := make([]map[string]interface{}, 0, 0)
	msg1 := make(map[string]interface{})
	msg1["type"] = "img"
	msg1["img"] = "http://image2.yuanfenba.net/uploads/oss/photo/201507/16/21134387344.jpg"
	msg1["text"] = "嗨购金秋，金币翻倍大放送,充五千送一千。赶快来抢呀"
	but1 := make(map[string]interface{})
	but1["tip"] = ""
	but1["cmd"] = notify.CMD_OPEN_ACCOUNT
	data := make(map[string]interface{})
	but1["data"] = data
	msg1["but"] = but1
	msgs = append(msgs, msg1)

	msg1 = make(map[string]interface{})
	msg1["type"] = "icon"
	msg1["img"] = "http://image1.yuanfenba.net/uploads/oss/avatar/201511/02/19201250400.png"
	msg1["text"] = "每天24张充值卡不间断送"
	but1 = make(map[string]interface{})
	but1["tip"] = ""
	but1["cmd"] = notify.CMD_OPEN_WEB
	data = make(map[string]interface{})
	data["url"] = "http://www.baidu.com/"
	but1["data"] = data
	msg1["but"] = but1
	msgs = append(msgs, msg1)

	msg1 = make(map[string]interface{})
	msg1["type"] = "text"
	msg1["text"] = "玩游戏爆京东卡，每天3个时段玩游戏将随机爆出京东卡，面值50元"
	but1 = make(map[string]interface{})
	but1["tip"] = ""
	but1["cmd"] = notify.CMD_OPEN_WEB
	data = make(map[string]interface{})
	data["url"] = "http://www.jd.com/"
	but1["data"] = data
	msg1["but"] = but1
	msgs = append(msgs, msg1)

	msg1 = make(map[string]interface{})
	msg1["type"] = "button"
	but1 = make(map[string]interface{})
	but1["tip"] = "立即充值"
	but1["cmd"] = notify.CMD_OPEN_ACCOUNT
	data = make(map[string]interface{})
	but1["data"] = data
	msg1["but"] = but1
	msgs = append(msgs, msg1)
	content["msgs"] = msgs

	msgid, e := general.SendMsg(fromuid, touid, content, "")

	if e != nil {
		return e
	}
	res := make(map[string]interface{})
	res["msgid"] = msgid
	res["from"] = fromuid
	res["to"] = touid
	res["content"] = content
	result["res"] = res
	return
}

// func (sm *FaceModule) SendUserTest(req *service.HttpRequest, result map[string]interface{}) (e error) {
// 	var touid, fromuid uint32
// 	if touid, e = utils.ToUint32(req.GetParam("touid")); e != nil {
// 		return e
// 	}
// 	if fromuid, e = utils.ToUint32(req.GetParam("fromuid")); e != nil {
// 		return e
// 	}

// 	content := make(map[string]interface{})
// 	content["type"] = common.MSG_TYPE_RICHTEXT
// 	// content["folder"] = common.FOLDER_OTHER
// 	content["tip"] = "登陆奖励金币"
// 	msgs := make([]map[string]interface{}, 0, 0)
// 	msg1 := make(map[string]interface{})
// 	msg1["type"] = "img"
// 	msg1["img"] = "http://img11.360buyimg.com/cms/jfs/t1303/122/483844403/61560/268819c6/557a8ebdN42b33dd0.jpg"
// 	msg1["text"] = "嗨购金秋，开放购买！无需抢购，欲购从速！！3GB大内存更畅快，4.6万次好评更荣耀！选择下方“移动老用户4G飞享合约”，无需换号，还有话费每月返还！"
// 	but1 := make(map[string]interface{})
// 	but1["tip"] = ""
// 	but1["cmd"] = notify.CMD_OPEN_WEB
// 	data := make(map[string]interface{})
// 	data["url"] = "http://www.baidu.com/"
// 	but1["Data"] = data
// 	msg1["but"] = but1
// 	msgs = append(msgs, msg1)

// 	msg1 = make(map[string]interface{})
// 	msg1["type"] = "icon"
// 	msg1["img"] = "http://img10.360buyimg.com/n9/jfs/t922/347/831102337/123361/4a673ce7/554c74b7N63e81dfb.jpg"
// 	msg1["text"] = "上午十一点买的，下午就收到了，京东快递员很负责，快递包装有点简单，里面要是包裹几层泡沫纸会更安全，手机运行很流畅，用了一天，没有卡顿的现象，对手机要求不高，黑色看起来很稳重，性价比很高。用京东白条付款减50，很划算。满意！"
// 	but1 = make(map[string]interface{})
// 	but1["tip"] = ""
// 	but1["cmd"] = notify.CMD_OPEN_WEB
// 	data = make(map[string]interface{})
// 	data["url"] = "http://www.baidu.com/"
// 	but1["Data"] = data
// 	msg1["but"] = but1
// 	msgs = append(msgs, msg1)

// 	msg1 = make(map[string]interface{})
// 	msg1["type"] = "text"
// 	msg1["text"] = "今天小米4C发布了，感觉上性价比比这款还高点了，华为这款手机可能也该停产了，荣耀7的外观没这款靓，特别是黑色的，普通使用还是够了。另外，听说在安卓系统里，关后台程序是以应用程序里能关闭，才最彻底，为啥不做个专门管理的快捷开关。上传的照片自己刚贴了膜，贴的不太好。 　"
// 	msgs = append(msgs, msg1)

// 	msg1 = make(map[string]interface{})
// 	msg1["type"] = "text"
// 	msg1["text"] = "今天小米4C发布了，感觉上性价比比这款还高点了，华为这款手机可能也该停产了，荣耀7的外观没这款靓，特别是黑色的，普通使用还是够了。另外，听说在安卓系统里，关后台程序是以应用程序里能关闭，才最彻底，为啥不做个专门管理的快捷开关。上传的照片自己刚贴了膜，贴的不太好。 　"
// 	but1 = make(map[string]interface{})
// 	but1["tip"] = ""
// 	but1["cmd"] = notify.CMD_OPEN_WEB
// 	data = make(map[string]interface{})
// 	data["url"] = "http://www.baidu.com/"
// 	but1["Data"] = data
// 	msg1["but"] = but1
// 	msgs = append(msgs, msg1)

// 	msg1 = make(map[string]interface{})
// 	msg1["type"] = "icon"
// 	msg1["img"] = "http://img10.360buyimg.com/n9/jfs/t922/347/831102337/123361/4a673ce7/554c74b7N63e81dfb.jpg"
// 	msg1["text"] = "上午十一点买的，下午就收到了，京东快递员很负责，快递包装有点简单，里面要是包裹几层泡沫纸会更安全，手机运行很流畅，用了一天，没有卡顿的现象，对手机要求不高，黑色看起来很稳重，性价比很高。用京东白条付款减50，很划算。满意！"
// 	msgs = append(msgs, msg1)

// 	msg1 = make(map[string]interface{})
// 	msg1["type"] = "img"
// 	msg1["img"] = "http://img11.360buyimg.com/cms/jfs/t1303/122/483844403/61560/268819c6/557a8ebdN42b33dd0.jpg"
// 	msg1["text"] = "嗨购金秋，开放购买！无需抢购，欲购从速！！3GB大内存更畅快，4.6万次好评更荣耀！选择下方“移动老用户4G飞享合约”，无需换号，还有话费每月返还！"

// 	msgs = append(msgs, msg1)

// 	msg1 = make(map[string]interface{})
// 	msg1["type"] = "button"
// 	but1 = make(map[string]interface{})
// 	but1["tip"] = "点击打开"
// 	but1["cmd"] = notify.CMD_OPEN_WEB
// 	data = make(map[string]interface{})
// 	data["url"] = "http://www.baidu.com/"
// 	but1["Data"] = data
// 	msg1["but"] = but1
// 	msgs = append(msgs, msg1)
// 	content["msgs"] = msgs

// 	msgid, e := general.SendMsg(fromuid, touid, content, "")

// 	if e != nil {
// 		return e
// 	}
// 	res := make(map[string]interface{})
// 	res["msgid"] = msgid
// 	res["from"] = fromuid
// 	res["to"] = touid
// 	res["content"] = content
// 	result["res"] = res
// 	return
// }

/*
发送大表情 SecSendBigface

请求URL：s/face/SendBigface

参数:

{
	"to":17151,//对方ID
	"content":{
		"id":1112221 //表情ID
	}
}

返回结果：

{
	"status": "ok",
	"res":{
		"msgid":111222//PUSH的消息ID
	}
}
*/
func (sm *FaceModule) SecSendBigface(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var to uint32
	// var tag string
	var content interface{}
	var faceid uint32
	if err := req.Parse("content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("to", &to, 0); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	switch value := content.(type) {
	case map[string]interface{}:
		if iid, ok := value["id"]; ok {
			if i, err := utils.ToUint32(iid); err != nil {
				return service.NewError(service.ERR_INVALID_PARAM, err.Error())
			} else {
				faceid = i
			}

		} else {
			return service.NewError(service.ERR_INVALID_PARAM, "gid 参数错误")
		}
	default:
		return service.NewError(service.ERR_INVALID_PARAM, "参数错误")
	}
	// var to, faceid uint32
	// if err := req.Parse("to", &to, "faceid", &faceid); err != nil {
	// 	return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	// }
	msgid, e := face.SendBigFace(req.Uid, to, faceid)
	if e != nil {
		return e
	}
	res := make(map[string]interface{})
	res["msgid"] = msgid
	result["res"] = res
	return
}

func (sm *FaceModule) TestBigface(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var touid, fromuid uint32
	if touid, e = utils.ToUint32(req.GetParam("touid")); e != nil {
		return e
	}
	if fromuid, e = utils.ToUint32(req.GetParam("fromuid")); e != nil {
		return e
	}

	msgid, e := face.SendBigFace(fromuid, touid, 10001)
	if e != nil {
		return e
	}
	res := make(map[string]interface{})
	res["msgid"] = msgid
	result["res"] = res

	return
}
