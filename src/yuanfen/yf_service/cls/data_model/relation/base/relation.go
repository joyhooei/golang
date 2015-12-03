package base

import (
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
)

var mdb *mysql.MysqlDB
var msgdb *mysql.MysqlDB
var sdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool
var mainLog *log.MLogger

func Init(env *cls.CustomEnv) {
	sdb = env.SortDB
	mdb = env.MainDB
	msgdb = env.MsgDB
	rdb = env.MainRds
	cache = env.CacheRds
	mainLog = env.MainLog
}

//ReplySayHello用户me回应用户him的认识一下请求，如果成功，则返回true。
func ReplySayHello(me uint32, him uint32) (success bool, e error) {
	//判断是否在黑名单中，如果在黑名单中，则不能回复
	isBlack, e := IsInBlacklist(me, him)
	if e != nil {
		return false, e
	}
	if isBlack {
		return false, service.NewError(service.ERR_IN_BLACKLIST, "he is in your blacklist.", "该用户在您的黑名单中")
	}
	mid, e := redis.Uint32(rdb.HGet(redis_db.REDIS_USER_DATA, me, general.MakeKey(common.LAST_SAYHELLO_TO_ME_PREFIX, him)))
	switch e {
	case nil:
	case redis.ErrNil:
		return false, nil
	default:
		return false, e
	}
	//检查回复的是否是一条合法的认识请求
	sql := "select `stat` from sayhello_msg where id=?"
	var stat int
	if e = mdb.QueryRow(sql, mid).Scan(&stat); e != nil {
		return false, e
	}
	if stat != common.SAYHELLO_MSG_READ && stat != common.SAYHELLO_MSG_UNREAD {
		return false, nil
	}
	//从him的认识请求列表中删除me
	if _, e = rdb.ZRem(redis_db.REDIS_SAYHELLO, general.MakeKey(common.SAYHELLO_TARGET_HIM, him), me); e != nil {
		return false, e
	}
	if e = rdb.HDel(redis_db.REDIS_USER_DATA, him, general.MakeKey(common.LAST_SAYHELLO_TO_HIM_PREFIX, me)); e != nil {
		return false, e
	}
	//添加好友
	if e = AddFriend(me, him); e != nil {
		return false, e
	}
	//把消息标记为已回复
	if _, e = mdb.Exec("update sayhello_msg set stat=? where id=?", common.SAYHELLO_MSG_REPLIED, mid); e != nil {
		return false, e
	}
	//发送回复消息
	return true, nil
}

func IsFriend(from, to uint32) (bool, error) {
	return rdb.ZExists(redis_db.REDIS_FRIENDS, from, to)
}

//查看用户是否在黑名单中
func IsInBlacklist(uid uint32, badUser uint32) (bool, error) {
	return rdb.ZIsMember(redis_db.REDIS_BLACKLIST, uid, badUser)
}

//添加为认识的人，仅供内部调用，用户不能直接加好友，必须通过认识一下相关接口添加好友
func AddFriend(from, to uint32) error {
	_, e := mdb.Exec("update follow set friend=1 where (f_uid=? and t_uid=?) or (f_uid=? and t_uid=?)", to, from, from, to)
	if e != nil {
		return e
	}
	e = rdb.ZAdd(redis_db.REDIS_FRIENDS, from, utils.Now.Unix(), to, to, utils.Now.Unix(), from)
	if e != nil {
		return e
	}
	stat.Append(from, stat.ACTION_GET_FRIEND, map[string]interface{}{"with": to})
	stat.Append(to, stat.ACTION_GET_FRIEND, map[string]interface{}{"with": from})
	return nil
}
