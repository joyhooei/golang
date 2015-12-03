package topic

import (
	"encoding/json"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

func AddRecentMessage(tid uint32, from uint32, content map[string]interface{}) (e error) {
	t := utils.ToString(content["type"])
	if t != common.MSG_TYPE_TEXT {
		return
	}
	con := rdb.GetWriteConnection(redis_db.REDIS_TOPIC_RECENT_MSG)
	defer con.Close()

	num, e := redis.Uint32(con.Do("LLEN", tid))
	switch e {
	case nil:
		l := len(utils.ToString(content["content"]))
		if num < 2 || l > 10 {
			uinfos, e := user_overview.GetUserObjects(from)
			if e != nil {
				return e
			}
			msg := map[string]interface{}{"from": from, "content": content}
			uinfo := uinfos[from]
			if uinfo != nil {
				msg["nickname"] = uinfo.Nickname
				msg["avatar"] = uinfo.Avatar
			}
			b, e := json.Marshal(msg)
			if e != nil {
				return e
			}
			if _, e = con.Do("RPUSH", tid, b); e != nil {
				return e
			}

			if num >= 2 {
				if _, e := con.Do("LTRIM", tid, -2, -1); e != nil {
					return e
				}
			} else if num == 0 {
				if _, e = con.Do("EXPIRE", tid, TOPIC_TIMEOUT); e != nil {
					return e
				}
			}
		}
	default:
		return e
	}

	return nil
}

func GetRecentMessages(tids ...uint32) (msgs map[uint32][]interface{}, e error) {
	msgs = make(map[uint32][]interface{})
	con := rdb.GetReadConnection(redis_db.REDIS_TOPIC_RECENT_MSG)
	defer con.Close()
	for _, tid := range tids {
		if e := con.Send("LRANGE", tid, -2, -1); e != nil {
			return msgs, e
		}
	}
	con.Flush()
	for _, tid := range tids {
		v, e := redis.Values(con.Receive())
		if e != nil {
			return nil, e
		}
		mb := make([][]byte, 0, 2)
		if e = redis.ScanSlice(v, &mb); e != nil {
			return nil, e
		}
		msg := make([]interface{}, 0, 2)
		for _, b := range mb {
			content := map[string]interface{}{}
			if e = json.Unmarshal(b, &content); e != nil {
				return nil, e
			}
			uid, e := utils.ToUint32(content["from"])
			if e != nil {
				return nil, e
			}
			uinfo, e := user_overview.GetUserObject(uid)
			if e != nil {
				return nil, e
			}
			if uinfo.Stat == common.USER_STAT_NORMAL {
				msg = append(msg, content)
			}
		}
		msgs[tid] = msg
	}
	return
}
