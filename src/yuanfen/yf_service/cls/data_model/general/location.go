package general

import (
	"fmt"
	"yf_pkg/lbs/baidu"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
)

func IsValidLocation(lat, lng float64) bool {
	return lat <= 90 && lat >= -90 && lng <= 180 && lng >= -180
}

//获取用户位置，如果找不到坐标，则返回common.LAT_NO_VALUE和common.LNG_NO_VALUE

func UserLocation(uid uint32) (lat, lng float64, e error) {
	type Location struct {
		Lat float64 `redis:"lat"`
		Lng float64 `redis:"lng"`
	}
	exist, e := rdb.Exists(redis_db.REDIS_LOCATION, uid)
	if e != nil {
		return 0, 0, e
	}
	if !exist {
		return common.LAT_NO_VALUE, common.LNG_NO_VALUE, nil
	}
	lcon := rdb.GetReadConnection(redis_db.REDIS_LOCATION)
	defer lcon.Close()
	var loc Location
	if reply, e := redis.Values(lcon.Do("HGETALL", uid)); e != nil {
		return 0, 0, e
	} else {
		if e := redis.ScanStruct(reply, &loc); e != nil {
			return 0, 0, e
		}
		return loc.Lat, loc.Lng, nil
	}
}

//批量获取用户位置
func MUserLocation(uids []uint32) (m map[uint32]utils.Coordinate, e error) {
	type Location struct {
		Lat float64 `redis:"lat"`
		Lng float64 `redis:"lng"`
	}
	ids := make([]interface{}, 0, len(uids))
	for _, uid := range uids {
		ids = append(ids, uid)
	}
	m = make(map[uint32]utils.Coordinate)
	rm, e := rdb.MHGetAll(redis_db.REDIS_LOCATION, ids...)
	if e != nil {
		return
	}
	for key, r := range rm {
		var loc Location
		if e = redis.ScanStruct(r, &loc); e != nil {
			return
		}
		id, e := utils.ToUint32(key)
		if e != nil {
			return m, e
		}
		c := utils.Coordinate{Lat: loc.Lat, Lng: loc.Lng}
		m[id] = c
	}
	return
}

func CityByUid(uid uint32) (city string, province string, e error) {
	lat, lng, e := UserLocation(uid)
	if e != nil {
		return
	}
	return City(lat, lng)
}

//根据经纬度获取省市名称
func City(lat, lng float64) (city string, province string, e error) {
	key := fmt.Sprintf("%.2f-%.2f", lat, lng)
	exist, e := rdb.Exists(redis_db.REDIS_GEO_CITY, key)
	if e != nil {
		return "", "", e
	}
	if !exist {
		city, province = "", ""
	} else {
		rcon := rdb.GetReadConnection(redis_db.REDIS_GEO_CITY)
		defer rcon.Close()
		var err error
		reply, e := redis.Values(rcon.Do("HMGET", key, "city", "province"))
		switch e {
		case nil:
			if _, err = redis.Scan(reply, &city, &province); err != nil {
				return "", "", err
			}
		default:
			return "", "", e
		}
	}
	if city == "" && province == "" {
		bCity, bProvince, e := baidu.GetCityByGPS(utils.Coordinate{lat, lng})
		if e != nil {
			return "", "", e
		}
		province, city = BaiduToOurProvinceCity(bProvince, bCity)
		wcon := rdb.GetWriteConnection(redis_db.REDIS_GEO_CITY)
		defer wcon.Close()
		wcon.Do("HMSET", key, "city", city, "province", province)
	}
	return
}

//是否是直辖市
func IsZXS(province string) bool {
	bProvince := OurToBaiduProvince(province)
	return baidu.IsZXS(bProvince)
}

//根据省市获取经纬度
func GetGPSByCity(city, province string) (lat, lng float64, e error) {
	key := fmt.Sprintf("%v-%v", city, province)
	exist, e := rdb.Exists(redis_db.REDIS_GEO_CITY, key)
	if e != nil {
		return 0, 0, e
	}
	if !exist {
		lat, lng = common.LAT_NO_VALUE, common.LNG_NO_VALUE
	} else {
		rcon := rdb.GetReadConnection(redis_db.REDIS_GEO_CITY)
		defer rcon.Close()
		var err error
		reply, e := redis.Values(rcon.Do("HMGET", key, "lat", "lng"))
		switch e {
		case nil:
			if _, err = redis.Scan(reply, &lat, &lng); err != nil {
				return 0, 0, err
			}
		default:
			return 0, 0, e
		}
	}
	if lat == common.LAT_NO_VALUE {
		pos, err := baidu.GetGPSByCity(city, province)
		if err != nil {
			return 0, 0, err
		}
		wcon := rdb.GetWriteConnection(redis_db.REDIS_GEO_CITY)
		defer wcon.Close()
		wcon.Do("HMSET", key, "lat", pos.Lat, "lng", pos.Lng)
		lat, lng = pos.Lat, pos.Lng
	}
	return lat, lng, nil
}
