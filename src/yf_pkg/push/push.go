package push

import (
	"encoding/json"
	"errors"
	"fmt"
	"yf_pkg/log"
	"yf_pkg/net/http"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/push/pusher/common"
	ycm "yuanfen/yf_service/cls/common"
)

type UserTags struct {
	Status string   `json:"status"`
	Code   int      `json:"code"`
	Msg    string   `json:"msg"`
	Tags   []string `json:"tags"`
}

var host string
var sys func(uint32) int
var logger *log.MLogger
var mode string
var appleDomain string

func DefaultSys(uid uint32) int {
	return SYSTEM_XIAOMI
}

func Init(addr string, l *log.MLogger, system func(uint32) int, app_mode string) {
	host = addr
	sys = system
	logger = l
	mode = app_mode
	if mode == "production" {
		appleDomain = "api.xmpush.xiaomi.com"
	} else {
		appleDomain = "sandbox.xmpush.xiaomi.com"
	}
}

func GetEndpoint(uid uint32) (address string, key uint32, e error) {
	body, e := http.HttpSend(host, "push/GetEndpoint", map[string]string{"uid": utils.Uint32ToString(uid)}, nil, nil)
	if e != nil {
		return "", 0, e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return "", 0, e
	}
	if m["status"] != "ok" {
		return "", 0, errors.New(fmt.Sprintf("<%v,%v>", m["code"], m["msg"]))
	}
	address = utils.ToString(m["address"])
	keyF, ok := m["key"].(float64)
	if !ok {
		return "", 0, errors.New("key must be type uint32")
	}
	key = uint32(keyF)
	return
}

func AddTag(uid uint32, tag string) error {
	if tag == "all" {
		return nil
	}
	body, e := http.HttpSend(host, "push/AddTag", map[string]string{"uid": utils.Uint32ToString(uid), "tag": tag}, nil, nil)
	if e != nil {
		return e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return e
	}
	if m["status"] != "ok" {
		return errors.New(fmt.Sprintf("<%v,%v>", m["code"], m["msg"]))
	}
	return nil
}

func InTag(uid uint32, tag string) (bool, error) {
	if tag == "all" {
		return true, nil
	}
	body, e := http.HttpSend(host, "push/InTag", map[string]string{"uid": utils.Uint32ToString(uid), "tag": tag}, nil, nil)
	if e != nil {
		return false, e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return false, e
	}
	if m["status"] != "ok" {
		return false, errors.New(fmt.Sprintf("<%v,%v>", m["code"], m["msg"]))
	}
	return utils.ToBool(m["in"])

}
func DelTag(uid uint32, tag string) error {
	if tag == "all" {
		return nil
	}
	body, e := http.HttpSend(host, "push/DelTag", map[string]string{"uid": utils.Uint32ToString(uid), "tag": tag}, nil, nil)
	if e != nil {
		return e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return e
	}
	if m["status"] != "ok" {
		return errors.New(fmt.Sprintf("<%v,%v>", m["code"], m["msg"]))
	}
	return nil
}

func ClearTag(tag string) error {
	body, e := http.HttpSend(host, "push/ClearTag", map[string]string{"tag": tag}, nil, nil)
	if e != nil {
		return e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return e
	}
	if m["status"] != "ok" {
		return errors.New(fmt.Sprintf("<%v,%v>", m["code"], m["msg"]))
	}
	return nil
}
func Kick(uid uint32) error {
	body, e := http.HttpSend(host, "push/Kick", map[string]string{"uid": utils.Uint32ToString(uid)}, nil, nil)
	if e != nil {
		return e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return e
	}
	if m["status"] != "ok" {
		return errors.New(fmt.Sprintf("<%v,%v>", m["code"], m["msg"]))
	}
	return nil
}

func SendMsg(from uint32, to uint32, content map[string]interface{}, tag string, ring, shake bool) (msgid uint64, e error) {
	msgid, _, e = SendMsgWithOnline(from, to, content, tag, true, ring, shake)
	return msgid, e
}

func SendMsgNoPush(from uint32, to uint32, content map[string]interface{}, tag string) (msgid uint64, e error) {
	msgid, _, e = SendMsgWithOnline(from, to, content, tag, false, false, false)
	return msgid, e
}

/*
发送实时消息给用户

参数：
	withPush: 是否使用第三方推送
*/

func SendMsgWithOnline(from uint32, to uint32, content map[string]interface{}, tag string, withPush, ring, shake bool) (msgid uint64, online bool, e error) {
	content["tm"] = utils.Now
	j, e := json.Marshal(content)
	if e != nil {
		return 0, false, e
	}
	params := make(map[string]string)
	params["from"] = utils.ToString(from)
	params["tag"] = tag
	params["to"] = "u_" + utils.ToString(to)
	body, e := http.HttpSend(host, "push/Send", params, nil, j)
	if e != nil {
		return 0, false, e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return 0, false, e
	}
	if m["status"] != "ok" {
		return 0, false, service.NewError(service.ERR_INTERNAL, utils.ToString(m["detail"]), utils.ToString(m["msg"]))
	}
	online, e = utils.ToBool(m["online"])
	if e != nil {
		return 0, false, e
	}
	msgid, e = utils.ToUint64(m["msgid"])
	if e != nil {
		return 0, false, errors.New("msgid must be type uint64")
	}
	if withPush && !online {
		sendThirdParty(from, to, msgid, common.USER_MSG, content, tag, ring, shake)
	}
	return msgid, online, nil
}

//准备发送消息，用于预先得到消息的msgid
func PrepareSendMsg(from uint32, to uint32, content map[string]interface{}, tag string) (msgid uint64, e error) {
	content["tm"] = utils.Now
	j, e := json.Marshal(content)
	if e != nil {
		return 0, e
	}
	params := make(map[string]string)
	params["from"] = utils.ToString(from)
	params["tag"] = tag
	params["to"] = "u_" + utils.ToString(to)
	body, e := http.HttpSend(host, "push/PrepareSend", params, nil, j)
	if e != nil {
		return 0, e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return 0, e
	}
	if m["status"] != "ok" {
		return 0, service.NewError(service.ERR_INTERNAL, utils.ToString(m["detail"]), utils.ToString(m["msg"]))
	}
	msgid, e = utils.ToUint64(m["msgid"])
	if e != nil {
		return 0, errors.New("msgid must be type uint64")
	}
	return msgid, nil
}

//实际发送消息
func ExecSend(msgid uint64, from uint32, to uint32, content map[string]interface{}, tag string, withPush, ring, shake bool) (online bool, e error) {
	content["tm"] = utils.Now
	j, e := json.Marshal(content)
	if e != nil {
		return false, e
	}
	params := make(map[string]string)
	params["from"] = utils.ToString(from)
	params["tag"] = tag
	params["msgid"] = utils.ToString(msgid)
	params["to"] = "u_" + utils.ToString(to)
	body, e := http.HttpSend(host, "push/ExecSend", params, nil, j)
	if e != nil {
		return false, e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return false, e
	}
	if m["status"] != "ok" {
		return false, service.NewError(service.ERR_INTERNAL, utils.ToString(m["detail"]), utils.ToString(m["msg"]))
	}
	online, e = utils.ToBool(m["online"])
	if e != nil {
		return false, e
	}
	if withPush && !online {
		sendThirdParty(from, to, msgid, common.USER_MSG, content, tag, ring, shake)
	}
	return online, nil
}

//向多个用户发送消息
func SendMsgM(from uint32, to []uint32, content map[string]interface{}, tag string, ring, shake bool) (msgid map[uint32]uint64, e error) {
	content["tm"] = utils.Now
	j, e := json.Marshal(content)
	if e != nil {
		return nil, e
	}
	params := make(map[string]string)
	params["from"] = utils.ToString(from)
	params["tag"] = tag
	v, e := utils.Join(to, ",")
	if e != nil {
		return nil, e
	}
	params["to"] = v
	body, e := http.HttpSend(host, "push/SendM", params, nil, j)
	if e != nil {
		return nil, e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return nil, e
	}
	if m["status"] != "ok" {
		return nil, service.NewError(service.ERR_INTERNAL, utils.ToString(m["detail"]), utils.ToString(m["msg"]))
	}
	msgid = map[uint32]uint64{}
	res, ok := m["res"]
	if ok {
		switch users := res.(type) {
		case map[string]interface{}:
			for uidStr, ret := range users {
				uid, e := utils.ToUint32(uidStr)
				if e != nil {
					return nil, errors.New(fmt.Sprintf("parse uid %v error:%v", uidStr, e.Error()))
				}
				switch tmp := ret.(type) {
				case map[string]interface{}:
					mid, _ := utils.ToUint64(tmp["msgid"])
					online, _ := utils.ToBool(tmp["online"])
					msgid[uid] = mid
					if !online {
						sendThirdParty(from, uid, mid, common.USER_MSG, content, tag, ring, shake)
					}
				}
			}
		}
	}
	return msgid, nil
}

func PrepareSendTagMsg(from uint32, tag string, content map[string]interface{}) (msgid uint64, e error) {
	content["tm"] = utils.Now
	j, e := json.Marshal(content)
	if e != nil {
		return 0, e
	}
	params := make(map[string]string)
	params["from"] = utils.Uint32ToString(from)
	params["to"] = "t_" + tag
	body, e := http.HttpSend(host, "push/PrepareSend", params, nil, j)
	if e != nil {
		return 0, e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return 0, e
	}
	if m["status"] != "ok" {
		return 0, errors.New(fmt.Sprintf("<%v,%v>", m["code"], m["msg"]))
	}
	msgid, e = utils.ToUint64(m["msgid"])
	return
}

func ExecSendTagMsg(msgid uint64, from uint32, tag string, content map[string]interface{}) (offline []uint32, e error) {
	content["tm"] = utils.Now
	j, e := json.Marshal(content)
	if e != nil {
		return nil, e
	}
	params := make(map[string]string)
	params["from"] = utils.ToString(from)
	params["to"] = "t_" + tag
	params["msgid"] = utils.ToString(msgid)
	body, e := http.HttpSend(host, "push/ExecSend", params, nil, j)
	if e != nil {
		return nil, e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return nil, e
	}
	if m["status"] != "ok" {
		return nil, errors.New(fmt.Sprintf("<%v,%v>", m["code"], m["msg"]))
	}
	offline = make([]uint32, 0)
	o, ok := m["offline"].([]interface{})
	if ok {
		for _, uid := range o {
			uidF, ok := uid.(float64)
			if ok {
				offline = append(offline, uint32(uidF))
			}
		}
	}
	return offline, nil
}

/*
SendTagMsgWithThirdParty发送标签消息时同时会走socket和第三方推送。

参数：
	tag: 标签，all是一个特殊的标签，表示所有用户
*/
func SendTagMsgWithThirdParty(from uint32, tag string, content map[string]interface{}) (msgid uint64, offline []uint32, e error) {
	msgid, offline, e = SendTagMsg(from, tag, content)
	sendThirdParty(from, 0, msgid, common.TAG_MSG, content, tag, true, true)
	return
}

func SendTagMsg(from uint32, tag string, content map[string]interface{}) (msgid uint64, offline []uint32, e error) {
	content["tm"] = utils.Now
	j, e := json.Marshal(content)
	if e != nil {
		return 0, nil, e
	}
	params := make(map[string]string)
	params["from"] = utils.ToString(from)
	params["to"] = "t_" + tag
	body, e := http.HttpSend(host, "push/Send", params, nil, j)
	if e != nil {
		return 0, nil, e
	}
	var m map[string]interface{}
	if e := json.Unmarshal(body, &m); e != nil {
		return 0, nil, e
	}
	if m["status"] != "ok" {
		return 0, nil, errors.New(fmt.Sprintf("<%v,%v>", m["code"], m["msg"]))
	}
	msgidF, ok := m["msgid"].(float64)
	if !ok {
		return 0, nil, errors.New("msgid must be type uint64")
	}
	offline = make([]uint32, 0)
	o, ok := m["offline"].([]interface{})
	if ok {
		for _, uid := range o {
			uidF, ok := uid.(float64)
			if ok {
				offline = append(offline, uint32(uidF))
			}
		}
	}
	return uint64(msgidF), offline, nil
}

func sendThirdParty(from uint32, to uint32, msgid uint64, group int8, content map[string]interface{}, tag string, ring, shake bool) {
	var by, target string
	switch to {
	case 0:
		if tag == "all" {
			by = BY_ALL
		} else {
			by = BY_TOPIC
			target = tag
		}
	default:
		by = BY_ALIAS
		target = utils.ToString(to)
	}
	data := map[string]interface{}{}
	data["msgid"] = msgid
	data["group"] = group
	data["tagid"] = tag
	data["sender"] = from
	data["content"] = content
	j, e := json.Marshal(data)
	if e != nil {
		return
	}
	//管道中积压消息过多，删掉老消息
	if len(MsgChan) > 500 {
		<-MsgChan
		logger.Append("too many push message in channel MsgChan > 500")
	}
	platform := sys(to)
	switch platform {
	case SYSTEM_APPLE:
		tp := utils.ToString(content["type"])
		switch tp {
		case ycm.MSG_TYPE_TEXT:
			MsgChan <- &Message{platform, by, MODE_NOTIFICATION, shake, ring, []string{target}, "私聊消息", utils.ToString(msgid), "你收到一条私聊消息"}
		case ycm.MSG_TYPE_VOICE:
			MsgChan <- &Message{platform, by, MODE_NOTIFICATION, shake, ring, []string{target}, "私聊消息", utils.ToString(msgid), "你收到一条语音消息"}
		case ycm.MSG_TYPE_PIC:
			MsgChan <- &Message{platform, by, MODE_NOTIFICATION, shake, ring, []string{target}, "私聊消息", utils.ToString(msgid), "你收到一张图片"}
		}
	default:
		MsgChan <- &Message{platform, by, MODE_NOTIFICATION, shake, ring, []string{target}, "私聊消息", string(j), "点击查看详情"}
	}
}

func SendThirdPartyDirect(mode int, by string, targets []string, title string, content map[string]interface{}, desc string) error {
	data := map[string]interface{}{}
	data["msgid"] = -1
	data["content"] = content
	j, e := json.Marshal(data)
	if e != nil {
		return e
	}
	//管道中积压消息过多，删掉老消息
	if len(MsgChan) > 500 {
		<-MsgChan
		err_desc := "too many push message in channel MsgChan > 500"
		logger.Append(err_desc)
		return errors.New(err_desc)
	}
	switch by {
	case BY_ALL, BY_TOPIC:
		MsgChan <- &Message{SYSTEM_XIAOMI, by, mode, true, true, targets, title, string(j), desc}
		MsgChan <- &Message{SYSTEM_XINGE, by, mode, true, true, targets, title, string(j), desc}
		MsgChan <- &Message{SYSTEM_APPLE, by, mode, true, true, targets, title, string(j), desc}
	case BY_ALIAS:
		for _, idStr := range targets {
			id, e := utils.ToUint32(idStr)
			if e != nil {
				return e
			}
			MsgChan <- &Message{sys(id), by, mode, true, true, []string{utils.ToString(id)}, title, string(j), desc}
		}
	default:
		err_desc := "only support BY_ALIAS, BY_TOPICS and BY_ALL"
		logger.Append(err_desc)
		return errors.New(err_desc)
	}
	return nil
}

func GetUserTags(uid uint32) (tags []string, e error) {
	body, e := http.HttpSend(host, "push/GetUserTags", map[string]string{"uid": utils.ToString(uid)}, nil, nil)
	if e != nil {
		return nil, e
	}
	var m UserTags
	if e := json.Unmarshal(body, &m); e != nil {
		return nil, e
	}
	if m.Status != "ok" {
		return nil, errors.New(fmt.Sprintf("<%v,%v>", m.Code, m.Msg))
	}
	return m.Tags, nil
}
