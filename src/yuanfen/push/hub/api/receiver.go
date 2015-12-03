package api

import (
	"errors"
	"fmt"
	"strings"
	"yf_pkg/log"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/push/hub/config"
	"yuanfen/push/hub/db"
	"yuanfen/push/hub/node_manager"
)

type Receiver struct {
	conf config.Config
	log  *log.MLogger
}

func (r *Receiver) Init(env *service.Env) error {
	r.conf = env.ModuleEnv.(config.Config)
	r.log = env.Log
	return nil
}

func (r *Receiver) GetEndpoint(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	uid, err := utils.ToUint32(req.GetParam("uid"))
	if err != nil {
		return service.NewError(service.ERR_INVALID_FORMAT, fmt.Sprintf("parse uid %v failed : %v", req.GetParam("uid"), err.Error()))
	}
	result, e := node_manager.GetEndpoint(uid)
	if e.Code == service.ERR_NOERR {
		res["address"] = result["address"]
		res["key"] = uint32(result["key"].(float64))
		res["timeout"] = uint32(result["timeout"].(float64))
	}
	return
}

func (r *Receiver) SendM(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {

	from, err := utils.ToUint32(req.GetParam("from"))
	if err != nil {
		desc := fmt.Sprintf("parse from %v failed : %v", req.GetParam("from"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}

	toStr := strings.Split(req.GetParam("to"), ",")
	uids := make([]uint32, 0, len(toStr))
	for _, uidStr := range toStr {
		uid, e := utils.ToUint32(uidStr)
		if e != nil {
			return service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("parse to error :", req.GetParam("to")))
		}
		uids = append(uids, uid)
	}
	var msgType string
	if err := req.ParseOpt("type", &msgType, ""); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}

	tag := req.GetParam("tag")
	e = node_manager.SendUsersMsg(from, uids, tag, msgType, req.BodyRaw, res)
	return
}

func (r *Receiver) PrepareSend(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	from, err := utils.ToUint32(req.GetParam("from"))
	if err != nil {
		desc := fmt.Sprintf("parse from %v failed : %v", req.GetParam("from"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}

	toStr := req.GetParam("to")
	if len(toStr) < 2 {
		return service.NewError(service.ERR_INVALID_PARAM, "send target must start with t_ or u_")
	}
	var msgType string
	if err := req.ParseOpt("type", &msgType, ""); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}

	switch toStr[0:2] {
	case "t_":
		e = node_manager.PrepareSendTagMsg(from, toStr[2:], msgType, req.BodyRaw, res)
	case "u_":
		to, err := utils.ToUint32(toStr[2:])
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, "user id must be unsigned integer")
		}
		if from == to {
			return service.NewError(service.ERR_INVALID_PARAM, "cannot send to yourself")
		}
		tag := req.GetParam("tag")
		e = node_manager.PrepareSendUserMsg(from, to, tag, msgType, req.BodyRaw, res)
	default:
		return service.NewError(service.ERR_INVALID_PARAM, "send target must start with t_ or u_")
	}
	return
}

func (r *Receiver) ExecSend(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	msgid, err := utils.ToUint64(req.GetParam("msgid"))
	if err != nil {
		desc := fmt.Sprintf("parse from %v failed : %v", req.GetParam("msgid"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}
	from, err := utils.ToUint32(req.GetParam("from"))
	if err != nil {
		desc := fmt.Sprintf("parse from %v failed : %v", req.GetParam("from"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}

	toStr := req.GetParam("to")
	if len(toStr) < 2 {
		return service.NewError(service.ERR_INVALID_PARAM, "send target must start with t_ or u_")
	}
	var msgType string
	if err := req.ParseOpt("type", &msgType, ""); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}

	switch toStr[0:2] {
	case "t_":
		e = node_manager.ExecSendTagMsg(msgid, from, toStr[2:], msgType, req.BodyRaw, res)
	case "u_":
		to, err := utils.ToUint32(toStr[2:])
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, "user id must be unsigned integer")
		}
		if from == to {
			return service.NewError(service.ERR_INVALID_PARAM, "cannot send to yourself")
		}
		tag := req.GetParam("tag")
		e = node_manager.ExecSendUserMsg(msgid, from, to, tag, msgType, req.BodyRaw, res)
	default:
		return service.NewError(service.ERR_INVALID_PARAM, "send target must start with t_ or u_")
	}
	return
}
func (r *Receiver) Send(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	from, err := utils.ToUint32(req.GetParam("from"))
	if err != nil {
		desc := fmt.Sprintf("parse from %v failed : %v", req.GetParam("from"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}

	toStr := req.GetParam("to")
	if len(toStr) < 2 {
		return service.NewError(service.ERR_INVALID_PARAM, "send target must start with t_ or u_")
	}
	var msgType string
	if err := req.ParseOpt("type", &msgType, ""); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}

	switch toStr[0:2] {
	case "t_":
		e = node_manager.SendTagMsg(from, toStr[2:], msgType, req.BodyRaw, res)
	case "u_":
		to, err := utils.ToUint32(toStr[2:])
		if err != nil {
			return service.NewError(service.ERR_INVALID_PARAM, "user id must be unsigned integer")
		}
		if from == to {
			return service.NewError(service.ERR_INVALID_PARAM, "cannot send to yourself")
		}
		tag := req.GetParam("tag")
		e = node_manager.SendUserMsg(from, to, tag, msgType, req.BodyRaw, res)
	default:
		return service.NewError(service.ERR_INVALID_PARAM, "send target must start with t_ or u_")
	}
	return
}

func (r *Receiver) InTag(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	tag := req.GetParam("tag")
	if tag == "" {
		return service.NewError(service.ERR_INVALID_PARAM, "no param tag provided")
	}
	uid, err := utils.ToUint32(req.GetParam("uid"))
	if err != nil {
		desc := fmt.Sprintf("parse uid %v failed : %v", req.GetParam("uid"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}
	exists, err := db.InTag(uid, tag)
	if err != nil {
		desc := fmt.Sprintf("IntTag %v of uid %v failed : %v", tag, uid, err.Error())
		return service.NewError(service.ERR_REDIS, desc)
	}
	res["in"] = exists
	return
}
func (r *Receiver) AddTag(req *service.HttpRequest, res map[string]interface{}) (e error) {
	tag := req.GetParam("tag")
	if tag == "" {
		return service.NewError(service.ERR_INVALID_PARAM, "no param tag provided")
	}
	uid, err := utils.ToUint32(req.GetParam("uid"))
	if err != nil {
		desc := fmt.Sprintf("parse uid %v failed : %v", req.GetParam("uid"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}
	return node_manager.AddTag(uid, tag)
}

func (r *Receiver) DelTag(req *service.HttpRequest, res map[string]interface{}) (e error) {
	tag := req.GetParam("tag")
	if tag == "" {
		return service.NewError(service.ERR_INVALID_PARAM, "no param tag provided")
	}
	uid, err := utils.ToUint32(req.GetParam("uid"))
	if err != nil {
		desc := fmt.Sprintf("parse uid %v failed : %v", req.GetParam("uid"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}
	return node_manager.DelTag(uid, tag)
}

func (r *Receiver) ClearTag(req *service.HttpRequest, res map[string]interface{}) (e error) {
	/*
		tag := req.GetParam("tag")
		if tag == "" {
			return service.NewError(service.ERR_INVALID_PARAM, "no param tag provided")
		}
		if err := db.ClearTag(tag); err != nil {
			desc := fmt.Sprintf("clear tag %v failed : %v", tag, err.Error())
			return service.NewError(service.ERR_REDIS, desc)
		}
	*/
	return errors.New("not implemented")
}

func (r *Receiver) Kick(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	uid, err := utils.ToUint32(req.GetParam("uid"))
	if err != nil {
		desc := fmt.Sprintf("parse uid %v failed : %v", req.GetParam("uid"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}
	_, e = node_manager.SendToUser(uid, "Kick", nil, nil)
	return
}

func (r *Receiver) OnlineUsers(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	//暂时没有用到的地方，先不实现
	return
}

func (r *Receiver) Ping(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	result, e := node_manager.SendToNodes("Ping", nil, nil)
	res["nodes"] = result
	return e
}

func (r *Receiver) TestOfflineReceiver(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	res["result"] = string(req.BodyRaw)
	return
}

/*
GetUserTags获取用户的所有标签

URI: push/GetUserTags?uid=123

参数：
	uid: 目标用户ID
返回值：
	{
		"tags": ["game","topic"],
		"status": "ok",
		"tm": 1442491885
	}
*/
func (r *Receiver) GetUserTags(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	uid, err := utils.ToUint32(req.GetParam("uid"))
	if err != nil {
		desc := fmt.Sprintf("parse uid %v failed : %v", req.GetParam("uid"), err.Error())
		return service.NewError(service.ERR_INVALID_FORMAT, desc)
	}
	if tags, err := db.GetUserTags(uid); err != nil {
		desc := fmt.Sprintf("get tags of uid %v failed : %v", uid, err.Error())
		return service.NewError(service.ERR_REDIS, desc)
	} else {
		res["tags"] = tags
		return
	}
}
