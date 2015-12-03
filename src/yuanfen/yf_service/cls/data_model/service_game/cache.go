package service_game

import (
	"encoding/json"
	"yf_pkg/redis"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
)

// 获取游戏列表缓存
func readGameDataCache() (exists bool, games []GameData, e error) {
	if exists, e = cache.Exists(redis_db.CACHE_GAME, common.CACHE_GAME_KEY_GAMELIST); e != nil || !exists {
		return
	}
	v, e := redis.Values(cache.LRange(redis_db.CACHE_GAME, common.CACHE_GAME_KEY_GAMELIST, 0, -1))
	if e != nil {
		return
	}
	vas := make([][]byte, 0, 300)
	if e = redis.ScanSlice(v, &vas); e != nil {
		return false, nil, e
	}

	games = make([]GameData, 0, len(v))
	for _, b := range vas {
		var g GameData
		if e = json.Unmarshal(b, &g); e != nil {
			return
		}
		games = append(games, g)
	}
	return true, games, nil
}

// 写入游戏列表缓存
func writeGameDataCache(games []GameData) (e error) {
	key := common.CACHE_GAME_KEY_GAMELIST
	v := make([]interface{}, 0, 50)
	for _, item := range games {
		b, e := json.Marshal(item)
		if e != nil {
			return e
		}
		v = append(v, b)
	}
	if e = cache.Del(redis_db.CACHE_GAME, key); e != nil {
		return
	}
	if _, e = cache.RPush(redis_db.CACHE_GAME, key, v...); e != nil {
		return
	}
	e = cache.Expire(redis_db.CACHE_GAME, 3600, key)
	return
}
