package status

import (
	"encoding/json"
	"yf_pkg/log"
	"yf_pkg/redis"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
)

type Status struct {
	Stype string `json:"stype"` //状态的类型
	Id    uint32 `json:"id"`
	Show  string `json:"show"`  //显示内容，状态生成方构造
	Extra string `json:"extra"` //其它信息
}

var defaultStatus Status = Status{"default", 0, "空闲", ""}

var rdb *redis.RedisPool
var mainLog *log.MLogger

func Init(env *cls.CustomEnv) {
	rdb = env.MainRds
	mainLog = env.MainLog
}

func ClearStatus(uid uint32) error {
	conn := rdb.GetWriteConnection(redis_db.REDIS_USER_STATUS)
	defer conn.Close()
	_, e := conn.Do("DEL", uid)
	return e
}

func UpdateMStatus(uids []uint32, status Status) error {
	conn := rdb.GetWriteConnection(redis_db.REDIS_USER_STATUS)
	defer conn.Close()
	j, e := json.Marshal(&status)
	if e != nil {
		return e
	}
	for _, uid := range uids {
		if e := conn.Send("SET", uid, j); e != nil {
			return e
		}
	}
	conn.Flush()
	for _, _ = range uids {
		if _, err := conn.Receive(); err != nil {
			return err
		}
	}
	return nil
}

func UpdateStatus(uid uint32, status Status) error {
	conn := rdb.GetWriteConnection(redis_db.REDIS_USER_STATUS)
	defer conn.Close()
	j, e := json.Marshal(&status)
	if e != nil {
		return e
	}
	_, e = conn.Do("SET", uid, j)
	return e
}

func GetStatus(uids ...uint32) (status map[uint32]*Status, e error) {
	status = make(map[uint32]*Status)
	conn := rdb.GetReadConnection(redis_db.REDIS_USER_STATUS)
	defer conn.Close()
	for _, uid := range uids {
		if e := conn.Send("GET", uid); e != nil {
			return nil, e
		}
	}
	conn.Flush()
	for _, uid := range uids {
		j, e := redis.Bytes(conn.Receive())
		switch e {
		case redis.ErrNil:
			status[uid] = &defaultStatus
		case nil:
			var s Status
			e = json.Unmarshal(j, &s)
			if e != nil {
				return nil, e
			}
			status[uid] = &s
		default:
			return nil, e
		}
	}
	return
}
