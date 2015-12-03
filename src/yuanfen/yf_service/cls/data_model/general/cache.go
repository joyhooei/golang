package general

import (
	"encoding/json"
	"yf_pkg/redis"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
)

// 取通用配置
func readConfigCache() (exists bool, values []*Config, e error) {
	key := common.CACHE_GAME_KEY_CONFIG
	if ex, e := cache.Exists(redis_db.CACHE_GAME, key); !ex || e != nil {
		return false, nil, e
	}
	con := cache.GetWriteConnection(redis_db.CACHE_GAME)
	defer con.Close()
	v, e := redis.Values(con.Do("LRANGE", key, 0, -1))
	if e != nil {
		return false, nil, e
	}
	arr := make([]*Config, 0, 30)
	if e := redis.ScanSlice(v, &arr); e != nil {
		//	fmt.Println("---3-1---", e.Error())
		return false, nil, e
	}
	return true, arr, nil
}

// 写入通用配置
func writeConfigCache(values []*Config) error {
	key := common.CACHE_GAME_KEY_CONFIG
	con := cache.GetWriteConnection(redis_db.CACHE_GAME)
	defer con.Close()
	v := make([]interface{}, 0, 100)
	v = append(v, key)
	for _, c := range values {
		v = append(v, c.Key, c.Value, c.Type)
	}
	if _, e := con.Do("DEL", key); e != nil {
		return e
	}
	if _, e := con.Do("RPUSH", v...); e != nil {
		return e
	}
	_, e := con.Do("EXPIRE", key, 24*60*60)
	//fmt.Println("set cahce data  key ", key)
	return e
}

// 获取版本缓存key
func getVersionKey(c_uid, c_sid, ver string) string {
	return common.CACHE_KEY_VERSION + "_" + c_uid + "_" + c_sid + "_" + ver
}

// 获取版本更新配置
func readVersionCache(c_uid, c_sid, ver string) (exists bool, version Version, e error) {
	key := getVersionKey(c_uid, c_sid, ver)
	if ex, e := cache.Exists(redis_db.CACHE_VERSION, key); !ex || e != nil {
		return false, version, e
	}
	con := cache.GetWriteConnection(redis_db.CACHE_VERSION)
	defer con.Close()
	v, e := redis.String(con.Do("GET", key))
	if e != nil {
		return
	}
	if e = json.Unmarshal([]byte(v), &version); e != nil {
		return
	}
	//	mlog.AppendObj(e, "get cahce data  key", key, version)
	return true, version, nil
}

// 写入版本通用配置
func writeVersionCache(c_uid, c_sid, ver string, version Version) (e error) {
	key := getVersionKey(c_uid, c_sid, ver)
	con := cache.GetWriteConnection(redis_db.CACHE_VERSION)
	defer con.Close()
	b, e := json.Marshal(version)
	if e != nil {
		return
	}
	if _, e = con.Do("SET", key, string(b)); e != nil {
		return
	}
	_, e = con.Do("EXPIRE", key, 60)
	//	mlog.AppendObj(e, "set cahce data  key", key)
	return
}

// 获取背景图片缓存
func readAppImgCache() (exists bool, arr []AppImg, e error) {
	if exists, e = cache.Exists(redis_db.CACHE_VERSION, common.CACHE_KEY_APPIMG); e != nil || !exists {
		return
	}
	v, e := redis.Values(cache.LRange(redis_db.CACHE_VERSION, common.CACHE_KEY_APPIMG, 0, -1))
	if e != nil {
		return
	}
	vas := make([][]byte, 0, 300)
	if e = redis.ScanSlice(v, &vas); e != nil {
		return false, nil, e
	}
	arr = make([]AppImg, 0, len(v))
	for _, b := range vas {
		var g AppImg
		if e = json.Unmarshal(b, &g); e != nil {
			return
		}
		arr = append(arr, g)
	}
	return true, arr, nil
}

// 写入背景列表缓存
func writeAppImgCache(arr []AppImg) (e error) {
	key := common.CACHE_KEY_APPIMG
	v := make([]interface{}, 0, 50)
	for _, item := range arr {
		b, e := json.Marshal(item)
		if e != nil {
			return e
		}
		v = append(v, b)
	}
	if e = cache.Del(redis_db.CACHE_VERSION, key); e != nil {
		return
	}
	if _, e = cache.RPush(redis_db.CACHE_VERSION, key, v...); e != nil {
		return
	}
	e = cache.Expire(redis_db.CACHE_VERSION, 3600, key)
	return
}
