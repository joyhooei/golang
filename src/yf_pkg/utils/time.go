/****
	高性能时间，精确到0.1秒
****/
package utils

import (
	"fmt"
	"time"
)

var Now time.Time
var Local *time.Location

func init() {
	Now = time.Now().Round(time.Second)
	Local, _ = time.LoadLocation("Local")
	go refresh()
}

func refresh() {
	for {
		Now = time.Now().Round(time.Second)
		time.Sleep(100 * time.Millisecond)
	}
}

// 获取时间界限，如：today  返回stm: 2015-05-01 00:00:00  etm: 2015-05-02: 00:00:00
func TmLime(tmflag string) (stm, etm string) {
	stm = "1970-01-01 00:00:00"
	etm = "2070-01-01 00:00:00"
	if "today" == tmflag {
		stm = Now.Format("2006-01-02") + " 00:00:00"
		etm_tm := Now.AddDate(0, 0, 1)
		etm = etm_tm.Format("2006-01-02") + " 00:00:00"
	} else if "yesterday" == tmflag {
		stm_tm := Now.AddDate(0, 0, -1)
		stm = stm_tm.Format("2006-01-02") + " 00:00:00"
		etm = Now.Format("2006-01-02") + " 00:00:00"
	}
	return
}

func FormatPrevLogin(tm time.Time) (st string) {
	if Now.Before(tm) {
		return "当前在线"
	}
	du := Now.Sub(tm)
	switch {
	case du.Minutes() < 60:
		return ToString(int(du.Minutes())) + "分钟前"
	case du.Hours() < 24:
		return ToString(int(du.Hours())) + "小时前"
	case du.Hours() < 24*7:
		return ToString(int(du.Hours()/24)) + "天前"
	default:
		return "七天前"
	}
	return
}

//打印超过exceed时长的时间
//Prams:
// 	key: 用于识别的关键字
// 	start: 起始时间
// 	exceed: 持续时间超过多久才打印
func PrintDuration(key string, start time.Time, exceed time.Duration) {
	dur := time.Now().Sub(start)
	if dur >= exceed {
		fmt.Println("Duration", key, ":", dur.Seconds())
	}
}

//从现在到[days]天后的[H:M:S]时刻的时长
func DurationTo(days int, H, M, S int) time.Duration {
	fmt.Println(Now.Minute(), Now.Second())
	seconds := (days*24+H-Now.Hour())*3600 + (M-Now.Minute())*60 + S - Now.Second()
	return time.Duration(seconds) * time.Second
}
