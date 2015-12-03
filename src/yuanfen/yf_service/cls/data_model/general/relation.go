package general

import (
	"time"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
)

/*
UpdateRecentChatUser更新聊天双方的最近聊天列表中的信息，如果最近聊天列表里没有，则添加。

参数：
	uid1,uid2: 聊天双方uid
	msgid: 最后一条聊天记录
	mtype: 消息类型
*/
func UpdateRecentChatUser(uid1 uint32, uid2 uint32, msgid uint64, mtype string) error {
	if !common.IsChatMessage(mtype) || IsSystemUser(uid1) || IsSystemUser(uid2) {
		return nil
	}
	if e := rdb.HMultiSet(redis_db.REDIS_USER_DATA, uid1, MakeKey(common.LAST_MSG_ID_PREFIX, uid2), msgid, uid2, MakeKey(common.LAST_MSG_ID_PREFIX, uid1), msgid); e != nil {
		return e
	}
	return rdb.ZAdd(redis_db.REDIS_RECENT_CHAT_USERS, uid1, utils.Now.Unix(), uid2, uid2, utils.Now.Unix(), uid1)
}

//GetInterestUids获取标记的用户集合，包括特别关注的用户，结果集是一个map，方便查找。
func GetInterestUids(uid uint32) (users map[uint32]time.Time, e error) {
	k := MakeKey("f", uid)
	items, total, e := rdb.ZREVRangeWithScores(redis_db.REDIS_FOLLOW, k, 0, -1)
	if e != nil {
		return nil, e
	}
	users = make(map[uint32]time.Time, total)
	for _, item := range items {
		u, e := utils.ToUint32(item.Key)
		if e != nil {
			return nil, e
		}
		users[u] = time.Unix(int64(item.Score), 0)
	}
	return
}

//GetInterestUids获取被标记的用户集合，包括特别关注的用户，结果集是一个map，方便查找。
func GetInterestedUids(uid uint32) (users map[uint32]time.Time, e error) {
	k := MakeKey("t", uid)
	items, total, e := rdb.ZREVRangeWithScores(redis_db.REDIS_FOLLOW, k, 0, -1)
	if e != nil {
		return nil, e
	}
	users = make(map[uint32]time.Time, total)
	for _, item := range items {
		u, e := utils.ToUint32(item.Key)
		if e != nil {
			return nil, e
		}
		users[u] = time.Unix(int64(item.Score), 0)
	}
	return
}

//清空用户的所有消息
func ClearAllMessages(uid uint32) (e error) {
	lastMsgID, e := GetLastMsgID()
	if e != nil {
		return e
	}
	//把第一条消息的游标移到当前最大msgid
	if e := rdb.HSet(redis_db.REDIS_USER_DATA, uid, common.MSG_START_POS, lastMsgID); e != nil {
		return e
	}
	//删除最近聊天列表
	e = rdb.Del(redis_db.REDIS_RECENT_CHAT_USERS, uid)
	return
}
