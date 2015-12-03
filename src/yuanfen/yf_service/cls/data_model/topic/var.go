package topic

import (
	"time"
	"yf_pkg/cachedb"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yuanfen/yf_service/cls"
)

const (
	BIRTHDAY_RANGE     = 5 * 365 * 24 * time.Hour
	RANGE_CITY         = 3
	RANGE_PROVINCE     = 4
	RANGE_DEFAULT      = RANGE_PROVINCE //默认范围ID
	RANGE_MAX          = 10000          //最大范围ID
	RANGE_SECOND_MAX   = 1              //次大范围ID
	RANGE_MIN          = 1              //最小范围ID
	RANGE_UNKNOWN      = 0              //未知范围
	TOPIC_MAX_CAPACITY = 100            //话题人数上限
	TOPIC_TIMEOUT      = 30 * 24 * 3600 //话题超时时间（秒)
	TOPIC_TOP_USER_NUM = 10             //返回的活跃用户总数
)

var mdb *mysql.MysqlDB
var msgdb *mysql.MysqlDB
var sdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool
var cdb *cachedb.CacheDB
var mainLog *log.MLogger

var TrendName map[int]string = map[int]string{0: "", 1: "火爆", 2: "非常火爆"}
var TREND_MAX int = 2

var Ranges RangeLevels

func Init(env *cls.CustomEnv) (e error) {
	sdb = env.SortDB
	mdb = env.MainDB
	msgdb = env.MsgDB
	rdb = env.MainRds
	cache = env.CacheRds
	cdb = env.CacheDB
	mainLog = env.MainLog

	Ranges, e = NewRangeLevels()
	//	msg.RegisterTag(common.TAG_PREFIX_TOPIC, Send)
	//	msg.RegisterDelTagMsgFunc(common.TAG_TYPE_TOPIC, DelMsg)
	//unread.Register(common.UNREAD_MY_TOPIC, UnreadNum)
	//	message.RegisterNotification(message.OFFLINE, userOffline)
	return e
}
