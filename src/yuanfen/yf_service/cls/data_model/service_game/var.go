package service_game

import (
	"fmt"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/message"
	sys_message "yuanfen/yf_service/cls/message"
	"yuanfen/yf_service/cls/unread"
)

var mdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool
var mlog *log.Logger

func Init(env *cls.CustomEnv, conf service.Config) {
	fmt.Println("init service_game")
	mdb = env.MainDB
	rdb = env.MainRds
	cache = env.CacheRds
	l, err := log.New2(conf.LogDir+"/sgame.log", 10000, conf.LogLevel)
	if err != nil {
		fmt.Println("初始化日志error:", err.Error())
	}
	mlog = l
	message.RegisterTag(common.TAG_PREFIX_GAME, Send)

	unread.Register(common.UNREAD_GAME, UnreadNum)
	unread.Register(common.UNREAD_VERSION, UnreadNum)

	sys_message.RegisterNotification(sys_message.OFFLINE, userOffline)
}

const (
	GAMEAUTH_STATUS_OK            = 0 // 授权正确
	GAMEAUTH_STATUS_NOAUTH        = 1 // 未授权
	GAMEAUTH_STATUS_CODE_TIMEOUT  = 2 // 授权码过期
	GAMEAUTH_STATUS_TOKEN_TIMEOUT = 3 // access token 过期
)

const (
	GAME_AUTH_PRI_KEY = "qiuqian_game_2015_win" // 与游戏服务通讯的密钥
)

/*
游戏授权对象
*/
type GameAuth struct {
	Uid     uint32 // 授权uid
	AppId   string // 授权游戏ID
	Code    string // 授权码
	CodeTm  string // 授权码有效期
	Token   string // access token
	TokenTm string // access token 有效期
}

// 新游戏对象
type GameData struct {
	Id     uint32 `json:"id"`
	AppId  string `json:"appid"`
	Name   string `json:"name"`
	Info   string `json:"info"`
	Img    string `json:"img"`
	Pack   string `json:"pack"`
	Url    string `json:"url"`
	Size   string `json:"size"`
	Class  string `json:"class"`
	Secret string `json:"secret"`
	IsHot  int    `json:"isHost"`
	IsNew  int    `json:"isNew"`
}

// 中奖通知对象（用于接受游戏服务端参数）
type GameAward struct {
	Uid     uint32 `json:"uid"`     // 中奖用户
	AwardId uint32 `json:"awardid"` // 中奖id
	Count   int    `json:"count"`   // 数量
	Info    string `json:"info"`    // 中奖原因
}

// 游戏奖品配置列表
type GameAwardConf struct {
	AppId   string `json:"appid"`   // appid
	AwardId uint32 `json:"awardid"` // 中奖id
	Num     int    `json:"num"`     // 总数
	Balance int    `json:"balane"`  // 奖品余额
}
