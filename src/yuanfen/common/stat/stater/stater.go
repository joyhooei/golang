package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"yf_pkg/mysql"
	"yf_pkg/utils"
)

//platform->province->city->channel->sub_channel->gender->version
type BigMap map[int]map[string]map[string]map[string]map[string]map[int]map[string]float64
type UidBigMap map[int]map[string]map[string]map[string]map[string]map[int]map[string]map[uint32]bool

type BigMapTm struct {
	bigMap BigMap
	tm     time.Time
}

const (
	INTERVAL_HOUR = "hour"
	INTERVAL_DAY  = "day"
)
const (
	STAT_TYPE_USER_COUNT = "user_count" //人数
	STAT_TYPE_SUM        = "sum"        //总和，根据data字段中的特定key计算
	STAT_TYPE_USER_AVG   = "user_avg"   //人均数量
	STAT_TYPE_COUNT      = "count"      //总和，根据记录条数计算
)

type StatItem struct {
	id       string
	action   int
	statType string
	from     time.Time
	to       time.Time
	extra    map[string]interface{}
}

var mdb *mysql.MysqlDB

//var calculated map[string]BigMap = map[string]BigMap{} //已经计算过的统计项
var from, to time.Time //计算的起始时间和结束时间
var interval string    //统计间隔
var statItems map[string]StatItem = map[string]StatItem{}

func StatRange(interval string, tm ...time.Time) {
	if len(tm) > 0 {
		from = tm[0]
		switch interval {
		case INTERVAL_HOUR:
			to = tm[0].Add(time.Hour)
		case INTERVAL_DAY:
			to = tm[0].Add(24 * time.Hour)
		}
	} else {
		switch interval {
		case INTERVAL_HOUR:
			from = time.Now().Add(-41 * time.Minute).Round(time.Hour)
			to = from.Add(time.Hour)
		case INTERVAL_DAY:
			t := time.Now().Add(-1 * time.Hour).Round(time.Hour)
			from = t.Add(-time.Duration(t.Hour()) * time.Hour)
			to = from.Add(24 * time.Hour)
		}
	}
}

func LoadConfig() error {
	sql := "select id,action,stat_type,extra from Config"
	rows, e := mdb.Query(sql)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var item StatItem
		var extra []byte = []byte{}
		if e := rows.Scan(&item.id, &item.action, &item.statType, &extra); e != nil {
			return e
		}
		if len(extra) > 0 {
			if e := json.Unmarshal(extra, &item.extra); e != nil {
				return e
			}
		}
		item.from = from
		item.to = to
		statItems[item.id] = item
	}
	return nil
}

func InsertToDB(values map[string]BigMapTm) error {
	sql := "insert into Stat(`interval`,`key`,ver,platform,province,city,channel,sub_channel,gender,value,tm)values(?,?,?,?,?,?,?,?,?,?,?)on duplicate key update value=?"
	stmt, e := mdb.PrepareExec(sql)
	if e != nil {
		return e
	}
	defer stmt.Close()
	for key, value := range values {
		for platform, x1 := range value.bigMap {
			for province, x2 := range x1 {
				for city, x3 := range x2 {
					for channel, x4 := range x3 {
						for sub_channel, x5 := range x4 {
							for gender, x6 := range x5 {
								for ver, v := range x6 {
									_, e := stmt.Exec(interval, key, ver, platform, province, city, channel, sub_channel, gender, v, value.tm, v)
									if e != nil {
										return e
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func Stat(statItem StatItem) error {
	fmt.Printf("stat %v\n", statItem.id)
	var values map[string]BigMapTm
	var e error
	switch statItem.statType {
	case STAT_TYPE_USER_AVG:
		values, e = StatUserAvg(statItem)
	case STAT_TYPE_SUM:
		values, e = StatSum(statItem)
	case STAT_TYPE_USER_COUNT:
		values, e = StatUserCount(statItem)
	case STAT_TYPE_COUNT:
		values, e = StatCount(statItem)
	}
	if e != nil {
		return e
	}
	//fmt.Println(values)
	if e = InsertToDB(values); e != nil {
		return e
	}
	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("invalid args : %s [connect string] [interval] [date(optional)]\n", os.Args[0])
		return
	}
	interval = os.Args[2]
	if len(os.Args) == 3 {
		StatRange(os.Args[2])
	} else {
		tm, e := utils.ToTime(os.Args[3])
		if e != nil {
			fmt.Println(e.Error())
			return
		}
		StatRange(os.Args[2], tm)
	}
	fmt.Printf("stat from %v to %v\n", from, to)
	var err error
	mdb, err = mysql.New(os.Args[1], []string{os.Args[1]})
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if e := LoadConfig(); e != nil {
		fmt.Println(e.Error())
		return
	}
	for _, item := range statItems {
		if e := Stat(item); e != nil {
			fmt.Println(e.Error())
			continue
		}
	}
}

func AddToUidMap(values UidBigMap, platform int, ver string, province string, city string, channel string, sub_channel string, gender int, uid uint32) {
	//fmt.Printf("AddToUidMap: platform=%v,province=%v,city=%v,channel=%v,sub_channel=%v,gender=%v,uid=%v\n", platform, province, city, channel, sub_channel, gender, uid)
	x1, ok := values[platform]
	if !ok {
		x1 = map[string]map[string]map[string]map[string]map[int]map[string]map[uint32]bool{}
		values[platform] = x1
	}
	x2, ok := x1[province]
	if !ok {
		x2 = map[string]map[string]map[string]map[int]map[string]map[uint32]bool{}
		x1[province] = x2
	}
	x3, ok := x2[city]
	if !ok {
		x3 = map[string]map[string]map[int]map[string]map[uint32]bool{}
		x2[city] = x3
	}
	x4, ok := x3[channel]
	if !ok {
		x4 = map[string]map[int]map[string]map[uint32]bool{}
		x3[channel] = x4
	}
	x5, ok := x4[sub_channel]
	if !ok {
		x5 = map[int]map[string]map[uint32]bool{}
		x4[sub_channel] = x5
	}
	x6, ok := x5[gender]
	if !ok {
		x6 = map[string]map[uint32]bool{}
		x5[gender] = x6
	}
	x7, ok := x6[ver]
	if !ok {
		x7 = map[uint32]bool{}
		x6[ver] = x7
	}
	x7[uid] = true
}

func AddToMap(values BigMap, platform int, ver string, province string, city string, channel string, sub_channel string, gender int, value float64) {
	//fmt.Printf("AddToMap: platform=%v,province=%v,city=%v,channel=%v,sub_channel=%v,gender=%v,value=%v\n", platform, province, city, channel, sub_channel, gender, value)
	x1, ok := values[platform]
	if !ok {
		x1 = map[string]map[string]map[string]map[string]map[int]map[string]float64{}
		values[platform] = x1
	}
	x2, ok := x1[province]
	if !ok {
		x2 = map[string]map[string]map[string]map[int]map[string]float64{}
		x1[province] = x2
	}
	x3, ok := x2[city]
	if !ok {
		x3 = map[string]map[string]map[int]map[string]float64{}
		x2[city] = x3
	}
	x4, ok := x3[channel]
	if !ok {
		x4 = map[string]map[int]map[string]float64{}
		x3[channel] = x4
	}
	x5, ok := x4[sub_channel]
	if !ok {
		x5 = map[int]map[string]float64{}
		x4[sub_channel] = x5
	}
	x6, ok := x5[gender]
	if !ok {
		x6 = map[string]float64{}
		x5[gender] = x6
	}
	x6[ver] += value
}
