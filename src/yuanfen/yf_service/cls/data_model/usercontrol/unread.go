package usercontrol

import (
	"time"
	"yf_pkg/redis"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/unread"
)

func RegUserUnread() (e error) {
	unread.Register(common.UNREAD_GIFT, unreadNum) //iunread.UnreadNum
	unread.Register(common.UNREAD_MYAWARD, unreadNum)
	unread.Register(common.UNREAD_MUMUID, unreadNum)
	unread.Register(common.UNREAD_BINDPHONE, unreadNum)
	unread.Register(common.UNREAD_WALLET, unreadNum)
	unread.Register(common.UNREAD_LOCALTAG, unreadNum)
	unread.Register(common.UNREAD_LOCALTAG_VIEWER, unreadNum)
	//	unread.Register(common.UNREAD_PROVID, unreadNum)

	return
}

//(u *UserUnread)
func unreadNum(uid uint32, key string, from time.Time) (result uint32, show string) {
	switch key {
	case common.UNREAD_GIFT:
		if err := mdb.QueryRow("select count(*) from gift_record where t_uid=? and status=0", uid).Scan(&result); err != nil {
			// fmt.Println("unread error " + err.Error())
			result = 0
		}
		// fmt.Println(fmt.Sprintf("unreadNum %v,count %v", from, result))
	case common.UNREAD_MYAWARD:
		if err := mdb.QueryRow("select count(*) from award_record where uid=? and status=0", uid).Scan(&result); err != nil {
			result = 0
		}
	case common.UNREAD_MUMUID, common.UNREAD_BINDPHONE:
		// fmt.Println(fmt.Sprintf("%v", from))
		if err := mdb.QueryRow("select count(*) from user_main where uid=? and reg_time>=?", uid, from).Scan(&result); err != nil {
			result = 0
		}
	case common.UNREAD_WALLET:
		if err := mdb.QueryRow("select count(*) from user_main where uid=? and reg_time>=?", uid, from).Scan(&result); err != nil {
			result = 0
		}
		if result > 0 {
			show = "充5000送400"
		}
	case common.UNREAD_LOCALTAG:
		if err := mdb.QueryRow("select count(*) from user_main where uid=? and reg_time>=?", uid, from).Scan(&result); err != nil {
			result = 0
		}
		if result > 0 {
			show = "[红点]"
		}
		/*	case common.UNREAD_PROVID: //我的供养
			if err := mdb.QueryRow("select count(*) from user_coin_log where uid=? and type=3 and forid>0 and create_time>=?", uid, from).Scan(&result); err != nil {
				result = 0
			}
		*/
	case common.UNREAD_LOCALTAG_VIEWER:
		con := rdb.GetReadConnection(redis_db.REDIS_LOCALTAG_VIEWERS)
		defer con.Close()
		result, _ = redis.Uint32(con.Do("ZCOUNT", uid, from.Unix(), "+inf"))
	}
	return
}
