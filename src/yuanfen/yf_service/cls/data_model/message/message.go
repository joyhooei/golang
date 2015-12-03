package message

import (
	sqlpkg "database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"yf_pkg/cachedb"
	"yf_pkg/format"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/push"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/relation/base"
	"yuanfen/yf_service/cls/word_filter"
)

var mdb *mysql.MysqlDB
var msgdb *mysql.MysqlDB
var sdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool
var cdb *cachedb.CacheDB
var mainLog *log.MLogger

func Init(env *cls.CustomEnv) (e error) {
	rdb = env.MainRds
	sdb = env.SortDB
	mdb = env.MainDB
	msgdb = env.MsgDB
	cache = env.CacheRds
	cdb = env.CacheDB
	mainLog = env.MainLog

	return e
}

func Send(from uint32, to uint32, tag string, content interface{}, res map[string]interface{}) (msgid uint64, e error) {
	switch to {
	case 0: //Tag消息
		msgid, e = SendTag(from, tag, content, res)
	default:
		msgid, e = SendUser(from, to, tag, content, res)
	}
	return
}

func addFriend(from, to uint32) (e error) {
	if !general.IsSystemUser(to) {
		if fr, e := base.IsFriend(from, to); e != nil {
			return e
		} else if !fr {
			//如果是回复认识一下的消息，则加为好友
			success, e := base.ReplySayHello(from, to)
			if e != nil {
				return e
			}
			if !success {
				return service.NewError(service.ERR_PERMISSION_DENIED, "not friend", "对方不是你认识的人")
			}
		}
	}
	return nil
}
func sendHelper(from uint32, to uint32, tag string, content map[string]interface{}, res map[string]interface{}) (msgid uint64, e error) {
	p, e := general.GetUserProtect(to)
	if e != nil {
		return 0, e
	}
	msgid, online, err := general.SendMsgWithOnline(from, to, content, tag, p.ChatNotNotify == 0)
	if err != nil {
		general.Alert("push", "send message fail")
		return 0, service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res["online"] = online
	if err := general.UpdateRecentChatUser(from, to, msgid, content["type"].(string)); err != nil {
		mainLog.Append("UpdateRecentChatUser error:" + err.Error())
	}
	stat.Append(from, stat.ACTION_SEND_MSG, map[string]interface{}{"to": to})
	return msgid, nil
}

//图片消息实际发送方法
func sendPicHelper(msgid uint64, from uint32, to uint32, tag string, content map[string]interface{}) (e error) {
	// 执行图片较验
	img := utils.ToString(content["img"])
	m, er := general.CheckImg(general.IMGCHECK_SEXY_AND_AD, img)
	if er != nil {
		general.Alert("push", "check img is error")
		mainLog.AppendObj(er, "sendPicHelper is error  ", msgid, content, from)
	}
	if v, ok := m[img]; ok && v.Status != 0 {
		general.DeleBadPicMessage(msgid, from, img, 0)
		mid, _ := general.SendMsg(to, from, map[string]interface{}{"type": common.MSG_TYPE_PIC_INVALID, "content": "图片审核未通过,发送失败", "msgid": msgid}, tag)
		mainLog.AppendObj(nil, "[send]sendPicHelper,img check staus: ", v, mid)
		return
	}
	if e = addFriend(from, to); e != nil {
		return e
	}
	p, e := general.GetUserProtect(to)
	if e != nil {
		return e
	}
	_, err := general.ExecSend(msgid, from, to, content, tag, p.ChatNotNotify == 0)
	if err != nil {
		general.Alert("push", "send message fail")
		mainLog.AppendObj(err, "---sendPicHelper---is error ")
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if err := general.UpdateRecentChatUser(from, to, msgid, common.MSG_TYPE_PIC); err != nil {
		mainLog.Append("UpdateRecentChatUser error:" + err.Error())
	}
	stat.Append(from, stat.ACTION_SEND_MSG, map[string]interface{}{"to": to})
	return nil
}

func SendUser(from uint32, to uint32, tag string, content interface{}, res map[string]interface{}) (msgid uint64, e error) {
	switch value := content.(type) {
	case map[string]interface{}:
		typ := utils.ToString(value["type"])
		switch typ {
		case common.MSG_TYPE_TEXT:
			n, replaced := word_filter.Replace(utils.ToString(value["content"]))
			if n > 0 {
				origin := value["content"]
				value["content"] = replaced + "[敏感词已过滤]"
				msgid, e := sendHelper(from, to, tag, value, res)
				if e != nil {
					return 0, e
				}
				sql := "insert into bad_message(id,`from`,origin,replaced,num)values(?,?,?,?,?)"
				if _, e = msgdb.Exec(sql, msgid, from, origin, replaced, n); e != nil {
					mainLog.Append(fmt.Sprintf("add to bad_message table error:%v", e.Error()))
				}
				return msgid, nil
			} else {
				if e = addFriend(from, to); e != nil {
					return 0, e
				}
				return sendHelper(from, to, tag, value, res)
			}
		case common.MSG_TYPE_VOICE:
			if e = addFriend(from, to); e != nil {
				return 0, e
			}
			return sendHelper(from, to, tag, value, res)
		case common.MSG_TYPE_PIC:
			// 图片消息，需要图片过滤
			msgid, e := general.PrepareSendMsg(from, to, value, tag)
			if e != nil {
				mainLog.AppendObj(e, "--PrepareSendMsg is error----", from, to, tag, value)
				return 0, e
			}
			// 异步图片检测,并实际推送消息
			go sendPicHelper(msgid, from, to, tag, value)
			return msgid, e

		case common.MSG_TYPE_READ, common.MSG_TYPE_LOCATION, common.MSG_TYPE_OTHER:
			return sendHelper(from, to, tag, value, res)
		default:
			return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("invalid content type:%v", typ))
		}
	default:
		return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("content must be a json:%v", content))
	}
}

/*
GetMessage根据消息ID获取消息内容
*/
func GetMessage(uid uint32, msgid uint64) (msg map[string]interface{}, e error) {
	msgs, e := general.GetMessageById([]uint64{msgid})
	if e != nil {
		return nil, e
	}
	m, ok := msgs[msgid]
	if !ok {
		return nil, service.NewError(service.ERR_NOT_FOUND, "msg not found", "您查找的消息不存在")
	}
	switch v := m.(type) {
	case map[string]interface{}:
		msg = v
	default:
		return nil, service.NewError(service.ERR_INTERNAL, "msg content is not json", "")
	}

	to, e := utils.ToUint32(msg["to"])
	if e != nil {
		return nil, e
	}
	from, e := utils.ToUint32(msg["from"])
	if e != nil {
		return nil, e
	}
	tag := utils.ToString(msg["tag"])
	if to != uid && from != uid {
		if to == 0 {
			in, e := push.InTag(uid, tag)
			if e != nil {
				return nil, e
			}
			if !in {
				return nil, errors.New("not joined")
			}
		} else {
			return nil, service.NewError(service.ERR_PERMISSION_DENIED, "not permit", "权限不足")
		}
	}

	return
}

/*
RecentMessages获取用户的离线消息,最多返回最近的1000条。

如果last_msgid=0，则表示是新安装的客户端，会同时也返回自己发的消息。

参数：
	last_msgid: 上次收到的最后一条消息。
*/
func RecentMessages(uid uint32, types []string, last_msgid uint64, count uint32) (msgs []map[string]interface{}, e error) {
	var rows *sqlpkg.Rows
	var sql, where string
	if count > 1000 {
		count = 1000
	}
	if len(types) > 0 {
		where = (" and id>=? and `type`" + mysql.In(types) + " order by tm desc limit ?")
	} else {
		where = " and id>=? order by tm desc limit ?"
	}
	if last_msgid == 0 {
		last_msgid, e = redis.Uint64(rdb.HGet(redis_db.REDIS_USER_DATA, uid, common.MSG_START_POS))
		switch e {
		case nil, redis.ErrNil:
		default:
			return nil, e
		}
		sql = "(select id,`from`,`to`,tag,content,tm from message where `to` = ? and `from` > ?" + where + ")union(select id,`from`,`to`,tag,content,tm from message where `from` = ? and `to` > ?" + where + ")order by id desc limit ?"
		rows, e = msgdb.Query(sql, uid, common.UID_MAX_SYSTEM, last_msgid, count, uid, common.UID_MAX_SYSTEM, last_msgid, count, count)
	} else {
		sql = "select id,`from`,`to`,tag,content,tm from message where `to` = ?" + where
		rows, e = msgdb.Query(sql, uid, last_msgid, count)
	}
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	msgs = make([]map[string]interface{}, 0, count)
	startPos, e := redis.Int64Map(rdb.HGetAll(redis_db.REDIS_USER_MSG_START_POS, uid))
	if e != nil {
		return nil, e
	}
	for rows.Next() {
		var msgid uint64
		var from, to uint32
		var content []byte
		var tmStr, tag string
		if e = rows.Scan(&msgid, &from, &to, &tag, &content, &tmStr); e != nil {
			return nil, e
		}
		if msgid != last_msgid {
			//判断是否是发给自己的群聊消息
			if to == 0 {
				in, e := push.InTag(uid, tag)
				if e != nil {
					return nil, e
				}
				if !in {
					continue
				}
			}
			//过滤掉用户曾经删除过的私聊消息
			if to != 0 {
				him := from
				if from == uid {
					him = to
				}
				start, ok := startPos[utils.ToString(him)]
				if ok && uint64(start) >= msgid {
					continue
				}
			}
			tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
			if e != nil {
				return nil, e
			}
			var j map[string]interface{}
			if e := json.Unmarshal(content, &j); e != nil {
				return nil, e
			}
			msgs = append(msgs, map[string]interface{}{"msgid": msgid, "from": from, "to": to, "tm": tm, "tag": tag, "content": j})
		}
	}
	return
}

/*
OfflineMessages获取用户的离线消息。

参数：
	last_msgid: 上次收到的最后一条消息。
	from_msgid: 已经获取到的离线消息中最老的一条的消息ID。
func OfflineMessages(uid uint32, types []string, last_msgid uint64, from_msgid uint64, count uint32) (msgs []map[string]interface{}, e error) {
	sql := ""
	where := ""
	if len(types) > 0 {
		where = (" and id>? and id<? and `type`" + mysql.In(types) + " order by id desc limit ?")
	} else {
		where = " and id>? and id<? order by id desc limit ?"
	}
	if last_msgid == 0 {
		sql = "(select id,`from`,`to`,content,tm from message where `to` = ?" + where + ")union(select id,`from`,`to`,content,tm from message where `from` = ?" + where + ")order by id desc limit ?"
	} else {
		sql = "select id,`from`,`to`,content,tm from message where `to` = ?" + where
	}
	if count > 100 {
		count = 100
	}
	if from_msgid == 0 {
		from_msgid = math.MaxUint64
	}
	var rows *sqlpkg.Rows
	if last_msgid == 0 {
		rows, e = msgdb.Query(sql, uid, last_msgid, from_msgid, count, uid, last_msgid, from_msgid, count, count)
	} else {
		rows, e = msgdb.Query(sql, uid, last_msgid, from_msgid, count)
	}
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	msgs = make([]map[string]interface{}, 0, count)
	for rows.Next() {
		var msgid uint64
		var from, to uint32
		var content []byte
		var tmStr string
		if e = rows.Scan(&msgid, &from, &to, &content, &tmStr); e != nil {
			return nil, e
		}
		tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
		if e != nil {
			return nil, e
		}
		var j map[string]interface{}
		if e := json.Unmarshal(content, &j); e != nil {
			return nil, e
		}
		msgs = append(msgs, map[string]interface{}{"msgid": msgid, "from": from, "to": to, "tm": tm, "content": j})
	}
	return
}
*/

/*
func TagMessages(uid uint32, tag string, types []string, last_msgid uint64, from_msgid uint64, count uint32) (msgs []map[string]interface{}, e error) {
	in, e := push.InTag(uid, tag)
	if e != nil {
		return nil, e
	}
	if !in {
		return nil, errors.New("not joined")
	}
	sql := ""
	if len(types) == 0 {
		types = []string{common.MSG_TYPE_TEXT, common.MSG_TYPE_VOICE, common.MSG_TYPE_PIC}
	}
	sql = "select id,`from`,content,tm from tag_message where `tag` = ? and id>? and id<? and `type`" + mysql.In(types) + " order by id desc limit ?"
	if count > 100 {
		count = 100
	}
	if from_msgid == 0 {
		from_msgid = math.MaxUint64
	}
	rows, e := msgdb.Query(sql, tag, last_msgid, from_msgid, count)
	if e != nil {
		general.Alert("mysql-message", "read tag_message fail")
		return nil, e
	}
	defer rows.Close()
	msgs = make([]map[string]interface{}, 0, count)
	for rows.Next() {
		var msgid uint64
		var from uint32
		var content []byte
		var tmStr string
		if e = rows.Scan(&msgid, &from, &content, &tmStr); e != nil {
			return nil, e
		}
		tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
		if e != nil {
			return nil, e
		}
		var j map[string]interface{}
		if e := json.Unmarshal(content, &j); e != nil {
			return nil, errors.New(fmt.Sprintf("parse %v error : %v", string(content), e.Error()))
		}
		msgs = append(msgs, map[string]interface{}{"msgid": msgid, "from": from, "tm": tm, "content": j})
	}
	return
}
*/

func RecentTagMessages(uid uint32, tag string, types []string, last_msgid uint64, count uint32) (msgs []map[string]interface{}, e error) {
	/*
		in, e := push.InTag(uid, tag)
		if e != nil {
			return nil, e
		}
			if !in {
				return nil, errors.New("not joined")
			}
	*/
	sql := "select tm from tag_message where id>=? order by id limit 1"
	tmStr := ""
	if e = msgdb.QueryRow(sql, last_msgid).Scan(&tmStr); e != nil {
		return nil, e
	}
	tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
	if len(types) == 0 {
		types = []string{common.MSG_TYPE_TEXT, common.MSG_TYPE_VOICE, common.MSG_TYPE_PIC}
	}
	sql = "select id,`from`,content,tm from tag_message where `tag` = ? and tm>=? and `type`" + mysql.In(types) + " order by tm desc limit ?"
	if count > 300 {
		count = 300
	}
	rows, e := msgdb.Query(sql, tag, tm, count)
	if e != nil {
		general.Alert("mysql-message", "read tag_message fail")
		return nil, e
	}
	defer rows.Close()
	msgs = make([]map[string]interface{}, 0, count)
	for rows.Next() {
		var msgid uint64
		var from uint32
		var content []byte
		var tmStr string
		if e = rows.Scan(&msgid, &from, &content, &tmStr); e != nil {
			return nil, e
		}
		if msgid != last_msgid {
			tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
			if e != nil {
				return nil, e
			}
			var j map[string]interface{}
			if e := json.Unmarshal(content, &j); e != nil {
				return nil, errors.New(fmt.Sprintf("parse %v error : %v", string(content), e.Error()))
			}
			msgs = append(msgs, map[string]interface{}{"msgid": msgid, "from": from, "tm": tm, "content": j})
		}
	}
	return
}

//////////////////////////////////////////////////////////////////

type DelTagMsgFunc func(uid uint32, tag string, msgid uint64) (e error)

var delTagMsgFuncs map[string]DelTagMsgFunc = map[string]DelTagMsgFunc{}

func RegisterDelTagMsgFunc(prefix string, fc DelTagMsgFunc) {
	delTagMsgFuncs[prefix] = fc
}
func Del(uid uint32, tag string, msgid uint64) (e error) {
	switch tag {
	case "":
		return delMsg(uid, msgid)
	default:
		return delTagMsg(uid, tag, msgid)
	}
}

func delMsg(uid uint32, msgid uint64) (e error) {
	return
}
func delTagMsg(uid uint32, tag string, msgid uint64) (e error) {
	for pre, tm := range delTagMsgFuncs {
		if strings.Index(tag, pre) == 0 {
			return tm(uid, tag, msgid)
		}
	}
	return service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("unknown tag prefix : %v", tag))
}
