package api

import (
	"fmt"
	"strings"
	"yf_pkg/log"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/push/pusher/common"
	"yuanfen/push/pusher/msg"
	"yuanfen/push/pusher/user"
)

type Receiver struct {
	conf common.Config
	log  *log.MLogger
}

func (r *Receiver) Init(env *service.Env) error {
	r.conf = env.ModuleEnv.(common.Config)
	r.log = env.Log
	return nil
}

func (r *Receiver) SecGetEndpoint(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	key := user.AddKey(req.Uid)
	res["key"] = key.Key
	res["timeout"] = uint32(key.Timeout)
	return
}

func (r *Receiver) SecSendM(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	msgidStr := req.GetParam("msgid")
	msgids := strings.Split(msgidStr, ",")
	to := req.GetParam("to")
	uids := strings.Split(to, ",")
	tag := req.GetParam("tag")
	if len(uids) != len(msgids) {
		return service.NewError(service.ERR_INVALID_PARAM, "msgid count must equals to count")
	}

	users := map[string]interface{}{}
	for i, uidStr := range uids {
		msgid, err := utils.ToUint64(msgids[i])
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("msgid %v format error : %v", msgids, err.Error()))
		}

		uid, e := utils.ToUint32(uidStr)
		if e != nil {
			return service.NewError(service.ERR_INVALID_PARAM, "user id must be unsigned integer")
		}
		m := msg.New(common.USER_MSG, msgid, req.Uid, req.BodyRaw, tag)
		online, err := user.SendMessage(uid, m)
		if err != nil {
			users[uidStr] = err.Error()
		} else {
			users[uidStr] = map[string]interface{}{"online": online, "msgid": msgid}
		}
	}
	res["res"] = users
	return
}

/*
SecSend向用户发送即时消息

URI: s/push/Send?msgid=123&to=t_all

params：
	msgid: 消息ID
	to: 发送目标，tag消息的前缀必须为"t_"，私聊消息的前缀为"u_"。特别的，"t_all"为发送给所有用户的消息
cookie:
	uid: 发送者uid
body:
	消息体
返回值：
	{
		"status": "ok",
		"tm": 1442491885
	}
*/
func (r *Receiver) SecSend(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	msgidStr := req.GetParam("msgid")
	msgid, err := utils.ToUint64(msgidStr)
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("msgid %v format error : %v", msgidStr, err.Error()))
	}

	to := req.GetParam("to")
	if len(to) < 2 {
		return service.NewError(service.ERR_INVALID_PARAM, "send target must start with t_ or u_")
	}
	switch to[0:2] {
	case "t_":
		m := msg.New(common.TAG_MSG, msgid, req.Uid, req.BodyRaw, to[2:])
		go user.SendTagMessage(to[2:], m)
	case "u_":
		uid, e := utils.ToUint32(to[2:])
		if e != nil {
			return service.NewError(service.ERR_INVALID_PARAM, "user id must be unsigned integer")
		}
		m := msg.New(common.USER_MSG, msgid, req.Uid, req.BodyRaw, req.GetParam("tag"))
		online, err := user.SendMessage(uid, m)
		res["online"] = online
		if err != nil {
			return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("send message to user %v error : %v", uid, err.Error()))
		}
	default:
		return service.NewError(service.ERR_INVALID_PARAM, "send target must start with t_ or u_")
	}
	return
}

func (r *Receiver) SecKick(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	user.DelUser(req.Uid)
	return
}

func (r *Receiver) OnlineUsers(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	res["users"] = user.OnlineUsers()
	return
}
func (r *Receiver) SecAddTag(req *service.HttpRequest, res map[string]interface{}) (e error) {
	user.AddTag(req.Uid, req.GetParam("tag"))
	return nil
}
func (r *Receiver) SecDelTag(req *service.HttpRequest, res map[string]interface{}) (e error) {
	user.DelTag(req.Uid, req.GetParam("tag"))
	return nil
}
