package general

import (
	"errors"
	"yf_pkg/push"
	"yf_pkg/utils"
)

func ringShake(uid uint32) (ring, shake bool, e error) {
	p, e := GetUserProtect(uid)
	if e != nil {
		return false, false, e
	}
	if p.NightRing == 0 && (utils.Now.Hour() >= 23 || utils.Now.Hour() < 7) {
		return false, false, nil
	}
	mlog.AppendObj(errors.New("00"), p, uid, "notRing: ", p.MsgNotRing, " notShake:", p.MsgNotShake)
	return p.MsgNotRing == 0, p.MsgNotShake == 0, nil
}
func SendMsg(from uint32, to uint32, content map[string]interface{}, tag string) (msgid uint64, e error) {
	ring, shake, e := ringShake(to)
	if e != nil {
		return 0, e
	}
	return push.SendMsg(from, to, content, tag, ring, shake)
}

func SendMsgNoPush(from uint32, to uint32, content map[string]interface{}, tag string) (msgid uint64, e error) {
	return push.SendMsgNoPush(from, to, content, tag)
}

/*
发送实时消息给用户

参数：
	withPush: 是否使用第三方推送
*/

func SendMsgWithOnline(from uint32, to uint32, content map[string]interface{}, tag string, withPush bool) (msgid uint64, online bool, e error) {
	ring, shake, e := ringShake(to)
	if e != nil {
		return 0, false, e
	}
	return push.SendMsgWithOnline(from, to, content, tag, withPush, ring, shake)
}

//准备发送消息，用于预先得到消息的msgid
func PrepareSendMsg(from uint32, to uint32, content map[string]interface{}, tag string) (msgid uint64, e error) {
	return push.PrepareSendMsg(from, to, content, tag)
}

//实际发送消息
func ExecSend(msgid uint64, from uint32, to uint32, content map[string]interface{}, tag string, withPush bool) (online bool, e error) {
	ring, shake, e := ringShake(to)
	if e != nil {
		return false, e
	}
	return push.ExecSend(msgid, from, to, content, tag, withPush, ring, shake)
}

//向多个用户发送消息，耗时较长，必须异步
func SendMsgM(from uint32, to []uint32, content map[string]interface{}, tag string) (msgid map[uint32]uint64, e error) {
	msgid = make(map[uint32]uint64, len(to))
	for _, uid := range to {
		if id, e := SendMsg(from, uid, content, tag); e == nil {
			msgid[uid] = id
		} else {
			msgid[uid] = 0
		}
	}
	return msgid, nil
}

func PrepareSendTagMsg(from uint32, tag string, content map[string]interface{}) (msgid uint64, e error) {
	return push.PrepareSendTagMsg(from, tag, content)
}

func ExecSendTagMsg(msgid uint64, from uint32, tag string, content map[string]interface{}) (offline []uint32, e error) {
	return push.ExecSendTagMsg(msgid, from, tag, content)
}

/*
SendTagMsgWithThirdParty发送标签消息时同时会走socket和第三方推送。

参数：
	tag: 标签，all是一个特殊的标签，表示所有用户
*/
func SendTagMsgWithThirdParty(from uint32, tag string, content map[string]interface{}) (msgid uint64, offline []uint32, e error) {
	return push.SendTagMsgWithThirdParty(from, tag, content)
}

func SendTagMsg(from uint32, tag string, content map[string]interface{}) (msgid uint64, offline []uint32, e error) {
	return push.SendTagMsg(from, tag, content)
}
