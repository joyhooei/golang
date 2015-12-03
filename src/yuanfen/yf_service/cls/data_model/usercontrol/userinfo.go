package usercontrol

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"yf_pkg/cachedb"
	"yf_pkg/format"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/coin"
	"yuanfen/yf_service/cls/data_model/discovery"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/relation"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/message"
	"yuanfen/yf_service/cls/notify"
	"yuanfen/yf_service/cls/unread"
)

var mdb *mysql.MysqlDB
var cache *redis.RedisPool
var cachedb2 *cachedb.CacheDB
var sdb *mysql.MysqlDB
var rdb *redis.RedisPool
var msgdb *mysql.MysqlDB
var statdb *mysql.MysqlDB
var mainlog *log.MLogger
var mode string

// var log1 *log.Logger

func Init(env *service.Env) {
	cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	cachedb2 = env.ModuleEnv.(*cls.CustomEnv).CacheDB
	sdb = env.ModuleEnv.(*cls.CustomEnv).SortDB
	rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	msgdb = env.ModuleEnv.(*cls.CustomEnv).MsgDB
	statdb = env.ModuleEnv.(*cls.CustomEnv).StatDB
	mainlog = env.Log
	mode = env.ModuleEnv.(*cls.CustomEnv).Mode
	InitRandomNick()
	RegUserUnread()
	InitAllot()
	InitQuestion()
	InitSchool()
	message.RegisterNotification(message.OFFLINE, userOffline)
	message.RegisterNotification(message.ONLINE, userOnline)
	message.RegisterNotification(message.CREATETOPIC, createTopic)
	// log1 = env.Log
}

func StringAge(birthday string) int {
	d, err := utils.ToTime(birthday)
	if err != nil {
		return 0
	}
	return utils.Now.Year() - d.Year()
}

// 获取用户详细信息
func GetUserInfo(uid uint32) (map[string]string, error) {
	dr, err := mdb.Query("select * from user_main a LEFT JOIN user_detail b on a.uid=b.uid where a.uid=?", uid) //req.Uid
	if err != nil {
		return nil, err
	}
	defer dr.Close()
	sqlr, err := utils.ParseSqlResult(dr)
	if err != nil {
		return nil, err
	}
	if len(sqlr) <= 0 {
		return nil, errors.New("没找到用户 ")
	}
	var rmap = sqlr[0]
	return rmap, nil
}

// 获取其他用户详细信息
func GetOtherInfo(fromid uint32, uid uint32) (result map[string]string, e error) {
	result, e = GetUserInfo(uid)
	if e != nil {
		return result, e
	}
	return
}

//设置用户主要信息
func SetUserMainInfo(uid uint32, info map[string]interface{}) (e error) {
	keys := make([]string, 0, 0)
	params := make([]interface{}, 0, 0)
	for k, v := range info {
		keys = append(keys, k)
		params = append(params, v)
	}

	sql := "update user_main set " + strings.Join(keys, "=?,")
	sql = sql + "=? where uid=?"

	_, err := mdb.Exec(sql, params, uid) //req.Uid
	if err != nil {

		return err
	}
	return
}

func SetUserGuide(uid uint32, gender int, age int, nickname, job, trade string) (e error) {

	birthday := utils.AgeToBirthday(age)

	if _, err := mdb.Exec("update user_detail set birthday=?,nickname=?,job=?,trade=? where uid=?", birthday, nickname, job, trade, uid); err != nil {
		return err
	}
	if _, err := mdb.Exec("update user_main set gender=? where uid=?", gender, uid); err != nil {
		return err
	}
	if gender != common.GENDER_BOTH {
		stat.Append(uid, stat.ACTION_FINISH_GUIDE, nil)
	}
	return
}

//设置用户详细信息
func SetUserDetailInfo(uid uint32, info map[string]interface{}) (e error) {

	keys := make([]string, 0, 0)
	params := make([]interface{}, 0, 0)
	for k, v := range info {
		keys = append(keys, "`"+k+"`")
		params = append(params, v)
	}
	params = append(params, uid)
	sql := "update user_detail set " + strings.Join(keys, "=?,")
	sql = sql + "=? where uid=?"
	// fmt.Println("sql " + sql)
	_, err := mdb.Exec(sql, params...) //req.Uid
	if err != nil {
		return err
	}
	updateDetailCount(uid)
	return
}

func SetUserProtect(uid uint32, info map[string]interface{}) (e error) {

	keys := make([]string, 0, 0)
	params := make([]interface{}, 0, 0)
	for k, v := range info {
		keys = append(keys, "`"+k+"`")
		params = append(params, v)
	}
	params = append(params, uid)
	sql := "update user_protect set " + strings.Join(keys, "=?,")
	sql = sql + "=? where uid=?"
	mdb.Exec("insert into user_protect (uid)values(?)", uid)
	_, err := mdb.Exec(sql, params...) //req.Uid
	if err != nil {
		return err
	}
	return
}

func GetUserProtect(uid uint32) (info map[string]interface{}, e error) {
	mdb.Exec("insert into user_protect (uid)values(?)", uid)
	var canfind, chatremind, stranger, praise, commit, msgnotring, msgnotshake, nightring int
	e = mdb.QueryRow("select canfind,chatremind,stranger,praise,commit,msgnotring,msgnotshake,nightring from user_protect where uid=?", uid).Scan(&canfind, &chatremind, &stranger, &praise, &commit, &msgnotring, &msgnotshake, &nightring)
	if e != nil {
		return
	}
	info = make(map[string]interface{})
	info["canfind"] = canfind
	info["chatremind"] = chatremind
	info["stranger"] = stranger
	info["praise"] = praise
	info["commit"] = commit
	info["msgnotring"] = msgnotring
	info["msgnotshake"] = msgnotshake
	info["nightring"] = nightring
	return
}

func updateDetailCount(uid uint32) (e error) {
	var aboutme, height, star, birthdaystat, job, tag, interest, require, looking, contact string

	if err := mdb.QueryRowFromMain("select aboutme,height,star,job,tag,interest,`require`,looking,contact,birthdaystat from user_main,user_detail where user_main.uid=? and user_detail.uid=user_main.uid", uid).Scan(&aboutme, &height, &star, &job, &tag, &interest, &require, &looking, &contact, &birthdaystat); err != nil {
		return err
	}
	count := 0
	if height != "0" {
		count++
	}
	if star != "0" {
		count++
	}
	if birthdaystat != "0" {
		count++
	}

	if aboutme != "" {
		count++
	}
	if job != "" {
		count++
	}
	if tag != "" {
		count++
	}
	if interest != "" {
		count++
	}
	if require != "" {
		count++
	}

	if looking != "" {
		count++
	}
	if contact != "" {
		count++
	}

	var photocount int
	if err := mdb.QueryRowFromMain("select count(*) from user_photo_album where uid=?", uid).Scan(&photocount); err != nil {
		return err
	}
	if photocount > 3 {
		photocount = 3
	}
	count = count + photocount + 4
	_, e = mdb.Exec("update user_detail set infocomplete=? where uid=?", count, uid)
	// fmt.Println(fmt.Sprintf("getDetailCount count %v", count))
	// vmap := map[string]string{aboutme',height,star,birthdaystat,job,tag,interest,require,looking,contact}
	return
}

var sqlSetLocalTag string = "insert into user_tag(uid,content,gender,tm,timeout,`range`,min_age,max_age,min_height,max_height,star,income,certify_phone,certify_video,certify_idcard,lat,lng)values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) on duplicate key update content=?,gender=?,tm=?,timeout=?,`range`=?,min_age=?,max_age=?,min_height=?,max_height=?,star=?,income=?,certify_phone=?,certify_video=?,certify_idcard=?,lat=?,lng=?"

//设置用户本地标签
func SetUserLocaltag(uid uint32, lt user_overview.Localtag) (e error) {
	lat, lng, e := general.UserLocation(uid)
	if e != nil {
		return e
	}
	_, e = mdb.Exec(sqlSetLocalTag, uid, lt.Content, lt.Req.Gender, lt.Tm, lt.Timeout, lt.Req.Range, lt.Req.MinAge, lt.Req.MaxAge, lt.Req.MinHeight, lt.Req.MaxHeight, lt.Req.Star, lt.Req.Income, lt.Req.CertifyPhone, lt.Req.CertifyVideo, lt.Req.CertifyIDcard, lat, lng, lt.Content, lt.Req.Gender, lt.Tm, lt.Timeout, lt.Req.Range, lt.Req.MinAge, lt.Req.MaxAge, lt.Req.MinHeight, lt.Req.MaxHeight, lt.Req.Star, lt.Req.Income, lt.Req.CertifyPhone, lt.Req.CertifyVideo, lt.Req.CertifyIDcard, lat, lng)
	if e != nil {
		return e
	}
	//stat.Append(uid, stat.ACTION_SET_LOCALTAG, nil)
	con := rdb.GetWriteConnection(redis_db.REDIS_LOCALTAG_VIEWERS)
	defer con.Close()
	con.Do("DEL", uid)
	_, e = con.Do("ZADD", uid, utils.Now.Unix()-1, uid)
	if e != nil {
		return e
	}
	_, e = con.Do("EXPIREAT", uid, lt.Timeout.Unix())
	if e = unread.UpdateReadTime(uid, common.UNREAD_LOCALTAG_VIEWER); e != nil {
		return e
	}
	user_overview.ClearUserObjects(uid)
	return
}

var sqlCloseLocalTag string = "delete from user_tag where uid=?"

// 获取用户图片列表
func GetUserPicture(uid uint32, cur, ps int) ([]map[string]string, int, error) {
	dr, err := mdb.Query("select * from user_photo_album where uid=? order by create_time LIMIT ?", uid, ps) //req.Uid
	if err != nil {
		return nil, 0, err
	}
	defer dr.Close()
	sqlr, err := utils.ParseSqlResult(dr)
	if err != nil {
		return nil, 0, err
	}
	return sqlr, len(sqlr), nil
}

//获取用户地址列表
func GetUserAddr(uid uint32) ([]map[string]string, error) {
	dr, err := mdb.Query("select * from user_address where uid=?", uid) //req.Uid
	if err != nil {
		return nil, err
	}
	defer dr.Close()
	sqlr, err := utils.ParseSqlResult(dr)
	if err != nil {
		return nil, err
	}
	return sqlr, nil
}

//获取用户礼物列表
func GetUserGift(touid uint32) ([]map[string]string, error) {
	dr, err := mdb.Query("select * from user_address where uid=?", touid) //req.Uid
	if err != nil {
		return nil, err
	}
	defer dr.Close()
	sqlr, err := utils.ParseSqlResult(dr)
	if err != nil {
		return nil, err
	}
	return sqlr, nil
}

// 获取送给我的送礼记录  分页
func GetGiftList(touid uint32, cur, ps int) ([]map[string]string, int, error) {
	sql := "select uid,pursue_uid   from  pursue  where  pursue_uid =  ?  " + utils.BuildLimit(cur, ps)
	p_sql := "select count(*) as count   from  pursue  where  pursue_uid =  ? "

	rows, err := mdb.Query(sql, touid)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var count int
	err3 := mdb.QueryRow(p_sql, touid).Scan(&count)
	if err3 != nil {
		count = 0
	}
	list, err := utils.ParseSqlResult(rows)
	if err != nil {
		return list, 0, nil
	}
	return list, count, nil
}

//如果用户没有头像则 把照片作为用户头像 并清缓存
func UpdateAvatar(uid uint32, pic string) (e error) {
	//	r, e := mdb.Exec("update user_detail set avatar=?,avatarlevel=-2 where uid=? and avatar=''", pic, uid)
	r, e := mdb.Exec("update user_detail set avatar=? where uid=? and avatar=''", pic, uid)
	if e != nil {
		return
	}
	ir, e := r.RowsAffected()
	if e != nil {
		return
	}
	if ir > 0 {
		user_overview.ClearUserObjects(uid)
		UpdateNickChange(uid)
	}
	return
}

//设置用户头像
func SetAvatar(uid uint32, pic string) (e error) {
	var count int
	if e = mdb.QueryRow("select count(*) from user_photo_album where uid=? and pic=?", uid, pic).Scan(&count); e != nil {
		return
	}
	if count <= 0 {
		_, err := mdb.Exec("insert into user_photo_album (uid,pic,create_time,`type`)values(?,?,?,?)", uid, pic, utils.Now, 0)
		if err != nil {
			return err
		}
	}
	//if _, e = mdb.Exec("update user_detail set avatar=?,avatarlevel=-2 where uid=?", pic, uid); e != nil {
	if _, e = mdb.Exec("update user_detail set avatar=? where uid=?", pic, uid); e != nil {
		return
	}
	user_overview.ClearUserObjects(uid)
	updateUserVideoCertify(uid)
	return
}

func AddPic(uid uint32, albumname string, pic string, picsmall string, picdesc string, tp int) (int, error) {
	sql := "insert into user_photo_album (uid,albumname,pic,picsmall,picdesc,create_time,`type`)values(?,?,?,?,?,?,?)"

	sqlr, err := mdb.Exec(sql, uid, albumname, pic, picsmall, picdesc, utils.Now, tp) //req.Uid
	if err != nil {
		return 0, err
	}
	if i, err := sqlr.LastInsertId(); err == nil {
		UpdateAvatar(uid, pic)
		updateDetailCount(uid)
		return int(i), nil
	} else {
		return 0, errors.New("LastInsertId error")
	}

}

func AddPics(uid uint32, tp int, pics ...string) (ids []uint32, e error) {
	ids = make([]uint32, 0, 0)
	for i, v := range pics {
		sql := "insert into user_photo_album (uid,pic,create_time,`type`)values(?,?,?,?)"
		sqlr, err := mdb.Exec(sql, uid, v, utils.Now, tp) //req.Uid
		if err != nil {
			return nil, err
		}
		if i, err := sqlr.LastInsertId(); err == nil {
			ids = append(ids, uint32(i))
		} else {
			return nil, errors.New("LastInsertId error")
		}
		if i == 0 {
			UpdateAvatar(uid, v)
		}
	}
	updateDetailCount(uid)
	return
}

// 拼图游戏使用说明
// ids对应图片相册id
func DoPhotosCheckImg(uid uint32, ids []uint32, pics []string) (e error) {
	cm, e := general.CheckImg(general.IMGCHECK_SEXY_AND_AD, pics...)
	if e != nil {
		return
	}
	um := make(map[string]uint32)
	for k, pic := range pics {
		um[pic] = ids[k]
	}
	// 需要删除刚上传的图片
	for pic_url, v := range cm {
		if id, ok := um[pic_url]; v.Status != 0 && ok {
			DelPic(uid, int(id))
		}
	}
	return
}

func DelPic(uid uint32, albumid int) (e error) {
	var oldpic string
	e = mdb.QueryRow("select pic from user_photo_album where uid=? and albumid=?", uid, albumid).Scan(&oldpic)
	if e != nil {
		return e
	}
	_, e = mdb.Exec("delete from user_photo_album where uid=? and albumid=?", uid, albumid) //req.Uid
	if e != nil {
		return e
	}
	var pic string
	rows, e := mdb.QueryFromMain("select pic from user_photo_album where uid=?", uid)
	if rows.Next() {
		if err := rows.Scan(&pic); err != nil {
			return err
		}
	}
	//_, e = mdb.Exec("update user_detail set avatar=?,avatarlevel=-2 where uid=? and avatar=?", pic, uid, oldpic)
	_, e = mdb.Exec("update user_detail set avatar=? where uid=? and avatar=?", pic, uid, oldpic)
	// updateDetailCount(uid)
	return
}

func GetAddr(uid uint32) (addrlist []map[string]interface{}, e error) {

	dr, err := mdb.Query("select * from user_address where uid=?", uid) //req.Uid
	if err != nil {
		return nil, err
	}
	defer dr.Close()
	sqlr, err2 := utils.ParseSqlResult(dr)
	if err2 != nil {
		return nil, err2
	}

	addrlist = make([]map[string]interface{}, 0, len(sqlr))
	for _, v := range sqlr {
		item := make(map[string]interface{})
		v2, _ := utils.StringToUint32(v["addrid"])
		item["addrid"] = v2              //地址ID
		item["phone"] = v["phone"]       //电话号码
		item["province"] = v["province"] //省份
		item["city"] = v["city"]         //城市
		item["address"] = v["address"]   //详细地址
		item["username"] = v["username"] //详细地址
		addrlist = append(addrlist, item)
	}
	return
}

func AddAddr(uid uint32, phone string, province string, city string, address string, username string) (addrid uint32, e error) {

	sql := "insert into user_address (uid,phone,province,city,address,username)values(?,?,?,?,?,?)"

	sqlr, err := mdb.Exec(sql, uid, phone, province, city, address, username) //req.Uid
	if err != nil {
		return 0, err
	}
	if i, err := sqlr.LastInsertId(); err == nil {
		addrid = uint32(i)
	} else {
		return 0, errors.New("LastInsertId ERROR")
	}
	return
}

func SetAddr(uid uint32, addrid uint32, phone string, province string, city string, address string, username string) (e error) {
	sql := "update user_address set phone=?,province=?,city=?,address=?,username=? where uid=? and addrid=?"
	_, err := mdb.Exec(sql, phone, province, city, address, username, uid, addrid)
	if err != nil {
		return err
	}
	return
}

func DelAddr(uid uint32, addrid int) (e error) {
	sql := "delete from user_address where uid=? and addrid=?"
	_, err := mdb.Exec(sql, uid, addrid)
	if err != nil {
		return err
	}
	return
}

func GetCoinLog(uid uint32, cur, ps int) (result []map[string]string, total int, e error) {
	sql := "select uid,forid,info,type,create_time as `time`,coin   from  user_coin_log  where  uid =  ?  order by create_time desc " + utils.BuildLimit(cur, ps)
	rows, err := mdb.Query(sql, uid)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list, err := utils.ParseSqlResult(rows)
	if err != nil {
		return nil, 0, err
	}
	err = mdb.QueryRow("select count(*) from user_coin_log where uid =?", uid).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func GiftShop() (result []map[string]string, e error) {

	sql := "select id as gid,n,info,img,price,earn,type,level,res from gift "
	rows, err := mdb.Query(sql)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	list, err := utils.ParseSqlResult(rows)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GiftLevel(uid uint32, toid uint32) (rlevel int, e error) {
	var alevel int
	sql := "select ifnull(max(gift.level),-1) from gift_record left join gift on gift.id=gift_record.gid where uid=? and t_uid=?"
	e = mdb.QueryRowFromMain(sql, uid, toid).Scan(&alevel)
	if e != nil {
		return 0, e
	}
	return alevel + 1, nil
}

//我的未兑换礼物
func GiftInfo(uid uint32) (result []map[string]string, e error) {
	sql := "select gid gid,n,info,price,earn,img, count(*) count from  gift_record a LEFT JOIN gift b on a.gid=b.id where a.`status`=1 and a.t_uid =? GROUP BY a.gid"
	rows, err := mdb.Query(sql, uid)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := utils.ParseSqlResult(rows)
	if err != nil {
		return nil, err
	}
	return list, nil
}

//我的礼物记录
func GiftList(uid uint32, cur, ps int) (result []map[string]string, total int, e error) {
	sql := "select a.id id,uid,gid,n,info,price,earn,img,a.`status` as stat,tm as `time`  from  gift_record a LEFT JOIN gift b on a.gid=b.id where a.t_uid =? order by tm desc " + utils.BuildLimit(cur, ps)
	p_sql := "select count(*) as count from gift_record where t_uid =? "
	rows, err := mdb.Query(sql, uid)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	if err != nil {
		return nil, 0, err
	}
	list, err := utils.ParseSqlResult(rows)
	if err != nil {
		return nil, 0, err
	}
	var itotal int
	err = mdb.QueryRow(p_sql, uid).Scan(&itotal)
	if err != nil {
		return nil, 0, err
	}
	return list, itotal, nil
}

//收取礼物 进到我的礼物数
func GiftReceive(uid uint32, logid int, tag string) (msgid uint64, earn int, e error) {
	var fromid uint32
	var n, info, img string
	sql := "select uid,n,info,img,earn from gift_record left join gift on gift.id=gift_record.gid where gift_record.t_uid =? and gift_record.id=?"
	e = mdb.QueryRow(sql, uid, logid).Scan(&fromid, &n, &info, &img, &earn)
	if e != nil {
		return 0, 0, e
	}

	tx, err := mdb.Begin()
	if err != nil {
		return 0, 0, e
	}
	sql = "update gift_record set `status`=1 where t_uid =? and id=? and `status`=0"
	r, err := tx.Exec(sql, uid, logid)
	if err != nil {
		tx.Rollback()
		return 0, 0, err
	}
	if i, err := r.RowsAffected(); err != nil {
		return 0, 0, err
	} else {
		if i <= 0 {
			return 0, 0, errors.New("没有记录被更改")
		}
	}
	serr := coin.UserCoinChange(tx, uid, 0, coin.EARN_GIFT, 0, earn, info)
	if serr.Code != service.ERR_NOERR {
		tx.Rollback()
		return 0, 0, serr
	}
	tx.Commit()

	mid, err := PushThx_present(uid, fromid, logid, n, info, img, tag)
	if err != nil {
		return 0, 0, err
	}
	return mid, earn, nil
}

//收取所有礼物
func GiftReceiveAll(uid uint32) (e error) {
	sql := "select uid,n,info,img,gift_record.id from gift_record left join gift on gift.id=gift_record.gid where gift_record.`status`=0 and gift_record.t_uid =?"
	rows, err := mdb.Query(sql, uid)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var fromid uint32
		var logid int
		var n, info, img string
		if err := rows.Scan(&fromid, &n, &info, &img, &logid); err != nil {
			return err
		}
		r, err := mdb.Exec("update gift_record set `status`=1 where t_uid =? and id=? and `status`=0", uid, logid)
		if err != nil {
			return err
		}
		if i, err := r.RowsAffected(); err != nil {
			return err
		} else {
			if i <= 0 {
				continue
			}
		}
		PushThx_present(uid, fromid, logid, n, info, img, "")
	}
	return nil
}

//拒绝礼物
func GiftReject(uid uint32, logid int, tag string) (msgid uint64, e error) {
	var fromid uint32
	var n, info, img string
	sql := "select uid,n,info,img from gift_record left join gift on gift.id=gift_record.gid where gift_record.t_uid =? and gift_record.id=?"
	e = mdb.QueryRow(sql, uid, logid).Scan(&fromid, &n, &info, &img)
	if e != nil {
		return 0, e
	}

	tx, err := mdb.Begin()
	if err != nil {
		return 0, e
	}
	sql = "update gift_record set `status`=4 where t_uid =? and id=? and `status`=0"
	r, err := tx.Exec(sql, uid, logid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	if i, err := r.RowsAffected(); err != nil {
		tx.Rollback()
		return 0, err
	} else {
		if i <= 0 {
			tx.Rollback()
			return 0, errors.New("没有记录被更改")
		}
	}
	// serr := coin.UserCoinChange(tx, uid, 0, coin.EARN_GIFT, 0, earn, info)
	// if serr.Code != service.ERR_NOERR {
	// 	tx.Rollback()
	// 	return 0, 0, serr
	// }
	tx.Commit()

	// mid, err := PushThx_present(uid, fromid, logid, n, info, img, tag)
	// if err != nil {
	// 	return 0, 0, err
	// }
	return msgid, nil
}

func GiftCount(uid uint32) (result int, e error) {
	sql := "select count(*) from user_coin_log where uid =? and type=?"
	err := mdb.QueryRow(sql, uid, coin.EARN_WORK).Scan(&result)
	if err != nil {
		return 0, err
	}
	return result, nil
}

func FendCount(uid uint32) (result int, e error) {
	sql := "select count(*) from gift_record where t_uid =?"
	err := mdb.QueryRow(sql, uid).Scan(&result)
	if err != nil {
		return 0, err
	}
	return result, nil
}

//礼物兑换金币
func GiftExchange(uid uint32, gid int) (earn int, e service.Error) {
	var iearn int
	var icount int64
	var name string
	sql1 := "select earn,n from gift where id=?"
	err := mdb.QueryRow(sql1, gid).Scan(&iearn, &name)
	if err != nil {
		return 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	sql := "update gift_record set `status`=2 where t_uid =? and gid=? and `status`=1"
	tx, err := mdb.Begin()
	r, err := tx.Exec(sql, uid, gid)
	if err != nil {
		tx.Rollback()
		return 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	if icount, err = r.RowsAffected(); err != nil {
		tx.Rollback()
		return 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	if icount <= 0 {
		tx.Rollback()
		return 0, service.NewError(service.ERR_UNKNOWN, "没有可兑换的礼物")
	}
	iinc := int(icount) * iearn
	info := "兑换了 " + utils.Int64ToString(icount) + " 个 " + name
	serr := coin.UserCoinChange(tx, uid, 0, coin.EARN_GIFT, 0, iinc, info)
	if serr.Code != service.ERR_NOERR {
		tx.Rollback()
		return 0, serr
	}
	tx.Commit()
	return iinc, service.NewError(service.ERR_NOERR, "")
}

func GiftSend(uid uint32, to_id uint32, giftid int, tag string) (id int, msgid uint64, gprice int, e service.Error) {
	sql := "select n,info,img,price,`level`,earn,res from gift where id=?"
	var n, info, img, res string
	var price, level, earn int
	err := mdb.QueryRow(sql, giftid).Scan(&n, &info, &img, &price, &level, &earn, &res)
	if err != nil {
		return 0, 0, 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	rl, err := GiftLevel(uid, to_id)
	if err != nil {
		return 0, 0, 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	if rl < level {
		return 0, 0, 0, service.NewError(service.ERR_PERMISSION_DENIED, "赠送礼物超过可送等级", "赠送礼物超过可送等级")
	}

	tx, err := mdb.Begin()
	if err != nil {
		return 0, 0, 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	e = coin.UserCoinChange(tx, uid, to_id, coin.EARN_GIFT, 0, -price, "赠送礼物"+n)
	if e.Code != service.ERR_NOERR {
		tx.Rollback()
		return 0, 0, 0, e
	}
	sql = "insert into gift_record (uid,t_uid,gid,`status`)values(?,?,?,2)"
	r, err := tx.Exec(sql, uid, to_id, giftid)
	if err != nil {
		tx.Rollback()
		return 0, 0, 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	var gid int
	if i, err := r.LastInsertId(); err != nil {
		return 0, 0, 0, service.NewError(service.ERR_MYSQL, err.Error())
	} else {
		gid = int(i)
	}

	e = coin.UserCoinChange(tx, to_id, 0, coin.EARN_GIFT, 0, earn, fmt.Sprintf("收到礼物 %v 获得 %v个钻石", n, earn))
	if e.Code != service.ERR_NOERR {
		tx.Rollback()
		return 0, 0, 0, e
	}
	tx.Commit()

	mid, err := PushGive_present(uid, to_id, gid, n, info, img, tag, earn, giftid, res)
	if err != nil {
		return 0, 0, 0, service.NewError(service.ERR_MYSQL, e.Error())
	}
	return gid, mid, price, service.NewError(service.ERR_NOERR, "")
}

func GiftTest(uid uint32, to_id uint32, giftid int, tag string) (msgid uint64, e error) {
	sql := "select n,info,img,price,`level`,earn,res from gift where id=?"
	var n, info, img, res string
	var price, level, earn int
	err := mdb.QueryRow(sql, giftid).Scan(&n, &info, &img, &price, &level, &earn, &res)
	if err != nil {
		return 0, err
	}

	sql = "insert into gift_record (uid,t_uid,gid,`status`)values(?,?,?,2)"
	r, err := mdb.Exec(sql, uid, to_id, giftid)
	if err != nil {
		return 0, err
	}
	var gid int
	if i, err := r.LastInsertId(); err != nil {
		return 0, err
	} else {
		gid = int(i)
	}

	mid, err := PushGive_present(uid, to_id, gid, n, info, img, tag, earn, giftid, res)
	if err != nil {
		return 0, err
	}
	return mid, nil
}

//获取奖品记录
func GetAwardLog(itype int, uid uint32, cur, ps int) (result []map[string]interface{}, total int, e error) {
	var sql string
	var sql2 string
	switch itype {
	case 1:
		sql = "select a.id,`name`,`type`,img,price,info,tm,`from`,a.`status`,log_id,cnum,frominfo from  award_record a LEFT JOIN award_config b on a.award_id=b.id where uid =? and (b.`type`=1 or b.`type`=4 or b.`type`=5) order by tm desc " + utils.BuildLimit(cur, ps)
		sql2 = "select count(*) from award_record a LEFT JOIN award_config b on a.award_id=b.id  where uid =? and (b.`type`=1 or b.`type`=4 or b.`type`=5)"
	default:
		sql = "select a.id,`name`,`type`,img,price,info,tm,`from`,a.`status`,log_id,cnum,frominfo from  award_record a LEFT JOIN award_config b on a.award_id=b.id where uid =? and (b.`type`=2 or b.`type`=3) order by tm desc " + utils.BuildLimit(cur, ps)
		sql2 = "select count(*) from award_record a LEFT JOIN award_config b on a.award_id=b.id  where uid =? and (b.`type`=2 or b.`type`=3)"
	}
	rows, err := mdb.Query(sql, uid)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	result = make([]map[string]interface{}, 0, 0)
	for rows.Next() {
		var cnum int
		var id, tp, price, from, status, logid uint32
		var name, img, info, stm, frominfo string
		if err := rows.Scan(&id, &name, &tp, &img, &price, &info, &stm, &from, &status, &logid, &cnum, &frominfo); err != nil {
			return nil, 0, err
		}
		item := make(map[string]interface{})
		item["id"] = id
		if tp == 5 {
			item["name"] = fmt.Sprintf("%v %v", cnum, name)
			item["info"] = fmt.Sprintf("%v %v", cnum, info)
			item["price"] = cnum
			item["type"] = 1
		} else {
			item["name"] = name
			item["info"] = info
			item["price"] = price
			item["type"] = tp
		}

		item["img"] = img
		item["from"] = from
		item["status"] = status
		item["log_id"] = logid
		item["frominfo"] = frominfo

		if t, err := utils.ToTime(stm); err != nil {
			item["tm"] = utils.Now
		} else {
			item["tm"] = t
		}
		result = append(result, item)
	}
	// list, err := utils.ParseSqlResult(rows)
	// if err != nil {
	// 	return nil, 0, err
	// }

	err = mdb.QueryRow(sql2, uid).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

//用户领奖
func AwardItem(uid uint32, id int, addrid int) (e error) {
	var phone, province, city, address, username string
	tx, e := mdb.Begin()
	sql := "select phone,province,city,address,username from  user_address where addrid=? and uid=?"
	err := tx.QueryRow(sql, addrid, uid).Scan(&phone, &province, &city, &address, &username)
	if err != nil {
		tx.Rollback()
		return err
	}
	sql = "insert into award_trans (address,address_phone,address_name,province,city)values(?,?,?,?,?)"
	r, err := tx.Exec(sql, address, phone, username, province, city)
	if err != nil {
		tx.Rollback()
		return err
	}
	i, err := r.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}
	log_id := int(i)
	sql = "update award_record set log_id=?,oper_tm=?,status=? where id=? and uid=? and status<2"
	r, err = tx.Exec(sql, log_id, utils.Now, 2, id, uid)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return
}

//领取话费
func AwardPhone(uid uint32, id int, phone string) (e error) {

	sql := "update award_record set charge_phone=?,oper_tm=?,status=? where id=? and uid=? and status<2"
	r, err := mdb.Exec(sql, phone, utils.Now, 2, id, uid)
	if err != nil {
		return err
	}
	if r, e := r.RowsAffected(); e != nil {
		return err
	} else {
		if r == 0 {
			return errors.New("无可充值奖品")
		}
	}
	return
}

//领取金币
func AwardCoin(uid uint32, id int) (price int, e error) {
	// var price int
	var awname string
	if err := mdb.QueryRow("select price,award_config.`name` from award_record left join award_config on award_config.id=award_record.award_id where award_record.id=?", id).Scan(&price, &awname); err != nil {
		return 0, err
	}
	tx, err := mdb.Begin()
	if err != nil {
		return 0, err
	}
	sql := "update award_record set oper_tm=?,status=? where id=? and uid=? and status<1"
	r, err := tx.Exec(sql, utils.Now, 3, id, uid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	if r, e := r.RowsAffected(); e != nil {
		tx.Rollback()
		return 0, err
	} else {
		if r == 0 {
			tx.Rollback()
			return 0, errors.New("无可用奖品")
		}
	}
	if err := coin.UserCoinChange(tx, uid, 0, coin.EARN_AWARD, 0, price, "领取奖品 "+awname); err.Code != service.ERR_NOERR {
		tx.Rollback()
		return 0, err
	}
	tx.Commit()
	return
}

//领取虚拟物品
func AwardVirtual(uid uint32, id int) (result map[string]interface{}, e error) {
	// var price int
	var awname string
	var virtualtype, virtualcount int
	if err := mdb.QueryRow("select award_config.`name`,virtualtype,virtualcount from award_record left join award_config on award_config.id=award_record.award_id where award_record.id=?", id).Scan(&awname, &virtualtype, &virtualcount); err != nil {
		return nil, err
	}
	tx, err := mdb.Begin()
	if err != nil {
		return nil, err
	}
	result = make(map[string]interface{})
	sql := "update award_record set oper_tm=?,status=? where id=? and uid=? and status<1"
	r, err := tx.Exec(sql, utils.Now, 3, id, uid)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if r, e := r.RowsAffected(); e != nil {
		tx.Rollback()
		return nil, err
	} else {
		if r == 0 {
			tx.Rollback()
			return nil, errors.New("无可用奖品")
		}
	}
	switch virtualtype {
	case 1: //领取飞行场次
		if err := UpdateGameNum(tx, uid, virtualcount); err != nil {
			return nil, err
		}
	}
	tx.Commit()
	return
}

//查看物流
func AwardTrans(uid uint32, log_id int) (result map[string]interface{}, e error) {
	var name, num, tm string

	sql := "select name,num,tm from award_trans where id=? "
	err := mdb.QueryRow(sql, log_id).Scan(&name, &num, &tm)
	if err != nil {
		return nil, err
	}
	result = make(map[string]interface{})
	result["name"] = name
	result["num"] = num
	result["tm"] = tm
	result["info"] = "测试"
	return
}

/*
查看奖品发货信息

请求URL：s/user/AwardInfo
	{
		"id":13552，//奖品ID
	}
返回结果：
	{
		"status": "ok",
		"res":{
			"type": 1, //物品类型 1：Y币 ，2：实物 ，3：充值卡  待扩展
			"status":1    //状态 奖品状态：0 待领取 1. 已领取 2.等待充值（发货） 3：已完成
		 	"charge_phone"//手机充值号手机充值卡有此字段，实物包含后面字段
		 	"province":"湖南",  //领奖省份
		 	"city":"长沙",  //领奖城市
			"address":"..."//发货地址
			"address_phone":"13811111111"//收货手机号
			"address_name":"..."//收货人姓名
			"name":"中通快递",//物流公司
		 	"num":"1123232",//物流单号
			"tm":..,//发货时间
		}
	}
*/
func AwardInfo(uid uint32, id int) (result map[string]interface{}, e error) {
	var atype, status, log_id int
	var charge_phone string
	sql := "select award_config.type,award_record.status,charge_phone,log_id from award_record left join award_config on award_record.award_id=award_config.id  where award_record.id=? and uid=?"
	err := mdb.QueryRow(sql, id, uid).Scan(&atype, &status, &charge_phone, &log_id)
	if err != nil {
		return nil, err
	}
	result = make(map[string]interface{})
	result["type"] = atype
	result["status"] = status
	switch atype {
	case 2: //实物
		if log_id == 0 {
			return
		}
		var address, address_phone, address_name, name, num, province, city string
		var tm string
		if err := mdb.QueryRow("select address, address_phone, address_name, name, num,tm,province,city from award_trans where id=?", log_id).Scan(&address, &address_phone, &address_name, &name, &num, &tm, &province, &city); err != nil {
			return nil, err
		}
		result["address"] = address
		result["address_phone"] = address_phone
		result["address_name"] = address_name
		result["name"] = name
		result["num"] = num
		result["province"] = province
		result["city"] = city
		result["tm"], _ = utils.ToTime(tm)
	case 3: //充值卡
		result["charge_phone"] = charge_phone
	}
	return
}

// func createOrderNo(tp int) (s string, e error) {
// 	return "334343434343", nil
// }

// 1 财付通 2 网银在线 3 银联，4支付宝,5手动,6手机充值卡,7支付宝wap,8微信,9银联语音
func postOrderNo(uid uint32, order_no string, tp int, money int, ip string, extra map[string]interface{}) (content interface{}, e error) {
	switch tp {
	case 3: //银联
		v := url.Values{}
		v.Set("subject", "银联支付")
		v.Set("body", "购买VIP付款")
		v.Set("total_fee", utils.ToString(money))
		v.Set("out_trade_no", order_no)
		v.Set("ip", ip)
		// v.Set("f", "1")
		body := ioutil.NopCloser(strings.NewReader(v.Encode()))
		resp, e := http.Post("http://pa.app.mumu123.cn/user/unionPay", "application/x-www-form-urlencoded", body)
		if e != nil {
			return nil, e
		}
		defer resp.Body.Close()

		sr, e2 := ioutil.ReadAll(resp.Body)
		if e2 != nil {
			return nil, e2
		}
		var rmap map[string]interface{}
		if e := json.Unmarshal(sr, &rmap); e != nil {
			return nil, e
		}
		if v, ok := rmap["result"]; ok {
			if utils.ToString(v) == "1" {
				if v, ok := rmap["content"]; ok {
					content = v
				} else {
					return nil, errors.New("无正确返回结果")
				}
			} else {
				return nil, errors.New("无正确返回结果")
			}
		} else {
			return nil, errors.New("无正确返回结果")
		}
		fmt.Println(fmt.Sprintf("银联支付 order_no:%v,uid:%v,money:%v,result %v", order_no, uid, money, content))
		return content, nil
	case 8: //微信
		v := url.Values{}
		v.Set("subject", "微信支付")
		v.Set("body", "购买VIP付款")
		v.Set("total_fee", utils.ToString(money))
		v.Set("out_trade_no", order_no)
		v.Set("ip", ip)
		// v.Set("f", "1")
		body := ioutil.NopCloser(strings.NewReader(v.Encode()))
		resp, e := http.Post("http://zhifu.mumu123.cn/user/weChatPay", "application/x-www-form-urlencoded", body)
		if e != nil {
			return nil, e
		}
		defer resp.Body.Close()

		sr, e2 := ioutil.ReadAll(resp.Body)
		if e2 != nil {
			return nil, e2
		}
		// var r map[string]interface{}
		if e := json.Unmarshal(sr, &content); e != nil {
			return nil, e
		}
		fmt.Println(fmt.Sprintf("微信支付 order_no:%v,uid:%v,money:%v,result %v", order_no, uid, money, content))
		return content, nil
	case 6: //手机卡
		v := url.Values{}
		v.Set("total_fee", utils.ToString(money))
		v.Set("out_trade_no", order_no)
		v.Set("cardMoney", utils.ToString(extra["cardMoney"]))
		v.Set("sn", utils.ToString(extra["sn"]))
		v.Set("password", utils.ToString(extra["password"]))
		if vv, ok := extra["cardType"]; ok {
			v.Set("cardType", utils.ToString(vv))
		}
		fmt.Println(fmt.Sprintf("手机卡支付 extra %v", extra))
		body := ioutil.NopCloser(strings.NewReader(v.Encode()))
		fmt.Println(fmt.Sprintf("手机卡支付 body %v", body))
		resp, e := http.Post("http://pa.app.mumu123.cn/user/cardPay", "application/x-www-form-urlencoded", body)
		if e != nil {
			return nil, e
		}
		defer resp.Body.Close()
		sr, e2 := ioutil.ReadAll(resp.Body)
		if e2 != nil {
			return nil, e2
		}

		var rmap map[string]interface{}
		if e := json.Unmarshal(sr, &rmap); e != nil {
			return nil, e
		}
		if v, ok := rmap["result"]; ok {
			if utils.ToString(v) == "1" {
				if v, ok := rmap["content"]; ok {
					content = v
				} else {
					return nil, errors.New("无正确返回结果")
				}
			} else {
				return nil, errors.New(utils.ToString(rmap["content"]))
			}
		} else {
			return nil, errors.New("无正确返回结果")
		}

		fmt.Println(fmt.Sprintf("手机卡支付 order_no:%v,uid:%v,money:%v,result %v", order_no, uid, money, string(sr)))
		return content, nil
	case 7: //支付宝
		fmt.Println(fmt.Sprintf("支付宝支付 order_no:%v,uid:%v,money:%v,result %v", order_no, uid, money, content))
		return content, nil
	}
	return
}

func intLenToStr(i int64, length int) string {
	// time.Parse(layout, value)

	s := utils.Now.Format("20060102")
	if mode != cls.MODE_PRODUCTION {
		s = "9" + s[1:] //测试环境 则把第一位改成9
	} else {
		s = "8" + s[1:] //秋千，订单号首位为8
	}
	scode := utils.ToString(i)
	scode = strings.Repeat("0", 10-len(scode)) + scode
	return s + scode
}

//1 财付通 2 网银在线 3 银联，4支付宝,5手动,6手机充值卡,7支付宝wap,8微信,9银联语音
//开始充值 生成订单
func PayBegin(uid uint32, tp int, productid int, ip string, extra map[string]interface{}) (order_no string, content interface{}, e error) {
	// fmt.Println(fmt.Sprintf("begin PayBegin time %v", time.Now()))
	var money int
	var info string
	if err := mdb.QueryRow("select money,name from product where id=?", productid).Scan(&money, &info); err != nil {
		return "", nil, err
	}
	info = "购买 " + info
	sql := "insert into charge (tp,uid,money,productid,create_tm,info)values(?,?,?,?,?,?)"
	r, err := mdb.Exec(sql, tp, uid, money, productid, utils.Now, info)
	if err != nil {
		return "", nil, err
	}
	i, err := r.LastInsertId()
	if err != nil {
		return "", nil, err
	}
	order_no = intLenToStr(i, 20)
	if err != nil {
		return "", nil, err
	}
	mdb.Exec("update charge set order_no=? where id=?", order_no, i)
	// fmt.Println(fmt.Sprintf("begin postOrderNo time %v", time.Now()))
	content, err = postOrderNo(uid, order_no, tp, money, ip, extra)
	// fmt.Println(fmt.Sprintf("end postOrderNo time %v", time.Now()))
	if err != nil {
		// fmt.Println(fmt.Sprintf("PayBegin err %v", err))
		return "", nil, err
	}
	return order_no, content, nil
}

// 检查支付结果
func checkOrderResult(tx utils.SqlObj, order_no string) (e error) {
	var paymoney, money, stat, productid int
	var uid uint32
	var info string
	sql := "select paymoney,money,stat,uid,info,productid from charge where order_no=?"
	err := tx.QueryRow(sql, order_no).Scan(&paymoney, &money, &stat, &uid, &info, &productid)
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("paymoney %v,money %v ,coin %v.", paymoney, money))
	StatCharge(uid, paymoney)
	if paymoney < money {
		msg := make(map[string]interface{})
		msg["content"] = info + " 支付金额不足"
		msg["type"] = common.MSG_TYPE_TEXT
		PushSysMessage(common.UID_SYSTEM, uid, common.FOLDER_OTHER, msg)
		// PushSysMessage(uid, common.SYS_MSG_PAY, msg)
	} else {
		icoin, igame, idcard, err := giveProduct(tx, uid, productid)
		if err != nil {
			return err
		}
		// coin.UserCoinChange(mdb, uid, 0, coin.EARN_CHARGE, 0, con, "充值金币 "+info)
		switch idcard {
		case 1:
			err := IdCardPayCallback(uid)
			if err != nil {
				mainlog.Append(fmt.Sprintf("IdCardPayCallback error uid %v,error %v", uid, err.Error()))
			}
			return
		default:
			msg := make(map[string]interface{})
			msg["content"] = info + " 支付完成"
			if icoin != 0 {
				var allgold int
				if err := tx.QueryRow("select goldcoin from user_main where uid=?", uid).Scan(&allgold); err != nil {

				} else {
					msg[common.USER_BALANCE] = allgold
				}
				msg[common.USER_BALANCE_CHANGE] = fmt.Sprintf("充值获得 %v金币", icoin)
			}
			if igame != 0 {
				un := make(map[string]interface{})
				//	un[common.UNREAD_PLANE_FREE] = 0
				//	unread.GetUnreadNum(uid, un)
				msg[common.UNREAD_KEY] = un
			}
			not, e := notify.GetNotify(uid, notify.NOTIFY_COIN, nil, "系统消息", fmt.Sprintf("充值获得 %v金币", icoin), uid)
			if e == nil {
				msg[notify.NOTIFY_KEY] = not
			}
			msg["type"] = common.MSG_TYPE_TEXT
			PushSysMessage(common.UID_SYSTEM, uid, common.FOLDER_OTHER, msg)
			// PushSysMessage(uid, common.SYS_MSG_PAY, msg)
		}
	}
	return
}

func UpdateGameNum(db utils.SqlObj, uid uint32, num int) (e error) {
	sql := "update user_main set plane_num = plane_num + ? where uid =? and plane_num + ? >=0 "
	rs, e := db.Exec(sql, num, uid, num)
	if e != nil {
		return e
	}
	if num, e := rs.RowsAffected(); e != nil || num <= 0 {
		return errors.New("飞机次数修改失败")
	}
	return nil
}

func giveProduct(tx utils.SqlObj, uid uint32, productid int) (count int, planecount int, idcard int, e error) {
	var ucoin int
	var name string
	var paytype int
	// fmt.Println(fmt.Sprintf("query productid %v,uid %v", productid, uid))
	if err := tx.QueryRow("select coin,`name`,planecount,paytype from product where id=?", productid).Scan(&ucoin, &name, &planecount, &paytype); err != nil {
		return 0, 0, 0, err
	}
	// fmt.Println(fmt.Sprintf("加金币 %v，加飞机 %v", ucoin, planecount))
	if ucoin != 0 {
		err := coin.UserCoinChange(tx, uid, 0, coin.EARN_CHARGE, 0, ucoin, "充值 "+name)
		if err.Code != service.ERR_NOERR {
			return 0, 0, 0, errors.New("加金币失败 " + err.Error())
		}
	}
	if planecount != 0 {
		err := UpdateGameNum(tx, uid, planecount)
		if err != nil {
			return 0, 0, 0, err
		}
	}
	if paytype == 3 {
		idcard = 1
	}
	return ucoin, planecount, idcard, nil
}

//php回调
func PayCallback(tp int, order_no string, stat int, money int) (e error) {
	tx, err := mdb.Begin()
	sql := "update charge set stat=?,paymoney=? where order_no=? and tp=? and stat=0 "
	r, err := tx.Exec(sql, stat, money, order_no, tp)

	if err != nil {
		tx.Rollback()
		return err
	}
	if r, e := r.RowsAffected(); e != nil {
		tx.Rollback()
		return err
	} else {
		if r == 0 {
			tx.Rollback()
			return errors.New("找不到订单")
		}
	}
	if stat == 1 {
		if err := checkOrderResult(tx, order_no); err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return
}

//客户端获取充值结果
func PayQuery(tp int, uid uint32, order_no string) (stat int, money int, info string, e error) {
	sql := "select stat,paymoney,info from charge left join product on product.id=charge.productid where order_no=? and uid=? and tp=? "

	err := mdb.QueryRow(sql, order_no, uid, tp).Scan(&stat, &money, &info)
	if err != nil {
		return 0, 0, "", err
	}
	return
}

func giveProductIos(tx utils.SqlObj, uid uint32, iosproduct string) (count int, planecount int, idcard int, info string, productid uint32, money int, e error) {
	var ucoin int
	var name string
	var paytype int
	// fmt.Println(fmt.Sprintf("query productid %v,uid %v", productid, uid))
	if err := tx.QueryRow("select coin,`name`,planecount,paytype,id,info,money from product where iapid=?", iosproduct).Scan(&ucoin, &name, &planecount, &paytype, &productid, &info, &money); err != nil {
		return 0, 0, 0, "", 0, 0, err
	}
	// fmt.Println(fmt.Sprintf("加金币 %v，加飞机 %v", ucoin, planecount))
	if ucoin != 0 {
		err := coin.UserCoinChange(tx, uid, 0, coin.EARN_CHARGE, 0, ucoin, "充值 "+name)
		if err.Code != service.ERR_NOERR {
			return 0, 0, 0, "", 0, 0, errors.New("加金币失败 " + err.Error())
		}
	}
	if planecount != 0 {
		err := UpdateGameNum(tx, uid, planecount)
		if err != nil {
			return 0, 0, 0, "", 0, 0, err
		}
	}
	if paytype == 3 {
		idcard = 1
	}
	return ucoin, planecount, idcard, info, productid, money, nil
}

//客户端获取充值结果
func PayIosQuery(uid uint32, ifsandbox int, receiptdata string, transaction string) (result map[string]interface{}, e error) {
	mainlog.AppendInfo(fmt.Sprintf("PayIosQuery uid %v,ifsandbox %v,receiptdata %v", uid, ifsandbox, receiptdata))
	count, Product_id, transaction_id, _, e := general.IapQuery(receiptdata, ifsandbox, transaction)
	if e != nil {
		mainlog.AppendInfo(fmt.Sprintf("IapQuery Error uid %v,ifsandbox %v,receiptdata %v,error %v", uid, ifsandbox, receiptdata, e))
		fmt.Println(fmt.Sprintf("IapQuery Error uid %v,ifsandbox %v,receiptdata %v,error %v", uid, ifsandbox, receiptdata, e))
		return
	}
	mainlog.AppendInfo(fmt.Sprintf("IapQuery Success uid %v,ifsandbox %v,receiptdata %v,Product_id %v,transaction_id %v", uid, ifsandbox, receiptdata, Product_id, transaction_id))
	fmt.Println(fmt.Sprintf("IapQuery Success uid %v,ifsandbox %v,receiptdata %v,Product_id %v,transaction_id %v", uid, ifsandbox, receiptdata, Product_id, transaction_id))
	// mdb.QueryRow("select ", receipt.Product_id)
	tx, e := mdb.Begin()
	if e != nil {
		return nil, e
	}
	var pcount int
	e = tx.QueryRow("select count(*) from charge where transaction_id=?", transaction_id).Scan(&pcount)
	if e != nil {
		tx.Rollback()
		return nil, e
	} else {
		if pcount > 0 {
			tx.Rollback()
			return nil, errors.New("支付订单号已存在")
		}
	}
	// fmt.Println("giveProductIos")
	icoin, igame, idcard, info, productid, money, err := giveProductIos(tx, uid, Product_id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	icoin = icoin * count
	igame = igame * count
	info = "购买 " + info
	// fmt.Println("sql insert")
	sql := "insert into charge (tp,uid,money,paymoney,productid,create_tm,info,stat,transaction_id)values(?,?,?,?,?,?,?,?,?)"
	r, e := tx.Exec(sql, 10, uid, money, money, productid, utils.Now, info, 1, transaction_id)
	if e != nil {
		tx.Rollback()
		return nil, e
	}
	// fmt.Println("sql insert end")
	i, e := r.LastInsertId()
	if e != nil {
		tx.Rollback()
		return nil, err
	}

	order_no := intLenToStr(i, 20)
	_, e = tx.Exec("update charge set order_no=? where id=?", order_no, i)
	if e != nil {
		tx.Rollback()
		return nil, e
	}
	// fmt.Println("update charge end")
	// if false {
	result = make(map[string]interface{})
	switch idcard {
	case 1:
		err := IdCardPayCallback(uid)
		if err != nil {
			mainlog.Append(fmt.Sprintf("IdCardPayCallback error uid %v,error %v", uid, err.Error()))
		}
		return
	default:

		result["content"] = info + " 支付完成"
		if icoin != 0 {
			var allgold int
			if err := tx.QueryRow("select goldcoin from user_main where uid=?", uid).Scan(&allgold); err != nil {

			} else {
				result[common.USER_BALANCE] = allgold
			}
			result[common.USER_BALANCE_CHANGE] = fmt.Sprintf("充值获得 %v钻石", icoin)
		}
		not, e := notify.GetNotify(uid, notify.NOTIFY_COIN, nil, "系统消息", fmt.Sprintf("充值获得 %v钻石", icoin), uid)
		if e == nil {
			result[notify.NOTIFY_KEY] = not
		}
	}
	tx.Commit()
	return
}

//商品列表
func ProductList(tp int, tp2 int, os int) (result []map[string]interface{}, e error) {
	var sql string
	if os == 0 {
		if tp2 == 1 {
			if tp == 0 {
				sql = "select id,`name`,info,money,img,recommend,coincost,iapid from product  where iapid=''"
			} else {
				sql = "select id,`name`,info,money,img,recommend,coincost,iapid from product  where iapid='' and paytype=" + utils.ToString(tp-1)
			}
		} else {
			if tp == 0 {
				sql = "select id,`name`,info,money,img,recommend,coincost,iapid from product  where iapid='' and `stat` =1"
			} else {
				sql = "select id,`name`,info,money,img,recommend,coincost,iapid from product  where iapid='' and `stat` =1 and paytype=" + utils.ToString(tp-1)
			}
		}
	} else {
		if tp2 == 1 {
			if tp == 0 {
				sql = "select id,`name`,info,money,img,recommend,coincost,iapid from product  where iapid<>''"
			} else {
				sql = "select id,`name`,info,money,img,recommend,coincost,iapid from product  where iapid<>'' and paytype=" + utils.ToString(tp-1)
			}
		} else {
			if tp == 0 {
				sql = "select id,`name`,info,money,img,recommend,coincost,iapid from product  where iapid<>'' and `stat` =1"
			} else {
				sql = "select id,`name`,info,money,img,recommend,coincost,iapid from product  where iapid<>'' and `stat` =1 and paytype=" + utils.ToString(tp-1)
			}
		}
	}
	rows, err := mdb.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result = make([]map[string]interface{}, 0, 0)
	var id, money, recommend, coincost int
	var name, info, img, iapid string
	for rows.Next() {
		if err := rows.Scan(&id, &name, &info, &money, &img, &recommend, &coincost, &iapid); err != nil {
			return nil, err
		}
		item := make(map[string]interface{})
		item["id"] = id
		item["name"] = name
		item["info"] = info
		fmoney, _ := utils.ToFloat64(money)
		item["money"] = fmoney / 100
		item["img"] = img
		item["recommend"] = recommend
		item["coincost"] = coincost
		item["iapid"] = iapid
		result = append(result, item)
	}
	return result, nil
}

func CoinBuy(uid uint32, productid int) (info string, blance int, e service.Error) {
	var coincost, planecount int
	var name string
	if err := mdb.QueryRow("select coincost,`name`,planecount from product where id=? and paytype<>0", productid).Scan(&coincost, &name, &planecount); err != nil {
		return "", 0, service.NewError(service.ERR_INTERNAL, err.Error())
	}
	info = "购买" + name + "消费" + utils.ToString(coincost) + "金币"
	tx, err := mdb.Begin()
	if err != nil {
		return "", 0, service.NewError(service.ERR_INTERNAL, err.Error())
	}
	e = coin.UserCoinChange(tx, uid, 0, coin.COST_BUYGAME, 0, -coincost, info)
	if e.Code != service.ERR_NOERR {
		tx.Rollback()
		e.Show = "余额不足"
		e.Code = service.ERR_NOT_ENOUGH_MONEY
		return "", 0, e
	}
	err = UpdateGameNum(tx, uid, planecount)
	if err != nil {
		tx.Rollback()
		return "", 0, service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if err := tx.QueryRow("select goldcoin from user_main where uid=? ", uid).Scan(&blance); err != nil {
		tx.Rollback()
		return "", 0, service.NewError(service.ERR_INTERNAL, err.Error())
	}
	tx.Commit()
	return
}

func ItemShop() (result []map[string]string, e error) {
	sql := "select * from ItemInfo"
	rows, err := mdb.Query(sql)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	list, err := utils.ParseSqlResult(rows)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func BuyItem(uid uint32, itemid int) (price int, e service.Error) {
	switch itemid {
	case 1:
		var icount int
		if err := sdb.QueryRow("select count(*) from recommend where id=? and left>0", uid).Scan(&icount); err != nil {
			return 0, service.NewError(service.ERR_MYSQL, err.Error())
		}
		if icount > 0 {
			return 0, service.NewError(service.ERR_CANTBY_ITEM, "暂时不能购买推荐展示")
		}
		uinfos, err := user_overview.GetUserObjects(uid)
		if err != nil {
			return 0, service.NewError(service.ERR_MYSQL, fmt.Sprintf("Get uid %v info error :%v", uid, err.Error()))
		}
		uinfo := uinfos[uid]
		if uinfo == nil {
			return 0, service.NewError(service.ERR_MYSQL, fmt.Sprintf("user %v info not found", uid))
		}
		price = 100
		if e := coin.UserCoinChange(mdb, uid, 0, coin.COST_ITEM, coin.ITEM_RECOMMEND, price, "购买 推荐展示 道具"); err != nil {
			return 0, e
		}

		if _, err := sdb.Exec("insert into recommend(id,gender,birthday,left)values(?,?,?,?) ON DUPLICATE KEY UPDATE left=left+?", uid, uinfo.Gender, uinfo.Birthday, 30, 30); err != nil {
			return 0, service.NewError(service.ERR_INTERNAL, err.Error())
		}
	}
	return
}

func SetLoginTime(uid uint32) (e error) {
	lcon := rdb.GetWriteConnection(redis_db.REDIS_LOGIN_TIME)
	defer lcon.Close()
	_, e = lcon.Do("SET", uid, utils.Now.Format(format.TIME_LAYOUT_1))
	return e
}

func CountExp(uid uint32) (e error) {
	lcon := rdb.GetReadConnection(redis_db.REDIS_LOGIN_TIME)
	defer lcon.Close()
	var itm string
	var tm time.Time
	if itm, e = redis.String(lcon.Do("GET", uid)); e != nil {
		return e
	}
	if tm, e = utils.ToTime(itm, format.TIME_LAYOUT_1); e != nil {
		return e
	}
	exp := utils.Now.Sub(tm).Seconds()
	if exp <= 0 {
		return
	}
	if exp > 3600 {
		exp = 3600
	}
	if _, e := mdb.Exec("update user_main set exp=exp+? where uid=?", exp, uid); e != nil {
		return e
	}
	return
}

//男性用户注册PUSH给客服
func NanReg(uid uint32, province, city string, gender int) {
	// fmt.Println("NanReg " + utils.ToString(uid) + "  " + province + city)
	switch gender {
	case common.GENDER_MAN:
		jid := AllotUid3(province, city)
		// jid = 1008603
		if jid > 0 {
			mainlog.AppendInfo(fmt.Sprintf("PushNanReg province %v,city %v,uid %v,kfid %v", province, city, uid, jid))
			// fmt.Println("PushNanReg " + utils.ToString(jid) + "  " + utils.ToString(uid))
			PushNanReg(jid, uid)
		}
	case common.GENDER_WOMAN:
		WriteRegTime(uid)
		jid := AllotUid(province, city)
		if jid > 0 {
			// fmt.Println(fmt.Sprintf("PushNvReg province %v,city %v,uid %v,kfid %v", province, city, uid, jid))
			mainlog.AppendInfo(fmt.Sprintf("PushNvReg province %v,city %v,uid %v,kfid %v", province, city, uid, jid))
			// jid = 1008603
			PushNvReg(jid, uid)
		}
	}
	return
}

func userOnline(msgid int, data interface{}) {
	switch v := data.(type) {
	case message.Online:
		if v.Uid < 5000000 {
			return
		}
		if IfUidAllot(v.Uid) {
			return
		}
		ue, err := user_overview.GetUserObjects(v.Uid)
		if err != nil {
			return
		}
		mp, ok := ue[v.Uid]
		if !ok {
			return
		}
		if mp.Gender == 2 {
			jid := AllotUid(mp.Province, mp.City)
			if jid > 0 {
				// jid = 1008602
				// fmt.Println(fmt.Sprintf("PushNvOnline uid %v,jid %v", v.Uid, jid))
				PushNvOnline(jid, v.Uid)
			}
		}
	}
	return
}

func createTopic(msgid int, data interface{}) {
	// fmt.Println("message createTopic")
	switch v := data.(type) {
	case message.CreateTopic:
		if v.Uid < 5000000 {
			return
		}
		ue, err := user_overview.GetUserObjects(v.Uid)
		if err != nil {
			return
		}
		mp, ok := ue[v.Uid]
		if !ok {
			return
		}
		jid := AllotUid2(mp.Province, mp.City)
		if jid > 0 {
			PushCreateTopic(jid, v.Uid, v.Tid)
		}
	}
	return
}

func userOffline(msgid int, data interface{}) {
	// switch v := data.(type) {
	// case message.Offline:
	// CountExp(v.Uid)
	// }
}

func ExpToGrade(exp int) int {
	return 1
}

func Feedback(uid uint32, info, strimg string, tp int) (e error) {
	if _, err := mdb.Exec("insert into user_feedback (uid,info,imgs,tp)values(?,?,?,?)", uid, info, strimg, tp); err != nil {
		return err
	}
	return
}

func GetUidArea(uid uint32) (lng int, lat int, e error) {
	var prov string
	if e = mdb.QueryRow("select province from user_detail where uid=?", uid).Scan(&prov); e != nil {
		return 0, 0, e
	}

	if e = mdb.QueryRow("select x,y from user_province_area where province=?", prov).Scan(&lng, &lng); e == nil {
		return
	}
	if e = mdb.QueryRow("select x,y from user_province_area where province=?", "北京市").Scan(&lng, &lng); e == nil {
		return
	}
	return
}

func AdminSetAvatarStat(list ...string) (e error) {
	for _, v := range list {
		sl := strings.Split(v, ",")
		var uid uint32
		var istat int
		if uid, e = utils.ToUint32(sl[0]); e != nil {
			return e
		}
		if istat, e = utils.ToInt(sl[1]); e != nil {
			return e
		}
		if istat == -1 {
			if _, err := mdb.Exec("update user_detail set avatarlevel=?,avatar=? where uid=?", istat, "http://image1.yuanfenba.net/uploads/oss/avatar/201511/03/17112044489.png", uid); err != nil {
				return err
			}
		} else {
			if _, err := mdb.Exec("update user_detail set avatarlevel=? where uid=?", istat, uid); err != nil {
				return err
			}
		}
		// if _, err := mdb.Exec("update user_detail set avatarlevel=? where uid=?", istat, uid); err != nil {
		// 	return err
		// }
		user_overview.ClearUserObjects(uid)
	}
	return
}

func SendPhotoChange(uid uint32, pcount int) {

	msg := make(map[string]interface{})
	data := map[string]interface{}{"uid": uid}
	not, e := notify.GetNotify(uid, notify.NOTIFY_NEW_PHOTO, data, "", "上传了"+utils.ToString(pcount)+"张照片", 0)
	if e != nil {
		return
	}
	woman_notify := not
	woman_notify.ShowType = 3

	msg[notify.NOTIFY_KEY] = not

	woman_msg := msg
	woman_msg[notify.NOTIFY_KEY] = woman_notify

	users, _, err := relation.GetFollowUids(true, uid, 1, 1000)
	if err != nil {
		return
	}
	// 分男女分别推送
	man_uids := make([]uint32, 0, 10)
	woman_uids := make([]uint32, 0, 10)
	um, e := user_overview.GetUserObjects(users...)
	if e != nil {
		return
	}
	for _, id := range users {
		if u, ok := um[id]; ok {
			if u.Gender == common.GENDER_MAN {
				man_uids = append(man_uids, id)
			} else if u.Gender == common.GENDER_WOMAN {
				woman_uids = append(woman_uids, id)
			}
		}
	}
	for _, v := range man_uids {
		PushInfoChange(v, uid, msg)
	}
	for _, v := range woman_uids {
		PushInfoChange(v, uid, woman_msg)
	}

	/*	msg2 := make(map[string]interface{})
		data2 := map[string]interface{}{"uid": uid}
		notify2, e2 := notify.GetNotify(uid, notify.NOTIFY_PNEW_PHOTO, data2, "", "上传了"+utils.ToString(pcount)+"张照片", 0)
		if e2 != nil {
			return
		}
			msg2[notify.NOTIFY_KEY] = notify2

			plist, _, err := service_pursue.GetPursueList(uid, 1, service.MAX_PS)
			if err != nil {
				return
			}
			for _, v := range plist {
				PushInfoChange(v, uid, msg2)
			}
	*/
}

func SendLocaltagChange(uid uint32, info string) {
	msg := make(map[string]interface{})
	u, e := user_overview.GetUserObject(uid)
	if e != nil {
		return
	}
	if u.Avatarlevel <= 0 {
		fmt.Println("用户头像审核不通过，不能通知其他用户")
		return
	}
	data := map[string]interface{}{"uid": uid}
	not, e := notify.GetNotify(uid, notify.NOTIFY_NEAR, data, "", info, 0)
	if e != nil {
		return
	}
	mainlog.AppendInfo(fmt.Sprintf("AdjacentUsers begin %v,%v", uid, u.Gender))
	if u.Gender == common.GENDER_MAN {
		not.ShowType = 3
	}
	users, err := discovery.AdjacentMatchedUsers(uid, 50, 0, 10)
	if err != nil {
		return
	}
	msg[notify.NOTIFY_KEY] = not
	for _, v := range users {
		if v.Gender == common.GENDER_MAN {
			not.ShowType = 7
		} else {
			not.ShowType = 3
		}
		PushInfoChange(v.Uid, uid, msg)
	}
}

//发送十个在前端的用户给客服关注
func SendTenFollow(fuid uint32) (count int, e error) {
	const (
		per_count    = 20
		follow_total = 10
	)
	rows, err := mdb.Query("SELECT user_online.uid from user_online LEFT JOIN user_main on user_main.uid=user_online.uid where user_online.ontop=1 and user_online.uid>=5000000  and user_main.gender=1")
	defer rows.Close()
	if err != nil {
		return 0, err
	}
	ids := make([]interface{}, 0, 0)
	for rows.Next() {
		var uid uint32
		if err := rows.Scan(&uid); err != nil {
			return 0, err
		}
		ids = append(ids, uid)
	}
	// fmt.Println(fmt.Sprintf("ids %v", ids))
	rsend := make([]uint32, 0, 0)
	ibegin := 0
	con := cache.GetWriteConnection(redis_db.CACHE_CAN_TEN)
	defer con.Close()
	for ibegin < len(ids) {
		iend := ibegin + per_count
		if iend > len(ids) {
			iend = len(ids)
		}
		ida := ids[ibegin:iend]
		ibegin = iend
		// fmt.Println(fmt.Sprintf("ida %v", ida))
		elems, e := redis.Values(con.Do("MGET", ida...))
		if e != nil {
			return 0, err
		}
		index := 0
		for len(elems) > 0 {
			var elem string
			elems, e = redis.Scan(elems, &elem)
			if e != nil {
				return 0, err
			}
			if elem == "" {
				rsend = append(rsend, ida[index].(uint32))
				count++
			}
			if count >= follow_total {
				break
			}
			index++
		}
		if count >= follow_total {
			break
		}
	}
	mainlog.AppendInfo(fmt.Sprintf("SendTenFollow uid %v,getids %v", fuid, rsend))
	// fmt.Println(fmt.Sprintf("rsend %v", rsend))
	for _, v := range rsend {
		relation.Follow(fuid, v)
		con.Do("SETEX", v, 3600*24, 1)
	}
	return count, nil
}
func AdminTj(aid uint32, date string) (info map[string]interface{}, e error) {
	mdb.Exec("insert into  manager_stat (date,uid,tm)values(?,?,?) ON DUPLICATE KEY UPDATE tm=?", date, aid, utils.Now)
	sl := strings.Split(date, "-")
	if len(sl) != 3 {
		return nil, errors.New("日期转换失败")
	}
	y, err := utils.ToInt(sl[0])
	if err != nil {
		return nil, err
	}
	m, err := utils.ToInt(sl[1])
	if err != nil {
		return nil, err
	}
	d, err := utils.ToInt(sl[2])
	if err != nil {
		return nil, err
	}
	var sdd time.Month
	sdd = time.Month(m)
	t1 := time.Date(y, sdd, d, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(y, sdd, d, 23, 59, 59, 0, time.UTC)
	rows, err := mdb.Query("select manager_users.uid,user_main.gender,user_detail.province from manager_users left join user_main on user_main.uid=manager_users.uid left join user_detail on user_detail.uid=manager_users.uid where admin_uid=?", aid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	uids := make([]uint32, 0, 0)
	var mencount, womencount int
	nanmap := make(map[string]int)
	nvmap := make(map[string]int)
	for rows.Next() {
		var uid uint32
		var gender int
		var province string
		if err := rows.Scan(&uid, &gender, &province); err != nil {
			return nil, err
		}
		if gender == 2 {
			nvmap[province] = 1
		} else {
			nanmap[province] = 1
		}
		uids = append(uids, uid)
		if gender == 2 {
			womencount++
		} else {
			mencount++
		}
	}
	var msgcount, msguser, msgcount2, msguser2 int
	//每个运营人员每天与多少用户聊天、一共发了多少消息
	if err := msgdb.QueryRow("select count(distinct `to`),count(`to`) from message where `from` "+mysql.In(uids)+" and type in('text','voice','pic') and( tm between ? and ?)", t1, t2).Scan(&msguser, &msgcount); err != nil {
		return nil, err
	}
	//每个运营人员每天手动与多少用户聊天、一共手动发了多少消息
	if err := mdb.QueryRow("select count(distinct `to`),count(`to`) from manager_msg where `from` "+mysql.In(uids)+" and type in('text','voice','pic') and( tm between ? and ?)", t1, t2).Scan(&msguser2, &msgcount2); err != nil {
		return nil, err
	}
	// 每个运营人员每天收到多少女性用户注册消息，回复了多少用户 回复了多少消息
	var nvreg, nrmsg, nruser, nrmsg2, nruser2, nruser3, nrmsg3 int
	rows4, err4 := msgdb.Query("select distinct `from` from message where `to` "+mysql.In(uids)+" and type in('nvreg') and (tm between ? and ?)", t1, t2)
	if err4 != nil {
		return nil, err4
	}
	defer rows4.Close()
	loginlist3 := make([]uint32, 0, 0)
	for rows4.Next() {
		var uid uint32
		if err := rows4.Scan(&uid); err != nil {
			return nil, err
		}
		loginlist3 = append(loginlist3, uid)
	}
	nvreg = len(loginlist3)
	//客服发的消息
	if err := msgdb.QueryRow("select count(distinct `to`),count(`to`) from message where `to` "+mysql.In(loginlist3)+" and `from` "+mysql.In(uids)+" and tm between ? and ? and type in('text','voice','pic')", t1, t2).Scan(&nruser, &nrmsg); err != nil {
		return nil, err
	}
	//女性用户回的消息
	if err := msgdb.QueryRow("select count(distinct `from`),count(`from`) from message where `from` "+mysql.In(loginlist3)+" and `to` "+mysql.In(uids)+" and tm between ? and ? and type in('text','voice','pic')", t1, t2).Scan(&nruser2, &nrmsg2); err != nil {
		return nil, err
	}
	//手动发的消息
	if err := mdb.QueryRow("select count(distinct `to`),count(`to`) from manager_msg where `to` "+mysql.In(loginlist3)+" and `from` "+mysql.In(uids)+" and tm between ? and ?", t1, t2).Scan(&nruser3, &nrmsg3); err != nil {
		return nil, err
	}

	// 每个运营人员每天收到多少女性用户登陆消息，回复了多少用户 回复了多少消息
	var nvlogin, nvmsg, nvuser, nvmsg2, nvuser2, nvuser3, nvmsg3 int
	rows2, err2 := msgdb.Query("select distinct `from` from message where `to` "+mysql.In(uids)+" and type in('login') and (tm between ? and ?)", t1, t2)
	if err2 != nil {
		return nil, err2
	}
	defer rows2.Close()
	loginlist := make([]uint32, 0, 0)
	for rows2.Next() {
		var uid uint32
		if err := rows2.Scan(&uid); err != nil {
			return nil, err
		}
		loginlist = append(loginlist, uid)
	}
	nvlogin = len(loginlist)
	//客服发的消息
	if err := msgdb.QueryRow("select count(distinct `to`),count(`to`) from message where `to` "+mysql.In(loginlist)+" and `from` "+mysql.In(uids)+" and tm between ? and ? and type in('text','voice','pic')", t1, t2).Scan(&nvuser, &nvmsg); err != nil {
		return nil, err
	}
	//女性用户回的消息
	if err := msgdb.QueryRow("select count(distinct `from`),count(`from`) from message where `from` "+mysql.In(loginlist)+" and `to` "+mysql.In(uids)+" and tm between ? and ? and type in('text','voice','pic')", t1, t2).Scan(&nvuser2, &nvmsg2); err != nil {
		return nil, err
	}
	//手动发的消息
	if err := mdb.QueryRow("select count(distinct `to`),count(`to`) from manager_msg where `to` "+mysql.In(loginlist)+" and `from` "+mysql.In(uids)+" and tm between ? and ?", t1, t2).Scan(&nvuser3, &nvmsg3); err != nil {
		return nil, err
	}

	// 每个运营人员每天收到多少男性用户注册消息，回复了多少用户 回复了多少消息
	var nanreg, nanmsg, nanuser, nanmsg2, nanuser2, nanmsg3, nanuser3 int
	rows3, err3 := msgdb.Query("select distinct `from` from message where `to` "+mysql.In(uids)+" and type in('menreg') and (tm between ? and ?)", t1, t2)
	if err3 != nil {
		return nil, err3
	}
	defer rows3.Close()
	loginlist2 := make([]uint32, 0, 0)
	for rows3.Next() {
		var uid uint32
		if err := rows3.Scan(&uid); err != nil {
			return nil, err
		}
		loginlist2 = append(loginlist2, uid)
	}
	nanreg = len(loginlist2)
	//客服发的消息
	if err := msgdb.QueryRow("select count(distinct `to`),count(`to`) from message where `to` "+mysql.In(loginlist2)+" and `from` "+mysql.In(uids)+" and tm between ? and ? and type in('visit')", t1, t2).Scan(&nanuser, &nanmsg); err != nil {
		return nil, err
	}
	//男性用户回的消息
	if err := msgdb.QueryRow("select count(distinct `from`),count(`from`) from message where `from` "+mysql.In(loginlist2)+" and `to` "+mysql.In(uids)+" and tm between ? and ? and type in('text','voice','pic')", t1, t2).Scan(&nanuser2, &nanmsg2); err != nil {
		return nil, err
	}
	//手动发的消息
	if err := mdb.QueryRow("select count(distinct `to`),count(`to`) from manager_msg where `to` "+mysql.In(loginlist2)+" and `from` "+mysql.In(uids)+" and tm between ? and ?", t1, t2).Scan(&nanuser3, &nanmsg3); err != nil {
		return nil, err
	}

	// 每个运营人员每天参与多少个圈子，在圈子里发了多少句消息
	var topiccount, topicmsg int
	if err := msgdb.QueryRow("select count(distinct `tag`),count(`tag`) from tag_message where `from` "+mysql.In(uids)+" and type in ('text','voice','pic') and tm between ? and ?", t1, t2).Scan(&topiccount, &topicmsg); err != nil {
		return nil, err
	}
	// 每个运营人员每天给多少用户送礼物、送了多少礼物，收到多少礼物
	var giftuser, giftsend, giftrecv int
	if err := mdb.QueryRow("select count(distinct `t_uid`),count(`t_uid`) from gift_record where uid "+mysql.In(uids)+" and tm between ? and ?", t1, t2).Scan(&giftuser, &giftsend); err != nil {
		return nil, err
	}
	if err := mdb.QueryRow("select count(`uid`) from gift_record where t_uid "+mysql.In(uids)+" and tm between ? and ?", t1, t2).Scan(&giftrecv); err != nil {
		return nil, err
	}

	var nanzhu, nvzhu int
	nanprovlist := make([]string, 0, 0)
	for k, _ := range nanmap {
		nanprovlist = append(nanprovlist, k)
	}
	nvprovlist := make([]string, 0, 0)
	for k, _ := range nvmap {
		nvprovlist = append(nvprovlist, k)
	}
	if err := mdb.QueryRow("select count(user_main.uid) from user_main left join user_detail on user_detail.uid=user_main.uid where (reg_time between ? and ?) and gender=2 and  province "+mysql.In(nanprovlist), t1, t2).Scan(&nvzhu); err != nil {
		return nil, err
	}
	if err := mdb.QueryRow("select count(user_main.uid) from user_main left join user_detail on user_detail.uid=user_main.uid where (reg_time between ? and ?) and gender=1 and  province "+mysql.In(nvprovlist), t1, t2).Scan(&nanzhu); err != nil {
		return nil, err
	}
	sqry := "replace into manager_stat(date,uid,tm,nanzhu,nvzhu,msgcount,msguser,msgcount2,msguser2,nvlogin,nvmsg,nvuser,nvmsg2,nvuser2,nvmsg3,nvuser3,nvreg,nrmsg,nruser,nrmsg2,nruser2,nrmsg3,nruser3,nanreg,nanmsg,nanuser,nanmsg2,nanuser2,nanmsg3,nanuser3,topiccount,topicmsg,giftuser,giftsend,giftrecv)values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	tm := utils.Now
	mdb.Exec(sqry, date, aid, tm, nanzhu, nvzhu, msgcount, msguser, msgcount2, msguser2, nvlogin, nvmsg, nvuser, nvmsg2, nvuser2, nvmsg3, nvuser3, nvreg, nrmsg, nruser, nrmsg2, nruser2, nrmsg3, nruser3, nanreg, nanmsg, nanuser, nanmsg2, nanuser2, nanmsg3, nanuser3, topiccount, topicmsg, giftuser, giftsend, giftrecv)
	return
}

func AdminUidTongji(aid uint32, date string) (info map[string]interface{}, e error) {
	sl := strings.Split(date, "-")
	if len(sl) != 3 {
		return nil, errors.New("日期转换失败")
	}
	y, err := utils.ToInt(sl[0])
	if err != nil {
		return nil, err
	}
	m, err := utils.ToInt(sl[1])
	if err != nil {
		return nil, err
	}
	d, err := utils.ToInt(sl[2])
	if err != nil {
		return nil, err
	}
	var sdd time.Month
	sdd = time.Month(m)
	// t1 := time.Date(y, sdd, d, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(y, sdd, d, 23, 59, 59, 0, time.UTC)
	var ErrNoRows = errors.New("sql: no rows in result set")
	var nanzhu, nvzhu, msgcount, msguser, msgcount2, msguser2, nvlogin, nvmsg, nvuser, nvmsg2, nvuser2, nvmsg3, nvuser3, nvreg, nrmsg, nruser, nrmsg2, nruser2, nrmsg3, nruser3, nanreg, nanmsg, nanuser, nanmsg2, nanuser2, nanmsg3, nanuser3, topiccount, topicmsg, giftuser, giftsend, giftrecv int
	var tm string
	sqry := "select tm,nanzhu,nvzhu,msgcount,msguser,msgcount2,msguser2,nvlogin,nvmsg,nvuser,nvmsg2,nvuser2,nvmsg3,nvuser3,nvreg,nrmsg,nruser,nrmsg2,nruser2,nrmsg3,nruser3,nanreg,nanmsg,nanuser,nanmsg2,nanuser2,nanmsg3,nanuser3,topiccount,topicmsg,giftuser,giftsend,giftrecv from manager_stat where date=? and uid=?"
	err = mdb.QueryRow(sqry, date, aid).Scan(&tm, &nanzhu, &nvzhu, &msgcount, &msguser, &msgcount2, &msguser2, &nvlogin, &nvmsg, &nvuser, &nvmsg2, &nvuser2, &nvmsg3, &nvuser3, &nvreg, &nrmsg, &nruser, &nrmsg2, &nruser2, &nrmsg3, &nruser3, &nanreg, &nanmsg, &nanuser, &nanmsg2, &nanuser2, &nanmsg3, &nanuser3, &topiccount, &topicmsg, &giftuser, &giftsend, &giftrecv)
	if err != nil {
		if err.Error() == ErrNoRows.Error() {
			go AdminTj(aid, date)
		} else {
			return nil, err
		}
	}
	t, errt := utils.ToTime(tm)
	if errt != nil {
		return nil, err
	}
	if t.Unix() > t2.Unix() { //最后的

	} else {
		if time.Since(t).Minutes() > 10 {
			go AdminTj(aid, date)
		}
	}

	info = make(map[string]interface{})
	info["mencount"] = nanzhu
	info["womencount"] = nvzhu
	info["msgcount"] = msgcount
	info["msguser"] = msguser
	info["msgcount2"] = msgcount2
	info["msguser2"] = msguser2
	info["nvlogin"] = nvlogin
	info["nvmsg"] = nvmsg
	info["nvuser"] = nvuser
	info["nvmsg2"] = nvmsg2
	info["nvuser2"] = nvuser2
	info["nvmsg3"] = nvmsg3
	info["nvuser3"] = nvuser3
	info["nvreg"] = nvreg
	info["nrmsg"] = nrmsg
	info["nruser"] = nruser
	info["nrmsg2"] = nrmsg2
	info["nruser2"] = nruser2
	info["nrmsg3"] = nrmsg3
	info["nruser3"] = nruser3
	info["nanreg"] = nanreg
	info["nanmsg"] = nanmsg
	info["nanuser"] = nanuser
	info["nanmsg2"] = nanmsg2
	info["nanuser2"] = nanuser2
	info["nanmsg3"] = nanmsg3
	info["nanuser3"] = nanuser3
	info["topiccount"] = topiccount
	info["topicmsg"] = topicmsg
	info["giftuser"] = giftuser
	info["giftsend"] = giftsend
	info["giftrecv"] = giftrecv
	return
}

func ManagerTJ(msgid uint64, from, to, manager_id uint32, tp, tag string) (e error) {
	_, err := mdb.Exec("insert into manager_msg (msgid,`from`,`to`,manager_id,`type`,tag)values(?,?,?,?,?,?)", msgid, from, to, manager_id, tp, tag)
	return err
}

func makeUsersInfo(items []redis.ItemScore) (users []User, e error) {
	//取出当前请求的uid集合
	uids := make([]uint32, 0, len(items))
	for _, u := range items {
		if uid, e := utils.ToUint32(u.Key); e != nil {
			return nil, e
		} else {
			uids = append(uids, uid)
		}
	}
	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, e
	}
	users = make([]User, 0, len(uids))
	for i, item := range items {
		if ui := uinfos[uids[i]]; ui != nil {
			users = append(users, User{uids[i], ui.Nickname, ui.Avatar, time.Unix(int64(item.Score), 0)})
		}
	}

	return users, nil
}

func SetBirthday(uid uint32, tm time.Time) (e error) {
	_, e = mdb.Exec("update user_detail set birthday=?,birthdaystat=? where uid=?", tm, 1, uid)
	if e != nil {
		return e
	}
	message.SendMessage(message.BIRTHDAY_CHANGE, message.Online{uid}, nil)
	user_overview.ClearUserObjects(uid)
	return
}

func SetWorkArea(uid uint32, placeid, name, address string, lat, lng float64) (e error) {
	if placeid != "" {
		_, e = mdb.Exec("insert into building (placeid,`name`,address,lat,lng)values(?,?,?,?,?) on duplicate key update usecount=usecount+1", placeid, name, address, lat, lng)
		if e != nil {
			return e
		}
	}
	_, e = mdb.Exec("update user_detail set workarea=?,workplaceid=? where uid=?", name, placeid, uid)
	if e != nil {
		return e
	}
	return
}

func SetWorkunit(uid uint32, placeid, name, address string, lat, lng float64) (e error) {
	if placeid != "" {
		_, e = mdb.Exec("insert into building (placeid,`name`,address,lat,lng)values(?,?,?,?,?) on duplicate key update usecount=usecount+1", placeid, name, address, lat, lng)
		if e != nil {
			return e
		}
	}
	_, e = mdb.Exec("update user_detail set workunit=?,unitplaceid=? where uid=?", name, placeid, uid)
	if e != nil {
		return e
	}
	return
}

func CheckVerify(uid uint32, key string) (e error) {
	var nickname, avatar, homeprovince, homecity, workarea, job, trade string
	var height, mustcomplete int
	e = mdb.QueryRowFromMain("select nickname,avatar,height,homeprovince,homecity,workarea,job,trade,mustcomplete from user_detail where uid=?", uid).Scan(&nickname, &avatar, &height, &homeprovince, &homecity, &workarea, &job, &trade, &mustcomplete)
	if mustcomplete == 0 {
		if nickname != "" && avatar != "" && homeprovince != "" && homecity != "" && workarea != "" && job != "" && trade != "" && height != 0 {
			AddVerifyRecord(uid, "all")
			if _, e := mdb.Exec("update user_detail set mustcomplete=1 where uid=?", uid); e != nil {
				return e
			}
			stat.Append(uid, stat.ACTION_MUST_COMPLETE, nil)
		}
	} else {
		if key == "avatar" || key == "nickname" || key == "aboutme" {
			AddVerifyRecord(uid, key)
		}
	}
	return
}
