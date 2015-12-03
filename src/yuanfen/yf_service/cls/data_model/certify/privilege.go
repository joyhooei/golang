package certify

import (
	"encoding/json"
	"errors"
	"fmt"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/user"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/notify"
)

//level,item,name,num,interval,tips
type HonestyPri struct {
	Level int
	Item  string
	Name  string
	Num   int
	Flag  int
	Tips  string
}

type Pri struct {
	Can bool       `json:"can"` // 是否具有该权限,true 有，false 无
	Bal int        `json:"bal"` // 特权余额
	Msg string     `json:"msg"` // 弹框文字
	But notify.But `json:"but"` // 按钮
}

var mdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool
var mlog *log.Logger

// 1，更具诚信等级获取，2 查询redis每天权限，3.查询redis永远,4.手机认证，5.视频认证，6.身份证认证
var pri_key map[string]int = map[string]int{common.PRI_SEARCH: 5, common.PRI_CONTACT: 4, common.PRI_PHONELOGIN: 4, common.PRI_ONLINE_AWARD: 6, common.PRI_NEARMSG_FILTER: 5, common.PRI_PRIVATE_PHOTOS: 5, common.PRI_SEE_REQUIRE: 6, common.PRI_NEARMSG: 1, common.PRI_BIGIMG: 1, common.PRI_SAYHI: 2, common.PRI_CHAT: 2, common.PRI_PURSUE: 2, common.PRI_FOLLOW: 3, common.PRI_SEEINFO_NOTIFY: 1, common.PRI_INVITE_NOTIFY: 1, common.PRI_AWARD_PHONECARD: 4}

var pri_get_arr []string = []string{common.PRI_GET_AVATAR, common.PRI_GET_PHOTOS, common.PRI_GET_INFO, common.PRI_GET_PHONE, common.PRI_GET_VIDEO, common.PRI_GET_IDCARD}

//各个特权弹框限制
var pri_tips_map map[string]map[string]string = map[string]map[string]string{common.PRI_SAYHI: nil}

// 在main。go中初始化
func Init(env *cls.CustomEnv, conf service.Config) {
	fmt.Println("init certify")
	mdb = env.MainDB
	rdb = env.MainRds
	cache = env.CacheRds
	l, err := log.New2(conf.LogDir+"/scertify.log", 10000, conf.LogLevel)
	if err != nil {
		fmt.Println("初始化日志error:", err.Error())
		return
	}
	l.Append("初始化日志成功")
	mlog = l
	fillTips()
}

/*
检测某用户是否具有某项特权
参数：
	uid: 用户uid
	pri: 权限key，对应于common 常量定义
返回值：
	canUse: 是否可以使用该特权
	p: Pri 结构，详情查看http://120.131.64.91:8182/pkg/yuanfen/yf_service/cls/data_model/certify/#Pri
*/
func CheckPrivilege(uid uint32, pri string) (canUse bool, p Pri) {
	m, _, e := GetPrivilege(uid, pri)
	if e != nil {
		mlog.AppendObj(e, "CheckPrivilege --- is error ", uid, pri)
		return false, p
	}
	if v, ok := m[pri]; ok {
		canUse = v.Can
		p = v
	}
	return
}

/*
获取某用户所有的权限
参数：
	uid: 用户uid
	pris:【可选】权限key，对应于common 常量定义,不传则默认查询用户的所有权限
返回值：
	r: 返回查询的该用户权限，如：
	{
		pri_bigimg:{
			can:fasle,	// 是否具有该特权
			bal:0,		// 权限余额
			msg:"xxx",	// 弹出框提示内容
			but:{		// 弹出框按钮执行操作
				tip:"查看",   		//按钮上的提示
				cmd:"cmd_idcard_pri",	//按钮执行cmd
				def:true              	//是否为默认操作
				data:{}			//cmd所需参数
		}
	}
	cmd 对应关系详见：http://120.131.64.91:8082/pkg/yuanfen/yf_service/cls/notify/#pkg-constants
*/
func GetPrivilege(uid uint32, pris ...string) (r map[string]Pri, u *user_overview.UserViewItem, e error) {
	r = make(map[string]Pri)
	u, e = user_overview.GetUserObjectNoCache(uid)
	if e != nil || u == nil {
		return r, u, errors.New("获取用户失败")
	}
	if len(pris) == 0 { // 查询该用户所有权限
		all_pri := make([]string, 0, len(pri_key))
		for k, _ := range pri_key {
			all_pri = append(all_pri, k)
		}
		pris = all_pri
	}
	m := getPriKeyArr(pris)

	// 如果是客服帐号，则将其设置为6星用户的权限
	if user.IsKfUser(uid) {
		u.HonestyLevel = 6
		//	mlog.AppendObj(nil, " GetPrivilege is kf add honsty_level to 6")
	}

	pri_m, e := GetHonestyByLevel(u.HonestyLevel)
	if e != nil {
		return r, u, e
	}
	//	mlog.AppendObj(nil, "---1----", m)
	for t, arr := range m {
		res, e := getHonestyByType(u, arr, t)
		if e != nil {
			return r, u, e
		}
		for k, v := range res {
			var can bool
			var msg, cmd_str, tip string
			data := make(map[string]interface{})
			tm, isok := pri_tips_map[k]
			// 目前消耗的点数都是1，所以只需要判断值是否大于0就行了
			if v > 0 {
				can = true
				if u.Avatarlevel == -1 && (k == common.PRI_SAYHI || k == common.PRI_CHAT || k == common.PRI_FOLLOW || k == common.PRI_PURSUE || k == common.PRI_INVITE_NOTIFY || k == common.PRI_SEEINFO_NOTIFY || k == common.PRI_NEARMSG) { //当如果用户头像审核未通过时，直接收回用户特权
					can = false
					if isok {
						cmd_str = notify.CMD_USER_INFO
						tip = "去上传头像"
						msg = tm[no_avatar]
						data["uid"] = uid
					}
				}
			} else {
				if u.Avatarlevel == -1 {
					cmd_str = notify.CMD_USER_INFO
					tip = "去上传头像"
					msg = tm[no_avatar]
					data["uid"] = uid

				} else {
					if isok {
						msg = tm[no_can]
						if h, ok := pri_m[k]; ok && h.Num > 0 { // 可以，但余额不足
							msg = fmt.Sprintf(tm[no_bal], h.Num)
						}
						cmd_str = tm[cmd]
						tip = tm[tip_msg]
					}
				}
			}
			but := notify.GetBut(tip, cmd_str, true, data)
			r[k] = Pri{can, v, msg, but}
		}
	}
	//	mlog.AppendObj(e, "---end----", r)
	return
}

// 根据pris获取获取类型分类
func getPriKeyArr(pris []string) (m map[int][]string) {
	m = make(map[int][]string)
	for _, pri := range pris {
		if t, ok := pri_key[pri]; ok {
			arr := make([]string, 0, 5)
			if v, ok := m[t]; ok {
				arr = v
			}
			arr = append(arr, pri)
			m[t] = arr
		}
	}
	return
}

// 根据不同类型获取不同值
func getHonestyByType(u *user_overview.UserViewItem, pris []string, t int) (r map[string]int, e error) {
	r = make(map[string]int)
	if len(pris) <= 0 {
		return r, errors.New("pris[] is nil")
	}
	// 根据不类型，获取值1，更具诚信等级获取，2 查询redis每天权限，3.查询redis永远,4.手机认证，5.视频认证，6.身份证认证
	switch t {
	case 1:
		m, e := GetHonestyByLevel(u.HonestyLevel)
		if e != nil {
			return nil, e
		}
		for _, pri := range pris {
			if h, ok := m[pri]; ok {
				r[pri] = h.Num
			} else if pri == common.PRI_SEEINFO_NOTIFY || pri == common.PRI_INVITE_NOTIFY { // 特殊处理，3星级可发
				r[pri] = 0
				if u.HonestyLevel >= 3 {
					r[pri] = 1
				}
			}
		}

	case 2, 3:
		flag := 0
		if t == 2 {
			flag = 1
		}
		res, e := GetPriFromRedis(u, pris, flag)
		if e != nil {
			return r, e
		}
		m, e := GetHonestyByLevel(u.HonestyLevel)
		if e != nil {
			return nil, e
		}
		for k, v := range res {
			// 余额需要用配置值减去当前使用值的差值
			if h, ok := m[k]; ok {
				bal := h.Num - v
				if bal < 0 {
					bal = 0
				}
				r[k] = bal
			}
		}
	case 4, 5, 6:
		num := 0
		if (t == 4 && u.CertifyPhone == 1) || (t == 5 && u.CertifyVideo == 1) || (t == 6 && u.CertifyIDcard == 1) {
			num = 1
		}
		for _, pri := range pris {
			r[pri] = num
		}
	}
	return r, nil
}

// 获取诚信值配置
func GetAllHonesty() (m map[int][]HonestyPri, e error) {
	m = make(map[int][]HonestyPri)
	if exists, arr, e := readHonestyConifCache(); exists && e == nil {
		return genAllHonestyMap(arr), nil
	}
	s := "select level,item,name,num,flag,tips from honesty_pri_config"
	rows, e := mdb.Query(s)
	if e != nil {
		return m, e
	}
	defer rows.Close()
	arr := make([]HonestyPri, 0, 36)
	for rows.Next() {
		var h HonestyPri
		if e := rows.Scan(&h.Level, &h.Item, &h.Name, &h.Num, &h.Flag, &h.Tips); e != nil {
			return m, e
		}
		arr = append(arr, h)
	}
	m = genAllHonestyMap(arr)
	if e := writeHonestyConfigCache(arr); e != nil {
		return nil, e
	}
	return m, nil
}

//生成诚信配置map
func genAllHonestyMap(a []HonestyPri) (m map[int][]HonestyPri) {
	m = make(map[int][]HonestyPri)
	if len(a) <= 0 {
		return
	}
	for _, h := range a {
		arr := make([]HonestyPri, 0, 10)
		if v, ok := m[h.Level]; ok {
			arr = v
		}
		arr = append(arr, h)
		m[h.Level] = arr
	}
	return
}

//根据等级获取诚信特权配置
func GetHonestyByLevel(level int) (m map[string]HonestyPri, e error) {
	m = make(map[string]HonestyPri)
	am, e := GetAllHonesty()
	if e != nil {
		return m, e
	}
	if v, ok := am[level]; ok {
		for _, h := range v {
			m[h.Item] = h
		}
	}
	return m, nil
}

// 从redis获取用户特权
func GetPriFromRedis(u *user_overview.UserViewItem, pris []string, flag int) (r map[string]int, e error) {
	key := utils.Uint32ToString(u.Uid)
	if flag == 0 {
		key = "f_" + utils.Uint32ToString(u.Uid)
	}
	r = make(map[string]int)
	if len(pris) <= 0 {
		return
	}
	rcon := rdb.GetWriteConnection(redis_db.REDIS_USER_PRI)
	defer rcon.Close()
	for _, pri := range pris {
		exit, e := redis.Int(rcon.Do("HEXISTS", key, pri))
		if e != nil {
			return r, e
		}
		r[pri] = 0
		if exit == 1 {
			// 如果该字段存在，则直接获取该值
			num, e := redis.Int(rcon.Do("HGET", key, pri))
			if e != nil {
				return r, e
			}
			r[pri] = num
		}
	}
	return
}

/*
使用特权接口

参数：
	uid: 用户uid , pri: 使用权限key , result : 返回结果，将权限自动填充到http的result中，push的msg中
*/
func UsePrivilege(uid uint32, pri string, result map[string]interface{}) (e error) {
	//1.判断是否可以使用该权限
	balance, _, e := GetPrivilege(uid, pri)
	if e != nil {
		return e
	}
	p, ok := balance[pri]
	if !ok {
		return errors.New("has no this pri")
	}
	if !p.Can {
		return errors.New(fmt.Sprintf("has no this pri,balance:%v", p.Bal))
	}
	u, e := user_overview.GetUserObjectNoCache(uid)
	if e != nil {
		return e
	}
	t, ok := pri_key[pri]
	if !ok {
		return errors.New(fmt.Sprintf("pri key %v is no foud", pri))
	}
	// 如果为redis ，需要修改值
	if t == 2 || t == 3 {
		key := utils.Uint32ToString(u.Uid)
		if t == 3 {
			key = "f_" + utils.Uint32ToString(u.Uid)
		}
		rcon := rdb.GetWriteConnection(redis_db.REDIS_USER_PRI)
		defer rcon.Close()
		if _, e := rcon.Do("HINCRBY", key, pri, 1); e != nil {
			return e
		}
		if t == 2 {
			_, tm_str := utils.TmLime("today")
			tm, e := utils.ToTime(tm_str)
			if e != nil {
				return e
			}
			//	mlog.AppendObj(e, "----设置时间---", t, tm_str, tm, tm.Unix())
			if _, e := rcon.Do("EXPIREAT", key, tm.Unix()); e != nil {
				return e
			}
		}
	}
	// 获取该特权的余额
	if result != nil {
		balance, _, e = GetPrivilege(uid, pri)
		if e != nil {
			mlog.AppendObj(e, " userPrivilege get pri is error ", uid, pri)
		}
		result[common.PRI_KEY] = balance
	}
	return
}

/*
诚信升级检测
参数：
	uid: 用户uid
	keys: [可选]诚信获取key，无则检测所有升级的方式,key定义common包下：
		PRI_GET_AVATAR = "pri_get_avatar" // 头像为真实用户头像并审核通过
		PRI_GET_PHOTOS = "pri_get_photos" // 相册至少上传3张生活照
		PRI_GET_INFO   = "pri_get_info"   // 基本必填项填写
		PRI_GET_PHONE  = "pri_get_phone"  // 手机认证
		PRI_GET_VIDEO  = "pri_get_video"  // 视频认证
		PRI_GET_IDCARD = "pri_get_idcard" // 身份证认证
*/
func CheckHonesty(uid uint32, keys ...string) (e error) {
	if len(keys) <= 0 {
		keys = pri_get_arr
	}
	// 获取已获得方式
	m, e := GetHonestyRecord(uid)
	if e != nil {
		return e
	}
	user_overview.ClearUserObjects(uid)
	delete_item := make([]int, 0, 2)
	add_item := make([]int, 0, 2)
	//完成项目1：头像为真实用户头像并审核通过，2：相册至少上传3张生活照，3：基本资料全部填写
	for _, key := range keys {
		switch key {
		case common.PRI_GET_AVATAR: // 头像审核通过
		case common.PRI_GET_PHOTOS: // 照片3张以上
			item := 2
			up, e := checkHonestyPhoto(uid)
			if e != nil {
				return e
			}
			_, ok := m[item] // 是否已经存在
			if up && !ok {   // 可以升级，但是为存在记录,插入新纪录
				add_item = append(add_item, item)
			} else if !up && ok { // 不能升级，但是记录存在
				delete_item = append(delete_item, item)
			}

		case common.PRI_GET_INFO: // 填写资料
			item := 3
			up, e := checkHonestyInfo(uid)
			if e != nil {
				return e
			}
			_, ok := m[item] // 是否已经存在
			if up && !ok {   // 可以升级，但是为存在记录,插入新纪录
				add_item = append(add_item, item)
			} else if !up && ok { // 不能升级，但是记录存在
				delete_item = append(delete_item, item)
			}

		case common.PRI_GET_PHONE: // 手机认证
		case common.PRI_GET_VIDEO: // 视频认证
		case common.PRI_GET_IDCARD: // 身份证认证

		}
	}

	if len(delete_item) > 0 {
		for _, item := range delete_item {
			if e := DeleteHonestRecord(mdb, uid, item); e != nil {
				return e
			}
		}
	}
	if len(add_item) > 0 {
		for _, item := range add_item {
			if e := AddHonestyRecord(mdb, uid, item); e != nil {
				return e
			}
		}
	}
	e = CheckUpGrade(uid, keys...)
	return
}

// 检测是否升级
func CheckUpGrade(uid uint32, keys ...string) (e error) {
	u, e := user_overview.GetUserObjectNoCache(uid)
	if e != nil {
		return e
	}
	m, e := GetHonestyRecord(uid)
	if e != nil {
		return e
	}
	honesty_level := 0
	honesty_level += len(m)
	if u.CertifyPhone == 1 {
		honesty_level += 1
	}
	if u.CertifyVideo == 1 {
		honesty_level += 1
	}
	if u.CertifyIDcard == 1 {
		honesty_level += 1
	}
	if u.Avatarlevel != -1 {
		honesty_level += 1
	}

	//	mlog.AppendObj(nil, "---CheckUpGrade--uid", uid, "old: ", u.HonestyLevel, " new :", honesty_level)
	// 现有等级和初始化等级不相等,需要将其修改honsty_level 和对应特权余额
	if honesty_level != u.HonestyLevel {
		if e := updateHonestyLevel(uid, honesty_level, u.HonestyLevel); e != nil {
			mlog.AppendObj(e, "CheckUpGrade is error ", uid, "new_level:", honesty_level, "old_level:", u.HonestyLevel)
			return e
		}
	}
	// 如果等级不相等,发送推送消息
	if honesty_level != u.HonestyLevel {
		msg := make(map[string]interface{})
		change := honesty_level - u.HonestyLevel
		//		mlog.AppendObj(nil, "CheckHonesty keys  ", keys, change)
		if change > 0 && len(keys) > 0 && (keys[0] == common.PRI_GET_INFO || keys[0] == common.PRI_GET_PHOTOS) {
			msg["type"] = common.MSG_TYPE_HONESTY_CHANGE
			msg["now_level"] = honesty_level
			msg["change"] = change
			hm, _ := GetHonestyByLevel(honesty_level)
			msg["honesty"] = hm
			content := fmt.Sprintf("你的诚信等级升级到%v星", honesty_level)
			not, e := notify.GetNotify(common.USER_SYSTEM, notify.NOTIFY_HONESTY_CHANGE, nil, "", content, uid)
			if e != nil {
				mlog.AppendObj(e, "getNotify is error ", msg)
				return e
			}
			msg[notify.NOTIFY_KEY] = not
		}
		if r, _, e := GetPrivilege(uid); e == nil {
			msg[common.PRI_KEY] = r
		}
		general.SendMsg(common.USER_SYSTEM, uid, msg, "")
	}

	return
}

//获取诚信获取记录
func GetHonestyRecord(uid uint32) (m map[int]int, e error) {
	s := "select item from honesty_record where uid = ?"
	rows, e := mdb.QueryFromMain(s, uid)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	m = make(map[int]int)
	for rows.Next() {
		var item int
		if e := rows.Scan(&item); e != nil {
			return m, e
		}
		m[item] = 1
	}
	return m, nil
}

// 检测填写资料符合升级
func checkHonestyInfo(uid uint32) (ok bool, e error) {
	s := "select aboutme,nickname,star,height,job,tag,interest,income,province,city from user_detail where uid =?"
	var aboutme, job, tag, interest, nickname, province, city string
	var star, height, income int
	if e := mdb.QueryRowFromMain(s, uid).Scan(&aboutme, &nickname, &star, &height, &job, &tag, &interest, &income, &province, &city); e != nil {
		return false, e
	}
	if aboutme != "" && job != "" && tag != "" && interest != "" && nickname != "" && province != "" && city != "" && (star > 0 && height > 0 && income > 0) {
		return true, nil
	}
	return false, nil
}

//检测照片是否符合增加诚信值
func checkHonestyPhoto(uid uint32) (ok bool, e error) {
	s := "select count(*) from user_photo_album where uid =? "
	var num int
	if e := mdb.QueryRowFromMain(s, uid).Scan(&num); e != nil {
		return false, e
	}
	if num >= 3 {
		ok = true
	}
	return
}

// 删除诚信获取记录
func DeleteHonestRecord(db utils.SqlObj, uid uint32, item int) (e error) {
	s := "delete from honesty_record where uid = ? and item = ? "
	_, e = db.Exec(s, uid, item)
	return
}

func AddHonestyRecord(db utils.SqlObj, uid uint32, item int) (e error) {
	s := "insert into honesty_record(uid,item) values(?,?)"
	_, e = db.Exec(s, uid, item)
	return
}

//修改诚信等级
func updateHonestyLevel(uid uint32, new_level, old_level int) (e error) {
	s := "update user_main set honesty_level = ? where uid = ?"
	tx, e := mdb.Begin()
	if e != nil {
		return e
	}
	_, e = tx.Exec(s, new_level, uid)
	if e != nil {
		tx.Rollback()
		return e
	}
	tx.Commit()
	return
}

// 检测认证等级(key 同CheckHonesty接口参数定义)
func CheckCretifyLevel(uid uint32, key ...string) (new_level int, e error) {
	u, e := user_overview.GetUserObjectNoCache(uid)
	if e != nil {
		return 0, e
	}
	certify_level := 0
	if u.CertifyPhone == 1 {
		certify_level += 1
	}
	if u.CertifyVideo == 1 {
		certify_level += 1
	}
	if u.CertifyIDcard == 1 {
		certify_level += 1
	}
	if u.CertifyLevel != certify_level {
		s := "update user_main set certify_level= ? where uid = ?"
		_, e = mdb.Exec(s, certify_level, uid)
		if e != nil {
			return 0, e
		}
	}
	e = CheckHonesty(uid, key...)
	return certify_level, nil
}

func GetMyHonestyInfo(uid uint32) (r map[string]interface{}, e error) {
	r = make(map[string]interface{})
	m, e := GetHonestyRecord(uid)
	if e != nil {
		return nil, e
	}
	ue, e := user_overview.GetUserObjects(uid)
	if e != nil {
		return nil, e
	}
	mp, ok := ue[uid]
	if !ok {
		return nil, errors.New("读取用户信息错误")
	}
	for _, v := range pri_get_arr {
		r[v] = 0
	}
	for i := 1; i <= 3; i++ {
		if j, ok := m[i]; ok {
			if j == 1 {
				r[pri_get_arr[i-1]] = 1
			}
		}
	}
	if mp.Avatarlevel != -1 {
		r[common.PRI_GET_AVATAR] = 1
	}
	r[common.PRI_GET_PHONE] = mp.CertifyPhone
	r[common.PRI_GET_VIDEO] = mp.CertifyVideo
	r[common.PRI_GET_IDCARD] = mp.CertifyIDcard
	r["honesty_level"] = mp.HonestyLevel
	return
}

/*
检测是否能够发送聊天

参数:
	fuid:发送方，tuid:接收方
返回值:
	canUse: 是否可以使用该特权
	value: 余额
	msg: 错误提示
	no_avatar: 无权限原因，0 不是无头像,1 无头像
*/
func CheckChatPrivilege(fuid, tuid uint32) (canUse bool, value int, msg string, no_avatar int) {
	//1.检查是否有权限
	msg = "星级不足,请提升星级"
	pri := common.PRI_CHAT
	m, u, e := GetPrivilege(fuid, pri)
	if e != nil {
		mlog.AppendObj(e, "CheckPrivilege --- is error ", fuid, pri)
		return
	}
	p, ok := m[pri]
	if !ok {
		mlog.AppendObj(e, "CheckPrivilege  pri is not in m  ", fuid, pri)
		return false, 0, "", 0
	}
	if p.Can {
		return true, p.Bal, "", 0
	}

	if u.Avatarlevel == -1 {
		return false, 0, p.Msg, 1
	}

	//2.无权限，需要查看是否已经聊过的用户
	key := utils.ToString(fuid)
	rcon := rdb.GetWriteConnection(redis_db.REDIS_USER_PRI_CHAT)
	defer rcon.Close()
	limit_tm := utils.Now.AddDate(0, 0, -30).Unix()
	rcon.Do("ZREMRANGEBYSCORE", key, 0, limit_tm)
	// 查询接受是否是我之前主动发起的
	exit, e := rcon.Do("ZSCORE", key, tuid)
	if e != nil {
		mlog.AppendObj(e, "CheckPrivilege  get socre -1 ", key, tuid)
		return false, 0, p.Msg, 0
	}
	if exit != nil { // 是我之前主动聊过的用户
		return true, 1, "", 0
	}
	key2 := utils.ToString(tuid)
	rcon.Do("ZREMRANGEBYSCORE", key2, 0, limit_tm)
	exit2, e := rcon.Do("ZSCORE", key2, fuid)
	if e != nil {
		mlog.AppendObj(e, "CheckPrivilege  get socre -2 ", key, tuid)
		return false, 0, p.Msg, 0
	}
	if exit2 != nil { // 是之前用户主动找我
		return true, 1, "", 0
	}

	return false, 0, p.Msg, 0
}

/*
使用聊天特权

参数：
	参数：
	fuid: 发送方uid ,tuid: 结束放uid , result : 返回结果，将权限自动填充到http的result中，push的msg中
*/
func UseChatPrivilege(fuid, tuid uint32, result map[string]interface{}) (e error) {
	//1.直接查询是否是之前聊过的，否则添加到我的主动set中，并就该权限值
	key := utils.ToString(fuid)
	rcon := rdb.GetWriteConnection(redis_db.REDIS_USER_PRI_CHAT)
	defer rcon.Close()
	// 查询接受是否是我之前主动发起的
	exit, e := rcon.Do("ZSCORE", key, tuid)
	if e != nil {
		return e
	}
	if exit != nil { // 是我之前主动聊过的用户
		rcon.Do("ZADD", key, utils.Now.Unix(), tuid)
		return
	}
	// 查询接受是否是我之前回复的
	key2 := utils.ToString(tuid)
	exit2, e := rcon.Do("ZSCORE", key2, fuid)
	if e != nil {
		return e
	}
	if exit2 != nil { // 是之前回复过的用户
		rcon.Do("ZADD", key, utils.Now.Unix(), tuid)
		return
	}
	// 是我主动发起的
	if _, e := rcon.Do("ZADD", key, utils.Now.Unix(), tuid); e != nil {
		return e
	}

	e = UsePrivilege(fuid, common.PRI_CHAT, result)
	return
}

func CheckHonesty2(uid uint32, keys ...string) (e error) {
	if len(keys) <= 0 {
		keys = pri_get_arr
	}
	// 获取已获得方式
	m, e := GetHonestyRecord(uid)
	if e != nil {
		return e
	}
	user_overview.ClearUserObjects(uid)
	delete_item := make([]int, 0, 2)
	add_item := make([]int, 0, 2)
	//完成项目1：头像为真实用户头像并审核通过，2：相册至少上传3张生活照，3：基本资料全部填写
	for _, key := range keys {
		switch key {
		case common.PRI_GET_AVATAR: // 头像审核通过
		case common.PRI_GET_PHOTOS: // 照片3张以上
			item := 2
			up, e := checkHonestyPhoto(uid)
			if e != nil {
				return e
			}
			_, ok := m[item] // 是否已经存在
			if up && !ok {   // 可以升级，但是为存在记录,插入新纪录
				add_item = append(add_item, item)
			} else if !up && ok { // 不能升级，但是记录存在
				delete_item = append(delete_item, item)
			}

		case common.PRI_GET_INFO: // 填写资料
			item := 3
			up, e := checkHonestyInfo(uid)
			if e != nil {
				return e
			}
			_, ok := m[item] // 是否已经存在
			if up && !ok {   // 可以升级，但是为存在记录,插入新纪录
				add_item = append(add_item, item)
			} else if !up && ok { // 不能升级，但是记录存在
				delete_item = append(delete_item, item)
			}

		case common.PRI_GET_PHONE: // 手机认证
		case common.PRI_GET_VIDEO: // 视频认证
		case common.PRI_GET_IDCARD: // 身份证认证

		}
	}

	if len(delete_item) > 0 {
		for _, item := range delete_item {
			if e := DeleteHonestRecord(mdb, uid, item); e != nil {
				return e
			}
		}
	}
	if len(add_item) > 0 {
		for _, item := range add_item {
			if e := AddHonestyRecord(mdb, uid, item); e != nil {
				return e
			}
		}
	}

	e = CheckUpGrade2(uid, keys...)
	return
}

// 检测是否升级
func CheckUpGrade2(uid uint32, keys ...string) (e error) {
	u, e := user_overview.GetUserObjectNoCache(uid)
	if e != nil {
		return e
	}
	m, e := GetHonestyRecord(uid)
	if e != nil {
		return e
	}
	honesty_level := 0
	honesty_level += len(m)
	if u.CertifyPhone == 1 {
		honesty_level += 1
	}
	if u.CertifyVideo == 1 {
		honesty_level += 1
	}
	if u.CertifyIDcard == 1 {
		honesty_level += 1
	}
	if u.Avatarlevel != -1 {
		honesty_level += 1
	}

	// 现有等级和初始化等级不相等,需要将其修改honsty_level 和对应特权余额
	if honesty_level != u.HonestyLevel {
		if e := updateHonestyLevel(uid, honesty_level, u.HonestyLevel); e != nil {
			return e
		}
	}

	return
}

/*
用于在http中返会特权弹窗,无权限的情况下返回错误errord

p: 无权限返回的对象,详见：http://120.131.64.91:8182/pkg/yuanfen/yf_service/cls/data_model/certify/#Pri
*/
func GetNotifyErrorPop(p Pri) (re service.Error) {
	if p.Msg == "" || p.But.Data == nil {
		return service.NewError(service.ERR_INTERNAL, "GetNotifyErrorPop p is empety ", "内部错误")
	}
	//title, content, img string, nid, flag, show_type int, uid uint32, save_flag int, buts ...But
	n := notify.GenNotify("", p.Msg, "", notify.NOTIFY_TYPE_SYSTEM, 0, 8, 0, 0, notify.GetDefBut(notify.BUT_IGNORE), p.But)
	b, e := json.Marshal(n)
	if e != nil {
		return service.NewError(service.ERR_INTERNAL, "json 解析错误")
	}
	return service.NewError(service.ERR_POP_NOTIFY, string(b))
}
