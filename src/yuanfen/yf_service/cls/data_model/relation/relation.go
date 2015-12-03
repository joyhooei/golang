package relation

import (
	"fmt"
	"time"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/relation/base"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/unread"
)

type RecentUser struct {
	Uid      uint32      `json:"uid"`
	Nickname string      `json:"nickname"`
	Gender   int         `json:"gender"`
	Avatar   string      `json:"avatar"`
	Tag      uint16      `json:"tag"`
	LastMsg  interface{} `json:"last_msg"`
	Tm       time.Time   `json:"tm"`
}

var mdb *mysql.MysqlDB
var msgdb *mysql.MysqlDB
var sdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool
var mainLog *log.MLogger

/*
type School struct {
	Id     int    `json:"id"`
	School string `json:"school"`
	Owner  string `json:"owner"`
	Area   string `json:"area"`
	Level  string `json:"level"`
	Tip    string `json:"tip"`
}

func exportSchool() {
	rows, _ := mdb.Query("select id,school,owner,area,level,tip from school")
	defer rows.Close()
	sc := []School{}
	for rows.Next() {
		var s School
		rows.Scan(&s.Id, &s.School, &s.Owner, &s.Area, &s.Level, &s.Tip)
		sc = append(sc, s)
	}
	j, e := json.Marshal(sc)
	if e != nil {
		fmt.Println("generate json error:", e.Error())
		return
	}
	mainLog.Append(string(j), log.DEBUG)
}
*/

func Init(env *cls.CustomEnv) {
	sdb = env.SortDB
	mdb = env.MainDB
	msgdb = env.MsgDB
	rdb = env.MainRds
	cache = env.CacheRds
	mainLog = env.MainLog
	unread.Register(common.UNREAD_FANS, UnreadNum)
	unread.Register(common.UNREAD_SAYHELLO, UnreadNum)
	base.Init(env)
	go checkDateRequest()
	//	exportSchool()
}

/*
DelRecentChatUser删除最近聊天列表的某个用户，同时也会把聊天记录的起始时间移动到当前时间，之前的聊天记录就再也不会返回给这个用户了。
*/
func DelRecentChatUser(me uint32, him uint32) error {
	lastMsgID, e := general.GetLastMsgID()
	if e != nil {
		return e
	}
	if e := rdb.HSet(redis_db.REDIS_USER_MSG_START_POS, me, him, lastMsgID); e != nil {
		return e
	}
	if e := rdb.HDel(redis_db.REDIS_USER_DATA, me, general.MakeKey(common.LAST_MSG_ID_PREFIX, him)); e != nil {
		return e
	}
	_, e = rdb.ZRem(redis_db.REDIS_RECENT_CHAT_USERS, me, him)
	return e
}

/*
GetRecentChatUserList获取最近聊天的用户列表
*/
func GetRecentChatUserList(uid uint32, cur int, ps int, res map[string]interface{}) (users []RecentUser, total int, e error) {
	items, total, e := rdb.ZREVRangeWithScoresPS(redis_db.REDIS_RECENT_CHAT_USERS, uid, cur, ps)
	if e != nil {
		return nil, 0, e
	}
	users, e = makeRecentUsersInfo(uid, items)
	var msgid uint64
	if e = msgdb.QueryRow("select max(id) from message").Scan(&msgid); e != nil {
		return
	}
	res["last_msgid"] = msgid
	return
}

//---------------------Private Functions-----------------------//

func makeRecentUsersInfo(uid uint32, items []redis.ItemScore) (users []RecentUser, e error) {
	//取出当前请求的uid集合
	uids := make([]uint32, 0, len(items))
	keys := make([]interface{}, 0, len(items))
	for _, u := range items {
		if tid, e := utils.ToUint32(u.Key); e != nil {
			return nil, e
		} else {
			uids = append(uids, tid)
			keys = append(keys, general.MakeKey(common.LAST_MSG_ID_PREFIX, tid))
		}
	}
	//获取用户信息
	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, e
	}
	//获取标记
	tags, e := GetFollowTags(uid, uids)
	if e != nil {
		return nil, e
	}

	//获取最后一条消息
	mids := make([]uint64, len(items))
	if e := rdb.HMGet(redis_db.REDIS_USER_DATA, &mids, uid, keys...); e != nil {
		return nil, e
	}
	msgs, e := general.GetMessageById(mids)
	if e != nil {
		return nil, e
	}
	fmt.Println("msgs:", msgs)

	users = make([]RecentUser, 0, len(uids))
	for i, item := range items {
		if ui := uinfos[uids[i]]; ui != nil {
			var msg interface{}
			if mids[i] != 0 {
				msg = msgs[mids[i]]
			}
			users = append(users, RecentUser{uids[i], ui.Nickname, ui.Gender, ui.Avatar, tags[uids[i]], msg, time.Unix(int64(item.Score), 0)})
		}
	}
	return users, nil
}
