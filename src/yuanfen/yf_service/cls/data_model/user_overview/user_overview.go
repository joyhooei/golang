package user_overview

import (
	"fmt"
	"strings"
	"time"
	"yf_pkg/cachedb"
	"yf_pkg/format"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/push"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
)

var mdb *mysql.MysqlDB
var cache *redis.RedisPool
var cachedb2 *cachedb.CacheDB
var sdb *mysql.MysqlDB
var rdb *redis.RedisPool

var mlog *log.MLogger

func Init(env *service.Env) {
	cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	cachedb2 = env.ModuleEnv.(*cls.CustomEnv).CacheDB
	sdb = env.ModuleEnv.(*cls.CustomEnv).SortDB
	rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	mlog = env.Log
}

func GetUserObject(uid uint32) (uinfo *UserViewItem, e error) {
	uinfo = &UserViewItem{}
	e = cachedb2.Get(uid, uinfo)
	return
}

//获取用户简略信息
// 包含key gender,username,isvip,grade,nickname,avatar,province,city,birthday,Avatarlevel,Localtag,Localtagtm,certify_phone, certify_video, certify_idcard, honesty_level,CertifyLevel
func GetUserObjects(uidlist ...uint32) (obj map[uint32]*UserViewItem, e error) {
	if len(uidlist) == 0 {
		return
	}
	users := make(map[interface{}]cachedb.DBObject)
	for _, v := range uidlist {
		users[utils.Uint32ToString(v)] = nil
	}
	e = cachedb2.GetMap(users, NewUserViewItem)
	obj1 := make(map[uint32]*UserViewItem)
	if e != nil {
		return nil, e
	} else {
		for id, user := range users {
			uid, e := utils.ToUint32(id)
			if e != nil {
				return nil, e
			}
			if user != nil {
				obj1[uid] = user.(*UserViewItem)
			}
		}
	}
	return obj1, nil
}

func ClearUserObjects(uid uint32) (e error) {
	return cachedb2.ClearCache(NewUserViewItem(uid))
}

//add by jiatao
func IsOnline(uids ...uint32) (users map[uint32]bool, e error) {
	sql := "select uid,tm from user_online where uid" + mysql.In(uids)
	res, e := mdb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer res.Close()
	users = make(map[uint32]bool)
	for _, uid := range uids {
		users[uid] = false
	}
	for res.Next() {
		var uid uint32
		var tmStr string
		if e := res.Scan(&uid, &tmStr); e != nil {
			return nil, e
		}
		tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
		if e != nil {
			return nil, e
		}
		users[uid] = tm.After(utils.Now)
	}
	return users, nil
}

//登陆时间
func LoginTime(uid uint32) (tm time.Time, e error) {
	sql := "select tm from user_online where uid =?"
	var tmStr string
	e = mdb.QueryRow(sql, uid).Scan(&tmStr)
	if e != nil {
		return tm, e
	}
	tm, e = utils.ToTime(tmStr, format.TIME_LAYOUT_1)
	return tm, e
}

//查询操作系统
func SystemInfo(uid uint32) (sysid int, e error) {
	sysid, err := redis.Int(cache.Get(redis_db.CACHE_USER_SYSTEM, uid))
	switch err {
	case nil:
		return sysid, nil
	case redis.ErrNil:
		sql := "select sysinfo from user_main where uid=?"
		if e := mdb.QueryRow(sql, uid).Scan(&sysid); e != nil {
			return push.SYSTEM_OTHER, e
		}
		e = cache.SetEx(redis_db.CACHE_USER_SYSTEM, uid, 3600, sysid)
		// _, e := con.Do("SETEX", uid, 86400, sysid)
		return sysid, e
	default:
		return push.SYSTEM_OTHER, err
	}
}

//更新操作系统
func SetSystemInfo(uid uint32, sysid int) (e error) {
	var user_client_type int
	switch sysid {
	case 2, 3, 4:
		user_client_type = 2
	case 1:
		user_client_type = 3
	default:
		user_client_type = 0
	}
	_, err := mdb.Exec("update user_main set sysinfo=?,user_client_type=? where uid=?", sysid, user_client_type, uid)
	if err != nil {
		return err
	}
	e = cache.SetEx(redis_db.CACHE_USER_SYSTEM, uid, 3600, sysid)
	// con := cache.GetWriteConnection(redis_db.CACHE_USER_SYSTEM)
	// defer con.Close()
	// _, err = con.Do("SETEX", uid, 86400, sysid)
	return e
}

//验证用户
func UserValid(uid uint32, key string) (valid bool, e error) {
	skey, err := redis.String(cache.Get(redis_db.CACHE_USER_VALID, uid))
	switch err {
	case nil:
		return key == skey, nil
	case redis.ErrNil:
		sql := "select sid from user_main where uid=? and `stat`<>5"
		var sid string
		if e := mdb.QueryRow(sql, uid).Scan(&sid); e != nil {
			return false, e
		}
		// fmt.Println("UserValid " + key + "   " + sid)
		e = cache.SetEx(redis_db.CACHE_USER_VALID, uid, 3600, sid)
		return (key == sid), e
	default:
		fmt.Println("UserValid error " + key + "   " + err.Error())
		return false, err
	}
}

//更新密码的缓存
func SetUserPassCache(uid uint32, key string) (e error) {
	e = cache.SetEx(redis_db.CACHE_USER_VALID, uid, 3600, key)
	return e
}

//删除密码的缓存
func DelUserPassCache(uid uint32) (e error) {
	e = cache.Del(redis_db.CACHE_USER_VALID, uid)
	return e
}

func SysInfoNoErr(uid uint32) int {
	sysid, _ := SystemInfo(uid)
	return sysid
}

//查询版本号
func VerInfo(uid uint32) (ver, channel string, e error) {
	sql := "select ver,channel from user_main  where uid=?"
	if e := mdb.QueryRow(sql, uid).Scan(&ver, &channel); e != nil {
		return "", "", e
	}
	return
}

/*
//查询是否完成了必填项

参数说明：
	uid : 用户uid
返回值：
	ifcomplete: 是否完成必填项
*/
func CompleteMust(uid uint32) (ifcomplete bool, e error) {
	uinfo, err := GetUserObjectByUid(uid)
	if err != nil {
		return false, err
	}
	if uinfo.Avatar == "" || uinfo.Height == 0 || uinfo.Homeprovince == "" || uinfo.WorkPlaceName == "" || uinfo.Trade == "" || uinfo.Job == "" {
		return false, nil
	}
	return true, nil
}

func GetAge(tm time.Time) int {
	return utils.Now.Year() - tm.Year()
}

func expToGrade(exp int) int {
	return 1
}

func GetUserObjectNoCache(uid uint32) (u *UserViewItem, e error) {
	// var birthday string
	u = new(UserViewItem)
	var mtag string
	e = mdb.QueryRowFromMain("select a.uid,gender,nickname,avatar,province,city,height,aboutme,avatarlevel,tag,stat,star,phonestat,certify_video,homeprovince,homecity,workarea,workunit,job,school,trade,age from user_main a LEFT JOIN user_detail b on a.uid=b.uid where a.uid=?", uid).Scan(&u.Uid, &u.Gender, &u.Nickname, &u.Avatar, &u.Province, &u.City, &u.Height, &u.Aboutme, &u.Avatarlevel, &mtag, &u.Stat, &u.Star, &u.CertifyPhone, &u.CertifyVideo, &u.Homeprovince, &u.Homecity, &u.WorkPlaceName, &u.Workunit, &u.Job, &u.School, &u.Trade, &u.Age)
	if e != nil {
		return nil, e
	}
	u.Tag = strings.Split(mtag, ",")
	u.Ltag = NewLocaltag()
	// u.Birthday, _ = utils.ToTime(birthday, format.TIME_LAYOUT_1)

	return

}

// 获取单个用户信息
func GetUserObjectByUid(uid uint32) (user *UserViewItem, e error) {
	u_map, err := GetUserObjects([]uint32{uid}...)
	if err != nil {
		return nil, err
	}
	if v, ok := u_map[uid]; ok {
		user = v
	}
	return user, nil
}

/*
根据uids拼接必要的用户信息
*/
func GenUserInfo(uids []uint32, flag int) (res []map[string]interface{}, e error) {
	res = make([]map[string]interface{}, 0, len(uids))
	if len(uids) <= 0 {
		return
	}
	m, e := GetUserObjects(uids...)
	if e != nil {
		return
	}
	for _, uid := range uids {
		r := make(map[string]interface{})
		if u, ok := m[uid]; ok {
			r["uid"] = u.Uid
			r["nickname"] = u.Nickname
			r["avatar"] = u.Avatar
			if flag == 1 {
				r["gender"] = u.Gender
			}
		}
		res = append(res, r)
	}
	return
}

/*
获取某用户最近动态图片集合和总动态数
参数说明：
	uid : 用户uid
	isSelf: 是否为本人查询
返回值：
	n: 动态总条数
	pics: 最近动态图片列表，字符串数组返回
*/
func GetUserLastDynamicPic(uid uint32, isSelf bool) (n uint64, pics []string, e error) {
	status_query := " status =0  "
	if isSelf {
		status_query = " status != 1 "
	}
	s := "select pic from dynamics where uid = ? and pic != '' and " + status_query + "  order by id desc limit 5 "
	rows, e := mdb.Query(s, uid)
	if e != nil {
		return
	}
	defer rows.Close()
	pics = make([]string, 0, 10)
	for rows.Next() {
		var pic_str string
		if e = rows.Scan(&pic_str); e != nil {
			return
		}
		if strings.TrimSpace(pic_str) != "" {
			arr := strings.Split(pic_str, ",")
			pics = append(pics, arr...)
			if len(pics) >= 5 {
				break
			}
		}
	}
	s2 := "select count(*) from dynamics where uid = ? and " + status_query
	e = mdb.QueryRow(s2, uid).Scan(&n)
	return
}

/*
设置用户动态页面的背景图片
uid:用户uid
url:新图片
*/
func SetUserDynamicImg(uid uint32, url string) (e error) {
	s := "update user_detail set dynamic_img = ? where uid = ?"
	_, e = mdb.Exec(s, url, uid)
	return
}

// 检测用户是否可以显示动态或者评论
func CheckUserDynamicStatus(uids []uint32) (m map[uint32]bool, e error) {
	m = make(map[uint32]bool)
	if len(uids) <= 0 {
		return
	}
	um, e := GetUserObjects(uids...)
	if e != nil {
		return
	}
	for _, v := range um {
		if v.Avatarlevel == 9 {
			m[v.Uid] = true
		}
	}
	return
}

// 检测单个用户是否可以显示动态或者评论
func CheckUserDynamicStatusByUid(uid uint32) (b bool, e error) {
	um, e := CheckUserDynamicStatus([]uint32{uid})
	if e != nil {
		return
	}
	_, b = um[uid]
	return
}
