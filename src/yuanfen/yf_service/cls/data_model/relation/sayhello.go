package relation

import (
	"errors"
	"fmt"
	"time"
	"yf_pkg/format"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/unread"
)

type SayHelloBriefMessage struct {
	Id      uint32    `json:"id"`      //消息ID
	Content string    `json:"content"` //消息内容
	Status  int       `json:"status"`  //消息状态
	Tm      time.Time `json:"tm"`      //消息发送时间
}

type SayHelloMessage struct {
	Id       uint32    `json:"id"`       //消息ID
	Uid      uint32    `json:"uid"`      //消息相关的用户，根据实际应用场景，可能是发送者，也可能是接收者
	Avatar   string    `json:"avatar"`   //相关用户的头像
	Nickname string    `json:"nickname"` //相关用户的昵称
	Content  string    `json:"content"`  //消息内容
	Status   int       `json:"status"`   //消息状态
	Tm       time.Time `json:"tm"`       //消息发送时间
}

//获取认识一下的状态
func GetSayHelloStatus(target string, me, him uint32) (status int, e error) {
	var mid uint32
	switch target {
	case common.SAYHELLO_TARGET_ME:
		mid, e = redis.Uint32(rdb.HGet(redis_db.REDIS_USER_DATA, me, general.MakeKey(common.LAST_SAYHELLO_TO_ME_PREFIX, him)))
	case common.SAYHELLO_TARGET_HIM:
		mid, e = redis.Uint32(rdb.HGet(redis_db.REDIS_USER_DATA, me, general.MakeKey(common.LAST_SAYHELLO_TO_HIM_PREFIX, him)))
	default:
		return 0, errors.New("unknown target " + target)
	}
	switch e {
	case nil:
	case redis.ErrNil:
		return 0, nil
	default:
		return 0, e
	}
	sql := "select stat from sayhello_msg where id=?"
	if e = mdb.QueryRow(sql, mid).Scan(&status); e != nil {
		return 0, e
	}
	return
}

//把him发送给me的最新消息标记为已读
func ReadSayHello(me uint32, him uint32) error {
	mid, e := redis.Uint32(rdb.HGet(redis_db.REDIS_USER_DATA, me, general.MakeKey(common.LAST_SAYHELLO_TO_ME_PREFIX, him)))
	if e != nil {
		return e
	}
	sql := "update sayhello_msg set stat=? where id=? and stat=?"
	fmt.Println("mid=", mid)
	res, e := mdb.Exec(sql, common.SAYHELLO_MSG_READ, mid, common.SAYHELLO_MSG_UNREAD)
	if e != nil {
		return e
	}
	if _, e := res.RowsAffected(); e != nil {
		return e
	} else {
		return nil
	}
}

func SayHelloList(target string, me uint32, him uint32, cur int, ps int) (msgs []SayHelloBriefMessage, e error) {
	sql := ""
	switch target {
	case common.SAYHELLO_TARGET_ME:
		sql = fmt.Sprintf("select id,content,stat,tm from sayhello_msg where `from`=%v and `to`=%v order by tm desc%v", him, me, utils.BuildLimit(cur, ps))
		if e = ReadSayHello(me, him); e != nil {
			return nil, e
		}
	case common.SAYHELLO_TARGET_HIM:
		sql = fmt.Sprintf("select id,content,stat,tm from sayhello_msg where `from`=%v and `to`=%v order by tm desc%v", me, him, utils.BuildLimit(cur, ps))
	default:
		return nil, service.NewError(service.ERR_INVALID_PARAM, "unkown target", "请求不合法")
	}
	msgs = make([]SayHelloBriefMessage, 0, ps)
	rows, e := mdb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var msg SayHelloBriefMessage
		var tmStr string
		if err := rows.Scan(&msg.Id, &msg.Content, &msg.Status, &tmStr); err != nil {
			return nil, err
		}
		msg.Tm, _ = utils.ToTime(tmStr, format.TIME_LAYOUT_1)
		msgs = append(msgs, msg)
	}
	return
}

func GetConnection(me uint32, him uint32) (connection []string, e error) {
	infos, e := user_overview.GetUserObjects(me, him)
	if e != nil {
		return nil, e
	}
	return findConnection(infos[me], infos[him]), nil
}

//获取打招呼用户的资料
func GetSayHelloUserInfo(me uint32, him uint32) (uinfo SayHelloUser, e error) {
	minfo, e := user_overview.GetUserObject(me)
	if e != nil {
		return uinfo, e
	}
	hinfo, e := user_overview.GetUserObject(him)
	if e != nil {
		return uinfo, e
	}
	mlat, mlng, e := general.UserLocation(me)
	if e != nil {
		return uinfo, e
	}
	hlat, hlng, e := general.UserLocation(him)
	if e != nil {
		return uinfo, e
	}
	uinfo.Distence = utils.Distence(utils.Coordinate{mlat, mlng}, utils.Coordinate{hlat, hlng})
	uinfo.Connection = findConnection(minfo, hinfo)
	uinfo.Uid = him
	uinfo.Nickname = hinfo.Nickname
	uinfo.Avatar = hinfo.Avatar
	uinfo.Age = hinfo.Age
	uinfo.Height = hinfo.Height
	if uinfo.DyNum, uinfo.Dynamics, e = user_overview.GetUserLastDynamicPic(him, false); e != nil {
		return uinfo, e
	}
	uinfo.Province = hinfo.Province
	uinfo.City = hinfo.City
	return uinfo, nil
}

func SayHelloUsers(target string, uid uint32, cur int, ps int) (users []*SayHelloMessage, total int, e error) {
	items, total, e := rdb.ZREVRangeWithScoresPS(redis_db.REDIS_SAYHELLO, general.MakeKey(target, uid), cur, ps)
	if e != nil {
		return nil, 0, e
	}
	users, e = makeSayHelloMessages(target, uid, items)
	if target == common.SAYHELLO_TARGET_ME {
		unread.UpdateReadTime(uid, common.UNREAD_SAYHELLO)
	}
	return
}

//删除想认识我或我想认识的用户
func DelSayHelloUser(target string, me, him uint32) (e error) {
	if _, e := rdb.ZRem(redis_db.REDIS_SAYHELLO, general.MakeKey(target, me), him); e != nil {
		return e
	}
	key := ""
	switch target {
	case common.SAYHELLO_TARGET_ME:
		key = general.MakeKey(common.LAST_SAYHELLO_TO_ME_PREFIX, him)
	case common.SAYHELLO_TARGET_HIM:
		key = general.MakeKey(common.LAST_SAYHELLO_TO_HIM_PREFIX, him)
	default:
		return errors.New("unkown target:" + target)
	}
	return rdb.HDel(redis_db.REDIS_USER_DATA, me, key)
}

func makeSayHelloMessages(target string, uid uint32, items []redis.ItemScore) (messages []*SayHelloMessage, e error) {
	if len(items) == 0 {
		return []*SayHelloMessage{}, nil
	}
	fmt.Println("items:", items)
	//取出当前请求的uid集合
	uids := make([]uint32, 0, len(items))
	mids := make([]uint32, 0, len(items))
	keys := make([]interface{}, 0, len(items))
	tmpUsers := make(map[uint32]*SayHelloMessage, len(items))
	for _, u := range items {
		if id, e := utils.ToUint32(u.Key); e != nil {
			return nil, e
		} else {
			uids = append(uids, id)
			keys = append(keys, fmt.Sprintf("%v_%v_%v", common.LAST_SAYHELLO_PREFIX, target, id))
			tmpUsers[id] = &SayHelloMessage{0, id, "", "", "", 0, time.Unix(int64(u.Score), 0)}
		}
	}
	fmt.Println("keys=", keys)
	if e = rdb.HMGet(redis_db.REDIS_USER_DATA, &mids, uid, keys...); e != nil {
		return nil, e
	}
	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, e
	}
	ufield := "from"
	if target == common.SAYHELLO_TARGET_HIM {
		ufield = "to"
	}
	sql := "select `" + ufield + "`,content,stat from sayhello_msg where id" + mysql.In(mids)
	rows, e := mdb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var id uint32
		var content string
		var stat int
		if err := rows.Scan(&id, &content, &stat); err != nil {
			return nil, err
		}
		tmpUsers[id].Content = content
		tmpUsers[id].Status = stat
	}
	messages = make([]*SayHelloMessage, 0, len(uids))
	for i, id := range uids {
		fmt.Println("get uinfo:", id)
		if ui := uinfos[id]; ui != nil {
			fmt.Println("exist")
			if tmpUsers[id].Status > 0 {
				tmpUsers[id].Nickname, tmpUsers[id].Avatar = ui.Nickname, ui.Avatar
				tmpUsers[id].Id = mids[i]
				messages = append(messages, tmpUsers[id])
			}
		}
	}

	return messages, nil
}

//寻找两个用户之间的联系，如果没有任何联系则返回空字符串
func findConnection(me *user_overview.UserViewItem, candidate *user_overview.UserViewItem) (connections []string) {
	if me == nil || candidate == nil {
		return nil
	}
	connections = make([]string, 0, 6)
	ta := "他"
	if candidate.Gender == common.GENDER_WOMAN {
		ta = "她"
	}
	if me.School != "" && me.School == candidate.School {
		//校友
		connections = append(connections, "你们都来自"+me.School)
	}
	if me.Homeprovince != "" && me.Homeprovince != me.Province && me.Homeprovince == candidate.Homeprovince {
		//家乡
		if me.Homecity != "" && me.Homecity == candidate.Homecity {
			connections = append(connections, "你们都是"+me.Homeprovince+me.Homecity+"人")
		} else {
			connections = append(connections, "在"+me.City+"的"+me.Homeprovince+"人")
		}
	}
	if me.WorkPlaceName != "" && me.WorkPlaceName == candidate.WorkPlaceName {
		connections = append(connections, "你们都在"+me.WorkPlaceName+"工作")
	} else if candidate.WorkPlaceId != "" && utils.Distence(utils.Coordinate{me.WorkLat, me.WorkLng}, utils.Coordinate{candidate.WorkLat, candidate.WorkLng}) <= common.LOCATION_RADIUS*1000 {
		connections = append(connections, ta+"在"+candidate.WorkPlaceName+"工作")
	}
	//择友要求
	if me.Require.Filled() >= 2 && me.Require.Match(candidate) {
		connections = append(connections, "它符合您的择友要求")
	}
	//感兴趣的类型
	for _, v := range me.Needtag {
		for _, t := range candidate.Tag {
			if t != "" && v == t {
				connections = append(connections, ta+v)
				break
			}
		}
	}
	//行业职业
	if me.Trade != "" && me.Trade == candidate.Trade {
		if me.Job != "" && me.Job == candidate.Job {
			if _, ok := common.JobNotRecommend[me.Job]; !ok {
				connections = append(connections, "你们都是"+me.Job)
			}
		}
		if me.Trade != "其它" {
			connections = append(connections, "你们是同行")
		}
	}
	return
}
