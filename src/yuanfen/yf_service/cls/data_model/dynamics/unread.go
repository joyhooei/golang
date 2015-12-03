package dynamics

import (
	"time"
	"yf_pkg/mysql"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/unread"
)

// 未读消息
func UnreadNum(uid uint32, key string, from time.Time) (total uint32, show string) {
	switch key {
	case common.UNREAD_DYNAMIC_MARK:
		n, e := getUnreadMarkDynamic(uid, from)
		if e != nil {
			mlog.AppendObj(e, "UnreadNum UNREAD_DYNAMIC_MARK is error: ", uid)
		}
		if n > 0 {
			total = n
			show = "[红点]"
		}
		return
	}
	return 0, ""
}

/*
获取标记用户未读动态
uid:当前用户uid
*/
func getUnreadMarkDynamic(uid uint32, from time.Time) (num uint32, e error) {
	// 获取我的标记用户
	m, e := general.GetInterestUids(uid)
	if e != nil {
		return
	}
	uids := make([]uint32, 0, len(m))
	for id, _ := range m {
		uids = append(uids, id)
	}
	if len(uids) <= 0 {
		return
	}
	s := "select count(*) from dynamics where  status = 0 and uid " + mysql.In(uids) + "  and tm > ?   "
	if e = mdb.QueryRow(s, from).Scan(&num); e != nil {
		mlog.AppendObj(e, "GetUnreadDynamicMsg--is wrong", uid, from, uids)
		return
	}
	return
}

/*
标记用户时，特殊角标通知,需要主意
uid:当前用户uid
返回unread节点
*/
func GetUnReadMarkDynamic(uid uint32) (un map[string]interface{}) {
	num, e := rdb.ZCard(redis_db.REDIS_DYNAMIC, GetUserDynamicKey(uid))
	if e != nil {
		mlog.AppendObj(e, "GetUnReadMarkDynamic is error", uid)
	}
	un = make(map[string]interface{})
	item := unread.Item{Num: 0, Show: ""}
	if num > 0 {
		item = unread.Item{Num: 1, Show: "[红点]"}
	}
	un[common.UNREAD_DYNAMIC_MARK] = item
	return
}
