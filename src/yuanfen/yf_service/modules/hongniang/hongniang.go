package hongniang

import (
	"fmt"
	"yf_pkg/log"
	"yf_pkg/service"
	"yuanfen/yf_service/cls/data_model/hongniang"
)

type HongniangModule struct {
	log *log.MLogger
}

func (sm *HongniangModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	return
}

func (sm *HongniangModule) SecSendToUser(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var to uint32
	var content interface{}
	if err := req.Parse("to", &to, "content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	msgid, e := hongniang.SendToUser(req.Uid, to, content)
	if e != nil {
		return e
	}
	res := make(map[string]interface{})
	res["msgid"] = msgid
	result["res"] = res
	return
}

func (sm *HongniangModule) SecSendToHongniang(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var content interface{}
	if err := req.Parse("content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	msgid, e := hongniang.SendToHongniang(req.Uid, content)
	if e != nil {
		return e
	}
	res := make(map[string]interface{})
	res["msgid"] = msgid
	result["res"] = res
	return
}

func (sm *HongniangModule) SecGetMessages(req *service.HttpRequest, result map[string]interface{}) (e error) {
	is, err := hongniang.IsHongniang(req.Uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if !is {
		return service.NewError(service.ERR_INVALID_USER, "must be hongniang")
	}

	var uid, count uint32
	var msgid uint64
	if err := req.Parse("uid", &uid, "msgid", &msgid, "count", &count); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	msgs, err := hongniang.GetMessages(uid, msgid, count)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("query messages error :%v", err.Error()))
	}

	res := make(map[string]interface{})
	res["msgs"] = msgs
	result["res"] = res
	return
}
