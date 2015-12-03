package certify

import (
	"encoding/json"
	"yf_pkg/redis"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
)

// 取通用配置
func readHonestyConifCache() (exists bool, values []HonestyPri, e error) {
	key := common.CACHE_KEY_HONESTY
	if ex, e := cache.Exists(redis_db.CACHE_USER_ABOUT, key); !ex || e != nil {
		return false, nil, e
	}
	con := cache.GetWriteConnection(redis_db.CACHE_USER_ABOUT)
	defer con.Close()
	vas := make([][]byte, 0, 300)
	v, e := redis.Values(con.Do("LRANGE", key, 0, -1))
	if e != nil {
		return false, nil, e
	}
	if e = redis.ScanSlice(v, &vas); e != nil {
		return false, nil, e
	}
	values = make([]HonestyPri, 0, 36)
	for _, b := range vas {
		var i HonestyPri
		if e = json.Unmarshal(b, &i); e != nil {
			return false, nil, e
		}
		values = append(values, i)
	}
	//fmt.Printf("get cahce data  %+v, rs : %+v \n", key, values)
	return true, values, nil
}

// 写入通用配置
func writeHonestyConfigCache(values []HonestyPri) error {
	key := common.CACHE_KEY_HONESTY
	con := cache.GetWriteConnection(redis_db.CACHE_USER_ABOUT)
	defer con.Close()
	v := make([]interface{}, 0, 0)
	v = append(v, key)
	for _, item := range values {
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
	_, e := con.Do("EXPIRE", key, 5*60)
	//	fmt.Println("set cahce data  key ", key)
	return e
}
