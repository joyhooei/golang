package login

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"yf_pkg/encrypt"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/data_model/discovery"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/data_model/usercontrol"
)

var mdb *mysql.MysqlDB
var rdb *redis.RedisPool
var mlog *log.MLogger
var cache *redis.RedisPool

func Init(env *service.Env) {
	mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	mlog = env.Log
	InitCityArea()
}

func newPass() (result string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := r.Int63()
	return encrypt.MD5Sum(utils.ToString(code))
}

func PostVer(uid uint32, imei, imsi, mac string) (e error) { // ver,
	if _, e := mdb.Exec("update user_main set imei=?,imsi=?,mac=? where uid=?", imei, imsi, mac, uid); e != nil {
		return e
	}
	return
}

func PostModel(uid uint32, sysver, factory, model string) (e error) {
	if _, e := mdb.Exec("update user_main set sysver=?,factory=?,model=? where uid=?", sysver, factory, model, uid); e != nil {
		return e
	}
	return
}

func PostChannel(uid uint32, channel, channel_sid string) (e error) {
	if _, e := mdb.Exec("update user_main set channel_uid=?,channel_sid=? where uid=?", channel, channel_sid, uid); e != nil {
		return e
	}
	return
}

func RegMain(fromtype int, thirdID string, username, spass string, gender int, ver, imei, imsi, mac, sysver, factory, model, channel, channel_sid string, reg_ip string, phone string, devid uint32) (uid uint32, sid string, e error) {
	var fieldname string
	switch fromtype {
	case 1:
		fieldname = "wx_username"
	case 2:
		fieldname = "qq_username"
	case 3:
		fieldname = "wb_username"
	}
	switch fromtype {
	case 1, 2, 3:
		var count int
		if err := mdb.QueryRow("select count(*) from user_main where "+fieldname+"=?", thirdID).Scan(&count); err != nil {
			return 0, "", err
		}
		if count > 0 {
			return 0, "", errors.New("绑定ID已存在")
		}
		r, err := mdb.Exec("insert into user_main ("+fieldname+",gender,reg_time,ver, imei, imsi, mac, sysver, factory, model, channel_uid, channel_sid,reg_ip)values(?,?,?,?,?,?,?,?,?,?,?,?,?)", thirdID, gender, utils.Now, ver, imei, imsi, mac, sysver, factory, model, channel, channel_sid, reg_ip)
		if err != nil {
			return 0, "", err
		}
		iid, err := r.LastInsertId()
		uid = uint32(iid)
	case 4:
		r, err := mdb.Exec("insert into user_main (gender,reg_time,ver, imei, imsi, mac, sysver, factory, model, channel_uid, channel_sid,reg_ip)values(?,?,?,?,?,?,?,?,?,?,?,?)", gender, utils.Now, ver, imei, imsi, mac, sysver, factory, model, channel, channel_sid, reg_ip)
		if err != nil {
			return 0, "", err
		}
		iid, err := r.LastInsertId()
		uid = uint32(iid)
		if err != nil {
			return 0, "", err
		}
	case 5:
		var count int
		if err := mdb.QueryRow("select count(*) from user_main where username=?", username).Scan(&count); err != nil {
			return 0, "", err
		}
		if count > 0 {
			return 0, "", errors.New("用户名已存在")
		}
		r, err := mdb.Exec("insert into user_main (username,gender,reg_time,ver, imei, imsi, mac, sysver, factory, model, channel_uid, channel_sid,reg_ip)values(?,?,?,?,?,?,?,?,?,?,?,?,?)", username, gender, utils.Now, ver, imei, imsi, mac, sysver, factory, model, channel, channel_sid, reg_ip)
		if err != nil {
			return 0, "", err
		}
		iid, err := r.LastInsertId()
		if err != nil {
			return 0, "", err
		}
		uid = uint32(iid)
		password := encrypt.MD5Sum(utils.ToString(uid) + spass)
		_, err = mdb.Exec("update user_main set password=? where uid=?", password, uid)
		if err != nil {
			return 0, "", err
		}
	case 6, 7:
		r, err := mdb.Exec("insert into user_main (phone,gender,reg_time,ver, imei, imsi, mac, sysver, factory, model, channel_uid, channel_sid,reg_ip,phonestat)values(?,?,?,?,?,?,?,?,?,?,?,?,?,?)", phone, gender, utils.Now, ver, imei, imsi, mac, sysver, factory, model, channel, channel_sid, reg_ip, 1)
		if err != nil {
			return 0, "", err
		}
		iid, err := r.LastInsertId()
		if err != nil {
			return 0, "", err
		}
		uid = uint32(iid)
		password := encrypt.MD5Sum(utils.ToString(uid) + spass)
		_, err = mdb.Exec("update user_main set password=? where uid=?", password, uid)
		if err != nil {
			return 0, "", err
		}
	default:
		return 0, "", errors.New("fromtype 类型不对")
	}
	sid, _ = ChangeSid(uid)
	_, err := mdb.Exec("insert into user_detail (uid)values(?)", uid)
	if err != nil {
		return 0, "", err
	}
	stat.Append(uid, stat.ACTION_REG, map[string]interface{}{})
	stat.SimpleAppendDev(devid, stat.DEV_ACTION_REG, map[string]interface{}{})
	// uidmoney(uid)
	return
}

func uidmoney(uid uint32) (e error) {
	mdb.Exec("update user_main set goldcoin=10000 where uid=?", uid)
	return
}

func RegDetail(uid uint32, nickname, avatar string, birthday time.Time, province, city string) (e error) {
	_, err := mdb.Exec("insert into user_detail (uid,nickname,birthday,avatar,infocomplete,province, city)values(?,?,?,?,4,?,?)", uid, nickname, birthday, avatar, province, city)
	if err != nil {
		return err
	}
	_, err = mdb.Exec("insert into user_protect (uid)values(?)", uid)
	if err != nil {
		return err
	}
	// usercontrol.AddPic(uid, "", avatar, "", "", 0)
	return
}

func BindMumu(uid uint32, username, spass string) (e error) {
	var count int
	if err := mdb.QueryRow("select count(*) from user_main where username=? or phone=?", username, username).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return errors.New("此账号已存在")
	}
	password := encrypt.MD5Sum(utils.ToString(uid) + spass)
	_, err := mdb.Exec("update user_main set username=?,password=? where uid=?", username, password, uid)
	if err != nil {
		return err
	}
	// sid, _ = ChangeSid(uid)
	return
}

func CheckThirdID(openid string, fieldname string) (ifnew bool, uid uint32, sid string, guidecomplete int, e error) {
	var stat int
	var gender int
	err := mdb.QueryRow("select uid,sid,stat,gender from user_main where "+fieldname+"=? ", openid).Scan(&uid, &sid, &stat, &gender)
	if err != nil {
		if err == sql.ErrNoRows {
			ifnew = true
			return
		} else {
			e = err
			return
		}
	} else {
		ifnew = false
		if stat == 5 {
			e = errors.New("此账号因违规已被封禁。")
			return
		}
	}
	sid, _ = ChangeSid(uid)
	if gender == 0 {
		guidecomplete = 0
	} else {
		guidecomplete = 1
	}
	return
}

func CheckUser(username string) (exists int, e error) {
	var count int
	err := mdb.QueryRow("select count(*) from user_main where username=?", username).Scan(&count)
	if err != nil {
		return 0, err
	} else {
		if count == 0 {
			exists = 1
		} else {
			exists = 2 //用户名不存在
		}
	}
	return
}

func SetPhonePwd(phone string, newpass string) (uid uint32, sid string, e error) {
	// var uid uint32
	var pwd string
	if err := mdb.QueryRow("select uid,password from user_main where phone=? and stat<>5 ", phone).Scan(&uid, &pwd); err != nil {
		return 0, "", err
	}
	password := encrypt.MD5Sum(utils.ToString(uid) + newpass)
	sid = newPass()
	// if pwd == password {

	// } else {
	r, err := mdb.Exec("update user_main set password=?,sid=? where phone=? and uid=?", password, sid, phone, uid)
	if err != nil {
		return 0, "", err
	}
	if i, err := r.RowsAffected(); err != nil {
		return 0, "", err
	} else {
		if i == 0 {
			return 0, "", errors.New("修改失败")
		}
	}
	user_overview.SetUserPassCache(uid, sid)
	// }
	return
}

func BandPhone(uid uint32, phone string, btype int) (e error) {
	if btype == 1 {
		var count int
		if err := mdb.QueryRow("select count(*) from user_main where phone=? or username=?", phone, phone).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return errors.New("填入的手机号已存在")
		}
	} else {
		var count int
		if err := mdb.QueryRow("select count(*) from user_main where phone=? and uid=?", phone, uid).Scan(&count); err != nil {
			return err
		}
		if count <= 0 {
			return errors.New("解除绑定失败,手机号不同")
		}
	}
	switch btype {
	case 1: //绑定
		_, err := mdb.Exec("update user_main set phone=?,phonestat=1 where uid=?", phone, uid)
		if err != nil {
			return err
		}
	case 2: //解除绑定
		_, err := mdb.Exec("update user_main set phone='',phonestat=0 where uid=?", uid)
		if err != nil {
			return err
		}
	}
	return
}

func BindPhone(uid uint32, phone string, newpass string) (e error) {

	var count int
	if err := mdb.QueryRow("select count(*) from user_main where phone=? or username=?", phone, phone).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return errors.New("填入的手机号已存在")
	}
	password := encrypt.MD5Sum(utils.ToString(uid) + newpass)
	_, err := mdb.Exec("update user_main set phone=?,phonestat=1,password=? where uid=?", phone, password, uid)
	if err != nil {
		return err
	}
	_, e = ChangeSid(uid)
	if e != nil {
		return e
	}
	return
}

func ChangePhone(uid uint32, oldphone, newphone, newpass string) (e error) {
	password := encrypt.MD5Sum(utils.ToString(uid) + newpass)
	r, err := mdb.Exec("update user_main set phone=?,password=? where  stat<>5 and uid=? and  phone=?", newphone, password, uid, oldphone)
	if err != nil {
		return err
	}
	if i, err := r.RowsAffected(); err != nil {
		return err
	} else {
		if i == 0 {
			return errors.New("修改失败")
		}
	}
	_, e = ChangeSid(uid)
	if e != nil {
		return e
	}
	return
}

func ChangePwd(uid uint32, upass string, oldpass string) (sid string, e error) {
	password := encrypt.MD5Sum(utils.ToString(uid) + upass)
	oldpassword := encrypt.MD5Sum(utils.ToString(uid) + oldpass)
	r, err := mdb.Exec("update user_main set password=? where  stat<>5 and uid=? and password=?", password, uid, oldpassword)
	if err != nil {
		return "", err
	}
	if i, err := r.RowsAffected(); err != nil {
		return "", err
	} else {
		if i == 0 {
			return "", errors.New("修改失败，密码不对")
		}
	}
	sid, err = ChangeSid(uid)
	if err != nil {
		return "", err
	}
	return
}

func ChangeSid(uid uint32) (sid string, e error) {
	sid = newPass()
	_, err := mdb.Exec("update user_main set sid=? where uid=?", sid, uid)
	if err != nil {
		return "", err
	}
	user_overview.SetUserPassCache(uid, sid)
	return
}

func UserLogin(username string, password string) (uid uint32, sid string, guidecomplete int, e error) {
	var gender int
	var upass string
	if len(username) < 1 {
		return 0, "", 0, errors.New("用户名不存在")
	}
	err := mdb.QueryRow("select uid,password,sid,gender from user_main where stat<>5 and (username=? or phone=?)", username, username).Scan(&uid, &upass, &sid, &gender)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", 0, errors.New("用户名不存在")
		}
		return 0, "", 0, err
	}
	spass := encrypt.MD5Sum(utils.ToString(uid) + password)
	if spass != upass {
		return 0, "", 0, errors.New("密码错误")
	}
	if uid >= 5000000 {
		sid, err = ChangeSid(uid)
	}
	if err != nil {
		return 0, "", 0, err
	}
	if gender == 0 {
		guidecomplete = 0
	} else {
		guidecomplete = 1
	}
	return
}

func UserOnTop(uid uint32, state int) (e error) {
	_, e = mdb.Exec("update user_online set ontop=? where uid=?", state, uid)
	if state == 1 {
		if v, e := cache.Incr(redis_db.CACHE_ONTOP, uid); e == nil {
			if v <= 2 {
				cache.Expire(redis_db.CACHE_ONTOP, int(utils.DurationTo(1, 0, 0, 0).Seconds()), uid)
			}
			if v >= 3 && v < 5 {
				stat.Append(uid, stat.ACTION_ACTIVE, nil)
			}
		}
	}
	return
}

func MyPhoneUser(uid uint32, phone string) (result bool, e error) {
	var sphone string
	e = mdb.QueryRow("select phone from user_main where uid=?", uid).Scan(&sphone)
	if e != nil {
		if e == sql.ErrNoRows {
			return false, nil
		}
		return false, e
	}
	return phone == sphone, nil
}

func GetPhone(uid uint32) (phone string, e error) {
	e = mdb.QueryRow("select phone from user_main where uid=?", uid).Scan(&phone)
	return
}

func PhoneInDb(phone string) (result bool, e error) {
	var count int
	// fmt.Println("bind begin")
	e = mdb.QueryRow("select count(*) from user_main where phone=?", phone).Scan(&count)

	if e != nil {
		fmt.Println("bind err " + e.Error())
		return
	}
	return count > 0, nil
}

func getCityArea(province, city string) (x, y float64, e error) {
	e = mdb.QueryRow("select x,y from user_province_area where city=?", city).Scan(&x, &y)
	if e == nil {
		return
	}
	e = mdb.QueryRow("select x,y from user_province_area where city=?", city+"市").Scan(&x, &y)
	if e == nil {
		return
	}
	e = mdb.QueryRow("select x,y from user_province_area where province=?", province).Scan(&x, &y)
	if e == nil {
		return
	}
	e = mdb.QueryRow("select x,y from user_province_area where province=?", province+"省").Scan(&x, &y)
	if e == nil {
		return
	}
	return x, y, e
}

//
func GetJQUser(admin_uid, index uint32) (list []map[string]interface{}, e error) {
	rows, err := mdb.Query("select user_main.uid,sid,avatar,nickname,province,city,x,y,localtag,gender,username from manager_users,user_main left join user_detail on user_detail.uid=user_main.uid where manager_users.uid>? and manager_users.admin_uid=? and  manager_users.uid=user_main.uid order by manager_users.uid limit 30", index, admin_uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// var x1, y1 float64go
	// x1, y1, _ = getCityArea("北京市", "北京市")
	list = make([]map[string]interface{}, 0, 0)
	for rows.Next() {
		var uid uint32
		var x, y float64
		var sid, avatar, nickname, province, city, localtag, username string
		var gender int
		if err := rows.Scan(&uid, &sid, &avatar, &nickname, &province, &city, &x, &y, &localtag, &gender, &username); err != nil {
			return nil, err
		}
		// if sid == "" {
		// 	sid, _ = CheangeSid(uid)
		// }
		item := make(map[string]interface{})
		item["uid"] = uid
		item["password"] = sid
		item["avatar"] = avatar
		item["nickname"] = nickname
		item["province"] = province
		item["city"] = city
		item["localtag"] = localtag
		item["gender"] = gender
		item["username"] = username
		if x == 0 {
			x, y = GetArea(province, city)
		}
		// if x == 0 {
		// 	x = x1
		// 	y = y1
		// }
		item["x"] = x
		item["y"] = y
		list = append(list, item)
	}
	return list, nil
}

func AddJQUid(admin_id uint32, uid uint32) (e error) {
	_, e = mdb.Exec("insert into manager_users (admin_uid,uid)values(?,?)", admin_id, uid)
	return e
}

func AdminLogin(username string, password string) (admin_id uint32, e error) {
	// var uid uint32
	var upass string
	e = mdb.QueryRow("select password,uid from manager where username=?", username).Scan(&upass, &admin_id)
	if e != nil {
		return
	}
	if upass != password {
		return 0, errors.New("密码错误")
	}
	return admin_id, nil
}

func CheckRelogin(uid uint32) (result bool, e error) {
	result = false
	ov, err := user_overview.IsOnline(uid)
	if err != nil {
		return false, nil
	}
	if ov2, ok := ov[uid]; ok {
		if ov2 {
			result = true
		}
	}
	if !result {
		return
	}
	return
}

func SetUidArea(uid uint32, x, y float64) (e error) {
	_, e = mdb.Exec("update user_detail set x=?,y=? where uid=?", x, y, uid)
	return
}

func BanUser(uid uint32) (e error) {
	_, e = mdb.Exec("update user_main set stat=5 where uid=?", uid)
	if e != nil {
		return e
	}
	e = user_overview.DelUserPassCache(uid)
	if e != nil {
		return e
	}
	e = user_overview.ClearUserObjects(uid)
	return
}

func UnBanUser(uid uint32) (e error) {
	_, e = mdb.Exec("update user_main set stat=0 where uid=?", uid)
	return
}

func CheckPhoneModel(model, factory string) int {
	models := []string{"MI NOTE", "MI 4W", "MI 3W&MI 3C", "HM 1STD HM2014501", "HM 1SC HM 1SW", "MI PAD", "HM NOTE 1LTETD HM NOTE 1LTEW", "HM NOTE 1LTETD", "MI 2S", "MI 2SC", "MI 2A", "2013022", "MI 3", "MI-ONE Plus", "MI 1S M1 1SC", "MI 1S M1 1SC", "MI2"}
	if factory == "Xiaomi" {
		return 2
	}
	for _, v := range models {
		if strings.Contains(model, v) {
			return 2
		}
	}
	if factory == "APPLE" {
		return 1
	}
	return 3
}

func UpdateUidArea(uid uint32, lat, lng float64, ip string, gender int) {
	// fmt.Println(fmt.Sprintf("updarea %v,%v,%v,%v", uid, lat, lng, ip))
	var province, city string
	if (lat != 0) && (lng != 0) {
		city, province, _ = general.City(lat, lng)
	}

	if province == "" {
		province, city, _ = general.QueryIpInfo(ip)
	}
	mlog.AppendInfo(fmt.Sprintf("UpdateUidArea Complete %v,%v,%v", uid, province, city))
	// fmt.Println(fmt.Sprintf("updarea complete %v,%v", province, city))
	mdb.Exec("update user_detail set province=?,city=?,infocomplete= 4 where uid=?", province, city, uid)
	// ,province,city
	discovery.UpdateDiscovery(uid, "city", city)
	usercontrol.NanReg(uid, province, city, gender)
	user_overview.ClearUserObjects(uid)

	return
}

func UpdateUidGps(uid uint32) {
	var province, city string
	lat, lng, e := general.UserLocation(uid)
	if e != nil {
		return
	}
	if general.IsValidLocation(lat, lng) {
		city, province, _ = general.City(lat, lng)
	}
	mlog.AppendInfo(fmt.Sprintf("UpdateUidGps Complete %v,%v,%v", uid, province, city))
	if city != "" {
		mdb.Exec("update user_detail set province=?,city=? where uid=?", province, city, uid)
		discovery.UpdateDiscovery(uid, "city", city)
	}

	return
}

func CheckImei(imei string, imsi string, mac string) (canreg bool, e error) {
	var count int
	if (imei == "") && (imsi == "") && (mac == "" || mac == "00:00:00:00:00:00") {
		return true, nil
	}
	// fmt.Println(fmt.Sprintf("Imei %v", imei))
	e = mdb.QueryRow("select count(*) from user_main where imei=? and imsi=? and mac=? and stat=5", imei, imsi, mac).Scan(&count)
	if e != nil {
		return false, e
	}
	canreg = count <= 0
	// fmt.Println(fmt.Sprintf("Imei count %v,canreg %v", count, canreg))
	return
}

func IpCanReg(ip string) (canreg bool, e error) {
	con := cache.GetWriteConnection(redis_db.CACHE_REGIP)
	defer con.Close()
	icpount, err := redis.Int(con.Do("GET", ip))
	switch err {
	case nil:
		canreg = icpount > 0
		return canreg, nil
	case redis.ErrNil:
		t := utils.Now
		tdu := 3600*24 - t.Hour()*3600 + t.Minute()*60 + t.Second()
		_, err2 := con.Do("SETEX", ip, tdu, 3)
		return true, err2
	default:
		return true, err
	}
	return
}

func IpRegDec(ip string) {
	con := cache.GetWriteConnection(redis_db.CACHE_REGIP)
	defer con.Close()
	con.Do("DECR", ip)
}

func SubstrByByte(str string, length int) string {
	begin := 0
	rs := []rune(str)
	lth := len(rs)

	// 简单的越界判断
	if begin < 0 {
		begin = 0
	}
	if begin >= lth {
		begin = lth
	}
	end := begin + length
	if end > lth {
		end = lth
	}
	return string(rs[begin:end])
}

func ClearIpReg(ip string) (e error) {
	con := cache.GetWriteConnection(redis_db.CACHE_REGIP)
	defer con.Close()
	con.Do("DEL", ip)
	return
}
