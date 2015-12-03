package hongniang

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"yf_pkg/format"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
)

var mdb *mysql.MysqlDB
var msgdb *mysql.MysqlDB
var rdb *redis.RedisPool
var mainLog *log.MLogger

func Init(env *cls.CustomEnv) {
	mdb = env.MainDB
	msgdb = env.MsgDB
	rdb = env.MainRds
	mainLog = env.MainLog
	msgdb = env.MsgDB
}

func SendToHongniang(from uint32, content interface{}) (msgid uint64, e error) {
	hnid, err := getHongniang(from)
	if err != nil {
		return 0, service.NewError(service.ERR_INTERNAL, err.Error())
	}
	switch value := content.(type) {
	case map[string]interface{}:
		typ := utils.ToString(value["type"])
		switch typ {
		case common.MSG_TYPE_TEXT, common.MSG_TYPE_VOICE, common.MSG_TYPE_PIC:
			//如果有红娘在线，先发给分配的红娘
			if hnid != common.USER_HONGNIANG {
				fmt.Printf("hongniang: from=%v, to=%v,value=%v\n", from, hnid, value)
				_, err := general.SendMsg(from, hnid, value, "")
				if err != nil {
					return 0, service.NewError(service.ERR_INTERNAL, err.Error())
				}
			}
			//再发给公共ID
			msgid, err := general.SendMsg(from, common.USER_HONGNIANG, value, "")
			if err != nil {
				return 0, service.NewError(service.ERR_INTERNAL, err.Error())
			}
			return msgid, e
		default:
			return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("invalid content type:%v", typ))
		}
	default:
		return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("content must be a json:%v", content))
	}
}

func SendToUser(from uint32, to uint32, content interface{}) (msgid uint64, e error) {
	is, err := IsHongniang(from)
	if err != nil {
		return 0, service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if !is {
		return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("you are not hongniang:%v", from))
	}
	switch value := content.(type) {
	case map[string]interface{}:
		typ := utils.ToString(value["type"])
		switch typ {
		case common.MSG_TYPE_TEXT, common.MSG_TYPE_VOICE, common.MSG_TYPE_PIC:
			msgid, err := general.SendMsg(common.USER_HONGNIANG, to, value, "")
			if err != nil {
				return 0, service.NewError(service.ERR_INTERNAL, err.Error())
			}
			return msgid, e
		default:
			return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("invalid content type:%v", typ))
		}
	default:
		return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("content must be a json:%v", content))
	}
}

func GetMessages(uid uint32, msgid uint64, count uint32) (msgs []map[string]interface{}, e error) {
	sql := "select id,`from`,`to`,content,tm from message where ((`to`=? and `from`=?) or (`to`=? and `from`=?)) and id<? order by id desc limit ?"
	if msgid == 0 {
		msgid = math.MaxUint32
	}
	rows, e := msgdb.Query(sql, uid, common.USER_HONGNIANG, common.USER_HONGNIANG, uid, msgid, count)
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

//-----------------------Private Functions-------------------------//

func IsHongniang(uid uint32) (bool, error) {
	count := 0
	sql := "select count(uid) from hongniang where uid=?"
	e := mdb.QueryRow(sql, uid).Scan(&count)
	if e != nil || count == 0 {
		return false, e
	} else {
		return true, nil
	}
}

func findOnlineHongniang(uid uint32) (uint32, error) {
	con := rdb.GetWriteConnection(redis_db.REDIS_HONGNIANG)
	defer con.Close()
	var hnid uint32
	s := "select hongniang.uid from hongniang,user_online where hongniang.uid=user_online.uid and user_online.tm > ? order by rand() limit 1"
	e := mdb.QueryRow(s, utils.Now).Scan(&hnid)
	switch e {
	case sql.ErrNoRows:
		return common.USER_HONGNIANG, nil //没有红娘在线，只能发给系统ID
	case nil:
		_, e := con.Do("SET", uid, hnid)
		return hnid, e
	default:
		return 0, e
	}
}
func getHongniang(uid uint32) (hnid uint32, e error) {
	con := rdb.GetReadConnection(redis_db.REDIS_HONGNIANG)
	defer con.Close()
	hnid, err := redis.Uint32(con.Do("GET", uid))
	switch err {
	case redis.ErrNil:
		hnid, e = findOnlineHongniang(uid)
	case nil:
		sql := "select count(uid) from user_online where uid=? and tm > ?"
		count := 0
		if e := mdb.QueryRow(sql, hnid, utils.Now).Scan(&count); e != nil {
			return 0, e
		}
		if count == 0 {
			hnid, e = findOnlineHongniang(uid)
		}
	default:
		return 0, err
	}
	return hnid, e
}
