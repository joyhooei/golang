package service_game

import (
	"time"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/notify"
	"yuanfen/yf_service/cls/unread"

	_ "github.com/go-sql-driver/mysql"
)

// 获取我的追求者未读消息,实现未读消息模块
func UnreadNum(uid uint32, key string, from time.Time) (total uint32, show string) {
	//fmt.Println("---unread get --", uid, key, from)
	switch key {
	case common.UNREAD_GAME:
		sql := "select name  from  game_config where tm > ? "
		var name string
		if e := mdb.QueryRow(sql, from).Scan(&name); e != nil {
			return 0, ""
		}
		return 1, name
	case common.UNREAD_VERSION:
		sysid, e := user_overview.SystemInfo(uid)
		if e != nil {
			return 0, ""
		}
		app_type := 1   // android
		if sysid == 1 { //苹果
			app_type = 2 // ios
		}
		var ver uint32
		agent_id := ""
		version := "0"
		if version, agent_id, e = user_overview.VerInfo(uid); e != nil {
			return 0, ""
		} else {
			if ver, e = utils.StringToUint32(version); e != nil {
				return 0, ""
			}
		}
		sql := "select title from app_version where app_type = ? and agent_id =?  and ver >? and  tm > ? limit 1 "
		var title string
		if e := mdb.QueryRow(sql, app_type, agent_id, ver, from).Scan(&title); e != nil {
			return 0, ""
		}
		return 1, title

	}
	return 0, ""
}

//通用1对1发通知
func DoPush(f_uid, t_uid uint32, msg map[string]interface{}, is_detail int) (bool, error) {
	msg["f_uid"] = f_uid
	msg["t_uid"] = t_uid
	if is_detail > 0 {
		u_map, e := user_overview.GetUserObjects([]uint32{f_uid}...)
		if e != nil {
			return false, e
		}
		if u, ok := u_map[f_uid]; ok {
			msg["nickname"] = u.Nickname
			msg["avatar"] = u.Avatar
			if is_detail > 2 {
				msg["gender"] = u.Gender
				msg["grape"] = u.Grade
				msg["age"] = u.Age
				msg["province"] = u.Province
				msg["city"] = u.City
			}
		}
	}
	if _, e := general.SendMsg(common.USER_SYSTEM, t_uid, msg, ""); e != nil {
		mlog.AppendObj(nil, "--doPush- is error d -", e.Error())
		return false, e
	}
	return true, nil
}

// 未读消息通用发送
func DoUnReadPush(f_uid, t_uid uint32, msg_type string, unread_m map[string]interface{}, is_detail int) (bool, error) {
	msg := make(map[string]interface{})
	if msg_type == common.MSG_TYPE_ADDLUCKY {
		if n, e := notify.GetNotify(f_uid, notify.NOTIFY_ADD_LUCKY, nil, "", "", t_uid); e == nil {
			msg[notify.NOTIFY_KEY] = n
		}
	}
	msg["type"] = msg_type
	if e := unread.GetUnreadNum(t_uid, unread_m); e != nil {
		return false, e
	}
	msg["unread"] = unread_m
	if ok, e := DoPush(f_uid, t_uid, msg, 1); e != nil || !ok {
		return false, e
	}
	return true, nil
}
