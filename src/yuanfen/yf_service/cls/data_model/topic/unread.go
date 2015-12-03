package topic

import (
	"time"
	"yf_pkg/mysql"
	"yuanfen/yf_service/cls/common"
)

var sql1 string = "select id from topic where uid=? and status=?"
var sql2 string = "select count(*) from tag_message where tag=? and tm>? and type" + mysql.In([]string{common.MSG_TYPE_TEXT, common.MSG_TYPE_VOICE, common.MSG_TYPE_GIVE_PRESENT})

func UnreadNum(uid uint32, k string, from time.Time) (total uint32, show string) {
	switch k {
	case common.UNREAD_MY_TOPIC:
		var tid uint32
		if e := mdb.QueryRow(sql1, uid, 1).Scan(&tid); e != nil {
			return 0, ""
		}
		if e := msgdb.QueryRow(sql2, RoomId(tid), from).Scan(&total); e != nil {
			return 0, ""
		}
	}
	return total, ""
}
