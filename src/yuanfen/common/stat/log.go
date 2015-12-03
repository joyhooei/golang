package stat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	"yf_pkg/mysql"
	"yf_pkg/utils"
	"yuanfen/common/user"
	"yuanfen/yf_service/cls/common"
)

//-------------------------------------------------//
type UserInfo struct {
	platform    int
	province    string
	ver         string
	city        string
	channel     string
	sub_channel string
	gender      int
}

type Record struct {
	uid    uint32
	action int
	data   map[string]interface{}
	tm     time.Time
}

var mdb *mysql.MysqlDB
var ulog, dlog Logger

type Logger struct {
	sdb         *mysql.MysqlDB
	records     chan Record
	cache       map[uint32]UserInfo
	getUserInfo func(*Logger, uint32) (UserInfo, error)
}

//-------------------------------------------------//

//------------------------------------------------//

func Init(statDB *mysql.MysqlDB, dstatDB *mysql.MysqlDB, mainDB *mysql.MysqlDB) {
	ulog.sdb = statDB
	dlog.sdb = dstatDB
	mdb = mainDB
	ulog.records = make(chan Record, 10000)
	ulog.cache = make(map[uint32]UserInfo)
	ulog.getUserInfo = getUserInfo
	dlog.records = make(chan Record, 10000)
	dlog.cache = make(map[uint32]UserInfo)
	dlog.getUserInfo = getDevUserInfo
	go ulog.insert()
	go dlog.insert()
}

//添加注册用户的行为日志
func Append(uid uint32, action int, data map[string]interface{}) {
	if !user.IsKfUser(uid) {
		if len(ulog.records) < 8000 {
			ulog.records <- Record{uid, action, data, utils.Now}
		}
	}
}

//添加设备的行为日志，要求uid之前已经上传过用户信息
func SimpleAppendDev(uid uint32, action int, data map[string]interface{}) {
	if len(dlog.records) < 8000 {
		dlog.records <- Record{uid, action, data, utils.Now}
	}
}

//没有uid的用这个接口添加日志，该接口的uid由客户端生成，每个设备保持不变
func AppendDev(uid uint32, action int, data map[string]interface{}, platform int, ver string, province string, city string, channel string, sub_channel string) {
	uinfo, ok := dlog.cache[uid]
	if !ok {
		s := "select platform,ver,channel,sub_channel,province,city from UserInfo where uid=?"
		e := dlog.sdb.QueryRow(s, uid).Scan(&uinfo.platform, &uinfo.ver, &uinfo.channel, &uinfo.sub_channel, &uinfo.province, &uinfo.city)
		switch e {
		case sql.ErrNoRows:
			s := "insert into UserInfo(uid,ver,platform,province,city,channel,sub_channel)values(?,?,?,?,?,?,?)"
			_, err := dlog.sdb.Exec(s, uid, ver, platform, province, city, channel, sub_channel)
			if err != nil {
				return
			}
			uinfo.platform, uinfo.ver, uinfo.province, uinfo.city, uinfo.channel, uinfo.sub_channel, uinfo.gender = platform, ver, province, city, channel, sub_channel, common.GENDER_MAN
			dlog.cache[uid] = uinfo
		case nil:
			uinfo.gender = common.GENDER_MAN
			dlog.cache[uid] = uinfo
		default:
			return
		}
	}
	if len(dlog.records) < 8000 {
		dlog.records <- Record{uid, action, data, utils.Now}
	}
}

func (l *Logger) insert() {
	for {
		l.insertHelper()
	}
}

func (l *Logger) insertHelper() {
	sql := "insert into Actions(type,ver,platform,province,city,channel,sub_channel,uid,gender,data,tm)values(?,?,?,?,?,?,?,?,?,?,?)"
	stmt, e := l.sdb.PrepareExec(sql)
	if e != nil {
		fmt.Println("connect to stat db error :", e.Error())
		time.Sleep(3 * time.Second)
		return
	}
	defer stmt.Close()
	for record := range l.records {
		if interval := utils.Now.Sub(record.tm); interval < 10*time.Second {
			time.Sleep(10*time.Second - interval)
		}
		uinfo, e := l.getUserInfo(l, record.uid)
		if e != nil {
			fmt.Println("getUserInfo error :", e.Error())
			continue
		}
		var b []byte = []byte{}
		if record.data != nil {
			b, e = json.Marshal(record.data)
			if e != nil {
				fmt.Println("json marshal error :", e.Error())
				continue
			}
		}
		stmt.Exec(record.action, uinfo.ver, uinfo.platform, uinfo.province, uinfo.city, uinfo.channel, uinfo.sub_channel, record.uid, uinfo.gender, b, record.tm)
	}
}

func getUserInfo(l *Logger, uid uint32) (UserInfo, error) {
	uinfo, ok := l.cache[uid]
	if !ok {
		sql := "select user_client_type,gender,ver,channel_uid,channel_sid,province,city from user_main,user_detail where user_main.uid=user_detail.uid and user_main.uid=?"
		e := mdb.QueryRow(sql, uid).Scan(&uinfo.platform, &uinfo.gender, &uinfo.ver, &uinfo.channel, &uinfo.sub_channel, &uinfo.province, &uinfo.city)
		if e != nil {
			return uinfo, e
		}
		if uinfo.gender != common.GENDER_BOTH {
			l.cache[uid] = uinfo
		}
	}
	return uinfo, nil
}
func getDevUserInfo(l *Logger, uid uint32) (UserInfo, error) {
	uinfo, ok := l.cache[uid]
	if !ok {
		s := "select platform,ver,channel,sub_channel,province,city from UserInfo where uid=?"
		e := dlog.sdb.QueryRow(s, uid).Scan(&uinfo.platform, &uinfo.ver, &uinfo.channel, &uinfo.sub_channel, &uinfo.province, &uinfo.city)
		if e != nil {
			return uinfo, e
		}
		l.cache[uid] = uinfo
	}
	return uinfo, nil
}
