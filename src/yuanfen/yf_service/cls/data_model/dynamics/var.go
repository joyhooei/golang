package dynamics

import (
	"fmt"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/unread"
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
	l, err := log.New2(conf.LogDir+"/sdynamics.log", 10000, conf.LogLevel)
	if err != nil {
		fmt.Println("初始化日志error:", err.Error())
	}
	mlog = l
	// 标记动态未读
	unread.Register(common.UNREAD_DYNAMIC_MARK, UnreadNum)
}

const (
	REDIS_DYNAMIC_KEY = "dynamic_list_" //动态redis列表
	//	REDIS_EX_DYNAMIC_KEY = "ex_dynamic_list_" //优秀推荐动态redis列表 ,省
	REDIS_EX_DYNAMIC_KEY = "ex_dynamic_list" //优秀推荐动态redis列表, 全国
	//动态状态（0 正常 1 删除 2 封禁 3 图片不符合，4 文字不符合 ）
	DYNAMIC_STATUS_OK         = 0
	DYNAMIC_STATUS_DELETE     = 1
	DYNAMIC_STATUS_BAN        = 2
	DYNAMIC_STATUS_PICINVALID = 3
	DYNAMIC_STATUS_TXTINVALID = 4
)

/*
 动态对象
*/
type Dynamic struct {
	Id         uint32 `json:"id"`       // 动态id
	Uid        uint32 `json:"uid"`      // 动态发布人uid
	Type       int    `json:"type"`     // 动态类型 1：用户动态 2：小游戏 3：文章
	Stype      int    `json:"stype"`    // 用户动态类型  0 主动发送 1 交友寄语 2 形象照
	Pic        string `json:"pic"`      // 动态图片，已英文逗号隔开, json返回格式中已字符串数组的形式返回
	Text       string `json:"text"`     // 动态文字内容
	Location   string `json:"location"` // 动态发布位置
	Tm         string `json:"tm"`       // 动态发布时间
	Report     int    `json:"report"`   // 动态被举报次数
	Likes      int    `json:"likes"`    // 动态点赞次数
	Comments   int    `json:"comments"` // 动态评论数
	Url        string `json:"url"`      // 当type等于3时，文章url，默认空字符串
	GamgeInit  string `json:"gameurl"`  // 当type等于2时，拼图游戏初始序列字符串英文逗号隔开，9个数字
	GamgeKey   int    `json:"gamekey"`  // 当type等于2时，拼图游戏key
	GameAnswer string `json:"-"`        // 当type等于2时，拼图游戏结果
	Status     int    `json:"status"`   // 动态状态（0 正常 1 删除 2 封禁 3待扩展 ）
	IsLike     int    `json:"isLike"`   // 是否点过赞
	Sign       int    `json:"sign"`     // 用户标记状态（0，无意义，1 已标记 2，未标记 ）
}

/*
拼图游戏用户和时间
*/
type DynamicGameTm struct {
	Uid uint32 `json:"uid"` // 用户uid
	Tm  int    `json:"tm"`  // 拼图时间
}
