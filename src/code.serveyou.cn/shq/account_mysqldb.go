/*
	mysql数据库连接适配器
*/
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/model"
	"code.serveyou.cn/pkg/format"
	_ "github.com/go-sql-driver/mysql"
)

type AccountDBAdapter struct {
	db *sql.DB
}

func NewAccountDBAdapter(db *sql.DB) (dba *AccountDBAdapter) {
	dba = new(AccountDBAdapter)
	dba.db = db
	return
}

func (a *AccountDBAdapter) SetVerifyCode(vcode *model.VerifyCode) (err error) {
	_, err = a.db.Exec(common.SQL_SetVerifyCode, vcode.Cellphone(), vcode.ForWhat(), vcode.Code(), vcode.Timeout().Format(format.TIME_LAYOUT_1), vcode.Code(), vcode.Timeout().Format(format.TIME_LAYOUT_1))
	return err
}

func (a *AccountDBAdapter) VerifyCode(vcode *model.VerifyCode) (err common.Error) {
	var t string
	e := a.db.QueryRow(common.SQL_VerifyCode, vcode.Cellphone(), vcode.ForWhat(), vcode.Code()).Scan(&t)
	if e != nil {
		if e == sql.ErrNoRows {
			err = common.NewError(common.ERR_VERIFY_FAIL, "verify failed")
			return
		} else {
			err = common.NewError(common.ERR_INTERNAL, e.Error())
			return
		}
	}
	to, e := time.Parse(format.TIME_LAYOUT_1, t)
	if e != nil {
		err = common.NewError(common.ERR_INVALID_FORMAT, e.Error())
		return
	}
	if to.After(time.Now()) {
		return
	} else {
		err = common.NewError(common.ERR_VCODE_TIMEOUT, "code timeout")
		return
	}
}

func (a *AccountDBAdapter) GetPassword(uid common.UIDType, devid string) (password string, err error) {
	err = a.db.QueryRow(common.SQL_GetPassword, uid, devid).Scan(&password)
	if err != nil {
		return
	}
	return
}

func (a *AccountDBAdapter) NewPassword(app *model.AppClient) (err error) {
	_, err = a.db.Exec(common.SQL_NewPassword, app.UserId(), app.Password(), app.DeviceName(), app.DeviceId(), app.Platform(), app.LastUpdate.Format(format.TIME_LAYOUT_1), app.DeviceName(), app.Password(), app.Platform(), app.LastUpdate.Format(format.TIME_LAYOUT_1))
	return
}

func (a *AccountDBAdapter) GetUid(cellphone string) (uid common.UIDType, err error) {
	err = a.db.QueryRow(common.SQL_GetUid, cellphone).Scan(&uid)
	if err != nil {
		return
	}
	return
}

func (a *AccountDBAdapter) GetUserByPhone(cellphone string) (u *model.User, err error) {
	u = model.NewUser()
	var uid common.UIDType
	var nn string
	err = a.db.QueryRow(common.SQL_GetUserByPhone, cellphone).Scan(&uid, &nn)
	if err != nil {
		return
	}
	u.SetId(uid)
	err = u.SetNickName(nn)
	return
}

func (a *AccountDBAdapter) CreateUser(u *model.User, tx *sql.Tx) (uid common.UIDType, err error) {
	var res sql.Result
	if tx == nil {
		res, err = a.db.Exec(common.SQL_CreateUser, u.NickName(), u.LastName(), u.FirstName(), u.Cellphone(), u.IdCardType(), u.IdCardNum(), u.Sex(), u.Birthday().Format(format.TIME_LAYOUT_2), u.Birthplace(), u.Mail(), u.Inviter(), u.HealthCardTimeout())
	} else {
		res, err = tx.Exec(common.SQL_CreateUser, u.NickName(), u.LastName(), u.FirstName(), u.Cellphone(), u.IdCardType(), u.IdCardNum(), u.Sex(), u.Birthday().Format(format.TIME_LAYOUT_2), u.Birthplace(), u.Mail(), u.Inviter(), u.HealthCardTimeout())
	}

	if err != nil {
		return
	}
	tmpid, err := res.LastInsertId()
	if err != nil {
		return
	}
	uid = common.UIDType(tmpid)
	return
}

func (a *AccountDBAdapter) CreateAccount(acc *model.Account, tx *sql.Tx) (err error) {
	if tx == nil {
		_, err = a.db.Exec(common.SQL_CreateAccount, acc.UserId, acc.Balance, acc.Score, acc.Coupons)
	} else {
		_, err = tx.Exec(common.SQL_CreateAccount, acc.UserId, acc.Balance, acc.Score, acc.Coupons)
	}
	return
}

func (a *AccountDBAdapter) CreateAppClient(ac *model.AppClient, tx *sql.Tx) (err error) {
	if tx == nil {
		_, err = a.db.Exec(common.SQL_CreateAppClient, ac.UserId(), ac.Password(), ac.DeviceName(), ac.DeviceId(), ac.Platform(), ac.LastUpdate.Format(format.TIME_LAYOUT_1))
	} else {
		_, err = tx.Exec(common.SQL_CreateAppClient, ac.UserId(), ac.Password(), ac.DeviceName(), ac.DeviceId(), ac.Platform(), ac.LastUpdate.Format(format.TIME_LAYOUT_1))
	}

	return
}

func (a *AccountDBAdapter) CreateAddress(ad *model.Address, tx *sql.Tx) (err error) {
	var res sql.Result
	if tx == nil {
		res, err = a.db.Exec(common.SQL_CreateAddress, ad.UserId(), ad.Community, ad.Addr(), ad.Status(), ad.LastUse.Format(format.TIME_LAYOUT_1))
	} else {
		res, err = tx.Exec(common.SQL_CreateAddress, ad.UserId(), ad.Community, ad.Addr(), ad.Status(), ad.LastUse.Format(format.TIME_LAYOUT_1))
	}
	if err != nil {
		return
	}
	aid, err := res.LastInsertId()
	if err != nil {
		return
	}
	ad.AddrId = uint64(aid)
	return
}
func (a *AccountDBAdapter) ListAddress(uid common.UIDType) (ads []model.Address, err error) {
	rows, err := a.db.Query(common.SQL_ListAddress, uid)
	if err != nil {
		return
	}
	defer rows.Close()
	ads = make([]model.Address, 0, 10)
	for rows.Next() {
		var addr model.Address
		var address string
		rows.Scan(&addr.AddrId, &addr.Community, &address)
		if err = addr.SetAddr(address); err != nil {
			return
		}
		ads = append(ads, addr)
	}
	return
}

func (a *AccountDBAdapter) IsValidCommunity(communityId uint) (v bool, err error) {
	v = false
	var id uint
	err = a.db.QueryRow(common.SQL_IsValidCommunity, communityId).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return v, nil
		} else {
			return
		}
	}
	return true, nil
}

/*
func (a *AccountDBAdapter) IsDefaultAddr(uid common.UIDType, aid uint64) (v bool, err error) {
	v = false
	var id uint64
	err = a.db.QueryRow(common.SQL_IsDefaultAddr, uid).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.New(fmt.Sprintf("invalid uid : [%d]", uid))
			return
		}
		return
	}
	return id == aid, nil
}
*/

func (a *AccountDBAdapter) UpdateUserCellphone(user *model.User) (err error) {
	_, err = a.db.Exec(common.SQL_UpdateUserCellphone, user.Cellphone(), user.Id())
	if err != nil {
		return errors.New("duplicate cellphone number")
	}
	return
}

/*
func (a *AccountDBAdapter) UpdateDefaultAddr(uid common.UIDType, adid uint64) (err error) {
	_, err = a.db.Exec(common.SQL_UpdateDefaultAddr, adid, uid)
	return
}
*/
func (a *AccountDBAdapter) DelAddress(uid common.UIDType, adid uint64) (err error) {
	_, err = a.db.Exec(common.SQL_DelAddress, uid, adid)
	return
}
func (a *AccountDBAdapter) UpdateAddress(addr *model.Address) (err error) {
	var upd_str string
	if addr.Community > 0 {
		upd_str = fmt.Sprintf("Community=%v", addr.Community)
	}
	if addr.Addr() != "" {
		if upd_str != "" {
			upd_str += ","
		}
		upd_str += fmt.Sprintf("Address='%s'", addr.Addr())
	}
	sql := fmt.Sprintf(common.SQL_UpdateAddress, upd_str)
	res, err := a.db.Exec(sql, addr.UserId(), addr.AddrId)
	if err != nil {
		return
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return
	}
	if ra == 0 {
		err = errors.New(fmt.Sprintf("address uid=[%v],aid=[%v] not found", addr.UserId(), addr.AddrId))
	}
	return
}

func (a *AccountDBAdapter) UpdateInviter(user *model.User) (err error) {
	_, err = a.db.Exec(common.SQL_UpdateInviter, user.Inviter(), user.Id())
	return
}

func (a *AccountDBAdapter) AddCoupons(acc *model.Account) (err error) {
	_, err = a.db.Exec(common.SQL_AddCoupons, acc.Coupons, acc.UserId)
	return
}
func (a *AccountDBAdapter) AddCollection(c *model.Collection) (err error) {
	_, err = a.db.Exec(common.SQL_AddCollection, c.Customer(), c.ServiceProvider(), c.Role(), c.Job(), c.CollectTime.Format(format.TIME_LAYOUT_1))
	return
}
func (a *AccountDBAdapter) DelCollection(c *model.Collection) (err error) {
	where := fmt.Sprintf("Customer=%v and ServiceProvider=%v", c.Customer(), c.ServiceProvider())
	if c.Role() > 0 {
		where += fmt.Sprintf(" and Role=%v", c.Role())
		if c.Job() > 0 {
			where += fmt.Sprintf(" and Job=%v", c.Job())
		}
	}
	sql := fmt.Sprintf(common.SQL_DelCollection, where)
	_, err = a.db.Exec(sql)
	return
}
func (a *AccountDBAdapter) ListCollection(c *model.Collection, pn uint, rn uint) (prs []common.UIDType, err error) {
	where := fmt.Sprintf("Customer=%v", c.Customer())
	if c.Role() != 0 {
		where += fmt.Sprintf(" and Role=%v", c.Role())
	}
	if c.Job() != 0 {
		where += fmt.Sprintf(" and Job=%v", c.Job())
	}

	sql := fmt.Sprintf(common.SQL_ListCollection, where)
	rows, err := a.db.Query(sql, pn, rn)
	if err != nil {
		return
	}
	defer rows.Close()
	prs = make([]common.UIDType, 0, rn)
	for rows.Next() {
		var id common.UIDType
		rows.Scan(&id)
		prs = append(prs, id)
	}
	return
}
func (a *AccountDBAdapter) ListHistory(h *model.History, pn uint, rn uint) (prs []model.History, err error) {
	where := fmt.Sprintf("Customer=%v", h.Customer())
	where += fmt.Sprintf(" and Role=%v", h.Role())
	where += fmt.Sprintf(" and Job=%v", h.Job())

	sql := fmt.Sprintf(common.SQL_ListHistory, where)
	rows, err := a.db.Query(sql, pn, rn)
	if err != nil {
		return
	}
	defer rows.Close()
	prs = make([]model.History, 0, rn)
	var history model.History
	if err = history.SetRole(h.Role()); err != nil {
		return
	}
	if err = history.SetJob(h.Job()); err != nil {
		return
	}
	for rows.Next() {
		var lastTime string
		var provider common.UIDType
		rows.Scan(&provider, &lastTime)
		if err = history.SetServiceProvider(provider); err != nil {
			return
		}
		history.LastTime, err = time.Parse(format.TIME_LAYOUT_1, lastTime)
		if err != nil {
			return
		}
		prs = append(prs, history)
	}
	return
}
func (a *AccountDBAdapter) GetHKPJobMaxVersion(uid common.UIDType, jobId uint) (j *model.HKPJob, err error) {
	j = model.NewHKPJob()
	j.Uid = uid
	j.JobId = jobId
	var t string
	err = a.db.QueryRow(common.SQL_GetHKPJobMaxVersion, uid, jobId).Scan(&t, &j.Price, &j.Desc, &j.Version)
	if err != nil {
		return
	}
	j.BeginTime, err = time.Parse(format.TIME_LAYOUT_1, t)
	if err != nil {
		return
	}
	return
}
func (a *AccountDBAdapter) GetBalance(uid common.UIDType) (b *model.Balance, err error) {
	b = model.NewBalance(uid)
	err = a.db.QueryRow(common.SQL_GetBalance, uid).Scan(&b.Balance, &b.Score, &b.Coupons)
	if err != nil {
		return
	}
	return
}

func (a *AccountDBAdapter) UpdateUser(user *model.User) (err error) {
	var upd_str string
	if user.Sex() != common.SEX_UNKNOWN {
		upd_str = fmt.Sprintf("Sex=%v", user.Sex())
	}
	if user.NickName() != "" {
		if upd_str != "" {
			upd_str += ","
		}
		upd_str += fmt.Sprintf("NickName='%s'", user.NickName())
	}
	if user.FirstName() != "" {
		if upd_str != "" {
			upd_str += ","
		}
		upd_str += fmt.Sprintf("FirstName='%s'", user.FirstName())
	}
	if user.LastName() != "" {
		if upd_str != "" {
			upd_str += ","
		}
		upd_str += fmt.Sprintf("LastName='%s'", user.LastName())
	}
	if user.Mail() != "" {
		if upd_str != "" {
			upd_str += ","
		}
		upd_str += fmt.Sprintf("Mail='%s'", user.Mail())
	}
	sql := fmt.Sprintf(common.SQL_UpdateUserInfo, upd_str)
	_, err = a.db.Exec(sql, user.Id())
	if err != nil {
		return
	}
	return
}
func (a *AccountDBAdapter) Feedback(uid common.UIDType, content string) (err error) {
	_, err = a.db.Exec(common.SQL_Feedback, uid, content)
	return
}
func (a *AccountDBAdapter) Vote(ids string) (err error) {
	sql := fmt.Sprintf(common.SQL_Vote, ids)
	_, err = a.db.Exec(sql)
	return
}
func (a *AccountDBAdapter) ListOptions() (ops []model.VoteOption, err error) {
	rows, err := a.db.Query(common.SQL_ListOptions)
	if err != nil {
		return
	}
	defer rows.Close()
	ops = make([]model.VoteOption, 0, 10)
	for rows.Next() {
		var option model.VoteOption
		rows.Scan(&option.Id, &option.Name, &option.Pic, &option.Detail)
		ops = append(ops, option)
	}
	return
}
