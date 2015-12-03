package usercontrol

import (
	"fmt"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/coin"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/topic"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/notify"
	"yuanfen/yf_service/cls/unread"
	// "yf_pkg/utils"
)

func PushSysMessage(from, to uint32, folder string, msg map[string]interface{}) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content[common.FOLDER_KEY] = folder
	// content["type"] = common.MSG_TYPE_TEXT
	for k, v := range msg {
		content[k] = v
	}
	return general.SendMsg(from, to, content, "")
}

func PushSysMessageOld(to uint32, tp string, msg map[string]interface{}) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_TEXT
	content["sys_type"] = tp
	for k, v := range msg {
		content[k] = v
	}
	// fmt.Println(fmt.Sprintf("PushSysMessage %v", content))
	return general.SendMsg(common.USER_SYSTEM, to, content, "")
}

func getNickname(uid uint32) (result string) {
	v, e := user_overview.GetUserObjects(uid)
	if e != nil {
		return ""
	}
	if vname, ok := v[uid]; ok {
		return vname.Nickname
	}
	return
}

//发送给礼物消息
func PushGive_present(f_uid uint32, t_uid uint32, gift_id int, n string, info string, img string, tag string, earn int, giftid int, res string) (msgid uint64, e error) { //id int,
	msg := make(map[string]interface{})
	msg["type"] = common.MSG_TYPE_GIVE_PRESENT
	msg["f_uid"] = f_uid
	msg["t_uid"] = t_uid
	msg["gid"] = giftid
	msg["gift_record_id"] = gift_id
	msg["gift_name"] = n
	msg["gift_info"] = info
	msg["gift_img"] = img
	msg["gift_res"] = res
	s_fnick := getNickname(f_uid)
	msg["f_nickname"] = s_fnick
	msg["t_nickname"] = getNickname(t_uid)
	un := make(map[string]interface{})
	un[common.UNREAD_GIFT] = 0
	unread.GetUnreadNum(t_uid, un)
	msg[common.UNREAD_KEY] = un

	if c, _, err := coin.GetUserCoinInfo(t_uid); err == nil {
		msg[common.USER_BALANCE] = c
	}
	msg[common.USER_BALANCE_CHANGE] = fmt.Sprintf("收到礼物获得 %v钻石", earn)

	data := map[string]interface{}{"uid": f_uid, "nickname": s_fnick}
	not, e := notify.GetNotify2(notify.NOTIFY_GIFT_SEND, data, "", s_fnick+"送了你一个"+n, img, t_uid)
	if e != nil {
		return
	}
	msg[notify.NOTIFY_KEY] = not

	msgid, err := general.SendMsg(f_uid, t_uid, msg, tag)
	if err != nil {
		return 0, err
	}

	if tag != "" {
		general.SendTagMsg(f_uid, tag, msg)
	}
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_HINT
	content["msgid"] = msgid
	content["isSave"] = true
	content["content"] = fmt.Sprintf("收到礼物，钻石+ %v", earn)
	general.SendMsg(f_uid, t_uid, content, "")
	return msgid, nil
}

//发送收礼答谢消息
func PushThx_present(f_uid uint32, t_uid uint32, gift_id int, n string, info string, img string, tag string) (msgid uint64, e error) { //id int,
	msg := make(map[string]interface{})
	msg["type"] = common.MSG_TYPE_THX_PRESENT
	msg["f_uid"] = f_uid
	msg["t_uid"] = t_uid
	// msg["id"] = id
	msg["gift_record_id"] = gift_id
	msg["gift_name"] = n
	msg["gift_info"] = info
	msg["gift_img"] = img
	s_fnick := getNickname(f_uid)
	msg["f_nickname"] = s_fnick
	msg["t_nickname"] = getNickname(t_uid)

	data := map[string]interface{}{"uid": f_uid, "nickname": s_fnick}
	not, e := notify.GetNotify2(notify.NOTIFY_GIFT_ACCEPT, data, "", s_fnick+"接受了你的礼物", img, t_uid)
	if e != nil {
		return
	}
	msg[notify.NOTIFY_KEY] = not

	msgid, err := general.SendMsg(f_uid, t_uid, msg, tag)
	if err != nil {
		return 0, err
	}
	if tag != "" {
		general.SendTagMsg(f_uid, tag, msg)
	}

	return msgid, nil
}

//推送女用户上线消息 给男客服
func PushNvOnline(to uint32, from uint32) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_LOGIN
	content["isSave"] = true
	return general.SendMsg(from, to, content, "")
}

//推送用户创建圈子消息
func PushCreateTopic(to uint32, from uint32, tid uint32) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_CREATE_TOPIC
	content["tid"] = tid

	tinfos, err := topic.GetTopics(tid)
	if err != nil {
		return 0, err
	}
	tpc := tinfos[tid]
	tpinfo := make(map[string]interface{})
	if tpc != nil {
		tpinfo["id"] = tpc.Id
		tpinfo["tag"] = tpc.Tag
		tpinfo["pics"] = tpc.Pics
		tpinfo["picsLevel"] = tpc.PicsLevel
		tpinfo["title"] = tpc.Title
		tpinfo["uid"] = tpc.Uid
		tpinfo["tm"] = tpc.Tm
	}
	content["tinfo"] = tpinfo
	return general.SendMsg(from, to, content, "")
}

//推送男用户注册消息
func PushNanReg(to uint32, from uint32) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_MEN_REG
	return general.SendMsg(from, to, content, "")
}

//推送女用户注册消息
func PushNvReg(to uint32, from uint32) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_WOMEN_REG
	return general.SendMsg(from, to, content, "")
}

//推送信息变更
func PushInfoChange(to uint32, from uint32, msg map[string]interface{}) (msgid uint64, e error) {
	content := make(map[string]interface{})
	for k, v := range msg {
		content[k] = v
	}
	mainlog.AppendInfo(fmt.Sprintf("PushInfoChange %v,%v,%v", from, to, msg))
	return general.SendMsg(from, to, content, "")
}

func PushInvalidMsg(to uint32, from uint32) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_HINT
	content["isSave"] = true
	content["content"] = "无法向目标发送此类消息"
	return general.SendMsg(from, to, content, "")
}

func PushHintMsg(to uint32, from uint32, msg string) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_HINT
	content["isSave"] = true
	content["content"] = msg
	return general.SendMsg(from, to, content, "")
}

func FeeHintContent(to uint32, from uint32) (result interface{}, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_HINT
	content["isSave"] = true
	content["content"] = "你的余额不足，请充值。"
	but := make(map[string]interface{})
	but["tip"] = "点击立即充值"
	but["cmd"] = notify.CMD_OPEN_ACCOUNT
	data := make(map[string]interface{})
	but["Data"] = data
	content["but"] = but
	result = content
	return
	// return general.SendMsg(from, to, content, "")
}

func PushRelogin(to uint32, imei string) (msgid uint64, e error) {
	msg := make(map[string]interface{})
	msg["content"] = ""
	msg["imei"] = imei
	msg["type"] = common.MSG_TYPE_KICK
	return PushSysMessage(common.USER_SYSTEM, to, common.FOLDER_HIDE, msg)
}
