package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"yf_pkg/utils"
)

func statUserAvg(statItem StatItem, baseAction int, bFrom, bTo time.Time) (values BigMapTm, e error) {
	//fmt.Println("statUserAvg ", key, "action=", statItem.action, "baseAction=", baseAction)
	sums, e := statSum(statItem, baseAction, bFrom, bTo)
	if e != nil {
		return values, e
	}
	values = BigMapTm{tm: bFrom, bigMap: BigMap{}}
	rangeUidBigMap(baseAction, bFrom, bTo, func(platform int, ver string, province string, city string, channel string, sub_channel string, gender int, uids map[uint32]bool) {
		if len(uids) == 0 {
			AddToMap(values.bigMap, platform, ver, province, city, channel, sub_channel, gender, 0)
		} else {
			sum, _ := getBigMapItem(sums.bigMap, platform, ver, province, city, channel, sub_channel, gender)
			//fmt.Println("sum=", sum, "len(uids)=", len(uids))
			AddToMap(values.bigMap, platform, ver, province, city, channel, sub_channel, gender, sum/float64(len(uids)))
		}
	})
	return
}

func statLc(statItem StatItem, statFunc func(StatItem, int, time.Time, time.Time) (BigMapTm, error), values map[string]BigMapTm) error {
	if interval == "day" {
		if l, ok := statItem.extra["lc"]; ok {
			switch lc := l.(type) {
			case map[string]interface{}:
				baseAction, e := utils.ToInt(lc["base_action"])
				if e != nil {
					return e
				}
				res, e := statFunc(statItem, baseAction, statItem.from, statItem.to)
				values[fmt.Sprintf("%v_lc_0", statItem.id)] = res
				for _, v := range lc["days"].([]interface{}) {
					days, e := utils.ToInt(v)
					//fmt.Println("stat days ", days)
					if e != nil {
						return e
					}
					if days != 0 {
						from := statItem.from.AddDate(0, 0, -days)
						to := statItem.to.AddDate(0, 0, -days)
						res, e := statFunc(statItem, baseAction, from, to)
						if e != nil {
							return e
						}
						values[fmt.Sprintf("%v_lc_%v", statItem.id, days)] = res
					}
				}
			default:
				return errors.New("[lc] node must be a map[string]interface{}")
			}
		}
	}
	return nil
}
func StatUserAvg(statItem StatItem) (values map[string]BigMapTm, e error) {
	values = map[string]BigMapTm{}
	baseAction, e := utils.ToInt(statItem.extra["base_action"])
	if e != nil {
		return nil, e
	}
	v, e := statUserAvg(statItem, baseAction, statItem.from, statItem.to)
	if e != nil {
		return nil, e
	}
	values[statItem.id] = v
	e = statLc(statItem, statUserAvg, values)
	if e != nil {
		return nil, e
	}
	return values, nil
}

func statSum(statItem StatItem, baseAction int, bFrom, bTo time.Time) (values BigMapTm, e error) {
	key := utils.ToString(statItem.extra["key"])
	values = BigMapTm{tm: bFrom, bigMap: BigMap{}}
	sql := "select uid,data,platform,ver,province,city,channel,sub_channel,gender from Actions where tm >= ? and tm < ? and type= ?"
	rows, e := mdb.Query(sql, statItem.from, statItem.to, statItem.action)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var platform, gender int
		var uid uint32
		var ver, channel, sub_channel string
		var province, city string
		var extra []byte
		if e := rows.Scan(&uid, &extra, &platform, &ver, &province, &city, &channel, &sub_channel, &gender); e != nil {
			return values, e
		}
		exist, e := existInUidBigMap(baseAction, bFrom, bTo, uid, platform, ver, province, city, channel, sub_channel, gender)
		if e != nil {
			return values, e
		}
		if !exist {
			//fmt.Println(uid, "not in base")
			continue
		}
		data := map[string]interface{}{}
		if e := json.Unmarshal(extra, &data); e != nil {
			return values, e
		}
		num, ok := data[key]
		if !ok {
			return values, errors.New(fmt.Sprintf("stat %v error : cannot find [%v] in data field", statItem.id, key))
		}
		value, e := utils.ToFloat64(num)
		if e != nil {
			return values, e
		}
		AddToMap(values.bigMap, platform, ver, province, city, channel, sub_channel, gender, value)
	}
	return values, nil
}

func StatSum(statItem StatItem) (values map[string]BigMapTm, e error) {
	values = map[string]BigMapTm{}
	v, e := statSum(statItem, 0, statItem.from, statItem.to)
	if e != nil {
		return nil, e
	}
	values[statItem.id] = v
	e = statLc(statItem, statSum, values)
	if e != nil {
		return nil, e
	}
	return values, nil
}

func statCount(statItem StatItem, baseAction int, bFrom, bTo time.Time) (values BigMapTm, e error) {
	values = BigMapTm{tm: bFrom, bigMap: BigMap{}}
	sql := "select uid,platform,ver,province,city,channel,sub_channel,gender from Actions where tm >= ? and tm < ? and type= ?"
	rows, e := mdb.Query(sql, from, to, statItem.action)
	if e != nil {
		return values, e
	}
	defer rows.Close()
	for rows.Next() {
		var platform, gender int
		var uid uint32
		var ver, channel, sub_channel string
		var province, city string
		if e := rows.Scan(&uid, &platform, &ver, &province, &city, &channel, &sub_channel, &gender); e != nil {
			return values, e
		}
		exist, e := existInUidBigMap(baseAction, bFrom, bTo, uid, platform, ver, province, city, channel, sub_channel, gender)
		if e != nil {
			return values, e
		}
		if !exist {
			continue
		}
		AddToMap(values.bigMap, platform, ver, province, city, channel, sub_channel, gender, 1)
	}
	return values, nil
}
func StatCount(statItem StatItem) (values map[string]BigMapTm, e error) {
	values = map[string]BigMapTm{}
	v, e := statCount(statItem, 0, statItem.from, statItem.to)
	if e != nil {
		return nil, e
	}
	values[statItem.id] = v
	e = statLc(statItem, statCount, values)
	if e != nil {
		return nil, e
	}
	return values, nil
}

func StatUserCount(statItem StatItem) (values map[string]BigMapTm, e error) {
	values = map[string]BigMapTm{}
	v, e := statUserCount(statItem, 0, statItem.from, statItem.to)
	if e != nil {
		return nil, e
	}
	values[statItem.id] = v
	e = statLc(statItem, statUserCount, values)
	if e != nil {
		return nil, e
	}
	return values, nil
}
func statUserCount(statItem StatItem, baseAction int, bFrom, bTo time.Time) (values BigMapTm, e error) {
	values = BigMapTm{tm: bFrom, bigMap: BigMap{}}
	sql := "select distinct uid,platform,ver,province,city,channel,sub_channel,gender from Actions where tm >= ? and tm < ? and type= ?"
	rows, e := mdb.Query(sql, from, to, statItem.action)
	if e != nil {
		return values, e
	}
	defer rows.Close()
	for rows.Next() {
		var platform, gender int
		var uid uint32
		var ver, channel, sub_channel string
		var province, city string
		if e := rows.Scan(&uid, &platform, &ver, &province, &city, &channel, &sub_channel, &gender); e != nil {
			return values, errors.New(fmt.Sprintf("StatUserCount error :%v", e.Error()))
		}
		exist, e := existInUidBigMap(baseAction, bFrom, bTo, uid, platform, ver, province, city, channel, sub_channel, gender)
		if e != nil {
			return values, e
		}
		if !exist {
			continue
		}
		AddToMap(values.bigMap, platform, ver, province, city, channel, sub_channel, gender, 1)
	}
	return values, nil
}

func getBigMapItem(values BigMap, platform int, ver, province, city, channel, sub_channel string, gender int) (value float64, exist bool) {
	if x1, ok := values[platform]; ok {
		if x2, ok := x1[province]; ok {
			if x3, ok := x2[city]; ok {
				if x4, ok := x3[channel]; ok {
					if x5, ok := x4[sub_channel]; ok {
						if x6, ok := x5[gender]; ok {
							if x7, ok := x6[ver]; ok {
								return x7, true
							}
						}
					}
				}
			}
		}
	}
	return 0, false
}

var actionCaches map[int]map[time.Time]map[time.Time]UidBigMap = map[int]map[time.Time]map[time.Time]UidBigMap{}

func getActionUidMapFromDB(action int, from, to time.Time) (UidBigMap, error) {
	values := UidBigMap{}
	sql := "select distinct uid,platform,ver,province,city,channel,sub_channel,gender from Actions where tm >= ? and tm < ? and type= ?"
	rows, e := mdb.Query(sql, from, to, action)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var platform, gender int
		var uid uint32
		var ver, channel, sub_channel string
		var province, city string
		if e := rows.Scan(&uid, &platform, &ver, &province, &city, &channel, &sub_channel, &gender); e != nil {
			return nil, e
		}
		AddToUidMap(values, platform, ver, province, city, channel, sub_channel, gender, uid)
	}
	return values, nil
}

func getActionUidMap(action int, from, to time.Time) (UidBigMap, error) {
	x1, ok := actionCaches[action]
	if !ok {
		x1 = map[time.Time]map[time.Time]UidBigMap{}
		actionCaches[action] = x1
	}
	x2, ok := x1[from]
	if !ok {
		x2 = map[time.Time]UidBigMap{}
		x1[from] = x2
	}
	x3, ok := x2[to]
	if !ok {
		v, e := getActionUidMapFromDB(action, from, to)
		if e != nil {
			return nil, e
		}
		x2[to] = v
		x3 = v
	}
	//fmt.Println("UidMap", action, x3)
	return x3, nil
}

func rangeUidBigMap(action int, from time.Time, to time.Time, calcFunc func(int, string, string, string, string, string, int, map[uint32]bool)) error {
	values, e := getActionUidMap(action, from, to)
	if e != nil {
		return e
	}
	for platform, x1 := range values {
		for province, x2 := range x1 {
			for city, x3 := range x2 {
				for channel, x4 := range x3 {
					for sub_channel, x5 := range x4 {
						for gender, x6 := range x5 {
							for ver, uids := range x6 {
								calcFunc(platform, ver, province, city, channel, sub_channel, gender, uids)
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func rangeBigMap(values BigMap, calcFunc func(int, string, string, string, string, string, int, float64)) {
	for platform, x1 := range values {
		for province, x2 := range x1 {
			for city, x3 := range x2 {
				for channel, x4 := range x3 {
					for sub_channel, x5 := range x4 {
						for gender, x6 := range x5 {
							for ver, value := range x6 {
								calcFunc(platform, ver, province, city, channel, sub_channel, gender, value)
							}
						}
					}
				}
			}
		}
	}
}

func existInUidBigMap(action int, from, to time.Time, uid uint32, platform int, ver, province, city, channel, sub_channel string, gender int) (bool, error) {
	if action == 0 {
		return true, nil
	}
	values, e := getActionUidMap(action, from, to)
	if e != nil {
		return false, e
	}
	if x1, ok := values[platform]; ok {
		if x2, ok := x1[province]; ok {
			if x3, ok := x2[city]; ok {
				if x4, ok := x3[channel]; ok {
					if x5, ok := x4[sub_channel]; ok {
						if x6, ok := x5[gender]; ok {
							if x7, ok := x6[ver]; ok {
								if _, ok := x7[uid]; ok {
									return true, nil
								}
							}
						}
					}
				}
			}
		}
	}
	return false, nil
}
