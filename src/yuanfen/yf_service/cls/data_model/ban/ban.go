package ban

import (
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yuanfen/yf_service/cls"
)

var mdb *mysql.MysqlDB
var rdb *redis.RedisPool
var msgdb *mysql.MysqlDB

func Init(env *service.Env) {
	mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	msgdb = env.ModuleEnv.(*cls.CustomEnv).MsgDB
}

//投诉用户
func UserComplain(id uint32, fromuid uint32, info string, tp int) (e error) {
	_, e = mdb.Exec("insert into user_complain (uid,fromuid,info,`type`)values(?,?,?,?) on duplicate key update count=count+1", id, fromuid, info, tp)
	return
}

//用户发风险消息
func MsgFilter(uid uint32, content string) (e error) {
	_, e = mdb.Exec("insert into on user_ban_message (uid,content)valuse(?,?) duplicate key update count=count+1", uid, content)
	return
}

//获取投诉用户列表
func GetComplainList(index int, count int) (logs []map[string]interface{}, e error) {
	rows, e := mdb.Query("select id,uid,stat,count from user_complain where id<? LIMIT ?", index, count)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	logs = make([]map[string]interface{}, 0, count)
	for rows.Next() {
		var id uint32
		var uid uint32
		var stat, mcount int
		if e = rows.Scan(&id, &uid, &stat, &mcount); e != nil {
			return nil, e
		}
		logs = append(logs, map[string]interface{}{"id": id, "uid": uid, "stat": stat, "count": mcount})
	}
	return
}

//获取风险消息列表
func GetFilterUser(index int, count int) (logs []map[string]interface{}, e error) {
	rows, e := mdb.Query("select id,uid,stat,count,content from user_ban_message where id<? LIMIT ?", index, count)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	logs = make([]map[string]interface{}, 0, count)
	for rows.Next() {
		var id uint32
		var uid uint32
		var stat, mcount int
		var content string
		if e = rows.Scan(&id, &uid, &stat, &mcount, content); e != nil {
			return nil, e
		}
		logs = append(logs, map[string]interface{}{"id": id, "uid": uid, "stat": stat, "count": mcount, "content": content})
	}
	return
}
