package unread

import (
	"time"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
)

type Item struct {
	Num  uint32 `json:"num"`
	Show string `json:"show"`
}

var required []string = []string{common.UNREAD_EVENT, common.UNREAD_DYNAMIC_MARK}

//获取未读数的通用接口
type UnreadNum func(uid uint32, key string, from time.Time) (uint32, string)

var unread map[string]UnreadNum
var rds *redis.RedisPool

func init() {
	unread = make(map[string]UnreadNum)
}

func Init(redis *redis.RedisPool) {
	rds = redis
}

func Register(key string, i UnreadNum) {
	unread[key] = i
}

//获取上次已读时间
func GetReadTime(uid uint32, key string) (time.Time, error) {
	conn := rds.GetReadConnection(redis_db.REDIS_UNREAD_TIME)
	defer conn.Close()
	s, e := redis.Int64(conn.Do("HGET", uid, key))
	switch e {
	case nil:
		return time.Unix(s, 0), nil
	case redis.ErrNil:
		return time.Time{}, nil
	default:
		return time.Time{}, e
	}
}

//获取所有上次已读时间
func getAllReadTime(uid uint32) (map[string]time.Time, error) {
	values := map[string]time.Time{}
	conn := rds.GetReadConnection(redis_db.REDIS_UNREAD_TIME)
	defer conn.Close()
	kv, e := redis.Values(conn.Do("HGETAll", uid))
	switch e {
	case nil:
		res := make([]struct {
			Key string
			Tm  int64
		}, 0, 10)
		if err := redis.ScanSlice(kv, &res); err != nil {
			return nil, err
		}
		for _, item := range res {
			values[item.Key] = time.Unix(item.Tm, 0)
		}
		return values, nil
	case redis.ErrNil:
		return values, nil
	default:
		return nil, e
	}
}

func UpdateUnread(uid uint32, key string, res map[string]interface{}) (e error) {
	if e = UpdateReadTime(uid, key); e != nil {
		return e
	}
	ur := map[string]interface{}{key: 0}
	e = GetUnreadNum(uid, ur)
	if e != nil {
		return e
	}
	res[common.UNREAD_KEY] = ur
	return nil
}

//更新已读时间到当前时间
func UpdateReadTime(uid uint32, key string, tm ...time.Time) error {
	s := utils.Now.Unix()
	if len(tm) > 0 {
		s = tm[0].Unix()
	}

	conn := rds.GetWriteConnection(redis_db.REDIS_UNREAD_TIME)
	defer conn.Close()
	if _, e := conn.Do("HSET", uid, key, s); e != nil {
		return e
	}
	return nil
}

func UpdateReadTimes(uid uint32, res map[string]interface{}) error {
	conn := rds.GetWriteConnection(redis_db.REDIS_UNREAD_TIME)
	defer conn.Close()
	for key, _ := range res {
		if _, e := conn.Do("HSET", uid, key, utils.Now.Unix()); e != nil {
			return e
		}
	}
	return nil
}

//获取res中每个key对应的未读数
//如果没找到对应的key，设置数量为0
func GetUnreadNum(uid uint32, res map[string]interface{}) error {
	for key, _ := range res {
		unreadNum, ok := unread[key]
		var item Item
		if ok {
			from, e := GetReadTime(uid, key)
			if e != nil {
				return e
			}
			item.Num, item.Show = unreadNum(uid, key, from)
			res[key] = item
		} else {
			res[key] = item
		}
	}
	return nil
}
func GetAllUnreadNum(uid uint32) (res map[string]interface{}, e error) {
	items, e := getAllReadTime(uid)
	if e != nil {
		return nil, e
	}
	res = map[string]interface{}{}
	for key, _ := range items {
		var item Item
		unreadNum, ok := unread[key]
		if ok {
			item.Num, item.Show = unreadNum(uid, key, items[key])
			res[key] = item
		} else {
			res[key] = item
		}
	}
	for _, key := range required {
		_, ok := res[key]
		if !ok {
			var item Item
			unreadNum, ok := unread[key]
			if ok {
				item.Num, item.Show = unreadNum(uid, key, time.Unix(0, 0))
				res[key] = item
			} else {
				res[key] = item
			}
		}

	}
	return res, nil
}
