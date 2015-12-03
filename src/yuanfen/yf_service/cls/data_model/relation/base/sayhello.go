package base

import (
	"fmt"
	"yf_pkg/log"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/notify"
	"yuanfen/yf_service/cls/unread"
)

//尝试发送之前被挂起的请求
func TryPendingSayHello(uid uint32) (e error) {
	mainLog.Append(fmt.Sprintf("%v Try send SayHello...", uid), log.DEBUG)
	sql := "select `to`,`content` from sayhello_msg_pending where `from`=? order by tm asc"
	rows, e := mdb.Query(sql, uid)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var to uint32
		var content string
		if e = rows.Scan(&to, &content); e != nil {
			return e
		}
		if _, _, e = SayHello(uid, to, content, true); e != nil {
			mainLog.Append("Try send SayHello error:" + e.Error())
		}
	}
	if _, e = mdb.Exec("delete from sayhello_msg_pending where `from`=?", uid); e != nil {
		return e
	}
	return
}

/*
SayHello发送认识一下请求，如果发送者还在审核中，则挂起，直到审核通过才会发出去。

参数：
	args: 可变参数，第一项表示是否是挂起的消息，不填表示不是挂起的消息。
*/
func SayHello(me uint32, him uint32, content string, args ...bool) (msgid uint64, online bool, e error) {
	isPendingMsg := false
	if len(args) > 0 && args[0] {
		isPendingMsg = true
	}
	fmt.Println("isPendingMsg:", isPendingMsg)
	onlines, e := user_overview.IsOnline(him)
	if e != nil {
		mainLog.Append("get user online error:" + e.Error())
		online = false
	} else {
		online = onlines[him]
	}
	uinfo, e := user_overview.GetUserObject(me)
	if e != nil {
		return 0, false, e
	}
	if !isPendingMsg {
		n, e := redis.Int(cache.Get(redis_db.CACHE_SAYHELLO_TIMES, general.MakeKey(me, him)))
		switch e {
		case nil:
			if n >= 3 {
				return 0, false, service.NewError(service.ERR_TOO_MANY, "too many sayhello", "每周最多给同一个人追加3条消息")
			}
		case redis.ErrNil:
			if e = cache.SetEx(redis_db.CACHE_SAYHELLO_TIMES, general.MakeKey(me, him), 86400*7, 0); e != nil {
				return 0, false, e
			}
		default:
			return 0, online, e
		}
		stat.Append(me, stat.ACTION_SAYHELLO, map[string]interface{}{"to": him})
		if !uinfo.IsValid() {
			//如果还在审核中，则暂存到挂起消息表中。
			sql := "insert into sayhello_msg_pending(`from`,`to`,`content`)values(?,?,?)"
			_, e := mdb.Exec(sql, me, him, content)
			if e != nil {
				return 0, false, e
			}
			cache.Incr(redis_db.CACHE_SAYHELLO_TIMES, general.MakeKey(me, him))
			return 0, false, nil
		}
	}

	//1.添加消息到mysql
	//2.添加到对方收到的请求列表
	//3.如果在对方的黑名单，则结束
	//4.否则，添加到对方收到的请求列表，发送一条推送给对方
	sql := "insert into sayhello_msg(`from`,`to`,`content`,`stat`)values(?,?,?,?)"
	res, e := mdb.Exec(sql, me, him, content, common.SAYHELLO_MSG_UNREAD)
	if e != nil {
		return 0, false, e
	}
	mid, e := res.LastInsertId()
	if e != nil {
		return 0, false, e
	}
	if e = rdb.HSet(redis_db.REDIS_USER_DATA, me, general.MakeKey(common.LAST_SAYHELLO_TO_HIM_PREFIX, him), mid); e != nil {
		return 0, false, e
	}
	if e = rdb.ZAdd(redis_db.REDIS_SAYHELLO, general.MakeKey(common.SAYHELLO_TARGET_HIM, me), utils.Now.Unix(), him); e != nil {
		return 0, false, e
	}
	isBlack, e := IsInBlacklist(him, me)
	if e != nil {
		return 0, false, e
	}
	if !isBlack {
		if e = rdb.HSet(redis_db.REDIS_USER_DATA, him, general.MakeKey(common.LAST_SAYHELLO_TO_ME_PREFIX, me), mid); e != nil {
			return 0, false, e
		}
		if e = rdb.ZAdd(redis_db.REDIS_SAYHELLO, general.MakeKey(common.SAYHELLO_TARGET_ME, him), utils.Now.Unix(), me); e != nil {
			return 0, false, e
		}
		p, e := general.GetUserProtect(him)
		if e != nil {
			mainLog.Append("get user protect error:" + e.Error())
		} else {
			ur := map[string]interface{}{common.UNREAD_SAYHELLO: 0}
			fmt.Println("unread")
			e := unread.GetUnreadNum(him, ur)
			if e != nil {
				mainLog.Append("get unread num error:" + e.Error())
			}
			not, e := notify.GetNotify(me, notify.NOTIFY_SAY_HELLO, nil, "", content, him)
			if e != nil {
				mainLog.Append("get notify error:" + e.Error())
			}
			msgid, _, e = general.SendMsgWithOnline(me, him, map[string]interface{}{"f_uid": me, "t_uid": him, "type": common.MSG_TYPE_SAYHELLO, "f_nickname": uinfo.Nickname, "content": content, common.UNREAD_KEY: ur, notify.NOTIFY_KEY: not}, "", p.SayHelloNotNotify == 0)
			stat.Append(me, stat.ACTION_SAYHELLO_SUC, map[string]interface{}{"to": him})
		}
	}
	//如果是发送挂起的消息，则不增加数量。
	if !isPendingMsg {
		cache.Incr(redis_db.CACHE_SAYHELLO_TIMES, general.MakeKey(me, him))
	}
	return
}
