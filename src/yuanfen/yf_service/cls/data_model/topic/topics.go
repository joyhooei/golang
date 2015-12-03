package topic

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"
	"yf_pkg/cachedb"
	"yf_pkg/format"
	"yf_pkg/mysql"
	"yf_pkg/push"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/relation"
	"yuanfen/yf_service/cls/data_model/tag"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/message"
	"yuanfen/yf_service/cls/notify"
	"yuanfen/yf_service/cls/status"
	"yuanfen/yf_service/cls/unread"
)

const TOP_USERS = "top_users"

type Topic struct {
	Id        uint32
	Uid       uint32
	Title     string
	Capacity  uint32
	Tag       string
	Tm        time.Time
	Status    uint8
	Pics      string
	PicsLevel int8
}

var sql_1 string = fmt.Sprintf("select id from topic where uid=? and status=%v and tm > ?", common.TOPIC_STATUS_ACTIVE)
var sql_2 string = "insert into topic(uid,title,capacity,tag,pics)values(?,?,?,?,?)"
var sql_3 string = "update discovery set tid=?,in_room=0,full=0,timeout=? where id=?"
var sql_4 string = fmt.Sprintf("select id from topic where uid=? and status=%v limit 1", common.TOPIC_STATUS_ACTIVE)
var sql_5 string = fmt.Sprintf("update topic set status=%v where id=? and uid=? and status=%v", common.TOPIC_STATUS_CLOSED, common.TOPIC_STATUS_ACTIVE)
var sql_6 string = "update discovery set tid=0 where id=? and tid=?"
var sql_7 string = "update topic set report=report+1 where id=?"
var sql_8 string = "insert into topic_record(uid,tid)values(?,?)on duplicate key update start_tm=?"
var sql_9 string = "select start_tm from topic_record where uid=? and tid=?"
var sql_10 string = "update topic_record set total_tm=total_tm+? where uid=? and tid=?"
var sql_11 string = "select start_tm from topic_record where uid=? and tid=?"
var sql_12 string = "update topic_record set total_tm=total_tm+? where uid=? and tid=?"
var sql_13 string = "delete from tag_message where tag=? and id = ? and type" + mysql.In([]string{common.MSG_TYPE_PIC, common.MSG_TYPE_TEXT, common.MSG_TYPE_VOICE})
var sql_14 string = "select count(tid) from topic_record where uid=?"

func RoomId(tid uint32) string {
	return fmt.Sprintf("%v%v", common.TAG_PREFIX_TOPIC, tid)
}

func CreateTopic(uid uint32, title string, tagid string, capacity int, pics string) (tid uint32, timeout time.Time, e error) {
	rows, e := mdb.Query(sql_1, uid, utils.Now.Add(-1*TOPIC_TIMEOUT*time.Second))
	if e != nil {
		return 0, timeout, e
	}
	defer rows.Close()
	if rows.Next() {
		if e := rows.Scan(&tid); e != nil {
			return 0, timeout, e
		}
		return tid, utils.Now, service.NewError(service.ERR_TOPIC_EXIST, "", "您只能创建一个话题，请先关闭已有的话题。")
	}
	if capacity > TOPIC_MAX_CAPACITY {
		return 0, timeout, service.NewError(service.ERR_INVALID_REQUEST, fmt.Sprintf("exceed max capacity %v", TOPIC_MAX_CAPACITY), "超过话题允许人数上限")
	}
	if capacity == -1 {
		capacity = TOPIC_MAX_CAPACITY
	}
	res, e := mdb.Exec(sql_2, uid, title, capacity, tagid, pics)
	if e != nil {
		return 0, timeout, e
	}
	timeout = utils.Now.Add(TOPIC_TIMEOUT * time.Second)
	id, e := res.LastInsertId()
	if e != nil {
		return 0, timeout, e
	}
	tid = uint32(id)
	ucon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_USERS)
	defer ucon.Close()
	if _, e = ucon.Do("ZADD", tid, math.MinInt64, uid); e != nil {
		return 0, timeout, e
	}
	if _, e = ucon.Do("EXPIRE", tid, TOPIC_TIMEOUT); e != nil {
		return 0, timeout, e
	}
	//创建黑名单
	bcon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_BLACKLIST)
	defer bcon.Close()
	if _, e = bcon.Do("ZADD", tid, utils.Now.Unix()-1, 0); e != nil {
		return 0, timeout, e
	}
	if _, e = bcon.Do("EXPIRE", tid, TOPIC_TIMEOUT); e != nil {
		return 0, timeout, e
	}

	//加入聊天室
	if e = push.AddTag(uid, RoomId(tid)); e != nil {
		return 0, timeout, e
	}
	tag.UseTag(uid, common.TAG_TYPE_TOPIC, tagid)
	//更新发现列表中的圈子信息
	if _, e = sdb.Exec(sql_3, id, timeout, uid); e != nil {
		return 0, timeout, e
	}
	for _, r := range Ranges {
		if err := cache.Del(redis_db.CACHE_TOPIC, key(uid, r.Radius)); err != nil {
			mainLog.Append(fmt.Sprintf("update status of user %v failed : %v", uid, err.Error()))
		}

	}
	if err := unread.UpdateReadTime(uid, common.UNREAD_MY_TOPIC); err != nil {
		mainLog.Append(fmt.Sprintf("update redis failed : %v", err.Error()))
	}
	go sendNotification(uid, tid, title)
	return tid, timeout, nil
}

func sendNotification(uid uint32, tid uint32, title string) {
	users, total, e := relation.GetFollowUids(false, uid, 1, common.MAX_FOLLOW_NUM)
	if e != nil {
		mainLog.Append(fmt.Sprintf("SendNotification error : %v", e.Error()))
		return
	}
	if total > common.MAX_FOLLOW_NUM {
		general.Alert("follow", fmt.Sprintf("%v has too much followers %v", uid, total))
	}
	uniq := map[uint32]bool{}
	for _, uid := range users {
		uniq[uid] = true
	}
	for to, _ := range uniq {
		not, e := notify.GetNotify(uid, notify.NOTIFY_CREATE_TOPIC, map[string]interface{}{"tid": tid}, "", title, to)
		if e != nil {
			mainLog.Append(fmt.Sprintf("SendNotification error : %v", e.Error()))
			continue
		}
		general.SendMsg(uid, to, map[string]interface{}{notify.NOTIFY_KEY: not}, "")

	}
}

func CloseTopic(tid uint32, uid uint32) error {
	if tid == 0 {
		e := mdb.QueryRow(sql_4, uid).Scan(&tid)
		if e != nil {
			return e
		}
	}
	if _, e := mdb.Exec(sql_5, tid, uid); e != nil {
		return e
	}
	ucon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_USERS)
	defer ucon.Close()
	if _, e := ucon.Do("DEL", tid); e != nil {
		return e
	}
	bcon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_BLACKLIST)
	defer bcon.Close()
	if _, e := bcon.Do("DEL", tid); e != nil {
		return e
	}
	//删除push服务的标签
	if e := push.ClearTag(RoomId(tid)); e != nil {
		return e
	}

	//更新发现列表
	if _, e := sdb.Exec(sql_6, uid, tid); e != nil {
		return e
	}
	return nil
}

func ReportTopic(tid uint32) error {
	_, e := mdb.Exec(sql_7, tid)
	return e
}

func ClearCache(tid uint32) (err error) {
	return cdb.ClearCache(NewTopicDBObject(tid))
}

func GetTopics(ids ...uint32) (topics map[uint32]*Topic, e error) {
	ts := make(map[interface{}]cachedb.DBObject)
	for _, id := range ids {
		ts[id] = nil
	}
	if e := cdb.GetMap(ts, NewTopicDBObject); e != nil {
		return nil, e
	}
	topics = make(map[uint32]*Topic)
	for id, topic := range ts {
		if topic != nil {
			var t Topic
			data := *(topic.(*TopicDBObject))
			t.Id = id.(uint32)
			if t.Uid, e = utils.ToUint32(data["uid"]); e != nil {
				return nil, e
			}
			if t.Capacity, e = utils.ToUint32(data["capacity"]); e != nil {
				return nil, e
			}
			t.Title = data["title"].(string)
			t.Tag = data["tag"].(string)
			t.Pics = data["pics"].(string)
			if t.PicsLevel, e = utils.ToInt8(data["picslevel"]); e != nil {
				return nil, e
			}
			if t.Status, e = utils.ToUint8(data["status"]); e != nil {
				return nil, e
			}
			t.Tm, _ = utils.ToTime(data["tm"], format.TIME_LAYOUT_1)
			topics[t.Id] = &t
		} else {
			topics[id.(uint32)] = nil
		}
	}
	return
}

func JoinTopic(uid uint32, tid uint32, result map[string]interface{}) error {
	//确认话题存在
	ucon := rdb.GetReadConnection(redis_db.REDIS_TOPIC_USERS)
	defer ucon.Close()
	onlines, e := redis.Uint32(ucon.Do("ZCARD", tid))
	switch e {
	case nil:
	case redis.ErrNil:
		return service.NewError(service.ERR_INVALID_REQUEST, "", "话题不存在")
	default:
		general.Alert("redis-mumu", "read topic_users fail")
		return e
	}

	//查看话题是否已满员
	topics, e := GetTopics(tid)
	if e != nil {
		return e
	}
	topic, _ := topics[tid]
	if topic == nil {
		return errors.New("topic not available")
	}
	if topic.Status != common.TOPIC_STATUS_ACTIVE {
		return service.NewError(common.ERR_TOPIC_CLOSED, "topic not available", "话题已关闭")
	}

	if topic.Capacity <= onlines {
		if err := topicFull(tid, true); err != nil {
			mainLog.Append(fmt.Sprintf("update sdb.discovery of field full failed : %v", err.Error()))
		}
		return service.NewError(service.ERR_INVALID_REQUEST, "topic full", "人数已满")
	}
	//如果是话题创建者，则更新in_room字段
	if topic.Uid == uid {
		if err := inRoom(tid, true); err != nil {
			mainLog.Append(fmt.Sprintf("update sdb.discovery of field in_room failed : %v", err.Error()))
		}
		if err := unread.UpdateReadTime(uid, common.UNREAD_MY_TOPIC); err != nil {
			mainLog.Append(fmt.Sprintf("update redis failed : %v", err.Error()))
		}
		ur := map[string]interface{}{common.UNREAD_MY_TOPIC: 0}
		result[common.UNREAD_KEY] = ur
	}

	//检查是否在黑名单中
	rcon := rdb.GetReadConnection(redis_db.REDIS_TOPIC_BLACKLIST)
	defer rcon.Close()
	timeout, e := redis.Uint64(rcon.Do("ZSCORE", tid, uid))
	if e == nil && timeout > uint64(utils.Now.Unix()) {
		return service.NewError(service.ERR_INVALID_REQUEST, "you are on blacklist", "您没有权限进入该话题")
	} else if e != nil && e != redis.ErrNil {
		return e
	}

	if topic.Uid != uid {
		//离开上一个加入的话题
		con := rdb.GetWriteConnection(redis_db.REDIS_USER_TOPIC)
		defer con.Close()
		lastTid, e := redis.Uint32(con.Do("GET", uid))
		switch e {
		case redis.ErrNil:
		case nil:
			if e := LeaveTopic(uid, lastTid, result); e != nil {
				return errors.New(fmt.Sprintf("leave last joined topic %v error : %v", lastTid, e.Error()))
			}
		default:
			return errors.New(fmt.Sprintf("leave last joined topic %v error : %v", lastTid, e.Error()))
		}
		if _, e := con.Do("SET", uid, tid); e != nil {
			return e
		}
		//加入话题
		wcon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_USERS)
		defer wcon.Close()
		if _, e = wcon.Do("ZADD", tid, -utils.Now.Unix(), uid); e != nil {
			return e
		}

		//加入聊天室
		if e = push.AddTag(uid, RoomId(tid)); e != nil {
			return e
		}
		//更新参与话题记录表
		_, e = mdb.Exec(sql_8, uid, tid, utils.Now)
	}

	users, total, e := topicUsers(tid, 0, TOPIC_TOP_USER_NUM)
	if e != nil {
		return e
	}
	push.SendTagMsg(common.USER_SYSTEM, RoomId(tid), map[string]interface{}{"type": common.MSG_TYPE_JOIN_TOPIC, "uid": uid, "tid": tid, "total": total, TOP_USERS: users})
	//更新用户状态
	if err := status.UpdateStatus(uid, status.Status{common.STATUS_TYPE_INTOPIC, tid, "在话题室", ""}); err != nil {
		mainLog.Append(fmt.Sprintf("update status of user %v failed : %v", uid, err.Error()))
	}
	to := topic.Tm.Add(TOPIC_TIMEOUT * time.Second)
	//stat.Append(uid, stat.ACTION_TOPIC_JOIN, nil)
	uinfos, e := user_overview.GetUserObjects(topic.Uid)
	if e != nil {
		return e
	}
	if uinfos[topic.Uid] == nil {
		return errors.New("topic creator info not found")
	}
	res := make(map[string]interface{})
	res["tag"] = RoomId(tid)
	res["total"] = total
	res["uid"] = topic.Uid
	res["nickname"] = uinfos[topic.Uid].Nickname
	res["timeout"] = to
	res[TOP_USERS] = users
	result["res"] = res
	return nil
}

func OutTopic(uid uint32, tid uint32, res map[string]interface{}) error {
	//检查是否是话题创建者
	topics, e := GetTopics(tid)
	if e != nil {
		return e
	}
	//更新用户状态
	if err := status.ClearStatus(uid); err != nil {
		mainLog.Append(fmt.Sprintf("clear status of user %v failed : %v", uid, err.Error()))
	}
	topic, _ := topics[tid]
	if topic != nil && topic.Uid == uid {
		if err := unread.UpdateReadTime(uid, common.UNREAD_MY_TOPIC); err != nil {
			mainLog.Append(fmt.Sprintf("update redis failed : %v", err.Error()))
		}
		ur := map[string]interface{}{common.UNREAD_MY_TOPIC: 0}
		res[common.UNREAD_KEY] = ur
		return inRoom(tid, false)
	}
	return nil
}

func LeaveTopic(uid uint32, tid uint32, res map[string]interface{}) error {
	//检查是否是话题创建者
	topics, e := GetTopics(tid)
	if e != nil {
		return e
	}
	//更新用户状态
	if err := status.ClearStatus(uid); err != nil {
		mainLog.Append(fmt.Sprintf("clear status of user %v failed : %v", uid, err.Error()))
	}
	topic, _ := topics[tid]
	if topic != nil && topic.Uid == uid {
		if err := unread.UpdateReadTime(uid, common.UNREAD_MY_TOPIC); err != nil {
			mainLog.Append(fmt.Sprintf("update redis failed : %v", err.Error()))
		}
		ur := map[string]interface{}{common.UNREAD_MY_TOPIC: 0}
		res[common.UNREAD_KEY] = ur
		return inRoom(tid, false)
	}
	//离开聊天室
	if e = push.DelTag(uid, RoomId(tid)); e != nil {
		return e
	}
	//离开话题
	wcon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_USERS)
	defer wcon.Close()
	if _, e = wcon.Do("ZREM", tid, uid); e != nil {
		return e
	}

	tcon := rdb.GetWriteConnection(redis_db.REDIS_USER_TOPIC)
	defer tcon.Close()
	if _, e := tcon.Do("DEL", uid); e != nil {
		return e
	}
	users, total, e := topicUsers(tid, 0, TOPIC_TOP_USER_NUM)
	if e != nil {
		return e
	}
	push.SendTagMsg(common.USER_SYSTEM, RoomId(tid), map[string]interface{}{"type": common.MSG_TYPE_LEAVE_TOPIC, "uid": uid, "tid": tid, "mode": "self", "total": total, TOP_USERS: users})
	if err := topicFull(tid, false); err != nil {
		mainLog.Append(fmt.Sprintf("update sdb.discovery of field full failed : %v", err.Error()))
	}
	//更新参与话题记录表
	var tmStr string
	e = mdb.QueryRow(sql_9, uid, tid).Scan(&tmStr)
	if e != nil {
		mainLog.Append(fmt.Sprintf("LeaveTopic error:%v", e.Error()))
		return nil
	}
	tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
	if e != nil {
		mainLog.Append(fmt.Sprintf("LeaveTopic error:%v", e.Error()))
		return nil
	}
	_, e = mdb.Exec(sql_10, int(utils.Now.Sub(tm)/time.Second), uid, tid)
	if e != nil {
		mainLog.Append(fmt.Sprintf("LeaveTopic error:%v", e.Error()))
	}
	return nil
}

func Kick(uid uint32, tid uint32, forTm uint32) error {
	//加入黑名单
	bcon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_BLACKLIST)
	defer bcon.Close()
	if _, e := bcon.Do("ZADD", tid, utils.Now.Add(time.Duration(forTm)*time.Minute).Unix(), uid); e != nil {
		return e
	}
	//离开聊天室
	if e := push.DelTag(uid, RoomId(tid)); e != nil {
		return e
	}
	//离开话题
	wcon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_USERS)
	defer wcon.Close()
	if _, e := wcon.Do("ZREM", tid, uid); e != nil {
		return e
	}
	users, total, e := topicUsers(tid, 0, TOPIC_TOP_USER_NUM)
	if e != nil {
		return e
	}
	push.SendTagMsg(common.USER_SYSTEM, RoomId(tid), map[string]interface{}{"type": common.MSG_TYPE_LEAVE_TOPIC, "uid": uid, "tid": tid, "mode": "kick", "total": total, TOP_USERS: users})
	if err := topicFull(tid, false); err != nil {
		mainLog.Append(fmt.Sprintf("update sdb.discovery of field full failed : %v", err.Error()))
	}
	//更新用户状态
	if err := status.ClearStatus(uid); err != nil {
		mainLog.Append(fmt.Sprintf("clear status of user %v failed : %v", uid, err.Error()))
	}
	//更新参与话题记录表
	var tmStr string
	e = mdb.QueryRow(sql_11, uid, tid).Scan(&tmStr)
	if e != nil {
		mainLog.Append(fmt.Sprintf("Kick error:%v", e.Error()))
		return nil
	}
	tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
	if e != nil {
		mainLog.Append(fmt.Sprintf("Kick error:%v", e.Error()))
		return nil
	}
	_, e = mdb.Exec(sql_12, int(utils.Now.Sub(tm)/time.Second), uid, tid)
	if e != nil {
		mainLog.Append(fmt.Sprintf("Kick error:%v", e.Error()))
	}

	return nil
}

func Send(from uint32, tag string, content interface{}, res map[string]interface{}) (msgid uint64, e error) {
	tid, err := utils.ToUint32(tag[len(common.TAG_PREFIX_TOPIC):])
	if err != nil {
		return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("invalid tag :%v", tag))
	}

	switch value := content.(type) {
	case map[string]interface{}:
		msgid, offlines, err := SendMsg(from, tid, value)
		if err != nil {
			return 0, service.NewError(service.ERR_INTERNAL, fmt.Sprintf("send message error :%v", err.Error()))
		}

		users, total, e := topicUsers(tid, 0, TOPIC_TOP_USER_NUM)
		if e != nil {
			return msgid, e
		}
		push.SendTagMsg(common.USER_SYSTEM, RoomId(tid), map[string]interface{}{"type": common.MSG_TYPE_TOPIC_TOP_USERS, "tid": tid, "total": total, TOP_USERS: users})
		res["offline"] = offlines
		//stat.Append(from, stat.ACTION_TOPIC_CHAT, nil)
		return msgid, nil
	default:
		return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("msg must be a json:%v", content))
	}
}

// 圈子图片消息检测并推送
func doPicCheckPush(msgid uint64, from uint32, tid uint32, content map[string]interface{}) (offline []uint32, e error) {
	img := utils.ToString(content["img"])
	if img == "" {
		return
	}
	m, er := general.CheckImg(general.IMGCHECK_SEXY_AND_AD, img)
	if er != nil {
		mainLog.AppendObj(er, "[topic send]error checkimg", content, from, msgid)
		general.Alert("push", "topic check img is error")
	}
	if v, ok := m[img]; ok && v.Status != 0 {
		general.DeleBadPicMessage(msgid, from, img, 1)
		// 发送失败提示消息
		mid, _ := general.SendMsg(common.USER_SYSTEM, from, map[string]interface{}{"type": common.MSG_TYPE_PIC_INVALID, "content": "图片审核未通过，消息发送失败", "msgid": msgid}, RoomId(tid))
		mainLog.AppendObj(nil, "[topic send]doPicCheckPush,img check staus: ", v, content, mid, RoomId(tid))
		return
	}
	//ExecSendTagMsg(msgid uint64, from uint32, tag string, content map[string]interface{}) (offline []uint32, e error)
	_, e = general.ExecSendTagMsg(msgid, from, RoomId(tid), content)
	if e != nil {
		mainLog.AppendObj(e, "[topic send]doPicTagPushCheck, push.ExecSendTagMsg is error  ", msgid, content, from, RoomId(tid))
	}
	//mainLog.AppendObj(nil, "-topic--doPicCheckPush---end ", content, msgid)
	return
}

func SendMsg(from uint32, tid uint32, content map[string]interface{}) (msgid uint64, offlines []uint32, e error) {
	ucon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_USERS)
	defer ucon.Close()
	score, e := redis.Float64(ucon.Do("ZSCORE", tid, from))
	switch e {
	case nil:
		typ := utils.ToString(content["type"])
		switch typ {
		case common.MSG_TYPE_TEXT, common.MSG_TYPE_VOICE, common.MSG_TYPE_PIC:
			if int64(score) > -utils.Now.Unix() {
				if _, e := ucon.Do("ZADD", tid, -utils.Now.Unix(), from); e != nil {
					return 0, nil, e
				}
			}
			content["tid"] = tid
			if typ == common.MSG_TYPE_PIC {
				//PrepareSendTagMsg(from uint32, tag string, content map[string]interface{}) (msgid uint64, e error)
				msgid, e = push.PrepareSendTagMsg(from, RoomId(tid), content)
				if e != nil {
					return
				}
				go doPicCheckPush(msgid, from, tid, content)
				mainLog.AppendObj(e, "[topic send] content:", content, "msgid :", msgid, from, RoomId(tid))
			} else {
				msgid, offlines, e = push.SendTagMsg(from, RoomId(tid), content)
				if e != nil {
					return
				}
			}
			if typ == common.MSG_TYPE_TEXT {
				AddRecentMessage(tid, from, content)
			}
			if in, e := isInRoom(tid); e == nil {
				if !in {
					topics, e := GetTopics(tid)
					if e != nil {
						return 0, nil, e
					}
					topic := topics[tid]
					if topic != nil {
						ur := map[string]interface{}{common.UNREAD_MY_TOPIC: 0}
						e := unread.GetUnreadNum(topic.Uid, ur)
						if e != nil {
							mainLog.Append("get unread num error:" + e.Error())
						}
						_, e = general.SendMsg(common.USER_SYSTEM, topic.Uid, map[string]interface{}{"type": common.MSG_TYPE_UNREAD, common.UNREAD_KEY: ur}, "")
						if e != nil {
							mainLog.Append("send topic message error:" + e.Error())
						}
					}
				}
			} else {
				return 0, nil, e
			}
			return
		default:
			return 0, nil, errors.New("invalid message type : " + typ)
		}
	case redis.ErrNil:
		return 0, nil, service.NewError(service.ERR_VERIFY_FAIL, "not joined", "您尚未加入该圈子")
	default:
		return 0, nil, e
	}
}

func TopicDetail(tid uint32) (topic Topic, e error) {
	//暂不实现
	return topic, errors.New("not implemented")
}

func DelMsg(uid uint32, tag string, msgid uint64) (e error) {
	tid, err := utils.ToUint32(tag[len(common.TAG_PREFIX_TOPIC):])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("invalid tag :%v", tag))
	}
	return DelMessage(uid, tid, msgid)
}

func DelMessage(uid uint32, tid uint32, msgid uint64) (e error) {
	tinfos, e := GetTopics(tid)
	if e != nil {
		return errors.New(fmt.Sprintf("GetTopics error : %v", e.Error()))
	}
	tinfo := tinfos[tid]
	if tinfo == nil {
		return errors.New("topic not found")
	}
	if tinfo.Uid != uid {
		return service.NewError(service.ERR_INVALID_REQUEST, "not topic owner", "权限不足")
	}
	if _, e = msgdb.Exec(sql_13, RoomId(tid), msgid); e != nil {
		return e
	}
	push.SendTagMsg(uid, RoomId(tid), map[string]interface{}{"type": common.MSG_TYPE_DEL_MSG, "msgid": msgid})
	return
}

func topicUsers(tid uint32, offset int, count int) (users []User, total uint32, e error) {
	var uids []uint32
	rcon := rdb.GetReadConnection(redis_db.REDIS_TOPIC_USERS)
	defer rcon.Close()
	total, e = redis.Uint32(rcon.Do("ZCARD", tid))
	if e != nil {
		return nil, 0, e
	}
	values, e := redis.Values(rcon.Do("ZRANGEBYSCORE", tid, "-inf", "+inf", "LIMIT", offset, count))
	if e != nil {
		return nil, 0, e
	}
	if e = redis.ScanSlice(values, &uids); e != nil {
		return nil, 0, e
	}
	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, 0, e
	}
	users = make([]User, 0, len(uids))

	for _, uid := range uids {
		uinfo := uinfos[uid]
		if uinfo != nil {
			users = append(users, User{uid, uinfo.Nickname, uinfo.Gender, uinfo.Age, uinfo.Avatar})
		}
	}
	return
}

func TopicUsers(tid uint32, cur int, ps int) (users []User, total uint32, e error) {
	return topicUsers(tid, (cur-1)*ps, ps)
}

func JoinHistory(uid uint32, cur int, ps int) (topics []TopicInfo, total int, e error) {
	e = mdb.QueryRow(sql_14, uid).Scan(&total)
	if e != nil {
		return nil, 0, e
	}
	var sql_15 string = "select tid from topic_record where uid=? order by start_tm desc" + utils.BuildLimit(cur, ps)
	rows, e := mdb.Query(sql_15, uid)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	tids := make([]uint32, 0, ps)
	for rows.Next() {
		var tid uint32
		if err := rows.Scan(&tid); err != nil {
			return nil, 0, e
		}
		tids = append(tids, tid)
	}
	if topics, e = makeTopicsInfoByTid(tids); e != nil {
		return nil, 0, e
	}
	if e = getOnlines(topics); e != nil {
		return nil, 0, e
	}
	if e = getTrends(topics); e != nil {
		return nil, 0, e
	}
	return
}

func makeTopicsInfoByTid(tids []uint32) (topics []TopicInfo, e error) {
	topics = make([]TopicInfo, 0, len(tids))
	for _, tid := range tids {
		var topic TopicInfo
		topic.Tid = tid
		topics = append(topics, topic)
	}
	tinfos, e := GetTopics(tids...)
	if e != nil {
		return nil, errors.New(fmt.Sprintf("GetTopics error : %v", e.Error()))
	}
	uids := make([]uint32, 0, len(topics))
	for i, topic := range topics {
		tinfo := tinfos[topic.Tid]
		if tinfo != nil {
			uids = append(uids, tinfo.Uid)
			topics[i].Uid = tinfo.Uid
			topics[i].Capacity = tinfo.Capacity
			topics[i].Pics = tinfo.Pics
			topics[i].PicsLevel = tinfo.PicsLevel
			topics[i].Tag = tinfo.Tag
			topics[i].Title = tinfo.Title
			topics[i].Tm = tinfo.Tm
		}
	}

	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, errors.New(fmt.Sprintf("GetUserObjects error : %v", e.Error()))
	}
	for i, topic := range topics {
		uinfo := uinfos[topic.Uid]
		if uinfo != nil {
			topics[i].Nickname = uinfo.Nickname
			topics[i].Age = uinfo.Age
			topics[i].Gender = uinfo.Gender
			topics[i].Avatar = uinfo.Avatar
			topics[i].Online = 0 //先随便存一个值，后续的方法会补上
		}
	}
	return
}

var sql_16 string = "update discovery set full=? where tid=?"

func topicFull(tid uint32, full bool) (e error) {
	_, e = sdb.Exec(sql_16, full, tid)
	return
}

var sql_17 string = "update discovery set in_room=? where tid=?"

func inRoom(tid uint32, in bool) (e error) {
	_, e = sdb.Exec(sql_17, in, tid)
	return
}

var sql_18 string = "select in_room from discovery where tid=? limit 1"

func isInRoom(tid uint32) (in bool, e error) {
	e = sdb.QueryRow(sql_18, tid).Scan(&in)
	return
}

var sql_19 string = fmt.Sprintf("update topic set status=%v where id=?", common.TOPIC_STATUS_CLOSED)
var sql_20 string = "update discovery set tid=0 where tid=?"

func IClose(tid uint32) error {
	if _, e := mdb.Exec(sql_19, tid); e != nil {
		return e
	}
	ucon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_USERS)
	defer ucon.Close()
	if _, e := ucon.Do("DEL", tid); e != nil {
		return e
	}
	bcon := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_BLACKLIST)
	defer bcon.Close()
	if _, e := bcon.Do("DEL", tid); e != nil {
		return e
	}
	//删除push服务的标签
	if e := push.ClearTag(RoomId(tid)); e != nil {
		return e
	}

	//更新发现列表
	if _, e := sdb.Exec(sql_20, tid); e != nil {
		return e
	}
	return nil
}

func userOffline(msgid int, data interface{}) {
	switch v := data.(type) {
	case message.Offline:
		//离开上一个加入的话题
		con := rdb.GetReadConnection(redis_db.REDIS_USER_TOPIC)
		defer con.Close()
		lastTid, e := redis.Uint32(con.Do("GET", v.Uid))
		res := map[string]interface{}{}
		switch e {
		case redis.ErrNil:
		case nil:
			if e := LeaveTopic(v.Uid, lastTid, res); e != nil {
				mainLog.Append(fmt.Sprintf("leave last joined topic %v error : %v", lastTid, e.Error()))
			}
		default:
			mainLog.Append(fmt.Sprintf("leave last joined topic %v error : %v", lastTid, e.Error()))
		}
		tid, e := GetMyTopic(v.Uid, common.TOPIC_STATUS_ACTIVE)
		switch e {
		case sql.ErrNoRows:
		case nil:
			if e := LeaveTopic(v.Uid, tid, res); e != nil {
				mainLog.Append(fmt.Sprintf("leave created topic %v error : %v", tid, e.Error()))
			}
		default:
			mainLog.Append(fmt.Sprintf("leave created topic %v error : %v", tid, e.Error()))
		}

	}
}

var sql_21 string = "select id from topic where uid=? and status=?"

func GetMyTopic(uid uint32, status int) (tid uint32, e error) {
	e = mdb.QueryRow(sql_21, uid, status).Scan(&tid)
	return
}
