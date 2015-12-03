package general

import (
	"strings"
	"yf_pkg/lbs/baidu"
	"yuanfen/redis_db"
)

func QueryIpInfo(ip string) (province string, city string, e error) {
	ss := strings.Split(ip, ".")
	ss = ss[:len(ss)-1]
	part := strings.Join(ss, ".")
	sql := "select region,city from ip_address_lib where ip=?"
	if err := mdb.QueryRow(sql, part).Scan(&province, &city); err == nil {
		return province, city, err
	}
	bProvince, bCity, e := baidu.GetCityByIP(ip)
	if e != nil {
		return "", "", e
	}
	province, city = BaiduToOurProvinceCity(bProvince, bCity)
	sql2 := "replace into ip_address_lib(ip,region,city)values(?,?,?)"
	_, e = mdb.Exec(sql2, part, province, city)
	return
}

func ClearIP(ip string) (e error) {
	return cache.Del(redis_db.CACHE_REGIP, ip)
}
