package comments

import (
	"fmt"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yuanfen/yf_service/cls"
)

var mdb *mysql.MysqlDB
var msgdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool
var mlog *log.Logger

func Init(env *cls.CustomEnv, conf service.Config) {
	mdb = env.MainDB
	rdb = env.MainRds
	msgdb = env.MsgDB
	cache = env.CacheRds
	l, err := log.New2(conf.LogDir+"/scomments.log", 10000, conf.LogLevel)
	if err != nil {
		fmt.Println("初始化日志error:", err.Error())
	}
	mlog = l
}

const (
	//评论状态（0 正常 1 删除)
	COMENT_STATUS_OK     = 0
	COMENT_STATUS_DELETE = 1
)

// 评论模块
type Comment struct {
	Id         uint32 `json:"id"`          // 评论id
	SourceId   uint32 `json:"source_id"`   // 评论资源id
	SourceType int    `json:"source_type"` // 资源类型，1，动态评论 2，扩展
	Uid        uint32 `json:"uid"`         // 评论用户uid
	Type       int    `json:"type"`        // 评论类型    1. 点赞   2. 评论 3.拼图游戏时间
	Ruid       uint32 `json:"ruid"`        // 回复用户uid，非回复则默认为0
	Content    string `json:"content"`     // 评论内容
	Status     int    `json:"status"`      // 评论状态（0 正常 1 删除）
	Tm         string `json:"tm"`          // 评论时间
}
