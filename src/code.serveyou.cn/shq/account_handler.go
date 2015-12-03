package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/model"
	"code.serveyou.cn/pkg/encrypt"
	"code.serveyou.cn/pkg/format"
)

type AccountHandler struct {
	adb   *AccountDBAdapter
	voted map[common.UIDType]time.Time
}

func NewAccountHandler(db *sql.DB) (a *AccountHandler) {
	a = new(AccountHandler)
	a.adb = NewAccountDBAdapter(db)
	a.voted = make(map[common.UIDType]time.Time)
	return
}

func (a *AccountHandler) ProcessSec(cmd string, r *http.Request, uid common.UIDType, devid string, body string, result map[string]interface{}) (err common.Error) {
	switch cmd {
	case "add_collect":
		err = a.addCollect(r, uid, body, result)
	case "del_collect":
		err = a.delCollect(r, uid, body, result)
	case "list_collect":
		err = a.listCollect(r, uid, body, result)
	case "list_history":
		err = a.listHistory(r, uid, body, result)
	case "get_balance":
		err = a.getBalance(r, uid, body, result)
	case "add_addr":
		err = a.addAddress(r, uid, body, result)
	case "del_addr":
		err = a.delAddress(r, uid, body, result)
	case "update_addr":
		err = a.updateAddress(r, uid, body, result)
	case "list_addr":
		err = a.listAddress(r, uid, body, result)
	case "update_info":
		err = a.updateInfo(r, uid, body, result)
	case "vote":
		err = a.vote(r, uid, body, result)
	case "feedback":
		err = a.feedback(r, uid, body, result)
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, "unknown sec command : "+cmd)
		return
	}
	return
}

func (a *AccountHandler) Process(cmd string, r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	switch cmd {
	case "vcode":
		err = a.vcode(r, result)
	case "register":
		err = a.register(r, body, result)
	case "reg_device":
		err = a.registerDevice(r, body, result)
	case "new_password":
		fmt.Println("new_password")
		err = a.newPassword(r, body, result)
	case "get_hkp_detail":
		err = a.getHKPDetail(r, body, result)
	case "variables":
		err = a.variables(r, body, result)
	case "list_options":
		err = a.listOptions(r, body, result)
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, "unknown command : "+cmd)
		return
	}
	return
}

func (a *AccountHandler) GetPassword(uid common.UIDType, devid string) (password string, err common.Error) {
	password, e := a.adb.GetPassword(uid, devid)
	if e != nil {
		if e == sql.ErrNoRows {
			err = common.NewError(common.ERR_VERIFY_FAIL, fmt.Sprintf("not found user %v", uid))
		} else {
			err = common.NewError(common.ERR_INTERNAL, e.Error())
		}
	}
	return
}

func (a *AccountHandler) vcodeRegLogin(phone string, result map[string]interface{}) (err common.Error) {
	t := "reg"
	_, e := a.adb.GetUid(phone)
	if e != sql.ErrNoRows {
		if e != nil {
			err = common.NewError(common.ERR_INTERNAL, e.Error())
			return
		} else {
			t = "login"
		}
	}
	vcode, e := model.NewVerifyCode(phone, encrypt.RandomNumeric(6),
		common.DB_VTYPE_REGLOGIN, time.Now().Add(time.Duration(common.Variables.VCodeTimeout())*time.Minute))
	if e != nil {
		err = common.NewError(common.ERR_INVALID_FORMAT, e.Error())
		return
	}
	e = a.adb.SetVerifyCode(vcode)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, e.Error())
		return
	}

	//send sms
	var sms string
	switch t {
	case "reg":
		sms = fmt.Sprintf("%v(生活圈注册验证码，请完成验证），如非本人操作，请忽略本短信。", vcode.Code())
	case "login":
		sms = fmt.Sprintf("%v(生活圈找回密码验证码，请完成验证），如非本人操作，请忽略本短信。", vcode.Code())
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("invalid vtype : %v", vcode.ForWhat()))
		return
	}
	e = common.SendSMS(phone, sms)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, e.Error())
		return
	}

	result["result"] = sms
	result["type"] = t
	return
}

func (a *AccountHandler) vcode(r *http.Request, result map[string]interface{}) (err common.Error) {
	phone, found := r.URL.Query()["phone"]
	if !found {
		err = common.NewError(common.ERR_INVALID_PARAM, "no [phone] provided.")
		return
	}
	forWhat, found := r.URL.Query()["for"]
	if !found {
		err = common.NewError(common.ERR_INVALID_PARAM, "no [for] provided.")
		return
	}
	vtype, e := format.ParseUint8(forWhat[0])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, "parse [for] error : "+e.Error())
		return
	}
	switch vtype {
	case 1:
		err = a.vcodeRegLogin(phone[0], result)
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, "invalid [for] value.")
		return
	}

	return
}

func (a *AccountHandler) registerDevice(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}

	requiredKeys := []string{"devid", "dname", "platform"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}

	app := model.NewAppClient(0)
	if e = app.SetPassword(encrypt.RandomAlphanumeric(16)); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set password error : %v", e.Error()))
		return
	}
	pid, e := format.ParseUint8(values["platform"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [platform] error : %v", e.Error()))
		return
	}
	if e = app.SetPlatform(uint8(pid)); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set platform error : %v", e.Error()))
		return
	}
	if e = app.SetDeviceName(values["dname"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set dname error : %v", e.Error()))
		return
	}
	if e = app.SetDeviceId(values["devid"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set devid error : %v", e.Error()))
		return
	}
	app.LastUpdate = time.Now()
	if e = a.adb.CreateAppClient(app, nil); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("CreateAppClient error : %s", e.Error()))
		return
	}
	return
}

func (a *AccountHandler) register(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}

	requiredKeys := []string{"phone", "role", "vcode", "devid", "dname", "platform"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}

	vcode, e := model.NewVerifyCode(values["phone"], values["vcode"], common.DB_VTYPE_REGLOGIN, common.InitDate)
	if e != nil {
		err = common.NewError(common.ERR_INVALID_FORMAT, e.Error())
		return
	}
	err = a.adb.VerifyCode(vcode)
	if err.Code != 0 {
		return
	}

	roleStr := values["role"]
	role, e := format.ParseUint(roleStr)
	if e != nil {
		err = common.NewError(common.ERR_INVALID_FORMAT, e.Error())
		return
	}

	var uid common.UIDType
	var aid uint64
	var password string
	switch role {
	case common.ROLE_CUSTOMER:
		uid, password, aid, err = a.registerCustomer(values)
	case common.ROLE_HKP:
		uid, password, err = a.registerHKP(values)
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("invalid [role] value : [%v]", roleStr))
		return
	}
	if err.Code == 0 {
		result["uid"] = uid
		result["password"] = password
		result["aid"] = aid
	}
	return
}

func (a *AccountHandler) newPassword(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}

	requiredKeys := []string{"phone", "dname", "devid", "vcode", "platform"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	vcode, e := model.NewVerifyCode(values["phone"], values["vcode"], common.DB_VTYPE_REGLOGIN, common.InitDate)
	if e != nil {
		err = common.NewError(common.ERR_INVALID_FORMAT, e.Error())
		return
	}
	err = a.adb.VerifyCode(vcode)
	if err.Code != 0 {
		return
	}
	uid, e := a.adb.GetUid(values["phone"])
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, e.Error())
		return
	}
	app := model.NewAppClient(uid)
	if e = app.SetPassword(encrypt.RandomAlphanumeric(16)); e != nil {
		err = common.NewError(common.ERR_INTERNAL, e.Error())
		return
	}
	if e = app.SetDeviceId(values["devid"]); e != nil {
		err = common.NewError(common.ERR_INVALID_FORMAT, e.Error())
		return
	}
	if e = app.SetDeviceName(values["dname"]); e != nil {
		err = common.NewError(common.ERR_INVALID_FORMAT, e.Error())
		return
	}
	if e = app.SetPlatformStr(values["platform"]); e != nil {
		err = common.NewError(common.ERR_INVALID_FORMAT, e.Error())
		return
	}
	app.LastUpdate = time.Now()
	e = a.adb.NewPassword(app)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, e.Error())
		return
	}
	user, e := a.adb.GetUserByPhone(vcode.Cellphone())
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, e.Error())
		return
	}
	result["uid"] = uid
	result["password"] = app.Password()
	result["nickname"] = user.NickName()

	return
}
func (a *AccountHandler) getHKPDetail(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}

	requiredKeys := []string{"job"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}

	uids := make(map[common.UIDType]bool, 5)
	u, found := values["uids"]
	if found {
		uidstrs := strings.Split(u, ",")
		for _, ustr := range uidstrs {
			uid64, e := format.ParseUint64(ustr)
			if e != nil {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [uids=%v] error : %v", u, e.Error()))
				return err
			}
			uid := common.UIDType(uid64)
			uids[uid] = false
		}
	}
	p, found := values["phones"]
	if found {
		pstrs := strings.Split(p, ",")
		for _, pstr := range pstrs {
			uid, found := Phones[pstr]
			if found {
				uids[uid] = true
			}
		}
	}
	jobid, e := format.ParseUint(values["job"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("invalid job id [%v] : %v", values["job"], e.Error()))
		return
	}
	re_users := make([]map[string]interface{}, 0, 5)
	for uid, _ := range uids {
		user, found := Users[uid]
		if !found {
			err = common.NewError(common.ERR_USER_NOT_FOUND, fmt.Sprintf("user %v not found", uid))
			return err
		}
		jobs, found := HKPs[uid]
		if !found {
			err = common.NewError(common.ERR_ROLE_NOT_MATCH, fmt.Sprintf("user %v is not an HKP", uid))
			return err
		}
		job, found := jobs[jobid]
		if !found {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("cannot find job [%v] information", jobid))
			return err
		}
		result_u := make(map[string]interface{})
		result_u["uid"] = user.Id()
		result_u["role"] = common.ROLE_HKP
		result_u["nickname"] = user.NickName()
		result_u["birthday"] = user.Birthday().Format(format.TIME_LAYOUT_2)
		result_u["birthplace"] = user.Birthplace()
		result_u["idcardtype"] = user.IdCardType()
		masked, e := common.MaskIDCardNum(user.IdCardNum())
		if e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("invalid id card %v", user.IdCardNum()))
			return err
		}
		result_u["idcardnum"] = masked
		result_u["sex"] = user.Sex()
		result_u["firstname"] = user.FirstName()
		result_u["lastname"] = user.LastName()
		result_u["price"] = job.Job.Price
		result_u["begin_time"] = job.Job.BeginTime.Format(format.TIME_LAYOUT_1)
		result_u["career"] = int(time.Since(job.Job.BeginTime).Hours() / 24 / 30)
		result_u["desc"] = job.Job.Desc
		result_u["times"] = job.Rank.Times
		result_u["fail_times"] = job.Rank.FailTimes
		result_u["speed"] = job.Rank.Speed()
		result_u["quality"] = job.Rank.Quality()
		result_u["attitude"] = job.Rank.Attitude()
		re_users = append(re_users, result_u)
		if uids[user.Id()] {
			result_u["phone"] = user.Cellphone()
		}
		if user.HealthCardTimeout().After(time.Now()) {
			result_u["health"] = true
		} else {
			result_u["health"] = false
		}

	}
	result["users"] = re_users
	return
}

func (a *AccountHandler) addCollect(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	requiredKeys := []string{"provider", "role", "job"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	collect := model.NewCollection()
	if e = collect.SetCustomer(uid); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set customer error : %v", e.Error()))
		return
	}
	if e = collect.SetServiceProviderStr(values["provider"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set provider error : %v", e.Error()))
		return
	}
	if e = collect.SetRoleStr(values["role"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set role error : %v", e.Error()))
		return
	}
	if e = collect.SetJobStr(values["job"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set job error : %v", e.Error()))
		return
	}
	collect.CollectTime = time.Now()
	e = a.adb.AddCollection(collect)
	if e != nil {
		if strings.Contains(e.Error(), "Duplicate") {
			err = common.NewError(common.ERR_COLLECT_EXISTS, e.Error())
		} else {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddCollection error : %v", e.Error()))
		}
	}
	return
}

func (a *AccountHandler) delCollect(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	requiredKeys := []string{"provider"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	collect := model.NewCollection()
	if e = collect.SetCustomer(uid); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set customer error : %v", e.Error()))
		return
	}
	if e = collect.SetServiceProviderStr(values["provider"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set provider error : %v", e.Error()))
		return
	}
	role, found := values["role"]
	if found {
		if e = collect.SetRoleStr(role); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set role error : %v", e.Error()))
			return
		}
	}
	job, found := values["job"]
	if found {
		if e = collect.SetJobStr(job); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set job error : %v", e.Error()))
			return
		}
	}
	if e = a.adb.DelCollection(collect); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("DelCollection error : %v", e.Error()))
	}
	return
}
func (a *AccountHandler) listCollect(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	requiredKeys := []string{"pn", "rn"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	pn, e := format.ParseUint(values["pn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse pn error : %v", e.Error()))
		return err
	}
	rn, e := format.ParseUint(values["rn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse rn error : %v", e.Error()))
		return err
	}
	collect := model.NewCollection()
	if e = collect.SetCustomer(uid); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set customer error : %v", e.Error()))
		return
	}
	role, found := values["role"]
	if found {
		if e = collect.SetRoleStr(role); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set role error : %v", e.Error()))
			return
		}
	}
	job, found := values["job"]
	if found {
		if e = collect.SetJobStr(job); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set job error : %v", e.Error()))
			return
		}
	}
	prs, e := a.adb.ListCollection(collect, pn, rn)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("query database error : %v", e.Error()))
		return
	}
	users := make([]interface{}, 0, rn)
	for _, id := range prs {
		user, found := Users[id]
		if !found {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("user %v not found", id))
			return err
		}
		jobs, found := HKPs[id]
		if !found {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("user %v is not an HKP", uid))
			return err
		}
		job, found := jobs[collect.Job()]
		if !found {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("cannot find job [%v] information", collect.Job()))
			return err
		}
		result_u := make(map[string]interface{})
		result_u["uid"] = user.Id()
		result_u["nickname"] = user.NickName()
		result_u["birthday"] = user.Birthday().Format(format.TIME_LAYOUT_2)
		result_u["birthplace"] = user.Birthplace()
		result_u["idcardtype"] = user.IdCardType()
		masked, e := common.MaskIDCardNum(user.IdCardNum())
		if e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("invalid id card %v", user.IdCardNum()))
			return err
		}
		result_u["idcardnum"] = masked
		result_u["sex"] = user.Sex()
		result_u["firstname"] = user.FirstName()
		result_u["lastname"] = user.LastName()
		result_u["price"] = job.Job.Price
		result_u["jobversion"] = job.Job.Version
		result_u["begin_time"] = job.Job.BeginTime.Format(format.TIME_LAYOUT_1)
		result_u["career"] = int(time.Since(job.Job.BeginTime).Hours() / 24 / 30)
		result_u["desc"] = job.Job.Desc
		result_u["times"] = job.Rank.Times
		result_u["fail_times"] = job.Rank.FailTimes
		result_u["speed"] = job.Rank.Speed()
		result_u["quality"] = job.Rank.Quality()
		result_u["attitude"] = job.Rank.Attitude()
		result_u["rank"] = job.Rank.RankAll()
		result_u["rank_desc"] = job.Rank.RankAllDesc()
		result_u["phone"] = user.Cellphone()
		if user.HealthCardTimeout().After(time.Now()) {
			result_u["health"] = true
		} else {
			result_u["health"] = false
		}
		users = append(users, result_u)
	}
	result["users"] = users
	return
}
func (a *AccountHandler) listHistory(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	requiredKeys := []string{"role", "job", "pn", "rn"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	pn, e := format.ParseUint(values["pn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse pn error : %v", e.Error()))
		return err
	}
	rn, e := format.ParseUint(values["rn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse rn error : %v", e.Error()))
		return err
	}
	history := model.NewHistory()
	if e = history.SetCustomer(uid); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set customer error : %v", e.Error()))
		return
	}
	if e = history.SetRoleStr(values["role"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set role error : %v", e.Error()))
		return
	}
	if e = history.SetJobStr(values["job"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set job error : %v", e.Error()))
		return
	}
	prs, e := a.adb.ListHistory(history, pn, rn)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("query database error : %v", e.Error()))
		return
	}
	users := make([]interface{}, 0, rn)
	for _, item := range prs {
		fmt.Printf("history : provider %v\n", item.ServiceProvider())
		user, found := Users[item.ServiceProvider()]
		m := make(map[string]interface{})
		if found {
			m["uid"] = user.Id()
			m["birthday"] = user.Birthday().Format(format.TIME_LAYOUT_2)
			m["birthplace"] = user.Birthplace()
			m["idcardtype"] = user.IdCardType()
			masked, e := common.MaskIDCardNum(user.IdCardNum())
			if e != nil {
				err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("invalid id card %v", user.IdCardNum()))
				return err
			}
			m["idcardnum"] = masked
			m["sex"] = user.Sex()
			m["nickname"] = user.NickName()
			m["firstname"] = user.FirstName()
			m["lastname"] = user.LastName()
			m["lasttime"] = item.LastTime
			switch history.Role() {
			case common.ROLE_HKP:
				job, e := a.adb.GetHKPJobMaxVersion(item.ServiceProvider(), history.Job())
				if e != nil {
					err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("query database error : %v", e.Error()))
					return err
				}
				jobs, found := HKPs[user.Id()]
				if !found {
					err = common.NewError(common.ERR_ROLE_NOT_MATCH, fmt.Sprintf("user %v is not an HKP", uid))
					return err
				}
				jobWRank, found := jobs[job.JobId]
				if !found {
					err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("cannot find job [%v] information", job.JobId))
					return err
				}
				m["price"] = job.Price
				m["jobversion"] = job.Version
				m["begin_time"] = job.BeginTime.Format(format.TIME_LAYOUT_1)
				m["career"] = int(time.Since(job.BeginTime).Hours() / 24 / 30)
				m["desc"] = job.Desc
				m["times"] = jobWRank.Rank.Times
				m["fail_times"] = jobWRank.Rank.FailTimes
				m["speed"] = jobWRank.Rank.Speed()
				m["quality"] = jobWRank.Rank.Quality()
				m["attitude"] = jobWRank.Rank.Attitude()
				m["rank"] = jobWRank.Rank.RankAll()
				m["rank_desc"] = jobWRank.Rank.RankAllDesc()
				if user.HealthCardTimeout().After(time.Now()) {
					m["health"] = true
				} else {
					m["health"] = false
				}
			default:
				err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("unknown role %v", history.Role()))
				return
			}
		}
		users = append(users, m)
	}
	result["users"] = users
	return
}
func (a *AccountHandler) getBalance(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	b, e := a.adb.GetBalance(uid)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("query database error : %v", e.Error()))
		return
	}
	result["balance"] = b.Balance
	result["score"] = b.Score
	result["coupons"] = b.Coupons
	return
}

func (a *AccountHandler) addAddress(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	requiredKeys := []string{"community", "address"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	ads, e := a.adb.ListAddress(uid)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("query database error : %v", e.Error()))
		return
	}
	if len(ads) >= common.MAX_ADDRESS_NUM {
		err = common.NewError(common.ERR_MAX_ADDRESS_LIMIT, fmt.Sprintf("too many addresses. max address number [%v]", common.MAX_ADDRESS_NUM))
		return
	}
	ad := model.NewAddress(uid)
	ad.LastUse = time.Now()
	ad.Community, e = format.ParseUint(values["community"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [community] error : %v", e.Error()))
		return
	}
	_, found := Communities[ad.Community]
	if !found {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("invalid community id [%v]", ad.Community))
		return
	}
	if e = ad.SetAddr(values["address"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set address error : %v", e.Error()))
		return
	}
	if e = ad.SetStatus(1); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set status error : %v", e.Error()))
		return
	}
	if e = a.adb.CreateAddress(ad, nil); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("create address error : %v", e.Error()))
		return
	}
	result["aid"] = ad.AddrId
	return
}

func (a *AccountHandler) listAddress(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	ads, e := a.adb.ListAddress(uid)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("query database error : %v", e.Error()))
		return
	}
	list := make([]map[string]interface{}, 0, 10)
	for _, ad := range ads {
		tmp := make(map[string]interface{})
		tmp["id"] = ad.AddrId
		tmp["cid"] = ad.Community
		comm, found := Communities[ad.Community]
		if !found {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("fatal error : cannot find community of address [%v]", ad.AddrId))
			return
		}
		tmp["cname"] = comm.Name()
		tmp["caddress"] = comm.Address()
		tmp["address"] = ad.Addr()
		tmp["city_id"] = comm.City
		city, found := Cities[comm.City]
		if !found {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("fatal error : cannot find city of community [%v]", ad.Community))
			return
		}
		tmp["city_name"] = city.Name
		list = append(list, tmp)
	}
	result["addresses"] = list
	return
}

func (a *AccountHandler) updateAddress(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	requiredKeys := []string{"aid"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	ad := model.NewAddress(uid)
	aid, _ := values["aid"]
	ad.AddrId, e = format.ParseUint64(aid)
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [aid] error : %v", e.Error()))
		return
	}
	comm, found := values["community"]
	if found {
		ad.Community, e = format.ParseUint(comm)
		if e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [community] error : %v", e.Error()))
			return
		}
		_, found := Communities[ad.Community]
		if !found {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("invalid community id [%v]", ad.Community))
			return
		}
	}
	addr, found := values["address"]
	if found {
		if addr == "" {
			err = common.NewError(common.ERR_ADDRESS_EMPTY, "cannot set address empty")
			return
		}
		if e = ad.SetAddr(addr); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set addr error : %v", e.Error()))
			return
		}
	} else {
		if e = ad.SetAddr(""); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set addr error : %v", e.Error()))
			return
		}
	}
	if e = a.adb.UpdateAddress(ad); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("update address error : %v", e.Error()))
		return
	}
	return
}

func (a *AccountHandler) delAddress(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	requiredKeys := []string{"aid"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	aid, e := format.ParseUint64(values["aid"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [aid] error : %v", e.Error()))
		return
	}
	if e = a.adb.DelAddress(uid, aid); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("del address error : %v", e.Error()))
		return
	}
	return
}

func (a *AccountHandler) registerCustomer(info map[string]string) (uid common.UIDType, password string, aid uint64, err common.Error) {
	user := model.NewUser()
	if nickName, found := info["nname"]; found {
		if e := user.SetNickName(nickName); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set nname error : %v", e.Error()))
			return
		}
	}
	if lastName, found := info["lname"]; found {
		if e := user.SetLastName(lastName); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set lname error : %v", e.Error()))
			return
		}
	}
	if firstName, found := info["fname"]; found {
		if e := user.SetFirstName(firstName); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set fname error : %v", e.Error()))
			return
		}
	}
	if sex, found := info["sex"]; found {
		n, e := format.ParseUint8(sex)
		if e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [sex] error : %v", e.Error()))
			return 0, "", 0, err
		}
		if e = user.SetSex(uint8(n)); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set sex error : %v", e.Error()))
			return 0, "", 0, err
		}
	}
	if mail, found := info["mail"]; found {
		if e := user.SetMail(mail); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set mail error : %v", e.Error()))
			return
		}
	}
	if e := user.SetCellphone(info["phone"]); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set phone error : %v", e.Error()))
		return
	}
	invPhone, found := info["inviter"]
	var inviter common.UIDType = 0
	var e error
	if found {
		inviter, e = a.adb.GetUid(invPhone)
		if e != nil {
			if e == sql.ErrNoRows {
				err = common.NewError(common.ERR_CELLPHONE_NOT_EXISTS, "cellphone not found")
				return 0, "", 0, err
			} else {
				err = common.NewError(common.ERR_INTERNAL, e.Error())
				return 0, "", 0, err
			}
		}
	}
	if e = user.SetInviter(inviter); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set inviter error : %v", e.Error()))
		return 0, "", 0, err
	}

	tx, e := a.adb.db.Begin()
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, "mysql transaction begin error : "+e.Error())
		return 0, "", 0, err
	}

	uid, e = a.adb.CreateUser(user, tx)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("CreateUser error : %s", e.Error()))
		tx.Rollback()
		return
	}
	user.SetId(uid)

	//然后再用这个uid创建账户记录
	account := model.NewAccount(uid)
	if e = a.adb.CreateAccount(account, tx); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("CreateAccount error : %s", e.Error()))
		tx.Rollback()
		return
	}
	_, _, _, e = common.AddBSC(common.TRANS_REG, uid, 0, 0, float32(common.Variables.CRegBonus()), 0, "", tx)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddScore error : %s", e.Error()))
		tx.Rollback()
		return 0, "", 0, err
	}

	//接着创建地址记录
	community, found := info["community"]
	if found {
		address := model.NewAddress(uid)
		address.SetStatus(1)
		cid, e := format.ParseUint(community)
		if e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [community] error : %v", e.Error()))
			tx.Rollback()
			return
		}
		isValid, e := a.adb.IsValidCommunity(uint(cid))
		if e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("call IsValidCommunity error : %s", e.Error()))
			tx.Rollback()
			return
		}
		if !isValid {
			err = common.NewError(common.ERR_INVALID_PARAM, "[community] not found")
			tx.Rollback()
			return
		}
		address.Community = uint(cid)
		address.LastUse = time.Now()
		if e = address.SetAddr(info["address"]); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set address error : %v", e.Error()))
			tx.Rollback()
			return
		}
		if e = a.adb.CreateAddress(address, tx); e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("CreateAddress error : %s", e.Error()))
			tx.Rollback()
			return
		}
		aid = address.AddrId
	} else {
		aid = 0
	}

	//创建设备记录
	app := model.NewAppClient(uid)
	if e = app.SetPassword(encrypt.RandomAlphanumeric(16)); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set password error : %v", e.Error()))
		tx.Rollback()
		return
	}
	pid, e := format.ParseUint8(info["platform"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [platform] error : %v", e.Error()))
		tx.Rollback()
		return
	}
	if e = app.SetPlatform(uint8(pid)); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set platform error : %v", e.Error()))
		tx.Rollback()
		return
	}
	if e = app.SetDeviceName(info["dname"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set dname error : %v", e.Error()))
		tx.Rollback()
		return
	}
	if e = app.SetDeviceId(info["devid"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set devid error : %v", e.Error()))
		tx.Rollback()
		return
	}
	app.LastUpdate = time.Now()
	if e = a.adb.CreateAppClient(app, tx); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("CreateAppClient error : %s", e.Error()))
		tx.Rollback()
		return
	}

	//给邀请者奖励
	if inviter > 0 {
		_, _, _, e = common.AddBSC(common.TRANS_INVITE, inviter, 0, 0, float32(common.Variables.CInvBonus()), 0, "", tx)
		if e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddScore error : %s", e.Error()))
			tx.Rollback()
			return 0, "", 0, err
		}
	}
	txerr := tx.Commit()
	if txerr != nil {
		tx.Rollback()
		err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
		return 0, "", 0, err
	}

	if inviter > 0 {
		content := fmt.Sprintf("感谢您邀请用户%v注册生活圈，已奖励%v元到您账户。", user.NickName(), common.Variables.CInvBonus())
		common.SendSMS(invPhone, content)
		//给获得奖励的邀请者发推送消息
		extras := make(map[string]interface{})
		extras["type"] = common.NOTI_REWARD
		alias := make([]common.UIDType, 0, 1)
		alias = append(alias, uid)
		e = common.PushNotification(alias, nil, "邀请注册奖励", fmt.Sprintf("感谢您邀请用户%v注册生活圈，已奖励%v元到您账户！", common.Variables.CInvBonus()), extras)
		if e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Push Notification error : %s", e.Error()))
			return
		}

	}

	return uid, app.Password(), aid, err
}

func (a *AccountHandler) registerHKP(info map[string]string) (uid common.UIDType, password string, err common.Error) {
	requiredKeys := []string{"idcardtype", "idcardnum", "sex", "birthday", "birthplace", "fname", "lname"}
	suc, key := format.Contains(info, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	return 0, "", err
}
func (a *AccountHandler) variables(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	result["server_time"] = time.Now().Unix()
	result["hkp_service_start"] = common.Variables.HKPServiceStart()
	result["hkp_service_stop"] = common.Variables.HKPServiceStop()
	result["hkp_service_confirm_time"] = common.Variables.HKPServiceConfirmTime()
	result["android_version"] = common.Variables.AndroidVersion()
	result["iphone_version"] = common.Variables.IPhoneVersion()
	result["android_url"] = common.Variables.AndroidUrl()
	result["iphone_url"] = common.Variables.IPhoneUrl()
	result["android_force"] = common.Variables.AndroidForce()
	result["iphone_force"] = common.Variables.IPhoneForce()
	return
}
func (a *AccountHandler) updateInfo(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	info, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	user := model.NewUser()
	user.SetId(uid)
	if nickName, found := info["nname"]; found {
		if e := user.SetNickName(nickName); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set nname error : %v", e.Error()))
			return
		}
	}
	if lastName, found := info["lname"]; found {
		if e := user.SetLastName(lastName); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set lname error : %v", e.Error()))
			return
		}
	}
	if firstName, found := info["fname"]; found {
		if e := user.SetFirstName(firstName); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set fname error : %v", e.Error()))
			return
		}
	}
	if sex, found := info["sex"]; found {
		n, e := format.ParseUint8(sex)
		if e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [sex] error : %v", e.Error()))
			return err
		}
		if e = user.SetSex(uint8(n)); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set sex error : %v", e.Error()))
			return err
		}
	}
	if mail, found := info["mail"]; found {
		if e := user.SetMail(mail); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set mail error : %v", e.Error()))
			return
		}
	}
	if e = a.adb.UpdateUser(user); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("update User Info error : %v", e.Error()))
		return
	}
	return
}

func (a *AccountHandler) feedback(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	requiredKeys := []string{"content"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	if e = a.adb.Feedback(uid, values["content"]); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Insert Feedback error : %v", e.Error()))
		return
	}
	return
}

func (a *AccountHandler) listOptions(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	options, e := a.adb.ListOptions()
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("ListOptions error : %v", e.Error()))
		return
	}
	os := make([]interface{}, 0, 5)
	for _, option := range options {
		o := make(map[string]interface{})
		o["id"] = option.Id
		o["name"] = option.Name
		o["pic"] = option.Pic
		o["detail"] = option.Detail
		os = append(os, o)
	}
	result["options"] = os
	return
}

func (a *AccountHandler) vote(r *http.Request, uid common.UIDType, body string, result map[string]interface{}) (err common.Error) {
	values, e := format.ParseKV(string(body), "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	requiredKeys := []string{"ids"}
	suc, key := format.Contains(values, requiredKeys)
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	if t, found := a.voted[uid]; found {
		if t.After(time.Now()) {
			err = common.NewError(common.ERR_VOTED, fmt.Sprintf("Has voted"))
			return
		}
	}
	i, found := values["ids"]
	if found {
		idstrs := strings.Split(i, ",")
		for _, istr := range idstrs {
			_, e := format.ParseUint(istr)
			if e != nil {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [ids=%v] error : %v", i, e.Error()))
				return err
			}
		}
	}

	if e = a.adb.Vote(values["ids"]); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Vote error : %v", e.Error()))
		return
	}
	a.voted[uid] = time.Now().Add(1 * time.Hour)
	return
}
