package award

import (
	"encoding/json"
	"yf_pkg/redis"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
)

// 获取游戏列表缓存
func readAwardCache() (exists bool, awards []*Award, e error) {
	if exists, e = cache.Exists(redis_db.CACHE_GAME, common.CACHE_GAME_KEY_AWARD); e != nil || !exists {
		return false, nil, e
	}
	vas := make([][]byte, 0, 100)
	conn := cache.GetReadConnection(redis_db.CACHE_GAME)
	defer conn.Close()
	v, e := redis.Values(conn.Do("LRANGE", common.CACHE_GAME_KEY_AWARD, 0, -1))
	if e != nil {
		return false, nil, e
	}
	if e := redis.ScanSlice(v, &vas); e != nil {
		return false, nil, e
	}
	awards = make([]*Award, 0, 20)
	for _, b := range vas {
		i := new(Award)
		if e = json.Unmarshal(b, &i); e != nil {
			return false, nil, e
		}
		awards = append(awards, i)
	}
	return true, awards, nil
}

// 写入游戏列表缓存
func writeAwardCache(awards []*Award) error {
	key := common.CACHE_GAME_KEY_AWARD
	con := cache.GetWriteConnection(redis_db.CACHE_GAME)
	defer con.Close()
	v := make([]interface{}, 0, 50)
	v = append(v, key)
	for _, item := range awards {
		b, e := json.Marshal(item)
		if e != nil {
			return e
		}
		v = append(v, b)
	}
	if _, e := con.Do("DEL", key); e != nil {
		return e
	}
	if _, e := con.Do("RPUSH", v...); e != nil {
		return e
	}
	_, e := con.Do("EXPIRE", key, 24*60*60)
	return e
}
