package discovery

import (
	"bytes"
	"container/heap"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
	"yf_pkg/format"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/building"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/message"
)

var mdb *mysql.MysqlDB
var sdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool
var mainLog *log.MLogger
var mode string

const (
	BIRTHDAY_RANGE        = 5 * 365 * 24 * time.Hour
	RECOMMEND_TIMEOUT     = 30 * time.Minute
	MAX_RECOMMEND_NUM     = 10000
	ADJACENT_USER_TIMEOUT = 3  //查询的附近的人至少在几天内登陆过
	RECOMMEND_EACH_TIME   = 25 //每次推荐的数量
)

const (
	REC_TYPE_WORKPLACE = iota //工作地点
	REC_TYPE_HOMETOWN         //同乡
	REC_TYPE_GRADUATE         //校友
	REC_TYPE_REQUIRE          //择友要求
	REC_TYPE_TRADE            //同行
	REC_TYPE_CHAR             //感兴趣的类型
	REC_TYPE_NUM              //推荐类型的总数
)

func Init(env *cls.CustomEnv) {
	fmt.Println("REC_TYPE_NUM=", REC_TYPE_NUM)
	sdb = env.SortDB
	mdb = env.MainDB
	rdb = env.MainRds
	cache = env.CacheRds
	mainLog = env.MainLog
	mode = env.Mode

	message.RegisterNotification(message.LOCATION_CHANGE, locationChanged)
	message.RegisterNotification(message.ONLINE, userOnline)
	message.RegisterNotification(message.RECOMMEND_CHANGE, recommendChange)
	message.RegisterNotification(message.OFFLINE, userOffline)
	message.RegisterNotification(message.CLEAR_CACHE, delUser)

	go delInactiveUsers()
}

func SearchByUid(uid uint32) (user AdjUser, e error) {
	user.Id = uid
	users := []AdjUser{user}
	if users, e = makeUsersInfo(users, false, true, uid, 0, 0); e != nil {
		return
	}
	return users[0], nil
}

func ClearRecommend(uid uint32) (e error) {
	if mode != "test" {
		return errors.New("can only run in test mode")
	}
	if e = cache.Del(redis_db.CACHE_RECOMMEND_USERS, uid); e != nil {
		return
	}
	return rdb.Del(redis_db.REDIS_RECOMMEND_USERS, uid)
}

func Recommend(uid uint32, lat, lng float64) (users []*RecUser, left int, isCache bool, e error) {
	//查看用户今天的推荐名额是否已经用完，用完返回空
	count, e := cache.LLen(redis_db.CACHE_RECOMMEND_USERS, uid)
	if e != nil {
		return nil, 0, false, e
	}
	//获取已标记的用户
	follow, e := general.GetInterestUids(uid)
	if e != nil {
		return nil, 0, false, e
	}
	//如果已经推荐完成，则返回当天的缓存数据
	if count >= MAX_RECOMMEND_NUM {
		result, e := redis.Values(cache.LRange(redis_db.CACHE_RECOMMEND_USERS, uid, 0, -1))
		if e != nil {
			return nil, 0, false, e
		}
		bs := make([][]byte, 0, MAX_RECOMMEND_NUM)
		if e = redis.ScanSlice(result, &bs); e != nil {
			return nil, 0, false, e
		}
		users = make([]*RecUser, 0, MAX_RECOMMEND_NUM)
		for _, b := range bs {
			var u RecUser
			if e = json.Unmarshal(b, &u); e != nil {
				return nil, 0, false, e
			}
			if _, ok := follow[u.Uid]; ok {
				u.Follow = true
			}
			users = append(users, &u)
		}
		return users, 0, true, nil
	}
	users = make([]*RecUser, 0, RECOMMEND_EACH_TIME)
	//实际可推荐的数量
	num := RECOMMEND_EACH_TIME
	if count+RECOMMEND_EACH_TIME > MAX_RECOMMEND_NUM {
		num = MAX_RECOMMEND_NUM - count
	}
	//获取已推荐过的用户
	res := make([]uint32, 0, 100)
	if _, e = rdb.ZRange(redis_db.REDIS_RECOMMEND_USERS, uid, 0, -1, false, &res); e != nil {
		return nil, 0, false, e
	}
	recommended := make(map[uint32]bool, len(res)*2)
	for _, uid := range res {
		recommended[uid] = true
	}

	//获取自己的信息
	uinfo, e := user_overview.GetUserObject(uid)
	if e != nil {
		return nil, 0, false, e
	}
	var location utils.Coordinate
	location.Lat, location.Lng, e = general.UserLocation(uid)
	if e != nil {
		return nil, 0, false, e
	}
	//把未填写的内容替换成默认值
	if uinfo.Require.Minage == -1 {
		uinfo.Require.Minage = 0
	}
	if uinfo.Require.Maxage == -1 {
		uinfo.Require.Maxage = common.MAX_AGE
	}
	if uinfo.Require.Minheight == -1 {
		uinfo.Require.Minheight = 0
	}
	if uinfo.Require.Maxheight == -1 {
		uinfo.Require.Maxheight = common.MAX_HEIGHT
	}
	if uinfo.Require.Minedu == -1 {
		uinfo.Require.Minedu = 0
	}
	if uinfo.Require.Province == "不限" {
		uinfo.Require.Province = ""
	}
	if uinfo.Require.City == "不限" {
		uinfo.Require.City = ""
	}

	tnums := getAvailableRecommends(num, uinfo, location)
	if tnums == nil {
		return
	}
	provinces := []string{uinfo.Province}
	//检查择友要求中的所在省市，确定sql语句中是否需要增加额外的省
	if tnums[REC_TYPE_REQUIRE] > 0 && uinfo.Require.Province != "" && uinfo.Require.Province != uinfo.Province {
		provinces = append(provinces, uinfo.Require.Province)
	}
	var minBirth, maxBirth time.Time
	gender := common.GENDER_MAN
	switch uinfo.Gender {
	case common.GENDER_MAN:
		gender = common.GENDER_WOMAN
		minBirth = uinfo.Birthday.AddDate(-3, 0, 0)
		maxBirth = uinfo.Birthday.AddDate(8, 0, 0)
	case common.GENDER_WOMAN:
		minBirth = uinfo.Birthday.AddDate(-8, 0, 0)
		maxBirth = uinfo.Birthday.AddDate(3, 0, 0)
	}
	//测试，所以扩大范围
	//minBirth = uinfo.Birthday.AddDate(-30, 0, 0)
	//maxBirth = uinfo.Birthday.AddDate(80, 0, 0)
	//sql := "select id from discovery where gender=? and birthday>? and birthday<? and province" + mysql.In(provinces) + "order by online_timeout desc"
	sql := "select id from discovery where gender=? and province" + mysql.In(provinces) + "order by online_timeout desc"
	selected := make(map[uint32]*RecUser, num) //选中的用户集合
	forZAdd := make([]interface{}, 0, num)
	recResults := map[uint32]string{}
	selectedUids := make([]uint32, 0, num)
	backup := make([]*RecUser, 0, 10)
	for i := 1; len(selected) < num; i++ {
		uids := make([]uint32, 0, 100) //备选uid列表
		e = func() error {
			//rows, e := sdb.Query(sql+utils.BuildLimit(i, RECOMMEND_EACH_TIME*3), gender, minBirth, maxBirth)
			rows, e := sdb.Query(sql+utils.BuildLimit(i, RECOMMEND_EACH_TIME*3), gender)
			if e != nil {
				return e
			}
			defer rows.Close()
			for rows.Next() {
				var id uint32
				if err := rows.Scan(&id); err != nil {
					return err
				}
				recResults[id] = ""
				uids = append(uids, id)
			}
			return nil
		}()
		if e != nil {
			return nil, 0, false, e
		}
		if len(uids) == 0 {
			//已经找不到更多的用户了
			break
		}
		uinfos, e := user_overview.GetUserObjects(uids...)
		if e != nil {
			return nil, 0, false, e
		}
		//过滤掉不允许被推荐的用户
		canRecs, e := general.GetUserProtects(uids...)
		if e != nil {
			return nil, 0, false, e
		}
		for uid, tinfo := range uinfos {
			canRec := canRecs[uid]
			if canRec != nil && canRec.DoNotFindMe != 0 {
				uinfos[uid] = nil
				recResults[uid] = "[×]对方不允许被推荐"
			} else if tinfo != nil && tinfo.Workunit != "" && tinfo.Workunit == uinfo.Workunit {
				//同工作单位的不推荐
				uinfos[uid] = nil
				recResults[uid] = "[×]同工作单位不推荐"
			}
		}
		for cid, candidate := range uinfos {
			if u, reason := filterUser(tnums, uinfo, candidate, location, recommended, follow); u != nil {
				if u.Reason.Text == "" {
					backup = append(backup, u)
				} else {
					selected[cid] = u
					selectedUids = append(selectedUids, u.Uid)
					forZAdd = append(forZAdd, uid, utils.Now.Unix(), cid)
					recResults[cid] = "[√]" + u.Reason.Text
					if len(selected) >= num {
						break
					}
				}
			} else {
				recResults[cid] = "[×]" + reason
			}
		}

	}
	for _, u := range backup {
		if len(selected) < num {
			selected[u.Uid] = u
			selectedUids = append(selectedUids, u.Uid)
			forZAdd = append(forZAdd, uid, utils.Now.Unix(), u.Uid)
			recResults[u.Uid] = "[√]推荐人数不足的补充用户"
		} else {
			break
		}
	}
	photos, e := user_overview.GetUserPhotos(selectedUids...)
	if e != nil {
		return nil, 0, false, errors.New("GetUserPhotos error :" + e.Error())
	}
	for uid, recUser := range selected {
		if plist := photos[uid]; plist != nil {
			recUser.PhotoList = plist.PhotoList
		}
	}
	//添加到已推荐列表
	if e = rdb.ZAdd(redis_db.REDIS_RECOMMEND_USERS, forZAdd...); e != nil {
		return nil, 0, false, e
	}
	//删除一个月之前的记录
	if e = rdb.ZRemRangeByScore(redis_db.REDIS_RECOMMEND_USERS, uid, 0, utils.Now.AddDate(0, -1, 0).Unix()); e != nil {
		mainLog.Append(fmt.Sprintf("ZRemRangeByScore error:%v", e.Error()))
	}

	bs := make([]interface{}, 0, len(selected))
	for _, u := range selected {
		b, e := json.Marshal(u)
		if e != nil {
			return nil, 0, false, e
		}
		bs = append(bs, b)
		users = append(users, u)
	}
	//添加推荐缓存
	if _, e = cache.RPush(redis_db.CACHE_RECOMMEND_USERS, uid, bs...); e != nil {
		return nil, 0, false, e
	}
	//如果是第一次，设置超时时间
	if count == 0 && len(bs) > 0 {
		dur := utils.DurationTo(1, 6, utils.Now.Minute(), utils.Now.Second())
		if err := cache.Expire(redis_db.CACHE_RECOMMEND_USERS, int(dur.Seconds()), uid); err != nil {
			return nil, 0, false, err
		}
	}
	left = MAX_RECOMMEND_NUM - count - len(users)
	fmt.Println("sql:", sql, "gender=", gender, "minBirth=", minBirth, "maxBirth=", maxBirth)
	fmt.Printf("用户[%v]的推荐结果：\n", uid)
	for uid, reason := range recResults {
		fmt.Println("\t", uid, "->", reason)
	}
	return
}

//根据用户信息找出可以推荐的类型，以数组的形式返回。
//其中，下标表示推荐类型，值示剩余可推荐的人数
func getAvailableRecommends(total int, uinfo *user_overview.UserViewItem, uloc utils.Coordinate) (tnums []int) {
	tnums = make([]int, MAX_RECOMMEND_NUM)
	availableNum := 0
	//家乡
	if uinfo.Homeprovince != "" {
		availableNum++
		tnums[REC_TYPE_HOMETOWN] = -1
	}
	//工作地点
	if uloc.Lat != common.LAT_NO_VALUE && uloc.Lng != common.LNG_NO_VALUE {
		availableNum++
		tnums[REC_TYPE_WORKPLACE] = -1
	}
	//毕业院校
	if uinfo.School != "" {
		availableNum++
		tnums[REC_TYPE_GRADUATE] = -1
	}
	//同行
	if uinfo.Trade != "" {
		availableNum++
		tnums[REC_TYPE_TRADE] = -1
	}
	//感兴趣的类型
	if len(uinfo.Needtag) > 0 {
		availableNum++
		tnums[REC_TYPE_CHAR] = -1
	}

	//择友条件
	if uinfo.Require.Filled() >= 2 {
		availableNum++
		tnums[REC_TYPE_REQUIRE] = -1
	}
	if availableNum == 0 {
		return nil
	}
	//每种类型比平均值多30%
	n := (total * 13 / availableNum) / 10
	if n == 0 {
		n = 1
	}
	for i := range tnums {
		if tnums[i] == -1 {
			tnums[i] = n
		}
	}
	return
}

//过滤出推荐的用户
func filterUser(tnums []int, me *user_overview.UserViewItem, candidate *user_overview.UserViewItem, myLoc utils.Coordinate, recommended map[uint32]bool, follow map[uint32]time.Time) (u *RecUser, nreason string) {
	if candidate == nil || !candidate.IsRecommend() {
		return nil, "头像不符合标准"
	}
	/*
		if _, found := recommended[candidate.Uid]; found {
			return nil, "已推荐过"
		}
	*/
	if _, found := follow[candidate.Uid]; found {
		return nil, "已被标记"
	}
	ta := "他"
	if candidate.Gender == common.GENDER_WOMAN {
		ta = "她"
	}
	reason := ""
	recType := 0
	if tnums[REC_TYPE_GRADUATE] > 0 && me.School == candidate.School {
		//if tnums[REC_TYPE_GRADUATE] > 0 {
		//校友
		reason = "你们都来自" + me.School
		recType = REC_TYPE_GRADUATE
		tnums[REC_TYPE_GRADUATE]--
	} else if tnums[REC_TYPE_HOMETOWN] > 0 && me.Homeprovince != me.Province && me.Homeprovince == candidate.Homeprovince {
		//} else if tnums[REC_TYPE_HOMETOWN] > 0 && rand.Intn(2) >= 0 {
		//家乡
		if me.Homecity != "" && me.Homecity == candidate.Homecity {
			reason = "你们都是" + me.Homeprovince + me.Homecity + "人"
		} else {
			reason = "在" + me.City + "的" + me.Homeprovince + "人"
		}
		recType = REC_TYPE_HOMETOWN
		tnums[REC_TYPE_HOMETOWN]--
	} else if tnums[REC_TYPE_WORKPLACE] > 0 {
		//工作地点
		/*
			//为了测试
			if candidate.WorkPlaceName != "" {
				reason = ta + "在" + candidate.WorkPlaceName + "工作"
			}
		*/
		if candidate.WorkPlaceId != "" && utils.Distence(utils.Coordinate{myLoc.Lat, myLoc.Lng}, utils.Coordinate{candidate.WorkLat, candidate.WorkLng}) <= common.WORKPLACE_RADIUS*1000 {
			reason = ta + "在" + candidate.WorkPlaceName + "工作"
		}
		if reason != "" {
			fmt.Println(candidate.WorkPlaceId, reason)
			recType = REC_TYPE_WORKPLACE
			tnums[REC_TYPE_WORKPLACE]--
		}
	}
	if reason == "" && tnums[REC_TYPE_REQUIRE] > 0 {
		//择友要求
		//if true || me.Require.Match(candidate) {
		if me.Require.Match(candidate) {
			reason = "它符合您的择友要求"
			recType = REC_TYPE_REQUIRE
			tnums[REC_TYPE_REQUIRE]--
		}
	}
	if reason == "" && tnums[REC_TYPE_CHAR] > 0 {
		//感兴趣的类型
		/*
			//为了测试
			reason = ta + "活泼可爱"
			recType = REC_TYPE_CHAR
			tnums[REC_TYPE_CHAR]--
		*/
		for _, v := range me.Needtag {
			for _, t := range candidate.Tag {
				if t != "" && v == t {
					reason = ta + v
					recType = REC_TYPE_CHAR
					tnums[REC_TYPE_CHAR]--
					break
				}
			}
			if reason != "" {
				//匹配成功
				break
			}
		}
	}
	if reason == "" && tnums[REC_TYPE_TRADE] > 0 && me.Trade == candidate.Trade {
		//if reason == "" && tnums[REC_TYPE_TRADE] > 0 {
		//行业职业
		if me.Job != "" && me.Job == candidate.Job {
			if _, ok := common.JobNotRecommend[me.Job]; !ok {
				reason = "你们都是" + me.Job
			}
		}
		if reason == "" && me.Trade != "其它" {
			reason = "你们是同行"
		}
		if reason != "" {
			recType = REC_TYPE_TRADE
			tnums[REC_TYPE_TRADE]--
		}
	}
	u = &RecUser{candidate.Uid, candidate.Nickname, candidate.Age, candidate.Avatar, nil, candidate.Height, candidate.Trade, candidate.Workunit, false, ReasonObj{recType, reason}}
	return
}

//根据用户名、昵称或uid搜索
func SearchByUsername(username string, cur, ps int) (users []AdjUser, e error) {
	users = make([]AdjUser, 0, 1)
	sql := "select distinct uid from user_main where username=? or uid=?" + utils.BuildLimit(cur, ps)
	users, e = queryUserIds(users, sql, username, username)
	if e != nil {
		return nil, e
	}
	if len(users) == 0 {
		sql := "select distinct uid from user_detail where nickname=?" + utils.BuildLimit(cur, ps)
		users, e = queryUserIds(users, sql, username)
		if e != nil {
			return nil, e
		}
		if len(users) == 0 {
			return users, nil
		}
	}
	if users, e = makeUsersInfo(users, false, true, 0, 0, 0); e != nil {
		return nil, e
	}
	return users, nil
}

func Search(uid uint32, gender, minAge, maxAge, minHeight, maxHeight int, province string, minEdu int, homeprovince string, cur, ps int, refresh bool) (users []interface{}, pages *utils.Pages, e error) {
	uinfo, e := user_overview.GetUserObject(uid)
	if e != nil {
		return nil, nil, errors.New(fmt.Sprintf("Get uid %v info error :%v", uid, e.Error()))
	}
	if province == "" {
		province = uinfo.Province
	}
	if gender == -1 {
		if uinfo.Gender == common.GENDER_WOMAN {
			gender = common.GENDER_MAN
		} else {
			gender = common.GENDER_WOMAN
		}
	}
	key := general.MakeKey("search", gender, minAge, maxAge, minHeight, maxHeight, province, minEdu, homeprovince, province, cur, ps)
	minBirthday := utils.AgeToBirthday(maxAge)
	maxBirthday := utils.AgeToBirthday(minAge)
	exists := false
	var num int = 0
	if !refresh {
		exists, users, num, e = readCache(key, 0, ps, SearchUser{})
		pages = utils.PageInfo(-1, cur, num)
		if e != nil {
			return nil, nil, e
		}
	} else {
		if e = clearCache(key); e != nil {
			return nil, nil, e
		}
	}
	if !exists {
		var sqlBuf bytes.Buffer
		args := make([]interface{}, 0, 10)
		sqlBuf.WriteString("select m.uid,m.gender,d.nickname,d.birthday,d.height,d.avatar,d.city,d.job,d.aboutme,o.tm from user_main m,user_detail d,user_online o where m.uid=d.uid and m.uid=o.uid and d.avatarlevel>?")
		args = append(args, common.AVLEVEL_INVALID)
		if gender > 0 {
			sqlBuf.WriteString(" and m.gender=?")
			args = append(args, gender)
		}
		if minAge > 0 {
			sqlBuf.WriteString(" and d.birthday<=?")
			args = append(args, maxBirthday)
		}
		if maxAge < common.MAX_AGE {
			sqlBuf.WriteString(" and d.birthday>=?")
			args = append(args, minBirthday)
		}
		if minHeight > 0 {
			sqlBuf.WriteString(" and d.height>=?")
			args = append(args, minHeight)
		}
		if maxHeight < common.MAX_HEIGHT {
			sqlBuf.WriteString(" and d.height<=?")
			args = append(args, maxHeight)
		}
		if minEdu > 0 {
			sqlBuf.WriteString(" and d.edu>=?")
			args = append(args, minEdu)
		}
		if homeprovince != "" {
			sqlBuf.WriteString(" and d.homeprovince=?")
			args = append(args, homeprovince)
		}
		sqlBuf.WriteString(" and province=? order by o.tm desc" + utils.BuildLimit(cur, ps))
		args = append(args, province)
		rows, e := mdb.Query(sqlBuf.String(), args...)
		if e != nil {
			return nil, nil, e
		}
		defer rows.Close()
		uids := make([]uint32, 0, ps)
		userAll := make([]SearchUser, 0, ps)
		for rows.Next() {
			var user SearchUser
			var bStr, tm string
			if e = rows.Scan(&user.Uid, &user.Gender, &user.Nickname, &bStr, &user.Height, &user.Avatar, &user.City, &user.Job, &user.AboutMe, &tm); e != nil {
				return nil, nil, e
			}
			user.OnlineTimeout, _ = utils.ToTime(tm, format.TIME_LAYOUT_1)
			birthday, _ := utils.ToTime(bStr, format.TIME_LAYOUT_1)
			user.Age = utils.BirthdayToAge(birthday)
			userAll = append(userAll, user)
			uids = append(uids, user.Uid)
		}
		protects, e := general.GetUserProtects(uids...)
		if e != nil {
			return nil, nil, errors.New("GetUserProtects error :" + e.Error())
		}
		users = make([]interface{}, 0, ps)
		for i, _ := range userAll {
			if protects[userAll[i].Uid].DoNotFindMe == 0 {
				users = append(users, userAll[i])
			}
		}

		//放入redis缓存
		if e = writeCache(key, users, 600); e != nil {
			return nil, nil, e
		}
		pages = utils.PageInfo(-1, cur, len(users))
	}
	return users, pages, nil
}

/*
寻找附近的用户，默认搜索半径为common.LOCATION_RADIUS。

参数:
	gender: 性别，-1-表示异性
	building: 所在工作地点，""表示不限
*/
func AdjacentUsers(gender int, building string, uid uint32, lat float64, lng float64, cur int, ps int, refresh bool) (users []AdjUser, page *utils.Pages, buildings []*building.Building, e error) {
	exists := false
	var total int
	userAll := make([]AdjUser, 0, ps)
	k := general.MakeKey("adjacent", uid, gender, building)
	if !refresh {
		var data []interface{}
		exists, data, total, e = readCache(k, cur, ps, AdjUser{})
		if e != nil {
			general.Alert("redis-cache", "read adjacnet failed")
			return nil, nil, nil, e
		}
		for _, user := range data {
			fmt.Println("user=", user)
			switch v := user.(type) {
			case AdjUser:
				userAll = append(userAll, v)
			}
		}
	} else {
		if e = clearCache(k); e != nil {
			return nil, nil, nil, e
		}
	}
	if !exists {
		maxLat, minLat := lat+common.LOCATION_RADIUS, lat-common.LOCATION_RADIUS
		maxLng, minLng := lng+common.LOCATION_RADIUS, lng-common.LOCATION_RADIUS

		uinfo, e := user_overview.GetUserObject(uid)
		if e != nil {
			return nil, nil, nil, errors.New(fmt.Sprintf("Get uid %v info error :%v", uid, e.Error()))
		}
		if gender == -1 {
			if uinfo.Gender == common.GENDER_WOMAN {
				gender = common.GENDER_MAN
			} else {
				gender = common.GENDER_WOMAN
			}
		}
		//构造sql
		sql := "select id,lat,lng,online_timeout from discovery where lat <= ? and lat >= ? and lng <= ? and lng >= ? and online_timeout > ?"
		args := make([]interface{}, 0, 10)
		args = append(args, maxLat, minLat, maxLng, minLng, utils.Now.AddDate(0, 0, -ADJACENT_USER_TIMEOUT))
		if gender != common.GENDER_BOTH {
			sql += " and gender=?"
			args = append(args, gender)
		}
		if building != "" {
			sql += " and workplace=?"
			args = append(args, building)
		}
		sql += " limit 1000"
		if userAll, e = queryUserItems(userAll, sql, args...); e != nil {
			general.Alert("mysql-sort", "query discovery failed")
			return nil, nil, nil, e
		}
		if userAll, buildings, e = makeAdjUsersInfo(building == "", userAll, uid, lat, lng); e != nil {
			return nil, nil, nil, e
		}
		total = len(userAll)
		fmt.Println("total", total)
		//放入redis缓存
		/*
			if e = writeCache(k, userAll, 600); e != nil {
				general.Alert("redis-cache", "write adjacnet failed")
				return nil, nil, nil, e
			}
		*/
	}
	start, end := utils.BuildRange(cur, ps, total)
	if start < total {
		users = userAll[start : end+1]
	}
	page = utils.PageInfo(total, cur, ps)
	return
}

//AdjacentMatchedUsers寻找满足用户本地标签要求的附近的用户，实际返回的用户可能会少于ps的数量。
//
//	radius: 半径（公里）
func AdjacentMatchedUsers(uid uint32, radius uint32, cur, ps int) (matched []*user_overview.UserViewItem, e error) {
	uinfo, e := user_overview.GetUserObject(uid)
	if e != nil {
		return nil, errors.New(fmt.Sprintf("Get uid %v info error :%v", uid, e.Error()))
	}
	lat, lng, e := general.UserLocation(uid)
	if e != nil {
		return
	}
	r := float64(radius)
	maxLat, minLat := lat+utils.KmToLat(r), lat-utils.KmToLat(r)
	maxLng, minLng := lng+utils.KmToLng(r), lng-utils.KmToLng(r)

	gender := uinfo.Ltag.Req.Gender
	if uinfo.Ltag.Req.Gender == -1 {
		if uinfo.Gender == common.GENDER_WOMAN {
			gender = common.GENDER_MAN
		} else {
			gender = common.GENDER_WOMAN
		}
	}
	users := make([]AdjUser, 0, ps)
	if gender == common.GENDER_BOTH {
		sql := "select id,lat,lng,online_timeout from discovery where lat <= ? and lat >= ? and lng <= ? and lng >= ? and online_timeout > ?" + utils.BuildLimit(cur, ps)
		if users, e = queryUserItems(users, sql, maxLat, minLat, maxLng, minLng, utils.Now.Add(3*time.Minute)); e != nil {
			return nil, e
		}
	} else {
		sql := "select id,lat,lng,online_timeout from discovery where gender = ? and lat <= ? and lat >= ? and lng <= ? and lng >= ? and online_timeout > ?" + utils.BuildLimit(cur, ps)
		if users, e = queryUserItems(users, sql, gender, maxLat, minLat, maxLng, minLng, utils.Now.Add(3*time.Minute)); e != nil {
			return nil, e
		}
	}
	//取出当前请求的uid集合
	tmpUids := make([]uint32, 0, len(users))
	for i := 0; i < len(users); i++ {
		tmpUids = append(tmpUids, users[i].Id)
	}
	uinfos, e := user_overview.GetUserObjects(tmpUids...)
	if e != nil {
		return nil, errors.New("GetUserObjects error :" + e.Error())
	}
	matched = make([]*user_overview.UserViewItem, 0, len(users))
	for i, _ := range users {
		if users[i].Id == uid {
			continue
		}
		if uinfos[users[i].Id] != nil {
			//fmt.Printf("附近的用户users[%v]=%v\n", i, users[i])
			dis := utils.Distence(utils.Coordinate{users[i].Lat, users[i].Lng}, utils.Coordinate{lat, lng})
			if uinfo.MatchMyLocaltag(uinfos[users[i].Id], dis/1000) {
				matched = append(matched, uinfos[users[i].Id])
			}
		}
	}
	return
}

/*
UpdateDiscovery更新discovery表，仅做update操作，不会insert，如果uid不存在，不会抛出异常。

参数：
	args: 要更新的字段名称和值，必须成对出现，参看qiuqian_sort.discovery表中的字段名
*/
func UpdateDiscovery(uid uint32, args ...interface{}) (e error) {
	if len(args) == 0 {
		return nil
	}
	if len(args)%2 != 0 {
		return errors.New("invalid number of args")
	}
	values := make([]interface{}, len(args)/2, len(args)/2+1)
	values[0] = args[1]
	sql_str := fmt.Sprintf("update discovery set `%s`=?", args[0])
	for i := 2; i < len(args); i += 2 {
		sql_str += fmt.Sprintf(",`%s`=?", args[i])
		values[i>>1] = args[i+1]
	}
	values = append(values, uid)
	sql_str += " where id=?"
	_, e = sdb.Exec(sql_str, values...)
	switch e {
	case sql.ErrNoRows:
		return nil
	default:
		return e
	}
	return
}

//-------------------------Private Functions------------------------//

func locationChanged(msgid int, data interface{}) {
	switch v := data.(type) {
	case message.LocationChange:
		var sql string
		sql = "update discovery set lat=?,lng=?,online_timeout=? where id=?"
		if _, e := sdb.Exec(sql, v.Lat, v.Lng, utils.Now.Add(common.ONLINE_TIMEOUT*time.Second), v.Uid); e != nil {
			mainLog.Append(fmt.Sprintf("update user location to sdb.discovery error : %v", e.Error()))
		}
		lcon := rdb.GetWriteConnection(redis_db.REDIS_LOCATION)
		defer lcon.Close()
		if _, e := lcon.Do("HMSET", v.Uid, "lat", v.Lat, "lng", v.Lng); e != nil {
			mainLog.Append(fmt.Sprintf("update user location to rdb.location error : %v", e.Error()))
		}
	}
}

func recommendChange(msgid int, data interface{}) {
	fmt.Println("recommendChange", msgid, data)
	switch v := data.(type) {
	case message.RecommendChange:
		//该用户记录已被删除
		uinfo, e := user_overview.GetUserObject(v.Uid)
		fmt.Println("recommendChange", v.Uid, uinfo)
		if e != nil {
			mainLog.Append(fmt.Sprintf("user_overview.GetUserObjects %v error : %v", v.Uid, e.Error()))
			return
		}
		if uinfo != nil {
			if uinfo.IsRecommend() {
				lat, lng, e := general.UserLocation(v.Uid)
				if e != nil {
					mainLog.Append(fmt.Sprintf("get user %v location error : %v", v.Uid, e.Error()))
					lat, lng = common.LAT_NO_VALUE, common.LNG_NO_VALUE
				}
				timeout := utils.Now.Add(common.ONLINE_TIMEOUT * time.Second)
				sql := "insert into discovery(id,gender,birthday,province,city,workplace,lat,lng,online_timeout)values(?,?,?,?,?,?,?,?,?) on duplicate key update online_timeout=?"
				_, e = sdb.Exec(sql, v.Uid, uinfo.Gender, uinfo.Birthday, uinfo.Province, uinfo.City, uinfo.WorkPlaceId, lat, lng, timeout, timeout)
				if e != nil {
					general.Alert("mysql-sort", "insert discovery failed")
					mainLog.Append(fmt.Sprintf("insert user data to sdb.discovery error : %v", e.Error()))
				}
			} else {
				_, e := sdb.Exec("delete from discovery where id=?", v.Uid)
				if e != nil {
					general.Alert("mysql-sort", "delete discovery failed")
					mainLog.Append(fmt.Sprintf("delete user data to sdb.discovery error : %v", e.Error()))
				}
			}
		}
	}
}

func userOnline(msgid int, data interface{}) {
	switch v := data.(type) {
	case message.Online:
		uinfo, e := user_overview.GetUserObject(v.Uid)
		if e != nil {
			mainLog.Append(fmt.Sprintf("user_overview.GetUserObjects %v error : %v", v.Uid, e.Error()))
			return
		}
		if uinfo != nil && uinfo.IsRecommend() {
			timeout := utils.Now.Add(common.ONLINE_TIMEOUT * time.Second)
			sql := "insert into discovery(id,gender,birthday,province,city,workplace,online_timeout)values(?,?,?,?,?,?,?) on duplicate key update online_timeout=?"
			_, e := sdb.Exec(sql, v.Uid, uinfo.Gender, uinfo.Birthday, uinfo.Province, uinfo.City, uinfo.WorkPlaceId, timeout, timeout)
			if e != nil {
				general.Alert("mysql-sort", "insert discovery failed")
				mainLog.Append(fmt.Sprintf("insert user data to sdb.discovery error : %v", e.Error()))
			}
		}
	}
}

func userOffline(msgid int, data interface{}) {
	switch v := data.(type) {
	case message.Offline:
		var sql string
		sql = "update discovery set online_timeout=? where id=?"
		if _, e := sdb.Exec(sql, utils.Now.Add(-1*time.Second), v.Uid); e != nil {
			mainLog.Append(fmt.Sprintf("set user offline to sdb.discovery error : %v", e.Error()))
		}
	}
}

func queryUserIds(users UserItems, sql string, args ...interface{}) (UserItems, error) {
	users = make(UserItems, 0)
	rows, e := mdb.Query(sql, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var item AdjUser
		if err := rows.Scan(&item.Id); err != nil {
			return nil, err
		}
		users = append(users, item)
	}
	return users, nil
}
func queryUserItems(users UserItems, sql string, args ...interface{}) (UserItems, error) {
	users = make(UserItems, 0)
	rows, e := sdb.Query(sql, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var item AdjUser
		var tm string
		if err := rows.Scan(&item.Id, &item.Lat, &item.Lng, &tm); err != nil {
			return nil, err
		}
		item.OnlineTimeout, _ = utils.ToTime(tm, format.TIME_LAYOUT_1)
		users = append(users, item)
	}
	return users, nil
}

func makeUsersLocation(users []AdjUser) (e error) {
	lcon := rdb.GetReadConnection(redis_db.REDIS_LOCATION)
	defer lcon.Close()
	for _, user := range users {
		if e := lcon.Send("HGETALL", user.Id); e != nil {
			return e
		}
	}
	lcon.Flush()
	type Location struct {
		Lat float64 `redis:"lat"`
		Lng float64 `redis:"lng"`
	}
	for i, _ := range users {
		reply, e := redis.Values(lcon.Receive())
		if e != nil {
			return e
		}
		var loc Location
		if e = redis.ScanStruct(reply, &loc); e != nil {
			return e
		}
		users[i].Lat = loc.Lat
		users[i].Lng = loc.Lng
	}
	return nil
}

func makeAdjUsersInfo(withBuildings bool, users []AdjUser, uid uint32, lat, lng float64) (ret []AdjUser, buildings []*building.Building, e error) {
	//取出当前请求的uid集合
	uids := make([]uint32, 0, len(users))
	for i := 0; i < len(users); i++ {
		uids = append(uids, users[i].Id)
	}

	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, nil, errors.New("GetUserObjects error :" + e.Error())
	}
	protects, e := general.GetUserProtects(uids...)
	if e != nil {
		return nil, nil, errors.New("GetUserProtects error :" + e.Error())
	}
	photos, e := user_overview.GetUserPhotos(uids...)
	if e != nil {
		return nil, nil, errors.New("GetUserPhotos error :" + e.Error())
	}
	ret = make([]AdjUser, 0, len(users))
	buildingMap := make(map[string]*building.Building, 20)
	for _, user := range users {
		uinfo := uinfos[user.Id]
		p := protects[user.Id]
		if uinfo != nil && p != nil {
			if uinfo.Uid != uid && uinfo.IsRecommend() && uinfo.Stat == common.USER_STAT_NORMAL && p.DoNotFindMe == 0 {
				user.Nickname = uinfo.Nickname
				user.Age = uinfo.Age
				user.Avatar = uinfo.Avatar
				if plist := photos[uinfo.Uid]; plist != nil {
					user.PhotoList = plist.PhotoList
				}
				user.Gender = uinfo.Gender
				user.Building = uinfo.WorkPlaceName
				user.AboutMe = uinfo.Aboutme
				user.Height = uinfo.Height
				user.Distence = utils.Distence(utils.Coordinate{lat, lng}, utils.Coordinate{user.Lat, user.Lng})
				if withBuildings && uinfo.WorkPlaceId != "" {
					if _, ok := buildingMap[uinfo.WorkPlaceId]; !ok {
						distence := utils.Distence(utils.Coordinate{lat, lng}, utils.Coordinate{uinfo.WorkLat, uinfo.WorkLng})
						buildingMap[uinfo.WorkPlaceId] = &building.Building{uinfo.WorkPlaceId, uinfo.WorkPlaceName, uinfo.WorkPlaceAddress, uinfo.WorkLat, uinfo.WorkLng, distence}
					}
				}
				ret = append(ret, user)
			}
		}
	}
	buildings = make([]*building.Building, 0, len(buildingMap))
	for _, building := range buildingMap {
		buildings = append(buildings, building)
	}
	//按照距离排序
	sort.Sort(UserItems(ret))
	sort.Sort(building.BuildingItems(buildings))
	return ret, buildings, nil
}
func makeUsersInfo(users []AdjUser, checkAvarta bool, needLocation bool, uid uint32, lat, lng float64) (ret []AdjUser, e error) {
	if needLocation {
		if e = makeUsersLocation(users); e != nil {
			return nil, e
		}
	}
	//取出当前请求的uid集合
	uids := make([]uint32, 0, len(users))
	for i := 0; i < len(users); i++ {
		uids = append(uids, users[i].Id)
	}

	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, errors.New("GetUserObjects error :" + e.Error())
	}
	ret = make([]AdjUser, 0, len(users))
	for _, user := range users {
		uinfo := uinfos[user.Id]
		if uinfo != nil {
			if uinfo.Uid != uid && (!checkAvarta || uinfo.IsRecommend()) {
				user.Nickname = uinfo.Nickname
				user.Age = uinfo.Age
				user.Avatar = uinfo.Avatar
				user.Gender = uinfo.Gender
				user.Building = uinfo.WorkPlaceName
				user.Distence = utils.Distence(utils.Coordinate{lat, lng}, utils.Coordinate{user.Lat, user.Lng})
				ret = append(ret, user)
			}
		}
	}
	return ret, nil
}

func readCache(key string, cur int, ps int, prototype interface{}) (exists bool, users []interface{}, total int, e error) {
	exists, e = cache.Exists(redis_db.CACHE_DISCOVERY, key)
	if e != nil {
		return false, nil, 0, e
	}
	total = 0
	uinfos := make([][]byte, 0, ps)
	if exists {
		conn := cache.GetReadConnection(redis_db.CACHE_DISCOVERY)
		defer conn.Close()
		total, e = redis.Int(conn.Do("LLEN", key))
		if e != nil {
			return false, nil, 0, e
		}
		fmt.Println("LLEN", key, total)
		start, end := utils.BuildRange(cur, ps, total)
		fmt.Println("LRANGE", key, "start", start, "end", end)
		v, e := redis.Values(conn.Do("LRANGE", key, start, end))
		if e != nil {
			return false, nil, 0, e
		}
		if e = redis.ScanSlice(v, &uinfos); e != nil {
			return false, nil, 0, e
		}
		users = make([]interface{}, 0, ps)
		switch user := prototype.(type) {
		case AdjUser:
			for _, b := range uinfos {
				if e = json.Unmarshal(b, &user); e != nil {
					return false, nil, 0, e
				}
				users = append(users, user)
			}
		case SearchUser:
			for _, b := range uinfos {
				if e = json.Unmarshal(b, &user); e != nil {
					return false, nil, 0, e
				}
				users = append(users, user)
			}
		}
		return true, users, total, nil
	} else {
		return false, nil, 0, nil
	}
}

func writeCache(key string, users interface{}, expire int) error {
	usersJson := make([]interface{}, 0, 100)
	usersJson = append(usersJson, key)
	switch v := users.(type) {
	case []AdjUser:
		if len(v) == 0 {
			return nil
		}
		for _, item := range v {
			b, e := json.Marshal(item)
			if e != nil {
				return e
			}
			usersJson = append(usersJson, b)
		}
	case []SearchUser:
		if len(v) == 0 {
			return nil
		}
		for _, item := range v {
			b, e := json.Marshal(item)
			if e != nil {
				return e
			}
			usersJson = append(usersJson, b)
		}
	default:
		return nil
	}
	conn := cache.GetWriteConnection(redis_db.CACHE_DISCOVERY)
	defer conn.Close()
	_, e := conn.Do("RPUSH", usersJson...)
	if e != nil {
		return e
	}
	_, e = conn.Do("EXPIRE", key, expire)
	return e
}

func clearCache(key string) error {
	return cache.Del(redis_db.CACHE_DISCOVERY, key)
}

func delInactiveUsers() {
	sql := "delete from discovery where online_timeout < ?"
	for {
		if _, e := sdb.Exec(sql, utils.Now.AddDate(0, -1, 0)); e != nil {
			mainLog.Append(fmt.Sprintf("delete inactive users in discovery error : %v", e.Error()))
		}
		time.Sleep(60 * time.Second)
	}
}

func delUser(msgid int, data interface{}) {
	switch v := data.(type) {
	case message.ClearCache:
		sql := "delete from discovery where id= ?"
		if _, e := sdb.Exec(sql, v.Uid); e != nil {
			mainLog.Append(fmt.Sprintf("delete user in discovery error : %v", e.Error()))
		}
	}
}

func RecommendStat(uid uint32, lat, lng float64) (data map[string]interface{}, e error) {
	if lat == common.LAT_NO_VALUE || lng == common.LNG_NO_VALUE {
		if lat, lng, e = general.UserLocation(uid); e != nil {
			return
		}
		if lat == common.LAT_NO_VALUE || lng == common.LNG_NO_VALUE {
			return
		}
	}
	data = map[string]interface{}{}
	city, province, e := general.City(lat, lng)
	if e != nil {
		return nil, e
	}
	//自己的省市
	data["province"] = province
	if general.IsZXS(province) {
		data["city"] = province
	} else {
		data["city"] = city
	}
	uinfo, e := user_overview.GetUserObject(uid)
	if e != nil {
		return nil, e
	}
	//自己的行业
	data["trade"] = uinfo.Trade

	maxLat, minLat := lat+common.LOCATION_RADIUS, lat-common.LOCATION_RADIUS
	maxLng, minLng := lng+common.LOCATION_RADIUS, lng-common.LOCATION_RADIUS

	//附近的人数
	sql := "select workplace from discovery where lat <= ? and lat >= ? and lng <= ? and lng >= ? and online_timeout > ?"
	var num int
	buildings := make(map[string]*building.Building, 10)
	rows, e := sdb.Query(sql, maxLat, minLat, maxLng, minLng, utils.Now.AddDate(0, 0, -ADJACENT_USER_TIMEOUT))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var bid string
		if e = rows.Scan(&bid); e != nil {
			return nil, e
		}
		buildings[bid] = nil
		num++
	}
	data["adj_users"] = num
	//附近大厦的人数
	if e = building.GetBuildingMap(buildings); e != nil {
		return nil, e
	}
	fmt.Println("buildings:", buildings)
	allBuildings := make(building.BuildingItems, 0, 10)
	for key, b := range buildings {
		if b != nil && key != "" {
			b.Distence = utils.Distence(utils.Coordinate{lat, lng}, utils.Coordinate{b.Lat, b.Lng})
			allBuildings = append(allBuildings, b)
		} else {
			fmt.Println("building", key, "not found")
		}
	}
	if len(allBuildings) > 0 {
		h := &allBuildings
		heap.Init(h)
		selected := make([]string, 0, 3)
		for i := 0; i < 3 && h.Len() > 0; i++ {
			b := heap.Pop(h).(*building.Building)
			selected = append(selected, b.Id)
		}
		sql = "select workplaceid,count(uid) from user_detail where workplaceid" + mysql.In(selected) + " and avatarlevel>? group by workplaceid"
		brows, e := mdb.Query(sql, common.AVLEVEL_INVALID)
		if e != nil {
			return nil, e
		}
		defer brows.Close()
		r := []interface{}{}
		for brows.Next() {
			var count int
			var id string
			if e = brows.Scan(&id, &count); e != nil {
				return nil, e
			}
			m := map[string]interface{}{}
			m["info"] = buildings[id]
			m["users"] = count
			r = append(r, m)
		}
		data["building"] = r
	}

	//同行人数
	sql = "select count(*) from user_detail where trade=? and province=?"
	if e = mdb.QueryRow(sql, uinfo.Trade, province).Scan(&num); e != nil {
		return nil, e
	}
	data["trade_users"] = num
	return
}
