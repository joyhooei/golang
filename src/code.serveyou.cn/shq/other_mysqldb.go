package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/location"
	"code.serveyou.cn/model"
	"code.serveyou.cn/pkg/format"
	_ "github.com/go-sql-driver/mysql"
)

func GetCommunityLocations() (elements []location.Element, err error) {
	elements = make([]location.Element, 0, 100)
	for _, c := range Communities {
		var e location.Element
		e.Id = c.Id
		e.Lat = c.Latitude()
		e.Lng = c.Longitude()
		elements = append(elements, e)
	}
	return
}

func GetRecommend(db *sql.DB) (data format.JSON, err error) {
	rows, err := db.Query(common.SQL_GetRecommend)
	if err != nil {
		return
	}
	defer rows.Close()
	items := make([]map[string]string, 0, 3)
	for rows.Next() {
		var pic, op, value string
		err = rows.Scan(&pic, &op, &value)
		if err != nil {
			return
		}
		item := make(map[string]string)
		item["pic"] = pic
		item["op"] = op
		item["value"] = value
		items = append(items, item)
	}
	return format.GenerateJSON(items), err
}

func readUserInfo(db *sql.DB, sql string) (err error) {
	rows, err := db.Query(sql)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var u model.User
		err = u.InitByDB(rows)
		if err != nil {
			return
		}
		Users[u.Id()] = u
		Phones[u.Cellphone()] = u.Id()
	}
	err = rows.Err()
	return
}
func InitHKPs(db *sql.DB) (err error) {
	//获取用户和小区的对应关系
	rows, err := db.Query(common.SQL_InitHKPs_C)
	if err != nil {
		return
	}
	defer rows.Close()
	var ucmap map[common.UIDType]uint = make(map[common.UIDType]uint)
	for rows.Next() {
		var uid common.UIDType
		var comm uint
		err = rows.Scan(&uid, &comm)
		if err != nil {
			return
		}
		ucmap[uid] = comm
	}

	//初始化用户信息
	i := 0
	var buf bytes.Buffer
	for uid, _ := range ucmap {
		if i < 100 {
			if i == 0 {
				buf.WriteString(fmt.Sprintf("%v", uid))
			} else {
				buf.WriteString(fmt.Sprintf(",%v", uid))
			}
			i++
		} else {
			i = 0
			sql := fmt.Sprintf(common.SQL_InitHKPs_U, buf.String)
			if err = readUserInfo(db, sql); err != nil {
				return
			}
			buf.Truncate(0)
		}
	}
	if i > 0 {
		i = 0
		sql := fmt.Sprintf(common.SQL_InitHKPs_U, buf.String())
		if err = readUserInfo(db, sql); err != nil {
			return
		}
		buf.Truncate(0)
	}

	//初始化Job信息
	rows, err = db.Query(common.SQL_InitHKPs_J)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var job model.HKPJob
		var bt string
		err = rows.Scan(&job.Uid, &bt, &job.JobId, &job.Price, &job.Desc, &job.Version)
		if err != nil {
			return
		}
		job.BeginTime, err = time.Parse(format.TIME_LAYOUT_1, bt)
		if err != nil {
			return
		}
		c, found := ucmap[job.Uid]
		if !found {
			err = errors.New(fmt.Sprintf("cannot find community id of uid=%v", job.Uid))
			return
		}
		jobs, found := CommunityHKPs[c]
		if !found {
			jobs = make(model.HKPJobUserMap)
			CommunityHKPs[c] = jobs
		}
		users, found := jobs[job.JobId]
		if !found {
			users = make(model.UserHKPDetailMap)
			jobs[job.JobId] = users
		}
		d := users[job.Uid]
		d.Job = job
		users[job.Uid] = d
	}

	//初始化ranking信息
	rows, err = db.Query(common.SQL_InitHKPs_R)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var r model.HKPRank
		err = rows.Scan(&r.Uid, &r.JobId, &r.Times, &r.TotalSpeed, &r.TotalQuality, &r.TotalAttitude, &r.RejectTimes, &r.FailTimes)
		if err != nil {
			return
		}
		c, found := ucmap[r.Uid]
		if !found {
			err = errors.New(fmt.Sprintf("cannot find community id of uid=%v", r.Uid))
			return
		}
		jobs, found := CommunityHKPs[c]
		if !found {
			jobs = make(model.HKPJobUserMap)
			CommunityHKPs[c] = jobs
		}
		users, found := jobs[r.JobId]
		if !found {
			users = make(model.UserHKPDetailMap)
			jobs[r.JobId] = users
		}
		d := users[r.Uid]
		d.Rank = r
		users[r.Uid] = d
		js, found := HKPs[r.Uid]
		if !found {
			js = make(map[uint]*model.HKPDetail)
			HKPs[r.Uid] = js
		}
		js[r.JobId] = &d
	}

	return
}

//寻找没时间的HKPs
//startTime：服务开始时间
func GetHKPsNoTime(db *sql.DB, elems []location.Element, role uint, startTime time.Time) (hkps map[common.UIDType]bool, err error) {
	hkps = make(map[common.UIDType]bool)
	//寻找时间冲突的的阿姨
	timeMin := startTime.Add(-4 * time.Hour)
	timeMax := startTime.Add(4 * time.Hour)
	var buf bytes.Buffer
	for i, elem := range elems {
		if i == 0 {
			buf.WriteString(fmt.Sprintf("%v", elem.Id))
		} else {
			buf.WriteString(fmt.Sprintf(",%v", elem.Id))
		}

	}
	sql := fmt.Sprintf(common.SQL_GetHKPsNoTime_FindOrders, buf.String())

	//寻找符合条件的订单
	oids := make(map[uint64]uint8, 2)
	fmt.Println(sql)
	rows, err := db.Query(sql, role, common.HKP_PHASE_SERVICE_COMPLETE, timeMin, timeMax)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id uint64
		var phase uint8
		err = rows.Scan(&id, &phase)
		if err != nil {
			return
		}
		oids[id] = phase
	}
	err = rows.Err()
	if err != nil {
		return
	}
	//根据订单id寻找没时间的HKP
	if len(oids) == 0 {
		return
	}
	buf.Reset()
	isFirst := true
	for oid, _ := range oids {
		if isFirst {
			buf.WriteString(fmt.Sprintf("%v", oid))
			isFirst = false
		} else {
			buf.WriteString(fmt.Sprintf(",%v", oid))
		}

	}
	sql = fmt.Sprintf(common.SQL_GetHKPsNoTime_FindHKPs, buf.String())
	fmt.Println(sql)
	rows, err = db.Query(sql)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id common.UIDType
		var confirm uint8
		var oid uint64
		err = rows.Scan(&id, &confirm, &oid)
		if err != nil {
			return
		}
		//只选取订单未完成的所有HKPs和订单已完成且已确认的HKPs
		if oids[oid] != common.HKP_PHASE_ORDER_SUCCESS || confirm == 1 {
			hkps[id] = true
		}
	}
	err = rows.Err()
	if err != nil {
		return
	}

	return
}
