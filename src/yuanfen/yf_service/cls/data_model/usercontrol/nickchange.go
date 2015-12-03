package usercontrol

import (
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
)

//根据注册时间判断 是否要自动回复
func GetNickChangedUsers(tm int64, ids []uint32) (rids []uint32, e error) {
	rids = make([]uint32, 0, 10)
	if len(ids) <= 0 {
		rids = ids
		return
	}
	keys := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, id)
	}
	arr, e := redis.Strings(rdb.MGet(redis_db.REDIS_CHANGE_NICKAVATAR, keys...))
	if e != nil {
		return
	}
	for i, v := range arr {
		if v == "" {
			continue
		}
		iv, e := utils.ToInt64(v)
		if e != nil {
			return nil, e
		}
		if iv >= tm {

			rids = append(rids, ids[i])
		}
	}
	return
}

func UpdateNickChange(uid uint32) (e error) {
	e = rdb.Set(redis_db.REDIS_CHANGE_NICKAVATAR, uid, utils.Now.Unix())
	return
}
