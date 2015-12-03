package relation

import (
	"time"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
)

// 加幸运未读消息
func UnreadNum(uid uint32, k string, from time.Time) (total uint32, show string) {
	switch k {
	case common.UNREAD_FANS:
		total, _ = rdb.ZCount(redis_db.REDIS_FOLLOW, general.MakeKey("t", uid), float64(from.Unix()), float64(utils.Now.Unix()))
	case common.UNREAD_SAYHELLO:
		total, _ = rdb.ZCount(redis_db.REDIS_SAYHELLO, general.MakeKey(common.SAYHELLO_TARGET_ME, uid), float64(from.Unix()), float64(utils.Now.Unix()))
	}
	return total, ""
}
