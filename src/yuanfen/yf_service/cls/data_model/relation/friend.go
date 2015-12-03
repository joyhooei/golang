package relation

import (
	"container/heap"
	"errors"
	"fmt"
	"sort"
	"time"
	"yf_pkg/lbs/baidu"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/notify"
)

const MAKE_DATE_KEY = "make_date"
const DATE_REQUEST_INTERVAL = 100 //约会请求通知发出的间隔(秒)

//删除认识关系
func DelFriend(from, to uint32) (e error) {
	if _, e := mdb.Exec("update follow set friend=0 where f_uid=? and t_uid=?", from, to); e != nil {
		return e
	}
	if e = DelRecentChatUser(from, to); e != nil {
		return e
	}
	_, e = rdb.ZRem(redis_db.REDIS_FRIENDS, from, to, to, from)
	return e
}

//GetFriendUids获取认识的人的列表
func GetFriendUids(uid uint32, cur, ps int) (users map[uint32]bool, total int, e error) {
	var items []uint32
	total, e = rdb.ZRangePS(redis_db.REDIS_FRIENDS, uid, cur, ps, true, &items)
	if e != nil {
		return nil, 0, e
	}
	users = make(map[uint32]bool, len(items)*2)
	for _, v := range items {
		users[v] = true
	}
	return
}

//Friends获取认识的人的列表
func Friends(uid uint32, cur, ps int) (users []Friend, total int, e error) {
	items, total, e := rdb.ZRangeWithScoresPS(redis_db.REDIS_FRIENDS, uid, cur, ps)
	if e != nil {
		return nil, 0, e
	}
	users, e = makeFriendUsersInfo(uid, items)
	return
}

//想要和对方约会
func WantDate(me, him uint32, res map[string]interface{}) (isAvailable bool, e error) {
	var uid1, uid2 uint32
	if me > him {
		uid1, uid2 = him, me
	} else if him > me {
		uid1, uid2 = me, him
	} else {
		return false, service.NewError(service.ERR_INVALID_REQUEST, "cannot date yourself", "不能和自己约会")
	}
	sqlStr := "select request1,request2,available from date_request where uid1=? and uid2=?"
	var request1, request2, available int
	err := mdb.QueryRow(sqlStr, uid1, uid2).Scan(&request1, &request2, &available)
	switch err {
	case mysql.ErrNoRows:
		sql := "insert into date_request(uid1,uid2,request1,request2)values(?,?,?,?)"
		if uid1 == me {
			_, e = mdb.Exec(sql, uid1, uid2, 1, 0)
		} else {
			_, e = mdb.Exec(sql, uid1, uid2, 0, 1)
		}
		if e != nil {
			return false, e
		}
	case nil:
		if available > 0 {
			return true, nil
		}
		if uid1 == me && request1 == 0 {
			request1 = 1
			sql := "update date_request set request1=1 where uid1=? and uid2=?"
			if _, e = mdb.Exec(sql, uid1, uid2); e != nil {
				return false, e
			}
		} else if uid2 == me && request2 == 0 {
			request2 = 1
			sql := "update date_request set request2=1 where uid1=? and uid2=?"
			if _, e = mdb.Exec(sql, uid1, uid2); e != nil {
				return false, e
			}
		}
		if request1+request2 >= 2 {
			if e := rdb.ZAddOpt(redis_db.REDIS_MISC, "NX", MAKE_DATE_KEY, utils.Now.Add(DATE_REQUEST_INTERVAL*time.Second).Unix(), general.MakeKey(uid1, uid2)); e != nil {
				return false, e
			}
		}
	default:
		return false, err
	}
	uinfo, e := user_overview.GetUserObject(him)
	if e != nil {
		return false, e
	}
	him_str := "他"
	if uinfo.Gender == 2 {
		him_str = "她"
	}
	str := "你已有意向与" + him_str + "见面，咖啡交友致力于为用户打造自然的交友体验，真正的能够到生活中去认识。"
	//	notify.AddTipMsg(res, fmt.Sprintf("如果\"%v\"也有意向见面，系统则会通知女方来选择约会地点和时间\n", uinfo.Nickname), "查看官方见面地点  >>", notify.CMD_DATEPLACE, map[string]interface{}{"uid": me})
	notify.AddTipMsg(res, str, "", "", nil)
	return false, nil
}

//取消想要和对方约会
func CancelDate(me, him uint32, res map[string]interface{}) (e error) {
	var uid1, uid2 uint32
	if me > him {
		uid1, uid2 = him, me
	} else if him > me {
		uid1, uid2 = me, him
	}
	notify.AddTipMsg(res, "您已取消见面意向", "", "", nil)
	sql := "select request1,request2,available from date_request where uid1=? and uid2=?"
	request1, request2, available := 0, 0, 0
	e = mdb.QueryRow(sql, uid1, uid2).Scan(&request1, &request2, &available)
	switch e {
	case mysql.ErrNoRows:
		return nil
	case nil:
		if available > 0 {
			return service.NewError(service.ERR_INVALID_REQUEST, "already dated", "不能取消已成功配对的约会请求")
		}
		if uid1 == me {
			if request1 == 1 {
				sql := "update date_request set request1=? where uid1=? and uid2=?"
				_, e = mdb.Exec(sql, 0, uid1, uid2)
			}
		} else {
			if request2 == 1 {
				sql := "update date_request set request2=? where uid1=? and uid2=?"
				_, e = mdb.Exec(sql, 0, uid1, uid2)
			}
		}
		if e != nil {
			return e
		}
		//删除有可能存在的约会请求消息
		if request1+request2 >= 2 {
			if _, e := rdb.ZRem(redis_db.REDIS_MISC, MAKE_DATE_KEY, general.MakeKey(uid1, uid2)); e != nil {
				return e
			}
		}
	default:
		return e
	}
	return
}
func GetRecommendDatePlaces(me, him uint32) (places []Place, mWorkPlace, hWorkPlace UserWorkPlace, e error) {
	uinfos, e := user_overview.GetUserObjects(me, him)
	if e != nil {
		return nil, mWorkPlace, hWorkPlace, e
	}
	minfo, hinfo := uinfos[me], uinfos[him]
	if minfo == nil || hinfo == nil {
		return nil, mWorkPlace, hWorkPlace, errors.New("get userinfo error")
	}
	if minfo.Province != hinfo.Province {
		return nil, mWorkPlace, hWorkPlace, service.NewError(service.ERR_NOT_SAME_PROVINCE, "need same province", "仅支持同省约会")
	}
	mWorkPlace = UserWorkPlace{minfo.Uid, minfo.Nickname, minfo.Gender, minfo.Avatar, utils.Coordinate{minfo.WorkLat, minfo.WorkLng}}
	hWorkPlace = UserWorkPlace{hinfo.Uid, hinfo.Nickname, hinfo.Gender, hinfo.Avatar, utils.Coordinate{hinfo.WorkLat, hinfo.WorkLng}}
	if minfo.WorkPlaceId == "" {
		if mWorkPlace.Location.Lat, mWorkPlace.Location.Lng, e = general.UserLocation(minfo.Uid); e != nil {
			return nil, mWorkPlace, hWorkPlace, e
		}
	}
	if hinfo.WorkPlaceId == "" {
		if hWorkPlace.Location.Lat, hWorkPlace.Location.Lng, e = general.UserLocation(hinfo.Uid); e != nil {
			return nil, mWorkPlace, hWorkPlace, e
		}
	}
	central := utils.Coordinate{(minfo.WorkLat + hinfo.WorkLat) / 2, (minfo.WorkLng + hinfo.WorkLng) / 2}
	sql := "select id,name,address,pic,lat,lng from date_place where province=?"
	rows, e := mdb.Query(sql, minfo.Province)
	if e != nil {
		return nil, mWorkPlace, hWorkPlace, e
	}
	defer rows.Close()
	allPlaces := make(PlaceItems, 0, 200)
	for rows.Next() {
		var p Place
		if e := rows.Scan(&p.Id, &p.Name, &p.Address, &p.Pic, &p.Lat, &p.Lng); e != nil {
			return nil, mWorkPlace, hWorkPlace, e
		}
		p.Distence = utils.Distence(mWorkPlace.Location, utils.Coordinate{p.Lat, p.Lng}) + utils.Distence(hWorkPlace.Location, utils.Coordinate{p.Lat, p.Lng})
		allPlaces = append(allPlaces, p)
	}
	h := &allPlaces
	heap.Init(h)
	places = make([]Place, 0, 20)
	for i := 0; i < 20 && h.Len() > 0; i++ {
		place := heap.Pop(h).(Place)
		place.Distence = utils.Distence(utils.Coordinate{place.Lat, place.Lng}, central)
		places = append(places, place)
	}
	sort.Sort(PlaceItems(places))
	return
}

//获取备选约会地点
func GetDatePlaces(lat, lng float64) (places []Place, e error) {
	//根据用户给的经纬度获取约会地点
	maxLat, minLat := lat+common.DATEPLACE_RADIUS, lat-common.DATEPLACE_RADIUS
	maxLng, minLng := lng+common.DATEPLACE_RADIUS, lng-common.DATEPLACE_RADIUS
	sql := "select id,name,address,pic,lat,lng from date_place where lat>=? and lat<=? and lng>=? and lng<=?"
	rows, e := mdb.Query(sql, minLat, maxLat, minLng, maxLng)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	allPlaces := make(PlaceItems, 0, 200)
	for rows.Next() {
		var p Place
		if e := rows.Scan(&p.Id, &p.Name, &p.Address, &p.Pic, &p.Lat, &p.Lng); e != nil {
			return nil, e
		}
		p.Distence = utils.Distence(utils.Coordinate{lat, lng}, utils.Coordinate{p.Lat, p.Lng})
		allPlaces = append(allPlaces, p)
	}
	h := &allPlaces
	heap.Init(h)
	places = make([]Place, 0, 10)
	for i := 0; i < 10 && h.Len() > 0; i++ {
		place := heap.Pop(h).(Place)
		places = append(places, place)
	}
	return
}

/*
获取约会状态
*/
func GetDateStatus(me, him uint32) (status int, e error) {
	var uid1, uid2 uint32
	if me > him {
		uid1, uid2 = him, me
	} else if him > me {
		uid1, uid2 = me, him
	} else {
		return 0, service.NewError(service.ERR_INVALID_REQUEST, "cannot date yourself", "不能和自己约会")
	}
	var available, request1, request2 int
	e = mdb.QueryRow("select request1,request2,available from date_request where uid1=? and uid2=?", uid1, uid2).Scan(&request1, &request2, &available)
	switch e {
	case mysql.ErrNoRows:
		return 0, nil
	case nil:
		if available > 0 {
			return 2, nil
		}
		if (me > him && request2 == 1) || (me < him && request1 == 1) {
			return 1, nil
		} else {
			return 0, nil
		}
	default:
		return 0, e
	}
	return
}

/*
发起约会

参数:
	pid: 约会地点ID
*/
func MakeDate(me, him uint32, tm time.Time, pid string, firstTime int) (msg map[string]interface{}, e error) {
	var uid1, uid2 uint32
	if me > him {
		uid1, uid2 = him, me
	} else if him > me {
		uid1, uid2 = me, him
	} else {
		return nil, service.NewError(service.ERR_INVALID_REQUEST, "cannot date yourself", "不能和自己约会")
	}
	count := 0
	if e = mdb.QueryRow("select count(*) from date_request where uid1=? and uid2=? and available=1", uid1, uid2).Scan(&count); e != nil {
		return nil, e
	}
	if count == 0 {
		return nil, service.NewError(service.ERR_PERMISSION_DENIED, "not all users requested", "对方还没有约会意愿")
	}
	if tm.Before(utils.Now.Add(utils.DurationTo(1, 8, 0, 0))) {
		return nil, service.NewError(service.ERR_INVALID_REQUEST, "date time too close", "约会时间必须在明天8点以后")
	}
	place, e := getDatePlace(pid)
	if e != nil {
		return nil, e
	}
	uinfos, e := user_overview.GetUserObjects(me, him, common.UID_DATE_MSG)
	if e != nil {
		return nil, e
	}
	minfo, hinfo, dinfo := uinfos[me], uinfos[him], uinfos[common.UID_DATE_MSG]
	if minfo == nil || hinfo == nil || dinfo == nil {
		return nil, e
	}
	if minfo.Province != hinfo.Province {
		return nil, e
	}
	mWorkPlace := UserWorkPlace{minfo.Uid, minfo.Nickname, minfo.Gender, minfo.Avatar, utils.Coordinate{minfo.WorkLat, minfo.WorkLng}}
	hWorkPlace := UserWorkPlace{hinfo.Uid, hinfo.Nickname, hinfo.Gender, hinfo.Avatar, utils.Coordinate{hinfo.WorkLat, hinfo.WorkLng}}
	if minfo.WorkPlaceId == "" {
		if mWorkPlace.Location.Lat, mWorkPlace.Location.Lng, e = general.UserLocation(minfo.Uid); e != nil {
			return nil, e
		}
	}
	if hinfo.WorkPlaceId == "" {
		if hWorkPlace.Location.Lat, hWorkPlace.Location.Lng, e = general.UserLocation(hinfo.Uid); e != nil {
			return nil, e
		}
	}
	sender := map[string]interface{}{"uid": me, "nickname": minfo.Nickname, "avatar": minfo.Avatar, "gender": minfo.Gender}
	dateUser := map[string]interface{}{"uid": common.UID_DATE_MSG, "nickname": dinfo.Nickname, "avatar": dinfo.Avatar, "gender": dinfo.Gender}
	msg = map[string]interface{}{"type": common.MSG_TYPE_DATE_REQUEST, "date_time": tm, "place": place, "sender": sender, "my_workplace": hWorkPlace, "him_workplace": mWorkPlace}
	ta := "他"
	if minfo.Gender == common.GENDER_WOMAN {
		ta = "她"
	}
	if firstTime == 1 {
		msg["text"] = fmt.Sprintf("“%v”也有意向见面认识，%v希望能在以下地点和时间见面", minfo.Nickname, ta)
	} else {
		msg["text"] = fmt.Sprintf("“%v”选择了见面地点和时间", minfo.Nickname)
	}
	msgid, e := general.SendMsg(common.UID_DATE_MSG, him, msg, "")
	if e != nil {
		return nil, e
	}
	msg["msgid"], msg["sender"], msg["my_workplace"], msg["him_workplace"] = msgid, dateUser, mWorkPlace, hWorkPlace
	msg["text"] = fmt.Sprintf("已告知“%v”地点和时间", hinfo.Nickname)
	return msg, nil
}

//定期检查约会状态
func checkDateRequest() {
	for {
		immediate, e := func() (bool, error) {
			items, total, e := rdb.ZRangeWithScores(redis_db.REDIS_MISC, MAKE_DATE_KEY, 0, 0)
			if e != nil {
				return false, e
			}
			if total == 0 || int64(items[0].Score) >= utils.Now.Unix() {
				return false, nil
			}
			affected, e := rdb.ZRem(redis_db.REDIS_MISC, MAKE_DATE_KEY, items[0].Key)
			if e != nil {
				return false, e
			}
			if len(affected) > 0 && affected[0] == 1 {
				keys := general.SplitKey(items[0].Key)
				if len(keys) != 2 {
					return false, errors.New("SplitKey " + items[0].Key + " failed")
				}
				sql := fmt.Sprintf("update date_request set available=1 where uid1=%s and uid2=%s", keys[0], keys[1])
				if _, e := mdb.Exec(sql); e != nil {
					return false, e
				}
				//发送消息
				uid1, e := utils.ToUint32(keys[0])
				if e != nil {
					return false, e
				}
				uid2, e := utils.ToUint32(keys[1])
				if e != nil {
					return false, e
				}
				stat.Append(uid1, stat.ACTION_CAFE, map[string]interface{}{"with": uid2})
				stat.Append(uid2, stat.ACTION_CAFE, map[string]interface{}{"with": uid1})
				uinfos, e := user_overview.GetUserObjects(uid1, uid2)
				if e != nil {
					return false, e
				}
				if uinfos[uid1] == nil || uinfos[uid2] == nil {
					return false, errors.New("do not find uinfo")
				}
				var to uint32
				var from *user_overview.UserViewItem
				if uinfos[uid1].Gender == uinfos[uid2].Gender {
					//同性，给UID小的用户发消息
					to = uid1
					from = uinfos[uid2]
				} else {
					//异性，给女方发消息
					if uinfos[uid1].Gender == common.GENDER_WOMAN {
						to = uid1
						from = uinfos[uid2]
					} else {
						to = uid2
						from = uinfos[uid1]
					}

				}
				sender := map[string]interface{}{"uid": from.Uid, "nickname": from.Nickname, "avatar": from.Avatar, "gender": from.Gender}
				if _, e := general.SendMsg(common.UID_DATE_MSG, to, map[string]interface{}{"type": common.MSG_TYPE_DATE_NOTIFY, "sender": sender}, ""); e != nil {
					return false, errors.New(fmt.Sprintf("send date notify msg error:", e.Error()))
				}

			}
			return true, nil
		}()
		if e != nil {
			mainLog.Append(fmt.Sprintf("checkDateRequest failed:%v", e.Error()))
		}
		if !immediate {
			time.Sleep(10 * time.Second)
		}
	}
}

//bprovinces里的省的名称要带"省"
func UpdateDatePlace(bprovinces []string, keywords ...string) (e error) {
	usql := "insert into date_place(id,province,city,name,address,lat,lng)values(?,?,?,?,?,?,?)on duplicate key update province=?,city=?,name=?,address=?,lat=?,lng=?"
	for _, bprovince := range bprovinces {
		bcities := []string{}
		if baidu.IsZXS(bprovince) {
			bcities = append(bcities, bprovince)
		} else {
			cityNums, total, e := baidu.SearchProvince(bprovince, 1, 20, keywords...)
			if e != nil {
				fmt.Println(e.Error())
				continue
			}
			for _, cityNum := range cityNums {
				fmt.Println("\t", cityNum.Name, cityNum.Num)
				bcities = append(bcities, cityNum.Name)
			}
			for i := 1; i*20 < total; i++ {
				cityNums, _, e := baidu.SearchProvince(bprovince, i+1, 20, keywords...)
				if e != nil {
					fmt.Println(e.Error())
					continue
				}
				for _, cityNum := range cityNums {
					fmt.Println("\t", cityNum.Name, cityNum.Num)
					bcities = append(bcities, cityNum.Name)
				}
			}
		}
		if bcities[0] == "北京市" && len(bcities) > 10 {
			continue
		}
		fmt.Println(bprovince, len(bcities))
		for _, bcity := range bcities {
			fmt.Printf("update %v %v...", bprovince, bcity)
			places, total, e := baidu.SearchCity(bcity, 1, 20, keywords...)
			if e != nil {
				fmt.Println(e.Error())
				continue
			}
			var allPlaces []baidu.Place = make([]baidu.Place, 0, 1000)
			allPlaces = append(allPlaces, places...)
			for i := 1; i*20 < total; i++ {
				places, total, e = baidu.SearchCity(bcity, i+1, 20, keywords...)
				if e != nil {
					fmt.Println(e.Error())
					continue
				}
				allPlaces = append(allPlaces, places...)
			}
			for _, place := range allPlaces {
				province, city := general.BaiduToOurProvinceCity(bprovince, bcity)
				if _, e = mdb.Exec(usql, place.Uid, province, city, place.Name, place.Address, place.Location.Lat, place.Location.Lng, province, city, place.Name, place.Address, place.Location.Lat, place.Location.Lng); e != nil {
					fmt.Println(e.Error())
				}
			}
			fmt.Println(len(allPlaces))
			time.Sleep(1 * time.Second)
		}
	}
	return nil
}

//---------------------Private Functions-----------------------//

func makeFriendUsersInfo(uid uint32, items []redis.ItemScore) (users []Friend, e error) {
	//取出当前请求的uid集合
	uids := make([]uint32, 0, len(items))
	keys := make([]interface{}, 0, len(items))
	for _, u := range items {
		if uid, e := utils.ToUint32(u.Key); e != nil {
			return nil, e
		} else {
			uids = append(uids, uid)
			keys = append(keys, uid)
		}
	}
	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, e
	}
	interests, e := rdb.ZMultiScore(redis_db.REDIS_FOLLOW, general.MakeKey("f", uid), keys...)
	if e != nil {
		return nil, e
	}
	focus, e := rdb.ZMultiScore(redis_db.REDIS_FOLLOW, general.MakeKey("sf", uid), keys...)
	if e != nil {
		return nil, e
	}
	users = make([]Friend, 0, len(uids))
	for i, item := range items {
		if ui := uinfos[uids[i]]; ui != nil {
			var tag uint16 = common.FOLLOW_TAG_NONE

			if _, ok := focus[uids[i]]; ok {
				tag = common.FOLLOW_TAG_FOCUS
			} else if _, ok := interests[uids[i]]; ok {
				tag = common.FOLLOW_TAG_INTEREST
			}
			users = append(users, Friend{uids[i], ui.Nickname, ui.Avatar, tag, time.Unix(int64(item.Score), 0)})
		}
	}
	return users, nil
}

//获取约会地点详情
func getDatePlace(id string) (place Place, e error) {
	sql := "select id,name,address,lat,lng from date_place where id=?"
	e = mdb.QueryRow(sql, id).Scan(&place.Id, &place.Name, &place.Address, &place.Lat, &place.Lng)
	return
}
