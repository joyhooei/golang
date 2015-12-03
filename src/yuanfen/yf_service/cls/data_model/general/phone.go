package general

import (
	"errors"
	// "fmt"
	"yf_pkg/lbs/baidu"
	"yf_pkg/redis"
	"yuanfen/redis_db"
)

//根据经纬度获取省市名称
func GetCityByPhone(phone string) (city string, province string, supplier string, e error) {
	if len(phone) < 7 {
		return "", "", "", errors.New("Invalid Phone")
	}
	key := phone[0:7]
	exist, e := rdb.Exists(redis_db.REDIS_PHONE_CITY, key)
	if e != nil {
		return "", "", "", e
	}
	if !exist {
		city, province = "", ""
	} else {
		rcon := rdb.GetReadConnection(redis_db.REDIS_PHONE_CITY)
		defer rcon.Close()
		var err error
		reply, e := redis.Values(rcon.Do("HMGET", key, "city", "province", "supplier"))
		switch e {
		case nil:
			if _, err = redis.Scan(reply, &city, &province, &supplier); err != nil {
				return "", "", "", err
			}
		default:
			return "", "", "", e
		}
	}
	if city == "" && province == "" {
		city, province, supplier, e = baidu.GetCityByPhone(phone)
		if e != nil {
			return "", "", "", e
		}
		wcon := rdb.GetWriteConnection(redis_db.REDIS_PHONE_CITY)
		defer wcon.Close()
		wcon.Do("HMSET", key, "city", city, "province", province, "supplier", supplier)
	}
	return
}
