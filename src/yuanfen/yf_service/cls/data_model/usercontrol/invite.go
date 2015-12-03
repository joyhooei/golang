package usercontrol

import (
	"errors"
	"fmt"
	"time"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/notify"
)

const (
	INVITE_KEY_CERTIFY        = "certify"        //邀请认证
	INVITE_KEY_CERTIFY_PHONE  = "certify_phone"  //邀请手机认证
	INVITE_KEY_CERTIFY_VIDEO  = "certify_video"  //邀请视频认证
	INVITE_KEY_CERTIFY_IDCARD = "certify_idcard" //邀请身份证认证
	INVITE_KEY_REQUIRE        = "require"        //邀请填写则有需求
	INVITE_KEY_PHOTO          = "photo"          //邀请上传照片
)

//InviteFill邀请用户完善资料
//	key: 邀请类型，INVITE_KEY_CERTIFY-认证邀请，INVITE_KEY_REQUIRE-择友条件
func InviteFill(inviter, receiver uint32, key string) (e error) {
	var notifyKey string
	switch key {
	case INVITE_KEY_CERTIFY:
		user, e := user_overview.GetUserObject(receiver)
		if e != nil {
			return e
		}
		if user.CertifyPhone != 0 || user.CertifyVideo != 0 || user.CertifyIDcard != 0 {
			return errors.New("user already certified")
		}
		notifyKey = notify.NOTIFY_INV_CERTIFY
	case INVITE_KEY_REQUIRE:
		info, e := GetUserInfo(receiver)
		if e != nil {
			return e
		}
		if info["answercount"] != "0" {
			return errors.New("user already submit require")
		}
		notifyKey = notify.NOTIFY_INV_FILL_REQUIRE
	case INVITE_KEY_PHOTO:
		notifyKey = notify.NOTIFY_INV_PHOTO
	default:
		return errors.New(fmt.Sprintf("invalid key : %v", key))
	}
	rkey := fmt.Sprintf("%v_%v", key, receiver)
	con := rdb.GetWriteConnection(redis_db.REDIS_INVITE_FILL)
	defer con.Close()
	_, e = con.Do("ZADD", rkey, utils.Now.Unix(), inviter)
	if e != nil {
		return
	}
	//stat.Append(receiver, stat.ACTION_INVITED, map[string]interface{}{"key": key})
	exist, e := cache.Exists(redis_db.CACHE_INVITE_FILL, receiver)
	if e != nil {
		mainlog.Append("check exists error:" + e.Error())
		return nil
	}
	//24小时只能收到一次
	if !exist {
		//		if pri, _ := certify.CheckPrivilege(inviter, common.PRI_INVITE_NOTIFY); !pri {
		//			//没有给对方发送通知的权限
		//			return nil
		//		}

		not, e := notify.GetNotify(inviter, notifyKey, nil, "", "", receiver)
		if e != nil {
			mainlog.Append("GetNotify error:" + e.Error())
			return nil
		}
		_, e = general.SendMsg(inviter, receiver, map[string]interface{}{"type": common.MSG_TYPE_INVITE_FILL, notify.NOTIFY_KEY: not}, "")
		if e != nil {
			mainlog.Append("send invite message error:" + e.Error())
			return nil
		}
		ccon := cache.GetWriteConnection(redis_db.CACHE_INVITE_FILL)
		defer ccon.Close()
		_, e = ccon.Do("SETEX", rkey, 86400, "1")
		if e != nil {
			mainlog.Append("set CACHE_INVITE_FILL error:" + e.Error())
		}
	}
	return nil
}

//InviteList获取邀请uid用户完善资料的用户列表
//	key: 邀请类型，INVITE_KEY_CERTIFY-认证邀请，INVITE_KEY_REQUIRE-择友条件，INVITE_KEY_PHOTO-添加照片
func InviteList(uid uint32, key string) (users []User, e error) {
	rkey := fmt.Sprintf("%v_%v", key, uid)
	t := time.Date(utils.Now.Year(), utils.Now.Month(), utils.Now.Day(), 0, 0, 0, 0, time.Local).Unix()
	items, e := rdb.ZRangeByScore(redis_db.REDIS_INVITE_FILL, rkey, t, "+inf")
	if e != nil {
		return nil, e
	}
	users, e = makeUsersInfo(items)
	return
}

//NotifyAndDelInviteList通知并删除邀请用户
//	key: 邀请类型，INVITE_KEY_REQUIRE-择友条件， INVITE_KEY_PHOTO-添加照片，INVITE_KEY_CERTIFY_PHONE-手机认证邀请，INVITE_KEY_CERTIFY_VIDEO-视频认证，INVITE_KEY_CERTIFY_IDCARD-身份证认证
func NotifyAndDelInviteList(uid uint32, key string) {
	//stat.Append(uid, stat.ACTION_INVITED_FILL, map[string]interface{}{"key": key})
	go notifyAndDelInviteList(uid, key)
}
func notifyAndDelInviteList(uid uint32, key string) {
	var k, notifyKey string
	switch key {
	case INVITE_KEY_REQUIRE:
		notifyKey = notify.NOTIFY_INV_FILL_REQUIRE_FINISH
		k = key
	case INVITE_KEY_PHOTO:
		notifyKey = notify.NOTIFY_INV_PHOTO_FINISH
		k = key
	case INVITE_KEY_CERTIFY_PHONE:
		notifyKey = notify.NOTIFY_INV_CERTIFY_PHONE_FINISH
		k = INVITE_KEY_CERTIFY
	case INVITE_KEY_CERTIFY_VIDEO:
		notifyKey = notify.NOTIFY_INV_CERTIFY_VIDEO_FINISH
		k = INVITE_KEY_CERTIFY
	case INVITE_KEY_CERTIFY_IDCARD:
		notifyKey = notify.NOTIFY_INV_CERTIFY_IDCARD_FINISH
		k = INVITE_KEY_CERTIFY
	default:
		mainlog.Append("notifyAndDelInviteList error: unknown key " + key)
		return
	}
	rkey := fmt.Sprintf("%v_%v", k, uid)
	items, _, e := rdb.ZREVRangeWithScores(redis_db.REDIS_INVITE_FILL, rkey, 0, 100)
	if e != nil {
		mainlog.Append("notifyAndDelInviteList error:" + e.Error())
		return
	}

	for _, item := range items {
		id, e := utils.ToUint32(item.Key)
		if e != nil {
			mainlog.Append("send fill_finish message error:" + e.Error())
		}

		not, e := notify.GetNotify(uid, notifyKey, nil, "", "", id)
		if e != nil {
			mainlog.Append("GetNotify error:" + e.Error())
		}
		_, e = general.SendMsg(uid, id, map[string]interface{}{"type": common.MSG_TYPE_FILL_FINISH, notify.NOTIFY_KEY: not}, "")
		if e != nil {
			mainlog.Append("send fill_finish message error:" + e.Error())
		}
	}

	if e = rdb.Del(redis_db.REDIS_INVITE_FILL, rkey); e != nil {
		mainlog.Append("notifyAndDelInviteList error:" + e.Error())
	}
}
