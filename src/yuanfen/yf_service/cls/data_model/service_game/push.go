package service_game

import (
	"errors"
	"fmt"
	"yf_pkg/push"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/award"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

//游戏大厅发送消息
func Send(from uint32, tag string, content interface{}, res map[string]interface{}) (msgid uint64, e error) {
	switch value := content.(type) {
	case map[string]interface{}:
		typ := utils.ToString(value["type"])
		if typ == common.MSG_TYPE_PIC { // 图片消息，需要异步进行图片验证
			/*		ok, msgid, e := doPicTagPush(from, tag, value, 0)
					if !ok || e != nil || msgid <= 0 {
						mlog.AppendObj(e, "[plane send] error ", content, from)
						return 0, errors.New("发送失败")
					}
					go doPicTagPushCheck(msgid, from, tag, value)
			*/
			msid, offline, e := push.SendTagMsg(from, tag, value)
			if e != nil {
				return 0, service.NewError(service.ERR_INTERNAL, fmt.Sprintf("send message error :%v", e.Error()))
			}
			res["offline"] = offline
			return msid, nil
		} else { // 非图片群消息，直接走
			msid, offline, e := push.SendTagMsg(from, tag, value)
			if e != nil {
				return 0, service.NewError(service.ERR_INTERNAL, fmt.Sprintf("send message error :%v", e.Error()))
			}
			res["offline"] = offline
			return msid, nil
		}
		return
	default:
		return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("msg must be a json:%v", content))
	}
}

/*
批量推送中奖想笑
*/
func DoMoreAwardPush(v []GameAward) (e error) {
	for _, ga := range v {
		if e = PushAward(ga); e != nil {
			return
		}
	}
	return
}

// 中奖推送消息
func PushAward(ga GameAward) (e error) {
	// 获取组队信息
	u, e := user_overview.GetUserObject(ga.Uid)
	if e != nil {
		return e
	}
	if u == nil {
		return errors.New("用户获取失败")
	}
	a, e := award.GetAwardById(ga.AwardId)
	if e != nil {
		return
	}
	// 拼接返回值和area格式一致
	msg := make(map[string]interface{})
	msg["type"] = common.MSG_TYPE_PLANE_AWARD
	msg["name"] = a.Name
	msg["uint"] = a.Unit
	msg["info"] = ga.Info
	msg["nikcname"] = u.Nickname
	msg["avatar"] = u.Avatar
	msg["uid"] = ga.Uid
	msg["count"] = ga.Count
	tag, e := GetGameTag(ga.Uid)
	if e != nil {
		return
	}
	msid, _, e := push.SendTagMsg(common.USER_SYSTEM, tag, msg)
	if e != nil || msid <= 0 {
		return errors.New("send msg is error")
	}
	return
}
