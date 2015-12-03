package dynamics

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/common/user"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/comments"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/data_model/usercontrol"
	"yuanfen/yf_service/cls/unread"
)

func checkDynamicFlag(flag int, key string) (b bool) {
	//flag 1111   右起第一位 1 是否需要添加是否标记状态 2位 是否需要添加是否点赞状态 3位是否需要查询距离 4位是否需要查询在线状态
	switch key {
	case "isLike":
		if flag&2 == 2 {
			b = true
		}
	case "showDistance":
		if flag&4 == 4 {
			b = true
		}
	case "showOnlineStatus":
		if flag&8 == 8 {
			b = true
		}
	}
	return
}

/*
用来拼接动态用户信息和必要时间转化，图片转字符串数据等
flag 1111   右起第一位 1 是否需要添加是否标记状态 2位 是否需要添加是否点赞状态 3位是否需要查询距离 4位是否需要查询在线状态
uid 当前获取信息用户uid
*/
func GenDynamicsRes(v []Dynamic, uid uint32, flag int) (res []map[string]interface{}, e error) {
	res = make([]map[string]interface{}, 0, len(v))
	if len(v) <= 0 {
		return
	}
	uids := make([]uint32, 0, len(v))
	ids := make([]uint32, 0, len(v))
	game_ids := make([]uint32, 0, 10)
	for _, d := range v {
		uids = append(uids, d.Uid)
		ids = append(ids, d.Id)
		if d.Type == common.DYNAMIC_TYPE_GAME { //拼图游戏，获取对应的id
			game_ids = append(game_ids, d.Id)
		}
	}
	join_m, e := CheckIsJoinDynamicGames(uid, game_ids...)
	if e != nil {
		return
	}
	// 获取参与游戏的人数
	game_m := make(map[uint32]int)
	if len(game_ids) > 0 {
		game_m, e = GetGamesJoinNum(game_ids...)
		if e != nil {
			return
		}
	}
	// 是否点赞用户
	isLike := checkDynamicFlag(flag, "isLike")
	// 是否显示距离
	showDistance := checkDynamicFlag(flag, "showDistance")
	// 是否显示在线状态
	showOnlineStatus := checkDynamicFlag(flag, "showOnlineStatus")
	// 查询用户信
	m, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return
	}
	like_dy_m := make(map[uint32]int)
	if isLike {
		if like_dy_m, e = comments.GetLikeComentInIds(ids, uid); e != nil {
			return
		}
	}
	// 是否需要获取用户距离
	distance_m := make(map[uint32]float64)
	if showDistance {
		if distance_m, e = GetUserDistance(uid, uids); e != nil {
			return
		}
	}
	// 是否需要获取用户在线
	online_m := make(map[uint32]bool)
	if showDistance {
		if online_m, e = user_overview.IsOnline(uids...); e != nil {
			return
		}
	}
	for _, d := range v {
		if d.Status == DYNAMIC_STATUS_DELETE { // 删除的
			continue
		} else if uid != d.Uid && d.Status != DYNAMIC_STATUS_OK {
			continue
		}
		r := make(map[string]interface{})
		dy := make(map[string]interface{})
		dy["id"] = d.Id
		dy["uid"] = d.Uid
		dy["type"] = d.Type
		dy["stype"] = d.Stype
		pic_arr := make([]string, 0, 3)
		if d.Pic != "" {
			pic_arr = strings.Split(d.Pic, ",")
		}
		dy["text"] = d.Text
		dy["location"] = d.Location

		tm, er := utils.ToTime(d.Tm)
		if er != nil {
			mlog.AppendObj(er, "GenDynamicsRes is error", d)
			continue
		}
		dy["tm"] = tm
		dy["url"] = d.Url
		dy["gamekey"] = d.GamgeKey
		dy["gameinit"] = d.GamgeInit
		var join_game_num int
		if n, ok := game_m[d.Id]; ok {
			join_game_num = n
		}
		dy["join_game_num"] = join_game_num

		u := make(map[string]interface{})
		user := new(user_overview.UserViewItem)
		dpic_arr := make([]string, 0, len(pic_arr))
		if d.Type == common.DYNAMIC_TYPE_ARTICLE {
			if len(pic_arr) <= 0 {
				continue
			}
			user.Avatar = pic_arr[0]
			dpic_arr = pic_arr[1:]
		} else {
			us, ok := m[d.Uid]
			if !ok || us == nil {
				mlog.AppendObj(errors.New("get user is error"), "GenDynamicsRes get user is error")
				continue
			}
			user = us
			dpic_arr = pic_arr
		}

		u["uid"] = user.Uid
		u["age"] = user.Age
		u["nickname"] = user.Nickname
		u["avatar"] = user.Avatar
		u["job"] = user.Job
		u["city"] = user.City

		if showDistance {
			if l, ok := distance_m[d.Uid]; ok {
				u["distance"] = l
			}
		}
		u["isOnline"] = false
		if showOnlineStatus {
			if b, ok := online_m[d.Uid]; ok {
				u["isOnline"] = b
			}
		}
		dy["pic"] = dpic_arr
		dy["likes"] = d.Likes
		dy["comments"] = d.Comments
		// 添加上是否已经点过赞的标记
		if isLike {
			if _, ok := like_dy_m[d.Id]; ok {
				d.IsLike = 1
			}
		}
		d.Sign = 0
		dy["is_join"] = false
		mlog.AppendObj(nil, "join_m : ", join_m)
		if _, ok := join_m[d.Id]; ok {
			dy["is_join"] = true
		}

		dy["isLike"] = d.IsLike
		dy["sign"] = d.Sign
		r["dynamic"] = dy
		r["user"] = u
		res = append(res, r)
	}
	return
}

func CheckDynamicValid(dy Dynamic) (e service.Error) {
	if dy.Id <= 0 {
		return service.NewError(service.ERR_INTERNAL, "该动态不存在", "该动态不存在")
	}
	if dy.Status == DYNAMIC_STATUS_TXTINVALID {
		return service.NewError(service.ERR_NOERR, "", "")
	}
	if dy.Status == DYNAMIC_STATUS_DELETE {
		return service.NewError(service.ERR_INTERNAL, "该动态已删除", "该动态已删除")
	}
	if dy.Status == DYNAMIC_STATUS_BAN {
		return service.NewError(service.ERR_INTERNAL, "该动态已被关闭", "该动态已被关闭")
	}
	if dy.Status != DYNAMIC_STATUS_OK {
		return service.NewError(service.ERR_INTERNAL, "该动态不合法", "该动态不合法")
	}
	return service.NewError(service.ERR_NOERR, "", "")
}

func GetDynamicById(id uint32) (dy Dynamic, e error) {
	if id <= 0 {
		return
	}
	a, e := GetDynamicsByIds(id)
	if e != nil {
		return
	}
	for _, v := range a {
		if v.Id == id {
			return v, nil
		}
	}
	return
}

/*
批量获取动态列表  ids 动态id
*/
func GetDynamicsByIds(ids ...uint32) (a []Dynamic, e error) {
	a = make([]Dynamic, 0, len(ids))
	if len(ids) <= 0 {
		return
	}
	dym, unids, e := readDynamicCache(ids)
	if e != nil {
		return
	}
	m := make(map[uint32]Dynamic)
	for _, d := range dym {
		m[d.Id] = d
	}
	// 缓存没有，需要查询数据库
	sarr := make([]Dynamic, 0, len(unids))
	if len(unids) > 0 {
		s := "select id,uid,type,stype,pic,text,location,comments,likes,report,tm,status,url,game_key,game_init from dynamics where id in(" + utils.Uint32ArrTostring(unids) + ")"
		rows, e := mdb.Query(s)
		if e != nil {
			return a, e
		}
		mlog.AppendObj(nil, "GetDynamicsByIds from sql : ", s, ids)
		defer rows.Close()
		for rows.Next() {
			var dy Dynamic
			if e = rows.Scan(&dy.Id, &dy.Uid, &dy.Type, &dy.Stype, &dy.Pic, &dy.Text, &dy.Location, &dy.Comments, &dy.Likes, &dy.Report, &dy.Tm, &dy.Status, &dy.Url, &dy.GamgeKey, &dy.GamgeInit); e != nil {
				return a, e
			}
			m[dy.Id] = dy
			sarr = append(sarr, dy)
		}
		if e = writeDymamicCache(sarr); e != nil {
			return a, e
		}
	}

	for _, id := range ids {
		if v, ok := m[id]; ok {
			a = append(a, v)
		}
	}
	return
}

/*
及时获取动态信息
*/
func GetDynamicByIdNoCache(tx utils.SqlObj, id uint32) (dy Dynamic, e error) {
	s := "select id,uid,type,stype,pic,text,location,comments,likes,report,tm,status,url,game_key,game_init from dynamics where id = ?"
	rows, e := tx.Query(s, id)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&dy.Id, &dy.Uid, &dy.Type, &dy.Stype, &dy.Pic, &dy.Text, &dy.Location, &dy.Comments, &dy.Likes, &dy.Report, &dy.Tm, &dy.Status, &dy.Url, &dy.GamgeKey, &dy.GamgeInit); e != nil {
			return
		}
	}
	return
}

/*
及时获取动态信息
*/
func GetDynamicByIdFromMain(id uint32) (dy Dynamic, e error) {
	s := "select id,uid,type,stype,pic,text,location,comments,likes,report,tm,status,url,game_key,game_init from dynamics where id = ?"
	rows, e := mdb.QueryFromMain(s, id)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&dy.Id, &dy.Uid, &dy.Type, &dy.Stype, &dy.Pic, &dy.Text, &dy.Location, &dy.Comments, &dy.Likes, &dy.Report, &dy.Tm, &dy.Status, &dy.Url, &dy.GamgeKey, &dy.GamgeInit); e != nil {
			return
		}
	}
	return
}

/*
获取历史动态列表,起始值为s ,获取后面的ps条数据
key:省动态key
s: 最后一条已请求动态id
gender : 查询的性别
ps: 请求数量
ex_ids:去除动态id
*/
func GetDynamicList(key string, s uint32, gender, ps int, ex_ids []uint32, requid uint32) (r []Dynamic, e error) {
	if s <= 0 {
		s = math.MaxUint32
		ps += 20
	}
	score := MakeDynamicScore(s, gender, 16)
	ids := make([]uint32, 0, ps)
doSeach:
	is, e := getDynamics(key, score, ps)
	if e != nil {
		return
	}
	mlog.AppendObj(nil, "GetDynamicList--is", score, is)
	if len(is) > 0 {
		uids := make([]uint32, 0, len(ids))
		var index int
		for _, i := range is {
			if gender == 0 || (gender > 0 && gender == GetGenderFromScore(int64(i.Score), 16)) {
				if uid, id, e := getFromItemScoreKey(i); e == nil && id > 0 {
					uids = append(uids, uid)
				}
			}
		}
		um, e := user_overview.CheckUserDynamicStatus(uids)
		if e != nil {
			return r, e
		}
		um[requid] = true
		mlog.AppendObj(nil, "GetDynamicList--um ", uids, len(is), um)
		for k, i := range is {
			index = k
			if uid, id, e := getFromItemScoreKey(i); e == nil && id > 0 {
				if b, ok := um[uid]; ok && b {
					ids = append(ids, id)
				}
				if len(ids) >= ps {
					break
				}
			}
		}
		mlog.AppendObj(nil, "GetDynamicList--ids", ids, score, len(ids))
		score = int64(is[index].Score)
		// 不够条数，需要再去列表中取一次
		if len(ids) < ps {
			goto doSeach
		}
	}
	final_ids := utils.Uint32ArrDiff(ids, ex_ids)
	r, e = GetDynamicsByIds(final_ids...)
	return
}

/*
获取标记动态列表,起始值为s ,获取后面的ps条数据
s: 最后一条已请求动态id
ps: 请求数量
*/
func GetMarkDynamicList(key string, luid, s uint32, ps int, mu map[uint32]time.Time) (r []Dynamic, e error) {
	if s <= 0 {
		s = math.MaxUint32
	}
	score := MakeDynamicScore(s, 0, 16)
	uids := make([]uint32, 0, len(mu))
	m := make(map[uint32]bool)
	for k, _ := range mu {
		uids = append(uids, k)
	}
	m, e = user_overview.CheckUserDynamicStatus(uids)
	if len(m) <= 0 {
		return
	}
	ids := make([]uint32, 0, ps)
doSeach:
	is, e := getDynamics(key, score, ps)
	if e != nil {
		return
	}
	mlog.AppendObj(nil, "GetMarkDynamicList--getRes:", score, len(is))
	if len(is) > 0 {
		var index int
		for k, i := range is {
			index = k
			uid, did, e := getFromItemScoreKey(i)
			if e != nil {
				mlog.AppendObj(e, " fromItem ", i)
				continue
			}
			// 是否为我的标记用户
			if _, ok := m[uid]; ok {
				ids = append(ids, did)
				if len(ids) >= ps {
					break
				}
			}
		}
		mlog.AppendObj(nil, "GetMarkDynamicList--res:", ids, score, index)
		score = int64(is[index].Score)
		// 不够条数，需要再去列表中取一次
		if len(ids) < ps {
			goto doSeach
		}
	}
	//如果查询完还是不够，则需要从unionstore中获取
	if len(ids) < ps {
		mlog.AppendObj(nil, "-------- all redis is not enough,need ger from unionstroe ", score, "all getNum: ", len(ids))
		muids := make([]uint32, 0, len(m))
		for uid, _ := range m {
			muids = append(muids, uid)
		}
		is, er := getDynamicFromUnionStrore(luid, score, ps, muids)
		if er != nil {
			return r, er
		}
		mlog.AppendObj(nil, "GetUnionMarkDynamicList--is", score, len(is), muids)
		if len(is) > 0 {
			for _, i := range is {
				_, did, e := getFromItemScoreKey(i)
				if e != nil {
					mlog.AppendObj(e, " fromItem ", i)
					continue
				}
				// 是否为我的标记用户
				ids = append(ids, did)
				if len(ids) >= ps {
					break
				}
			}
		}
	}
	r, e = GetDynamicsByIds(ids...)
	return
}

/*
通过unionstroe中获取列表
*/
func getDynamicFromUnionStrore(uid uint32, score int64, ps int, muids []uint32) (r []redis.ItemScore, e error) {
	key, e := doUnionStroe(uid, muids)
	if e != nil {
		return
	}
	fmt.Println("getDynamicFromUnionStrore key: ", key)
	return rdb.ZREVRangeByScoreWithScores(redis_db.REDIS_DYNAMIC, key, math.MinInt64, score, ps)
}

func doUnionStroe(uid uint32, muids []uint32) (key string, e error) {
	key = "union_dynamic_list_" + utils.ToString(uid)
	// 首先检测是否已经有存在的unionStroe，如果已经有则返回key
	ex, e := rdb.Exists(redis_db.REDIS_DYNAMIC, key)
	if e != nil {
		return
	}
	mlog.AppendObj(nil, "doUnionStroe ex :", ex)
	if ex {
		return
	}
	keys := make([]interface{}, 0, len(muids))
	weights := make([]interface{}, 0, len(muids))
	for _, uid := range muids {
		keys = append(keys, GetUserDynamicKey(uid))
		weights = append(weights, 1)
	}
	// 没有则需要先进行unionStroe 然后在返回
	e = rdb.ZUnionSrore(redis_db.REDIS_DYNAMIC, key, 3600, keys, weights, "SUM")
	return
}

/*
根据itemscore获取key中的uid和id，已经score中的tm 和gender
*/
func getFromItemScoreKey(i redis.ItemScore) (uid, id uint32, e error) {
	a := strings.Split(i.Key, ",")
	if len(a) < 2 {
		mlog.AppendObj(errors.New("get ItemScore is error"), "getFromItemScoreKey is error", i)
		return
	}
	uid, e = utils.ToUint32(a[0])
	if e != nil {
		return
	}
	id, e = utils.ToUint32(a[1])
	return
}

func MakeDynamicKey(id, uid uint32) string {
	return utils.ToString(uid) + "," + utils.ToString(id)
}

/*
获取历史动态列表,起始值为s ,获取后面的ps条数据
s: 最后一条已请求动态id
gender : 查询的性别
ps: 请求数量
*/
func getDynamics(key string, score int64, ps int) (r []redis.ItemScore, e error) {
	return rdb.ZREVRangeByScoreWithScores(redis_db.REDIS_DYNAMIC, key, math.MinInt64, score, ps)
}

/*
新增一条动态,需要有事务
*/
func AddDynamic(dy Dynamic) (id uint32, e error) {
	s := "insert into dynamics(uid,type,stype,pic,text,location,url,game_key,game_init,status) values(?,?,?,?,?,?,?,?,?,?)"
	tx, e := mdb.Begin()
	if e != nil {
		return
	}
	rs, e := tx.Exec(s, dy.Uid, dy.Type, dy.Stype, dy.Pic, dy.Text, dy.Location, dy.Url, dy.GamgeKey, dy.GamgeInit, dy.Status)
	if e != nil {
		tx.Rollback()
		return
	}
	i, e := rs.LastInsertId()
	if e != nil {
		tx.Rollback()
		return
	}
	id = uint32(i)
	d, e := GetDynamicByIdNoCache(tx, id)
	if e != nil || d.Id <= 0 {
		tx.Rollback()
		return
	}
	tx.Commit()
	go stat.Append(dy.Uid, stat.ACTION_DYNAMICS, nil)
	return
}

/*
 发布动态（异步图片检测，消息push）
 id : 动态id
 needCheck: 是否需要图片检测，true 需要图片检测，false 不需要
*/
func DoCheckPicAndPush(id uint32, needCheck bool) (e error) {
	d, e := GetDynamicByIdFromMain(id)
	if e != nil || d.Id <= 0 {
		return
	}
	error_pic := make([]string, 0, 5)
	// 如果有图片执行图片审核
	if needCheck && d.Pic != "" {
		pic_arr := strings.Split(d.Pic, ",")
		if cm, er := general.CheckImg(general.IMGCHECK_SEXY_AND_AD, pic_arr...); er == nil {
			for pic_url, v := range cm {
				if v.Status != DYNAMIC_STATUS_OK { // 成功
					error_pic = append(error_pic, pic_url)
				}
			}
		} else {
			mlog.AppendObj(er, "add dynamic CheckImg is error ", d.Uid, d)
		}
	}
	// 如果图片有审核失败的，
	if len(error_pic) > 0 {
		e = UpdateDynamicStatus(mdb, id, DYNAMIC_STATUS_PICINVALID)
		mlog.AppendObj(e, "add dynamic CheckImg has valid ", d, error_pic)
	} else if d.Pic == "" && len(d.Text) <= 20 {
		// 纯文本动态，文字是否小于20字符
		e = UpdateDynamicStatus(mdb, id, DYNAMIC_STATUS_TXTINVALID)
		mlog.AppendObj(nil, "text is less than 20  no PushDynamicMsg ", d.Uid, d)
	}
	//添加到reids中
	if d.Type != common.DYNAMIC_TYPE_ARTICLE {
		if e = AddDynamicRedis(d.Uid, id); e != nil {
			return
		}
	}
	if b, e := user_overview.CheckUserDynamicStatusByUid(d.Uid); e == nil && b {
		e = PushDynamicMsg(d.Uid, d.Id)
	}
	return
}

/*
将动态添加到redis中
*/
func AddDynamicRedis(uid, id uint32) (e error) {
	u, e := user_overview.GetUserObject(uid)
	if e != nil || u == nil {
		return
	}
	// 未完成资料必填项，不加入到同城和个人redis中
	if addRedis, e := user_overview.CompleteMust(uid); e != nil || !addRedis {
		mlog.AppendObj(e, "CompleteMust is false ", uid, addRedis)
		return e
	}
	score := MakeDynamicScore(id, u.Gender, 16)
	key := MakeProvinceDyanmicKey(u.Province)
	if e = rdb.ZAdd(redis_db.REDIS_DYNAMIC, key, score, MakeDynamicKey(id, uid)); e != nil {
		return
	}
	e = rdb.ZAdd(redis_db.REDIS_DYNAMIC, GetUserDynamicKey(uid), score, MakeDynamicKey(id, uid))
	go deleteDynamicRedis(key)
	return
}

/*
个人动态记录Key
*/
func GetUserDynamicKey(uid uint32) string {
	return "user_dynamic_list_" + utils.ToString(uid)
}

func deleteDynamicRedis(key string) (e error) {
	tm := utils.Now.AddDate(0, -3, 0)
	s := "select id from dynamics where tm < ? and type =1 order by id desc limit 1"
	var id uint32
	rows, e := mdb.Query(s, tm)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&id); e != nil {
			return
		}
	}
	if id <= 0 {
		return
	}
	// 计算出超过3个月的时间的score值，并将其从有序集合中移除
	score := MakeDynamicScore(id, 2, 16)
	e = rdb.ZRemRangeByScore(redis_db.REDIS_DYNAMIC, key, 0, score)
	return
}

/*
根据id，修改动态status字段
*/
func UpdateDynamicStatus(tx utils.SqlObj, id uint32, status int) (e error) {
	s := "update dynamics set status = ? where id = ?"
	_, e = tx.Exec(s, status, id)
	return
}

/*
根据id，修改动态评论和点赞数,举报数
*/
func UpdateComDynamic(tx utils.SqlObj, id uint32, like_add, comments_add, reprot_add int) (e error) {
	s := "update dynamics set likes = likes + ?,comments = comments + ? ,report = report+? where id = ? "
	_, e = tx.Exec(s, like_add, comments_add, reprot_add, id)
	go ClearDynamicCache(id)
	return
}

/*
查询某用户是否参与某动态游戏
*/
func CheckIsJoinDynamicGame(id, uid uint32) (cid uint32, e error) {
	s := "select id from comment where source_id = ? and uid = ? and source_type =1 and type=3 limit 1"
	rows, e := mdb.Query(s, id, uid)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&cid); e != nil {
			return
		}
	}
	return
}

/*
查询某用户是否参与某几个动态游戏（map[id]uid
*/
func CheckIsJoinDynamicGames(uid uint32, ids ...uint32) (m map[uint32]uint32, e error) {
	m = make(map[uint32]uint32)
	if len(ids) <= 0 {
		return
	}
	s := "select source_id from comment where source_id  " + mysql.In(ids) + " and uid = ? and source_type =1 and type=3 "
	mlog.AppendObj(nil, "sql : ", s, uid)
	rows, e := mdb.Query(s, uid)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var cid uint32
		if e = rows.Scan(&cid); e != nil {
			return
		}
		m[cid] = cid
	}
	return
}

/*
查询拼图游戏参与人数
*/func GetGamesJoinNum(ids ...uint32) (m map[uint32]int, e error) {
	m = make(map[uint32]int)
	if len(ids) <= 0 {
		return
	}
	s := "select source_id,IFNULL(count(*),0) as num from  comment where source_id " + mysql.In(ids) + "  group by source_id having num >0 "
	rows, e := mdb.Query(s)
	if e != nil {
		return
	}
	for rows.Next() {
		var n int
		var id uint32
		if e = rows.Scan(&id, &n); e != nil {
			return
		}
		m[id] = n
	}
	return
}

/*
获取文章
*/
func GetArticle() (dy Dynamic, e error) {
	if d, e := readDynamicArticleCache(); e == nil && d.Id > 0 {
		fmt.Println("GetArticle from cache", d)
		return d, nil
	}
	s := "select id,uid,type,stype,pic,text,location,comments,likes,report,tm,status,url,game_key,game_init from dynamics where type = 3 and status = 0 order by tm desc limit 1 "
	rows, e := mdb.Query(s)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&dy.Id, &dy.Uid, &dy.Type, &dy.Stype, &dy.Pic, &dy.Text, &dy.Location, &dy.Comments, &dy.Likes, &dy.Report, &dy.Tm, &dy.Status, &dy.Url, &dy.GamgeKey, &dy.GamgeInit); e != nil {
			return
		}
		if er := writeDymamicArticleCache(dy); er != nil {
			return dy, er
		}
	}
	return
}

/*
个人中心，获取我的动态接口
s: 当前最新动态id
uid:用户uid
ps: 请求数量
*/
func GetMydynamicList(s, uid uint32, ps int) (r []Dynamic, e error) {
	if s <= 0 {
		s = math.MaxUint32
	}
	sql := "select id from dynamics where uid = ? and (status != " + utils.ToString(DYNAMIC_STATUS_DELETE) + " ) and id <? order by id desc limit ?"
	ids := make([]uint32, 0, ps)

	rows, e := mdb.Query(sql, uid, s, ps)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id uint32
		if e = rows.Scan(&id); e != nil {
			return
		}
		ids = append(ids, id)
	}
	mlog.AppendObj(nil, "GetDynamicList--ids", ids)
	r, e = GetDynamicsByIds(ids...)
	return
}

/* 获取我的动态
uid:用户uid
score : 根据时间构造score上线
ps: 请求数量
*/
func getMyDynamics(uid uint32, score int64, ps int) (r []redis.ItemScore, e error) {
	return rdb.ZREVRangeByScoreWithScores(redis_db.REDIS_DYNAMIC, GetUserDynamicKey(uid), math.MinInt64, score, ps)
}

/*
标记用户新增动态通知
*/
func PushDynamicMsg(uid, id uint32) (e error) {
	uids := make([]uint32, 0, 10)
	m, e := general.GetInterestedUids(uid)
	if e != nil {
		return
	}
	if len(m) <= 0 {
		return
	}
	for k, _ := range m {
		uids = append(uids, k)
	}
	msg := make(map[string]interface{})
	unread_res := map[string]interface{}{common.UNREAD_DYNAMIC_MARK: unread.Item{Num: 1, Show: "红点"}}
	msg[common.UNREAD_KEY] = unread_res
	_, e = general.SendMsgM(uid, uids, msg, "")
	mlog.AppendObj(e, "--push dynamic unread--", uid, uids, msg)
	return
}

/*
发送评论和点赞消息（评论，点赞，玩游戏）
*/
func PushCommentMsg(c comments.Comment, d Dynamic) (e error) {
	if c.Id <= 0 || d.Id <= 0 {
		return
	}
	if b, e := user_overview.CheckUserDynamicStatusByUid(c.Uid); e != nil || !b {
		return e
	}
	var t, folder string
	var tuid, fuid uint32
	if c.Ruid > 0 { // 回复评论 通知被回复人
		if c.Ruid != d.Uid { // 被回复人非动态用户
			// 非自己的被回复
			t = common.MSG_TYPE_DYNAMIC_MSG
			tuid = c.Ruid
			folder = common.FOLDER_OTHER
			fuid = common.UID_COMMENT

		} else {
			//自己的动态被回复
			t = common.MSG_TYPE_MYDYNAMIC_MSG
			tuid = d.Uid
			folder = common.FOLDER_HIDE
			fuid = common.UID_COMMENT_TOME
		}
	} else { // 评论，一定通知动态用户
		t = common.MSG_TYPE_MYDYNAMIC_MSG
		tuid = d.Uid
		folder = common.FOLDER_HIDE
		fuid = common.UID_COMMENT_TOME
	}
	if e = comments.UpdateCommentFlag(mdb, c.Id, 1); e != nil {
		return
	}

	if user.IsKfUser(tuid) {
		return
	}
	if c.Uid == tuid {
		mlog.AppendObj(nil, "push is to myself", c.Uid, tuid)
		return
	}
	// 获取用户隐私设置，确定是否进行第三方推送
	var isPush bool
	p, e := general.GetUserProtect(tuid)
	if e != nil {
		mlog.AppendObj(e, "get user project is error")
		return
	}
	if c.Type == common.COMMENT_TYPE_LIKE { // 点赞
		isPush = p.LikeNotNotify == 0
	} else { // 评论
		isPush = p.CommentNotNotify == 0
	}

	cres, e := comments.GenCommentInfo([]comments.Comment{c}, c.Uid)
	if e != nil || len(cres) < 1 {
		return
	}
	dres, e := GenDynamicsRes([]Dynamic{d}, tuid, 3)
	if e != nil || len(dres) < 1 {
		return
	}
	msg := make(map[string]interface{})
	msg["type"] = t
	msg["folder"] = folder
	msg["comment_info"] = cres[0]
	msg["dynamic_info"] = dres[0]
	msid, _, e := general.SendMsgWithOnline(fuid, tuid, msg, "", isPush)
	mlog.AppendObj(e, "--push dynamic msg--", fuid, tuid, msid, msg, isPush)
	if e != nil {
		return
	}
	return
}

// 同城动态列表(REDIS_DYNAMIC_KEY+pro)
func MakeProvinceDyanmicKey(pro string) (key string) {
	return REDIS_DYNAMIC_KEY + pro
}

// 优秀同城动态列表(REDIS_EX_DYNAMIC_KEY+pro)
func MakeExProvinceDyanmicKey(pro string) (key string) {
	//return REDIS_EX_DYNAMIC_KEY + pro
	return REDIS_EX_DYNAMIC_KEY
}

//生成score，id存储在64位整型的前(64-bits)位，gender会存储在后bits位。
func MakeDynamicScore(id uint32, gender int, bits uint) (score int64) {
	score = (int64(id) << bits) + int64(gender)
	fmt.Printf("tm=%b,tag=%b,score=%b(%u)\n", id, gender, score)
	return
}

//从score中提取gender
func GetGenderFromScore(score int64, bits uint) int {
	fmt.Printf("score=%b,tag=%b\n", score, score&((int64(1)<<bits)-1))
	return int(score & ((int64(1) << bits) - 1))
}

//从score中提取动态id
func GetIdFromScore(score int64, bits uint) uint32 {
	fmt.Printf("tm=%b\n", score>>bits)
	return uint32(score >> bits)
}

/*
 封禁用户后，封禁该用户的动态
*/func CloseUserDynamic(uid uint32) (e error) {
	/*	u, e := user_overview.GetUserObject(uid)
		if e != nil {
			return
		}
		s := "select id from dynamics where uid = ? and status = 0 "
		rows, e := mdb.Query(s, uid)
		if e != nil {
			return
		}
		defer rows.Close()
		key := MakeProvinceDyanmicKey(u.Province)
		args := make([]interface{}, 0, 10)
		for rows.Next() {
			var id uint32
			if e = rows.Scan(&id); e != nil {
				return
			}
			args = append(args, key, MakeDynamicKey(id, u.Uid))
		}
		if len(args) <= 0 {
			return
		}
		// 批量移除同城动态和各自动态redis列表
		if _, e = rdb.ZRem(redis_db.REDIS_DYNAMIC, args...); e != nil {
			return
		}
		// 删除自己redis个人
		if e = rdb.Del(redis_db.REDIS_DYNAMIC, GetUserDynamicKey(u.Uid)); e != nil {
			return
		}
	*/
	// 并修改自己的所有未被封禁改为封禁
	s2 := "update dynamics set status = 2 where uid = ? and status =" + utils.ToString(DYNAMIC_STATUS_OK)
	_, e = mdb.Exec(s2, uid)
	return
}

// 检测是否需要推荐
func CheckRecomdDynamic(uid uint32) (r bool) {
	u, e := usercontrol.GetUserInfo(uid)
	if e != nil {
		return
	}
	tm, e := utils.ToTime(u["reg_time"])
	if e != nil {
		return
	}
	return utils.Now.AddDate(0, 0, -1).Before(tm)
}

/*
随机获取n推荐动态
key:省动态key
gender : 查询的性别
*/
func GetExDynamics(key string, gender, n int) (r []Dynamic, ex_ids []uint32, e error) {
	score := MakeDynamicScore(math.MaxUint32, gender, 16)
	ps := 100
	ids := make([]uint32, 0, ps)
	is, e := getDynamics(key, score, ps)
	if e != nil {
		return
	}
	for _, i := range is {
		if gender == 0 || (gender > 0 && gender == GetGenderFromScore(int64(i.Score), 16)) {
			if _, id, e := getFromItemScoreKey(i); e == nil && id > 0 {
				ids = append(ids, id)
			}
		}
	}
	ex_ids = make([]uint32, 0, n)
	if len(ids) > n {
		for _, i := range general.RandNumMap(0, len(ids)-1, n) {
			ex_ids = append(ex_ids, ids[i])
		}
	} else {
		ex_ids = ids
	}
	r = make([]Dynamic, 0, n)
	if len(ex_ids) <= 0 {
		return
	}
	r, e = GetDynamicsByIds(ex_ids...)
	return
}
