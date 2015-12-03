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

type TransactionDBAdapter struct {
	db *sql.DB
}

func NewTransactionDBAdapter(db *sql.DB) (dba *TransactionDBAdapter) {
	dba = new(TransactionDBAdapter)
	dba.db = db
	return
}

func (d *TransactionDBAdapter) Begin() (tx *sql.Tx, err error) {
	tx, err = d.db.Begin()
	return
}

func (d *TransactionDBAdapter) GetCSR() (csr []uint, err error) {
	rows, err := d.db.Query(common.SQL_GetCSR)
	if err != nil {
		return
	}
	defer rows.Close()
	csr = make([]uint, 0, 5)
	for rows.Next() {
		var id uint
		rows.Scan(&id)
		csr = append(csr, id)
	}
	fmt.Printf("CSR : %v\n", csr)
	return
}

func (d *TransactionDBAdapter) GetUserAddress(uid common.UIDType, adId uint64) (ad *model.Address, err error) {
	ad = model.NewAddress(uid)
	var addr string
	err = d.db.QueryRow(common.SQL_GetUserAddress, uid, adId).Scan(&ad.Community, &addr)
	if err != nil {
		return
	}
	if err = ad.SetAddr(addr); err != nil {
		return
	}
	ad.AddrId = adId
	return
}

func (d *TransactionDBAdapter) GetHKPsNoTime(elems []location.Element, order *model.HKPOrder) (hkps map[common.UIDType]bool, err error) {
	hkps = make(map[common.UIDType]bool)
	//把用户选择的阿姨也要排除在外
	for _, p := range order.Providers {
		fmt.Printf("exclude user %v\n", p.UserId)
		hkps[p.UserId] = true
	}
	//寻找时间冲突的的阿姨
	timeMin := order.ServiceStart().Add(-4 * time.Hour)
	timeMax := order.ServiceStart().Add(4 * time.Hour)
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
	rows, err := d.db.Query(sql, order.Role(), common.HKP_PHASE_SERVICE_COMPLETE, timeMin, timeMax)
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
	rows, err = d.db.Query(sql)
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

func (d *TransactionDBAdapter) AddHKPOrder(o *model.HKPOrder) (err error) {
	//要更新order中的id
	tx, err := d.Begin()
	if err != nil {
		return
	}
	fmt.Printf("o.StartTime() : %v\n", o.StartTime())
	res, err := tx.Exec(common.SQL_AddOrder, o.Role(), o.Job(), o.Customer(), o.Address(), o.Community(), o.Persons(), o.StartTime().Format(format.TIME_LAYOUT_1), o.Phase(), o.PhaseTime().Format(format.TIME_LAYOUT_1), o.PhaseReason(), o.CSR(), o.ServiceStart().Format(format.TIME_LAYOUT_1), o.Duration(), o.Description())
	if err != nil {
		tx.Rollback()
		return
	}
	tmpid, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return
	}
	for _, p := range o.Providers {
		fmt.Printf("add provider : uid=%v, oid=%v, isRec=%v\n", p.UserId, tmpid, p.Recommend)
		_, err = tx.Exec(common.SQL_AddHKPOrderProvider, tmpid, p.UserId, p.Job.Version, p.Recommend, p.AvailableTime, p.Confirm, p.Priority, 0, 0, 0)
		if err != nil {
			tx.Rollback()
			return
		}
	}
	o.SetOrderId(uint64(tmpid))
	err = tx.Commit()
	return
}

func (d *TransactionDBAdapter) GetOrder(oid uint64) (o *model.Order, err error) {
	o = model.NewOrder()
	o.Id = oid
	var startTime, phaseTime, serviceStart string
	err = d.db.QueryRow(common.SQL_GetOrder, oid).Scan(&o.Role, &o.Job, &o.Customer, &o.Address, &o.Community, &o.Persons, &startTime, &o.Phase, &phaseTime, &o.PhaseReason, &o.CSR, &serviceStart, &o.Duration, &o.Description)
	if err != nil {
		return
	}
	o.StartTime, err = time.Parse(format.TIME_LAYOUT_1, startTime)
	if err != nil {
		return
	}
	o.PhaseTime, err = time.Parse(format.TIME_LAYOUT_1, phaseTime)
	if err != nil {
		return
	}
	fmt.Printf("serviceStart: %s\n", serviceStart)
	o.ServiceStart, err = time.Parse(format.TIME_LAYOUT_1, serviceStart)
	if err != nil {
		return
	}
	return
}
func (d *TransactionDBAdapter) GetHKPJob(uid common.UIDType, jobId uint, jobVersion uint) (j *model.HKPJob, err error) {
	j = model.NewHKPJob()
	j.Uid = uid
	j.JobId = jobId
	j.Version = jobVersion
	var t string
	err = d.db.QueryRow(common.SQL_GetHKPJob, uid, jobId, jobVersion).Scan(&t, &j.Price, &j.Desc)
	if err != nil {
		return
	}
	j.BeginTime, err = time.Parse(format.TIME_LAYOUT_1, t)
	if err != nil {
		return
	}
	return
}
func (d *TransactionDBAdapter) ListOrders(uid common.UIDType, phases []uint8, orderby string, desc bool, pn uint, rn uint) (orders []model.Order, err error) {
	s := "desc"
	if desc == false {
		s = "asc"
	}
	var where string
	if len(phases) == 0 {
		where = fmt.Sprintf("Customer=%v order by %s %s limit %v,%v", uid, orderby, s, pn*rn, rn)
	} else {
		var buf bytes.Buffer
		for i, p := range phases {
			if i == 0 {
				buf.WriteString(fmt.Sprintf("%v", p))
			} else {
				buf.WriteString(fmt.Sprintf(",%v", p))
			}

		}
		where = fmt.Sprintf("Customer=%v and Phase in(%s) order by %s %s limit %v,%v", uid, buf.String(), orderby, s, pn*rn, rn)
	}
	sql := fmt.Sprintf(common.SQL_ListOrder, where)
	rows, err := d.db.Query(sql)
	if err != nil {
		return
	}
	defer rows.Close()
	orders = make([]model.Order, 0, 5)
	for rows.Next() {
		var o model.Order
		var startTime, phaseTime, serviceStart string
		rows.Scan(&o.Id, &o.Role, &o.Job, &o.Customer, &o.Address, &o.Community, &o.Persons, &startTime, &o.Phase, &phaseTime, &o.PhaseReason, &o.CSR, &serviceStart, &o.Duration, &o.Description)
		o.StartTime, err = time.Parse(format.TIME_LAYOUT_1, startTime)
		if err != nil {
			return
		}
		o.PhaseTime, err = time.Parse(format.TIME_LAYOUT_1, phaseTime)
		if err != nil {
			return
		}
		fmt.Printf("serviceStart: %s\n", serviceStart)
		o.ServiceStart, err = time.Parse(format.TIME_LAYOUT_1, serviceStart)
		if err != nil {
			return
		}
		orders = append(orders, o)
	}
	return
}
func (d *TransactionDBAdapter) UpdateOrder(o *model.Order, tx *sql.Tx) (err error) {
	if tx == nil {
		_, err = d.db.Exec(common.SQL_UpdateOrder, o.Role, o.Job, o.Customer, o.Address, o.Community, o.Persons, o.StartTime.Format(format.TIME_LAYOUT_1), o.Phase, o.PhaseTime.Format(format.TIME_LAYOUT_1), o.PhaseReason, o.CSR, o.ServiceStart.Format(format.TIME_LAYOUT_1), o.Duration, o.Description, o.Id)
		if err != nil {
			return
		}
	} else {
		_, err = tx.Exec(common.SQL_UpdateOrder, o.Role, o.Job, o.Customer, o.Address, o.Community, o.Persons, o.StartTime.Format(format.TIME_LAYOUT_1), o.Phase, o.PhaseTime.Format(format.TIME_LAYOUT_1), o.PhaseReason, o.CSR, o.ServiceStart.Format(format.TIME_LAYOUT_1), o.Duration, o.Description, o.Id)
		if err != nil {
			tx.Rollback()
			return
		}
	}
	return
}
func (d *TransactionDBAdapter) GetHKPProviders(oid uint64, jobid uint) (p map[common.UIDType]model.HKPOrderProvider, err error) {
	rows, err := d.db.Query(common.SQL_GetHKPProviders, oid, jobid)
	if err != nil {
		return
	}
	defer rows.Close()
	p = make(map[common.UIDType]model.HKPOrderProvider)
	for rows.Next() {
		var prov model.HKPOrderProvider
		var spd, qua, att uint8
		var beginTime, comment, availableTime string
		err = rows.Scan(&prov.UserId, &prov.Job.Version, &prov.Recommend, &availableTime, &prov.Confirm, &prov.Priority, &spd, &qua, &att, &comment, &beginTime, &prov.Job.Price, &prov.Job.Desc)
		if err != nil {
			return
		}
		prov.OrderId = oid
		prov.AvailableTime, err = time.Parse(format.TIME_LAYOUT_1, availableTime)
		if err != nil {
			return
		}
		prov.Job.BeginTime, err = time.Parse(format.TIME_LAYOUT_1, beginTime)
		if err != nil {
			return
		}
		if err = prov.SetRank(spd, qua, att); err != nil {
			err = errors.New(fmt.Sprintf("SetRank of user [%v] error : ", prov.UserId, err.Error()))
			return
		}
		if err = prov.SetComment(comment); err != nil {
			err = errors.New(fmt.Sprintf("SetComment of user [%v] error : ", prov.UserId, err.Error()))
			return
		}
		p[prov.UserId] = prov
	}
	err = rows.Err()
	if err != nil {
		return
	}
	return
}
func (d *TransactionDBAdapter) GetConfirmedHKPProviders(oid uint64, jobid uint) (p map[common.UIDType]model.HKPOrderProvider, err error) {
	rows, err := d.db.Query(common.SQL_GetConfirmedHKPProviders, oid, jobid, common.HKP_PROVIDER_CONFIRM_YES)
	if err != nil {
		return
	}
	defer rows.Close()
	p = make(map[common.UIDType]model.HKPOrderProvider)
	for rows.Next() {
		var prov model.HKPOrderProvider
		var spd, qua, att uint8
		var beginTime, availableTime string
		err = rows.Scan(&prov.UserId, &prov.Job.Version, &prov.Recommend, &availableTime, &prov.Priority, &spd, &qua, &att, &beginTime, &prov.Job.Price, &prov.Job.Desc)
		if err != nil {
			return
		}
		prov.OrderId = oid
		if err = prov.SetAttitude(att); err != nil {
			return
		}
		if err = prov.SetQuality(qua); err != nil {
			return
		}
		if err = prov.SetSpeed(spd); err != nil {
			return
		}
		prov.Job.JobId = jobid
		prov.Confirm = common.HKP_PROVIDER_CONFIRM_YES
		prov.AvailableTime, err = time.Parse(format.TIME_LAYOUT_1, availableTime)
		if err != nil {
			return
		}
		prov.Job.BeginTime, err = time.Parse(format.TIME_LAYOUT_1, beginTime)
		if err != nil {
			return
		}
		if err = prov.SetRank(spd, qua, att); err != nil {
			err = errors.New(fmt.Sprintf("SetRank of user [%u] error : ", prov.UserId, err.Error()))
			return
		}
		p[prov.UserId] = prov
	}
	err = rows.Err()
	if err != nil {
		return
	}
	return
}
func (d *TransactionDBAdapter) UpdateHKPProvider(p model.HKPOrderProvider, tx *sql.Tx) (err error) {
	var needCommit bool = false
	if tx == nil {
		tx, err = d.Begin()
		if err != nil {
			return
		}
		needCommit = true
	}
	_, err = tx.Exec(common.SQL_UpdateHKPOrderProvider, p.Job.Version, p.Recommend, p.AvailableTime.Format(format.TIME_LAYOUT_1), p.Confirm, p.Priority, p.Speed(), p.Quality(), p.Attitude(), p.Comment(), p.OrderId, p.UserId)
	if err != nil {
		if needCommit {
			err = tx.Rollback()
		}
		return err
	}
	if needCommit {
		err = tx.Commit()
	}
	return
}
func (d *TransactionDBAdapter) UpdateHKPProviders(providers []model.HKPOrderProvider, tx *sql.Tx) (err error) {
	var needCommit bool = false
	if tx == nil {
		tx, err = d.Begin()
		if err != nil {
			return
		}
		needCommit = true
	}
	for _, p := range providers {
		_, err := tx.Exec(common.SQL_UpdateHKPOrderProvider, p.Job.Version, p.Recommend, p.AvailableTime.Format(format.TIME_LAYOUT_1), p.Confirm, p.Priority, p.Speed(), p.Quality(), p.Attitude(), p.Comment(), p.OrderId, p.UserId)
		if err != nil {
			if needCommit {
				err = tx.Rollback()
			}
			return err
		}
	}
	if needCommit {
		err = tx.Commit()
	}
	return
}
func (d *TransactionDBAdapter) UpdateHKPRanks(rank []model.HKPRank, tx *sql.Tx) (err error) {
	var needCommit bool = false
	if tx == nil {
		tx, err = d.Begin()
		if err != nil {
			return
		}
		needCommit = true
	}
	for _, r := range rank {
		_, err := tx.Exec(common.SQL_UpdateHKPRank, r.Times, r.TotalSpeed, r.TotalQuality, r.TotalAttitude, r.RejectTimes, r.FailTimes, r.Uid, r.JobId)
		if err != nil {
			if needCommit {
				err = tx.Rollback()
			}
			return err
		}
	}
	if needCommit {
		err = tx.Commit()
	}
	return
}
func (d *TransactionDBAdapter) AddHKPRankTimes(uid common.UIDType, job uint, tx *sql.Tx) (err error) {
	var needCommit bool = false
	if tx == nil {
		tx, err = d.Begin()
		if err != nil {
			return
		}
		needCommit = true
	}
	_, err = tx.Exec(common.SQL_AddHKPRankTimes, uid, job)
	if err != nil {
		if needCommit {
			err = tx.Rollback()
		}
		return err
	}
	if needCommit {
		err = tx.Commit()
	}
	return
}
func (d *TransactionDBAdapter) AddHKPsRankTimes(uids []common.UIDType, job uint, tx *sql.Tx) (err error) {
	var needCommit bool = false
	if tx == nil {
		tx, err = d.Begin()
		if err != nil {
			return
		}
		needCommit = true
	}
	for _, uid := range uids {
		_, err := tx.Exec(common.SQL_AddHKPRankTimes, uid, job)
		if err != nil {
			if needCommit {
				err = tx.Rollback()
			}
			return err
		}
	}
	if needCommit {
		err = tx.Commit()
	}
	return
}
func (d *TransactionDBAdapter) GetHKPRanks(uids []common.UIDType, jobId uint) (ranks map[common.UIDType]model.HKPRank, err error) {
	var buf bytes.Buffer
	for i, uid := range uids {
		if i == 0 {
			buf.WriteString(fmt.Sprintf("%v", uid))
		} else {
			buf.WriteString(fmt.Sprintf(",%v", uid))
		}

	}
	sql := fmt.Sprintf(common.SQL_GetHKPRanks, buf.String(), jobId)
	rows, err := d.db.Query(sql)
	if err != nil {
		return
	}
	defer rows.Close()
	ranks = make(map[common.UIDType]model.HKPRank)
	for rows.Next() {
		var r model.HKPRank
		r.JobId = jobId
		err = rows.Scan(&r.Uid, &r.Times, &r.TotalSpeed, &r.TotalQuality, &r.TotalAttitude, &r.RejectTimes, &r.FailTimes)
		if err != nil {
			return
		}
		ranks[r.Uid] = r
	}
	return
}
func (d *TransactionDBAdapter) AddHKPFailTimes(providers map[common.UIDType]model.HKPOrderProvider, jobid uint, times uint) (err error) {
	for uid, _ := range providers {
		_, err = d.db.Exec(common.SQL_AddHKPFailTimes, times, uid, jobid)
		if err != nil {
			return
		}
	}
	return
}
func (d *TransactionDBAdapter) AddHistory(h *model.History) (err error) {
	_, err = d.db.Exec(common.SQL_AddHistory, h.Customer(), h.ServiceProvider(), h.Role(), h.Job(), h.LastTime.Format(format.TIME_LAYOUT_1), h.LastTime.Format(format.TIME_LAYOUT_1))
	return
}
func (d *TransactionDBAdapter) UpdateAddressLastUse(aid uint64) (err error) {
	_, err = d.db.Exec(common.SQL_UpdateAddressLastUse, time.Now().Format(format.TIME_LAYOUT_1), aid)
	return
}

//临时用HKPOrderProvider里的UserID属性装载CustomerID
func (d *TransactionDBAdapter) HKPComments(uid common.UIDType, pn uint, rn uint) (comments []model.HKPOrderProvider, err error) {
	rows, err := d.db.Query(common.SQL_HKPComments, uid, pn, rn)
	if err != nil {
		return
	}
	defer rows.Close()
	comments = make([]model.HKPOrderProvider, 0, rn)
	for rows.Next() {
		var p model.HKPOrderProvider
		var s, q, a uint8
		var c string
		err = rows.Scan(&p.UserId, &s, &q, &a, &c)
		if err = p.SetAttitude(a); err != nil {
			return
		}
		if err = p.SetQuality(q); err != nil {
			return
		}
		if err = p.SetSpeed(s); err != nil {
			return
		}
		if err = p.SetComment(c); err != nil {
			return
		}
		if err != nil {
			return
		}
		comments = append(comments, p)
	}
	return
}

func (d *TransactionDBAdapter) IsFirstOrder(uid common.UIDType) (first bool, err error) {
	var oid uint64
	err = d.db.QueryRow(common.SQL_IsFirstOrder, uid, common.HKP_PHASE_RANK_COMPLETE).Scan(&oid)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		} else {
			return false, err
		}
	}
	return false, nil
}
func (d *TransactionDBAdapter) ListTransactions(uid common.UIDType, pn uint, rn uint) (trans []model.Transaction, err error) {
	rows, err := d.db.Query(common.SQL_ListTransactions, uid, pn, rn)
	if err != nil {
		return
	}
	defer rows.Close()
	trans = make([]model.Transaction, 0, rn)
	for rows.Next() {
		var t model.Transaction
		var timeStr string
		rows.Scan(&t.Id, &t.Type, &timeStr, &t.OrderId, &t.AddScore, &t.AddBalance, &t.AddCoupons, &t.Score, &t.Balance, &t.Coupons, &t.Desc)
		t.Time, err = time.Parse(format.TIME_LAYOUT_1, timeStr)
		if err != nil {
			return
		}
		trans = append(trans, t)
	}
	return
}
func (d *TransactionDBAdapter) ListGoods(pn uint, rn uint) (goods []model.Goods, err error) {
	rows, err := d.db.Query(common.SQL_ListGoods, pn, rn)
	if err != nil {
		return
	}
	defer rows.Close()
	goods = make([]model.Goods, 0, rn)
	for rows.Next() {
		var g model.Goods
		rows.Scan(&g.Id, &g.Name, &g.Pic, &g.Price, &g.Detail)
		goods = append(goods, g)
	}
	return
}
func (d *TransactionDBAdapter) GetGoods(gid uint) (g model.Goods, err error) {
	err = d.db.QueryRow(common.SQL_GetGoods, gid).Scan(&g.Id, &g.Name, &g.Pic, &g.Price, &g.Detail, &g.Explain)
	if err != nil {
		return
	}
	return
}

func (d *TransactionDBAdapter) GetNeedNotifyOrders() (orders []model.Order, err error) {
	rows, err := d.db.Query(common.SQL_GetNeedNotifyOrders, time.Now().Add(-10*time.Hour).Format(format.TIME_LAYOUT_1), time.Now().Format(format.TIME_LAYOUT_1), common.HKP_PHASE_ORDER_SUCCESS)
	if err != nil {
		return
	}
	defer rows.Close()
	orders = make([]model.Order, 0, 10)
	for rows.Next() {
		var o model.Order
		var serviceStart string
		rows.Scan(&o.Id, &o.Customer, &serviceStart, &o.Duration)
		fmt.Printf("order %v\n", o.Id)
		o.ServiceStart, err = time.ParseInLocation(format.TIME_LAYOUT_1, serviceStart, time.Local)
		if err != nil {
			return
		}
		complete := o.ServiceStart.Add(time.Duration(o.Duration+30) * time.Minute)
		fmt.Println(complete)
		if time.Now().After(complete) && time.Now().Before(complete.Add(10*time.Minute)) {
			fmt.Println("append")
			orders = append(orders, o)
		}
	}
	return
}
