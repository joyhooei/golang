package scanuser

import (
	"fmt"
	"time"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/utils"
	// "yuanfen/yf_service/cls/common"
)

var mdb *mysql.MysqlDB
var statdb *mysql.MysqlDB
var mainlog *log.MLogger
var prevtm time.Time

// func Init(statDB *mysql.MysqlDB, mainDB *mysql.MysqlDB, logger *log.MLogger) {

// 	mdb = mainDB
// 	statdb = statdb
// 	mainlog = logger
// 	scan()
// }

func Scan(statDB *mysql.MysqlDB, mainDB *mysql.MysqlDB, logger *log.MLogger) {
	mdb = mainDB
	statdb = statdb
	mainlog = logger
	// fmt.Println("scan")
	for {

		y, m, d := prevtm.Date()
		y1, m1, d1 := utils.Now.Date()
		// fmt.Println(fmt.Sprintf("prevtm date year %v,mouth %v,day %v ", y, m, d))
		// fmt.Println(fmt.Sprintf("Now year %v,mouth %v,day %v ", y1, m1, d1))
		if (y != y1) || (m != m1) || (d != d1) {
			prevtm = utils.Now
			mainlog.AppendInfo("scan star begin")
			scantoday(2)
			scantoday(3)
		}
		time.Sleep(time.Second)
	}
}

//获取最近7日登陆次数
func getlogincount(uid uint32) (ct int, e error) {
	tm2 := utils.Now.AddDate(0, 0, -7)
	e = mdb.QueryRow("select count(*) from user_online_award_log where uid=? and tm>?", uid, tm2).Scan(&ct)
	if e != nil {
		return 0, e
	}
	return
}

//检查当天天某个等级的用户是否该被降级
func scantoday(lv int) {
	fmt.Println("scantoday " + utils.ToString(lv))
	tm2 := utils.Now.AddDate(0, 0, -7)
	rows, err := mdb.Query("select uid,level from user_star_level where tm>=? and level=?", tm2, lv)
	if err != nil {
		return
	}
	defer rows.Close()
	uids := make([]uint32, 0, 0)
	for rows.Next() {
		var uid uint32
		var level int

		if err := rows.Scan(&uid, &level); err != nil {
			return
		}
		var rlevel int
		count, e := getlogincount(uid)
		if e != nil {
			continue
		}
		if count >= 5 {
			rlevel = 3
		} else {
			if count >= 4 {
				rlevel = 2
			} else {
				rlevel = 1
			}
		}
		if rlevel < level {
			uids = append(uids, uid)
		}
	}
	if len(uids) > 0 {
		_, e := mdb.Exec("update user_star_level set level=?,changes=-1 where uid "+mysql.In(uids), lv-1)
		if e != nil {
			mainlog.AppendInfo(" update user_star_level SQL error " + e.Error())
		} else {
			mainlog.AppendInfo(fmt.Sprintf("update user_star_level level %v,%v", lv, uids))
		}
	}
}
