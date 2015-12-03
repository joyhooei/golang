package topic

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
	"yf_pkg/format"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/common/user"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

func key(uid uint32, r uint32) string {
	return fmt.Sprintf("discov_topic_%v_%v", uid, r)
}

/*
func key2(gender, cur, ps int) string {
	return fmt.Sprintf("discov_topic_%v_%v_%v", gender, ps, cur)
}
*/
func key2(cur, ps int) string {
	return fmt.Sprintf("discov_topic_any_%v_%v", ps, cur)
}

//r，半径（公里）
func Discovery(uid uint32, r uint32, lat float64, lng float64, cur int, ps int, refresh bool) (topics []TopicInfo, page *utils.Pages, e error) {
	var total int
	radius := utils.KmToLng(float64(r))
	k := key(uid, r)
	exists := false
	if !refresh {
		exists, topics, total, e = readCache(k, cur, ps)
		if e != nil {
			return nil, nil, e
		}
	} else {
		conn := cache.GetWriteConnection(redis_db.CACHE_TOPIC)
		defer conn.Close()
		_, e := conn.Do("DEL", k)
		if e != nil {
			return nil, nil, e
		}
	}
	if !exists {
		uinfos, e := user_overview.GetUserObjects(uid)
		if e != nil {
			return nil, nil, errors.New(fmt.Sprintf("Get uid %v info error :%v", uid, e.Error()))
		}
		uinfo := uinfos[uid]
		if uinfo == nil {
			return nil, nil, errors.New(fmt.Sprintf("user %v info not found", uid))
		}
		//	gender := common.GENDER_WOMAN
		base := TopicInfo{}
		switch uinfo.Gender {
		case common.GENDER_MAN:
			base.Uid, base.Lat, base.Lng, base.Birthday = uid, lat, lng, uinfo.Birthday.Add(3*360*24*time.Hour)
		case common.GENDER_WOMAN:
			//	gender = common.GENDER_MAN
			base.Uid, base.Lat, base.Lng, base.Birthday = uid, lat, lng, uinfo.Birthday.Add(-3*360*24*time.Hour)
		}
		maxLat, minLat := lat+radius, lat-radius
		maxLng, minLng := lng+radius, lng-radius

		topicsAll := make([]TopicInfo, 0, ps)

		sql := "(select distinct id,tid,birthday,in_room,online_timeout,lat,lng,priority from discovery where timeout > ? and tid > 0 and full=0 and  lat <= ? and lat >= ? and lng <= ? and lng >= ? order by timeout desc limit 200)union(select distinct id,tid,birthday,in_room,online_timeout,lat,lng,priority from discovery where id = ? and tid > 0 and timeout > ?)union(select distinct id,tid,birthday,in_room,online_timeout,lat,lng,priority from discovery where priority > 0 and tid > 0 and timeout > ?)"
		if topicsAll, e = queryTopicItems(topics, sql, utils.Now.Add(10*time.Minute), maxLat, minLat, maxLng, minLng, uid, utils.Now, utils.Now.Add(10*time.Minute)); e != nil {
			return nil, nil, e
		}
		if topicsAll, e = makeTopicsInfo(topicsAll); e != nil {
			return nil, nil, e
		}
		if e = getTrends(topicsAll); e != nil {
			return nil, nil, e
		}
		total = len(topicsAll)
		//按照权重排序
		for i, _ := range topicsAll {
			DiscovScore(&topicsAll[i], &base)
		}
		sort.Sort(TopicItems(topicsAll))
		//放入redis缓存
		if e = writeRedis(k, topicsAll); e != nil {
			return nil, nil, e
		}
		start, end := utils.BuildRange(cur, ps, total)
		if start < total {
			topics = topicsAll[start : end+1]
		}
	}
	page = utils.PageInfo(total, cur, ps)
	if total == 0 || (len(topics) == 0 && cur*ps > 50) {
		if topics, e = getAnyTopics(uid, cur-page.Pn, ps); e != nil {
			return nil, nil, e
		}
		if e = getTrends(topics); e != nil {
			return nil, nil, e
		}
	}
	page = utils.PageInfo(-1, cur, ps)
	if e = getOnlines(topics); e != nil {
		return nil, nil, e
	}
	if user.IsKfUser(uid) {
		if e = getRealOnlines(topics); e != nil {
			return nil, nil, e
		}
	}

	return topics, page, e
}

//-------------------------Private Functions------------------------//

func getAnyTopics(uid uint32, cur, ps int) (topics []TopicInfo, e error) {
	uinfos, e := user_overview.GetUserObjects(uid)
	if e != nil {
		return nil, errors.New(fmt.Sprintf("Get uid %v info error :%v", uid, e.Error()))
	}
	uinfo := uinfos[uid]
	if uinfo == nil {
		return nil, errors.New(fmt.Sprintf("user %v info not found", uid))
	}

	var exists bool
	k := key2(cur, ps)
	exists, topics, _, e = readCache(k, 0, ps)
	if e != nil {
		return nil, e
	}

	if !exists {
		sql := "select distinct id,tid,birthday,in_room,online_timeout,lat,lng,priority from discovery where timeout > ? and tid > 0 and full=0 order by online_timeout desc" + utils.BuildLimit(cur, ps)
		if topics, e = queryTopicItems(topics, sql, utils.Now.Add(10*time.Minute)); e != nil {
			return nil, e
		}
		if topics, e = makeTopicsInfo(topics); e != nil {
			return nil, e
		}
		//放入redis缓存
		if e = writeRedis(k, topics); e != nil {
			return nil, e
		}
	}
	return
}

func queryTopicItems(topics TopicItems, sql string, args ...interface{}) (TopicItems, error) {
	topics = make(TopicItems, 0)
	rows, e := sdb.Query(sql, args...)
	if e != nil {
		return nil, e
	}
	added := make(map[uint32]bool, 20)
	defer rows.Close()
	for rows.Next() {
		var item TopicInfo
		var birth, onlineTimeout string
		if err := rows.Scan(&item.Uid, &item.Tid, &birth, &item.InRoom, &onlineTimeout, &item.Lat, &item.Lng, &item.Priority); err != nil {
			return nil, err
		}
		if _, ok := added[item.Tid]; ok {
			continue
		} else {
			added[item.Tid] = true
		}
		item.Birthday, _ = utils.ToTime(birth, format.TIME_LAYOUT_1)
		item.OnlineTimeout, _ = utils.ToTime(onlineTimeout, format.TIME_LAYOUT_1)
		topics = append(topics, item)
	}
	return topics, nil
}

func makeTopicsInfo(topics []TopicInfo) (filtered []TopicInfo, e error) {
	//取出当前请求的uid集合
	uids := make([]uint32, 0, len(topics))
	tids := make([]uint32, 0, len(topics))
	for i := 0; i < len(topics); i++ {
		uids = append(uids, topics[i].Uid)
		tids = append(tids, topics[i].Tid)
	}

	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, e
	}
	tinfos, e := GetTopics(tids...)
	if e != nil {
		return nil, e
	}
	msgs, e := GetRecentMessages(tids...)
	if e != nil {
		return nil, e
	}
	filtered = make([]TopicInfo, 0, len(topics))
	for i, topic := range topics {
		tinfo := tinfos[topic.Tid]
		if tinfo != nil && tinfo.PicsLevel > 0 {
			uinfo := uinfos[topic.Uid]
			if uinfo != nil {
				topics[i].Nickname = uinfo.Nickname
				topics[i].Age = uinfo.Age
				topics[i].Avatar = uinfo.Avatar
				topics[i].Gender = uinfo.Gender
				topics[i].Province = uinfo.Province
				topics[i].Online = 0 //先随便存一个值
				topics[i].Capacity = tinfo.Capacity
				topics[i].Pics = tinfo.Pics
				topics[i].PicsLevel = tinfo.PicsLevel
				topics[i].Tag = tinfo.Tag
				topics[i].Title = tinfo.Title
				topics[i].IsAdmin = general.IsAdmin(topic.Uid)
				topics[i].Tm = tinfo.Tm
				msg := msgs[topic.Tid]
				if msg != nil {
					topics[i].Messages = msg
				}
				filtered = append(filtered, topics[i])
			}
		}
	}
	return
}

func readCache(key string, cur int, ps int) (exists bool, topics []TopicInfo, total int, e error) {
	exists, e = cache.Exists(redis_db.CACHE_TOPIC, key)
	if e != nil {
		return false, nil, 0, e
	}
	total = 0
	uinfos := make([][]byte, 0, ps)
	if exists {
		conn := cache.GetReadConnection(redis_db.CACHE_TOPIC)
		defer conn.Close()
		total, e = redis.Int(conn.Do("LLEN", key))
		if e != nil {
			return false, nil, 0, e
		}
		start, end := utils.BuildRange(cur, ps, total)
		v, e := redis.Values(conn.Do("LRANGE", key, start, end))
		if e != nil {
			return false, nil, 0, e
		}
		if e = redis.ScanSlice(v, &uinfos); e != nil {
			return false, nil, 0, e
		}
		topics = make([]TopicInfo, 0, ps)
		for _, b := range uinfos {
			var topic TopicInfo
			if e = json.Unmarshal(b, &topic); e != nil {
				return false, nil, 0, e
			}
			topics = append(topics, topic)
		}
		return true, topics, total, nil
	} else {
		return false, nil, 0, nil
	}
}

func writeRedis(key string, topics []TopicInfo) error {
	if len(topics) == 0 {
		return nil
	}
	topicsJson := make([]interface{}, 0, len(topics))
	topicsJson = append(topicsJson, key)
	for _, item := range topics {
		b, e := json.Marshal(item)
		if e != nil {
			return e
		}
		topicsJson = append(topicsJson, b)
	}
	conn := cache.GetWriteConnection(redis_db.CACHE_TOPIC)
	defer conn.Close()
	_, e := conn.Do("RPUSH", topicsJson...)
	if e != nil {
		return e
	}
	_, e = conn.Do("EXPIRE", key, 600)
	return e
}

func getRealOnlines(topics []TopicInfo) (e error) {
	conn := rdb.GetReadConnection(redis_db.REDIS_TOPIC_USERS)
	defer conn.Close()
	for i, _ := range topics {
		if e := conn.Send("ZRANGE", topics[i].Tid, 0, -1); e != nil {
			return e
		}
	}
	conn.Flush()
	for i, _ := range topics {
		reply, e := redis.Values(conn.Receive())
		if e != nil {
			return e
		}
		uids := []uint32{}
		e = redis.ScanSlice(reply, &uids)
		if e != nil {
			return e
		}
		isOnlines, e := user_overview.IsOnline(uids...)
		if e != nil {
			return e
		}
		var n uint32 = 0
		for uid, online := range isOnlines {
			if user.IsKfUser(uid) {
				continue
			}
			if online {
				n++
			}
		}
		topics[i].RealOnline = n
	}
	return
}
func getOnlines(topics []TopicInfo) (e error) {
	conn := rdb.GetReadConnection(redis_db.REDIS_TOPIC_USERS)
	defer conn.Close()
	for i, _ := range topics {
		if e := conn.Send("ZCARD", topics[i].Tid); e != nil {
			return e
		}
	}
	conn.Flush()
	for i, _ := range topics {
		n, e := redis.Uint32(conn.Receive())
		if e != nil {
			return e
		}
		if n > 0 {
			isOnlines, e := user_overview.IsOnline(topics[i].Uid)
			if e != nil {
				isOnlines[topics[i].Uid] = false
			}
			if !isOnlines[topics[i].Uid] {
				n--
			}
		}
		topics[i].Online = n
	}
	return
}
func getTrends(topics []TopicInfo) (e error) {
	conn := rdb.GetReadConnection(redis_db.REDIS_TOPIC_USERS)
	defer conn.Close()
	for i, _ := range topics {
		if e := conn.Send("ZCOUNT", topics[i].Tid, "-inf", -utils.Now.Add(-10*time.Minute).Unix()); e != nil {
			return e
		}
	}
	conn.Flush()
	for i, _ := range topics {
		n, e := redis.Int(conn.Receive())
		if e != nil {
			return e
		}
		trend, ok := TrendName[n/10]
		if !ok {
			trend = TrendName[TREND_MAX]
		}
		topics[i].Trend = trend
		topics[i].TrendValue = n
	}
	return
}
