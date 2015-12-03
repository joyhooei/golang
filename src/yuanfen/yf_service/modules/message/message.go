package message

import (
	"errors"
	"fmt"
	"time"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/push"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/hongniang"
	msg "yuanfen/yf_service/cls/data_model/message"
	"yuanfen/yf_service/cls/data_model/relation/base"
	"yuanfen/yf_service/cls/message"
)

type MessageModule struct {
	log   *log.MLogger
	mdb   *mysql.MysqlDB
	rdb   *redis.RedisPool
	cache *redis.RedisPool
}

var special = map[uint32]utils.Coordinate{}

func (sm *MessageModule) initSpecialUsers() {
	for {
		rows, e := sm.mdb.Query("select uid,lat,lng from special_user")
		if e != nil {
			fmt.Println("init special_user error:", e.Error())
			return
		}
		defer rows.Close()
		for rows.Next() {
			var uid uint32
			var lat, lng float64
			if e = rows.Scan(&uid, &lat, &lng); e != nil {
				fmt.Println("init special_user error:", e.Error())
				break
			}
			special[uid] = utils.Coordinate{lat, lng}
		}
		time.Sleep(10 * time.Second)
	}
}

func (sm *MessageModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds

	go sm.initSpecialUsers()
	return
}

/*
SecSend发送消息

URI: s/message/Send

参数:
		{
			"to":345,	//接收者uid
			"tag":"game",	//标签
			"content":{}	//消息内容，具体格式参见common包
		}
返回值:

	{
		"res": {
			"msgid": 25694
		},
		"status": "ok",
		"tm": 1443071652
	}
*/
func (sm *MessageModule) SecSend(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var to uint32
	var tag string
	var content interface{}
	if err := req.Parse("content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("to", &to, 0, "tag", &tag, ""); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	res := make(map[string]interface{})
	var msgid uint64
	switch to {
	case 0: //Tag消息
		if msgid, e = msg.SendTag(req.Uid, tag, content, res); e != nil {
			return
		}
	case common.USER_HONGNIANG:
		if msgid, e = hongniang.SendToHongniang(req.Uid, content); e != nil {
			return
		}
	default:
		if bl, e := base.IsInBlacklist(to, req.Uid); e != nil {
			return e
		} else if bl {
			return service.NewError(service.ERR_IN_BLACKLIST, "", "")
		}

		if msgid, e = msg.SendUser(req.Uid, to, tag, content, res); e != nil {
			return
		}
	}
	res["msgid"] = msgid
	result["res"] = res
	return
}

/*
Notify来自push服务的用户状态变化通知

URI: s/message/Notify

参数:
		{
			"type":"location",	//通知类型。location-位置变化，需要传uid,lat,lng；online-用户上线，需要传uid；offline-下线，需要传uid。
			"uid":123123,
			"lat":45.22,
			"lng":23.11
		}
返回值:

	{
		"status": "ok",
		"tm": 1443071652
	}
*/
func (sm *MessageModule) Notify(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var typ string
	if err := req.Parse("uid", &uid, "type", &typ); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	switch typ {
	case "offline":
		message.SendMessage(message.OFFLINE, message.Offline{uid}, result)
	case "online":
		message.SendMessage(message.ONLINE, message.Online{uid}, result)
	case "location":
		var lat, lng float64
		if pos, ok := special[uid]; ok {
			lat, lng = pos.Lat, pos.Lng
		} else {
			lat, e = utils.ToFloat64(req.Body["lat"])
			if e != nil {
				return service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("parse lat=%v to float64 error :%v", req.Body["lat"], e.Error()))
			}
			lng, e = utils.ToFloat64(req.Body["lng"])
			if e != nil {
				return service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("parse lng=%v to float64 error :%v", req.Body["lng"], e.Error()))
			}
		}
		message.SendMessage(message.LOCATION_CHANGE, message.LocationChange{uid, lat, lng}, result)
	}

	return
}
func (sm *MessageModule) SecCustomPush(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	var targets []string
	var mode int
	var title, by, desc string
	var content map[string]interface{}
	if err := req.Parse("title", &title, "by", &by, "mode", &mode, "desc", &desc, "content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("targets", &targets, []string{}); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if e = push.SendThirdPartyDirect(mode, by, targets, title, content, desc); e != nil {
		return e
	}
	res := make(map[string]interface{})
	result["res"] = res
	return
}

/*
SecRecentMessages获取最近收到的消息

如果是新装的客户端，则只返回最近的私聊消息，同时也会返回我发送的私聊消息

URI: s/message/RecentMessages

参数:
		{
			"last_msgid":123,	//上次收到的最后一条消息的ID，如果不填或者为0，则认为是新装的客户端
		}
返回值:

	{
		"res": {
			"msgs": [
			{
				"content": {
					"content": "",
					"imei": "\u003cnil\u003e",
					"sys_type": "relogin",
					"tm": "2015-07-27T11:31:47+08:00",
					"type": "text"
				},
				"from": 1,
				"msgid": 25694,
				"tm": "2015-07-27T11:31:47+08:00",
				"to": 5000779
			}
			]
		},
		"status": "ok",
		"tm": 1443071652
	}
*/
func (sm *MessageModule) SecRecentMessages(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var last_msgid uint64
	if err := req.Parse("last_msgid", &last_msgid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	types := []string{}
	if last_msgid == 0 {
		types = common.ChatMessageTypes
	}
	msgs, err := msg.RecentMessages(req.Uid, types, last_msgid, 200)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("query offline messages error :%v", err.Error()))
	}

	res := make(map[string]interface{})
	res["msgs"] = msgs
	result["res"] = res
	return
}

/*
func (sm *MessageModule) SecTagMessages(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var count uint32
	var last_msgid, from_msgid uint64
	var tag string
	var types []string
	if err := req.Parse("tag", &tag, "count", &count, "last_msgid", &last_msgid, "from_msgid", &from_msgid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("types", &types, []string{}); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	msgs, err := msg.TagMessages(req.Uid, tag, types, last_msgid, from_msgid, count)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("query tag messages error :%v", err.Error()))
	}

	res := make(map[string]interface{})
	res["msgs"] = msgs
	result["res"] = res
	return
}
*/

func (sm *MessageModule) SecRecentTagMessages(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var last_msgid uint64
	var tag string
	types := []string{common.MSG_TYPE_GAME_INVITE, common.MSG_TYPE_TEXT, common.MSG_TYPE_VOICE, common.MSG_TYPE_PIC}
	if err := req.Parse("tag", &tag, "last_msgid", &last_msgid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	msgs, err := msg.RecentTagMessages(req.Uid, tag, types, last_msgid, 20)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("query tag messages error :%v", err.Error()))
	}

	res := make(map[string]interface{})
	res["msgs"] = msgs
	result["res"] = res
	return
}

func (sm *MessageModule) SecDel(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var msgid uint64
	var tag string
	if err := req.Parse("msgid", &msgid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("tag", &tag, ""); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	e = msg.Del(req.Uid, tag, msgid)
	if e != nil {
		return e
	}
	res := make(map[string]interface{})
	result["res"] = res
	return
}

/*
SecGetMessage根据消息ID获取消息内容

URI: s/message/GetMessage

参数:
		{
			"msgid":123,	//消息ID
		}
返回值:

	{
		"res": {
			"content": {
				"content": "",
				"imei": "\u003cnil\u003e",
				"sys_type": "relogin",
				"tm": "2015-07-27T11:31:47+08:00",
				"type": "text"
			},
			"from": 1,
			"msgid": 25694,
			"tm": "2015-07-27T11:31:47+08:00",
			"to": 5000779
		},
		"status": "ok",
		"tm": 1443071652
	}
*/
func (sm *MessageModule) SecGetMessage(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var msgid uint64
	if err := req.Parse("msgid", &msgid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	message, err := msg.GetMessage(req.Uid, msgid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("query offline messages error :%v", err.Error()))
	}

	result["res"] = message
	return
}

/*
SecAdminSend管理员发送消息的接口

URI: s/message/AdminSend

参数:
		{
			"from":123,	//发送者uid
			"to":345,	//接收者uid
			"tag":"game",	//标签
			"content":{}	//消息内容，具体格式参见common包
		}
返回值:

	{
		"res": {
			"msgid": 25694
		},
		"status": "ok",
		"tm": 1443071652
	}
*/
func (sm *MessageModule) SecAdminSend(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	var from, to uint32
	var tag string
	var content interface{}
	if err := req.Parse("content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("from", &from, 1, "to", &to, 0, "tag", &tag, ""); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	res := make(map[string]interface{})
	var msgid uint64
	switch to {
	case 0: //Tag消息
		if msgid, e = msg.SendTag(from, tag, content, res); e != nil {
			return
		}
	case common.USER_HONGNIANG:
		if msgid, e = hongniang.SendToHongniang(from, content); e != nil {
			return
		}
	default:
		if msgid, e = msg.SendUser(from, to, tag, content, res); e != nil {
			return
		}
	}
	res["msgid"] = msgid
	result["res"] = res
	return
}

/*
SecAdminAddTag为用户添加标签

URI: s/message/AddTag

参数:
		{
			"uid":123,	//要加标签的用户ID
			"tag":"game"	//要加的标签名称
		}
返回值:

	{
		"status": "ok",
		"tm": 1443071652
	}
*/
func (sm *MessageModule) SecAdminAddTag(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	var uid uint32
	var tag string
	if err := req.Parse("uid", &uid, "tag", &tag); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	return push.AddTag(uid, tag)
}

/*
SecClearAllMessages清空所有消息

URI: s/message/ClearAllMessages

参数:
		{
		}
返回值:

	{
		"status": "ok",
		"tm": 1443071652
	}
*/
func (sm *MessageModule) SecClearAllMessages(req *service.HttpRequest, result map[string]interface{}) (e error) {
	return general.ClearAllMessages(req.Uid)
}
