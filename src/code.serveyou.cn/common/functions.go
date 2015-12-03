package common

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"code.serveyou.cn/pkg/format"
)

func SetUid(u UIDType, uid *UIDType) (err error) {
	*uid = u
	return
}
func SetUidStr(u string, uid *UIDType) (err error) {
	ui, err := format.ParseUint(u)
	if err != nil {
		err = errors.New("convert [" + u + "] to uint64 error : " + err.Error())
		return
	}
	err = SetUid(UIDType(ui), uid)
	return
}

func SetRole(r uint, role *uint) (err error) {
	if r != ROLE_HKP {
		err = errors.New(fmt.Sprintf("role must be HKP(%v)", ROLE_HKP))
		return
	}
	*role = r
	return
}
func SetRoleStr(r string, role *uint) (err error) {
	ro, err := format.ParseUint(r)
	if err != nil {
		err = errors.New("convert [" + r + "] to uint error : " + err.Error())
		return
	}
	err = SetRole(ro, role)
	return
}

func SetJob(role uint, j uint, job *uint) (err error) {
	if role == 0 {
		err = errors.New("must set role first")
		return
	}
	if !RoleJob.IsValidJob(role, j) {
		err = errors.New(fmt.Sprintf("invalid jobid [%v] for role [%v]", j, role))
		return
	}
	*job = j
	return
}
func SetJobStr(role uint, j string, job *uint) (err error) {
	jo, err := format.ParseUint(j)
	if err != nil {
		err = errors.New("convert [" + j + "] to uint error : " + err.Error())
		return
	}
	err = SetJob(role, jo, job)
	return
}

func MaskIDCardNum(n string) (masked string, err error) {
	if len(n) < 15 {
		err = errors.New(fmt.Sprintf("invalid idcardnum [%v]", n))
		return
	}
	masked = n[0:len(n)-10] + "*****" + n[len(n)-5:]
	return
}

func NewDB(dsn string) (db *sql.DB, err error) {
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return
}

func AddBSC(typ uint, uid UIDType, oid uint64, score int, balance float32, coupons int, desc string, tx *sql.Tx) (newScore int, newBalance float32, newCoupons int, err error) {
	var oldBalance float32
	var oldCoupons, oldScore int
	fmt.Printf("AddBSC : uid=%v\n", uid)
	err = tx.QueryRow(SQL_GetBSC, uid).Scan(&oldBalance, &oldCoupons, &oldScore)
	if err != nil {
		return
	}
	newScore = oldScore + score
	if newScore < 0 {
		err = errors.New("not enough score")
		return
	}
	newBalance = oldBalance + balance
	if newBalance < 0 {
		err = errors.New("not enough money")
		return
	}
	newCoupons = oldCoupons + coupons
	if newCoupons < 0 {
		err = errors.New("not enough coupons")
		return
	}
	_, err = tx.Exec(SQL_AddTransaction, typ, time.Now(), oid, uid, score, coupons, balance, desc, oldBalance, oldScore, oldCoupons)
	if err != nil {
		return
	}
	_, err = tx.Exec(SQL_UpdateAccount, newBalance, newScore, newCoupons, uid)
	if err != nil {
		return
	}
	return
}
