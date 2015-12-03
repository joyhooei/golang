package relation

import (
	"errors"
	"fmt"
	"time"
	"yf_pkg/format"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/relation/base"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

func Follow(from uint32, to uint32) (e error) {
	exists, e := rdb.ZExists(redis_db.REDIS_FOLLOW, general.MakeKey("f", from), to)
	if e != nil {
		return e
	}
	if exists {
		//仅更新标记时间
		if _, e := mdb.Exec("update follow set tm=? where f_uid=? and t_uid=?", utils.Now, from, to); e != nil {
			return e
		}
		return rdb.ZAdd(redis_db.REDIS_FOLLOW, general.MakeKey("f", from), utils.Now.Unix(), to, general.MakeKey("t", to), utils.Now.Unix(), from)
	}
	//如果在黑名单中，从黑名单中删除
	if e = DelFromBlacklist(from, to); e != nil {
		return e
	}
	num, e := rdb.ZCard(redis_db.REDIS_FOLLOW, general.MakeKey("f", from))
	if e != nil {
		return e
	}
	if num > common.MAX_FOLLOW_NUM {
		return errors.New("too many follow")
	}
	friend := 0
	if isFriend, e := base.IsFriend(from, to); e != nil {
		return e
	} else {
		if isFriend {
			friend = 1
		}
	}
	sql := "insert into follow(f_uid,t_uid,friend)values(?,?,?)"
	if _, e = mdb.Exec(sql, from, to, friend); e != nil {
		return e
	}
	stat.Append(from, stat.ACTION_FOLLOW, map[string]interface{}{"target": to})
	return rdb.ZAdd(redis_db.REDIS_FOLLOW, general.MakeKey("f", from), utils.Now.Unix(), to, general.MakeKey("t", to), utils.Now.Unix(), from)
}

/*
UpdateFollowTag更新标记的类型。
*/
func UpdateFollowTag(from uint32, to uint32, tag uint16) (e error) {
	switch tag {
	case common.FOLLOW_TAG_NONE:
		return UnFollow(from, to)
	case common.FOLLOW_TAG_UNLIKE:
		//如果是标记为不喜欢，则添加到黑名单中
		if e = AddToBlacklist(from, to); e != nil {
			return e
		}
		return nil
	case common.FOLLOW_TAG_FOCUS:
		//要先检查是否已经特别关注，只有没有特别关注才会更新关注时间
		focused, e := rdb.ZExists(redis_db.REDIS_FOLLOW, general.MakeKey("sf", from), to)
		if e != nil {
			return service.NewError(service.ERR_INTERNAL, "ZExists error:"+e.Error(), "")
		}
		if !focused {
			if e = Follow(from, to); e != nil {
				return e
			}
			if _, e := mdb.Exec("update follow set focus=1,tm=? where f_uid=? and t_uid=?", utils.Now, from, to); e != nil {
				return e
			}
			if e = rdb.ZAdd(redis_db.REDIS_FOLLOW, general.MakeKey("sf", from), utils.Now.Unix(), to, general.MakeKey("st", to), utils.Now.Unix(), from); e != nil {
				return e
			}
		}
		return nil
	case common.FOLLOW_TAG_INTEREST:
		if e = Follow(from, to); e != nil {
			return e
		}
		if _, e := mdb.Exec("update follow set focus=0,tm=? where f_uid=? and t_uid=?", utils.Now, from, to); e != nil {
			return e
		}
		if _, e := rdb.ZRem(redis_db.REDIS_FOLLOW, general.MakeKey("sf", from), to, general.MakeKey("st", to), from); e != nil {
			return e
		}
		return nil
	default:
		return errors.New(fmt.Sprintf("unkown tag:%v", tag))
	}
}

/*
UnFollow取消标记。也要删除特别关注。
*/
func UnFollow(from uint32, to uint32) error {
	DelFromBlacklist(from, to)
	if _, e := mdb.Exec("delete from follow where f_uid=? and t_uid=?", from, to); e != nil {
		return e
	}
	_, e := rdb.ZRem(redis_db.REDIS_FOLLOW, general.MakeKey("sf", from), to, general.MakeKey("st", to), from, general.MakeKey("f", from), to, general.MakeKey("t", to), from)
	return e
}

/*
IsFollow查看from是否标记了to，并且返回标记的类型。

返回值：
	tag: 0-未标记，1-有好感，2-特别关注，3-不喜欢(拉黑)
*/
func IsFollow(from uint32, to uint32) (tag uint16, e error) {
	_, e = rdb.ZScore(redis_db.REDIS_FOLLOW, general.MakeKey("f", from), to)
	switch e {
	case nil:
		_, e2 := rdb.ZScore(redis_db.REDIS_FOLLOW, general.MakeKey("sf", from), to)
		switch e2 {
		case nil:
			return common.FOLLOW_TAG_FOCUS, nil
		case redis.ErrNil:
			return common.FOLLOW_TAG_INTEREST, nil
		default:
			return common.FOLLOW_TAG_NONE, e2
		}
	case redis.ErrNil:
		if bl, e := rdb.ZIsMember(redis_db.REDIS_BLACKLIST, from, to); e != nil {
			return common.FOLLOW_TAG_NONE, e
		} else if bl {
			return common.FOLLOW_TAG_UNLIKE, nil
		}
		return common.FOLLOW_TAG_NONE, nil
	default:
		return common.FOLLOW_TAG_NONE, e
	}
}

//FollowedNum返回用户被标记的数量，这里不区分标记的类型，不包括不喜欢这个类型。
func FollowedNum(uid uint32) (total uint64, e error) {
	return rdb.ZCard(redis_db.REDIS_FOLLOW, general.MakeKey("t", uid))
}

//FollowingNum返回用户标记的数量，这里不区分标记的类型，不包括不喜欢这个类型。
func FollowingNum(uid uint32) (total uint64, e error) {
	return rdb.ZCard(redis_db.REDIS_FOLLOW, general.MakeKey("f", uid))
}

//GetFocusUids获取特别关注的用户集合，结果集是一个map，方便查找。
func GetFocusUids(uid uint32) (users map[uint32]time.Time, e error) {
	k := general.MakeKey("sf", uid)
	items, total, e := rdb.ZREVRangeWithScores(redis_db.REDIS_FOLLOW, k, 0, -1)
	if e != nil {
		return nil, e
	}
	users = make(map[uint32]time.Time, total)
	for _, item := range items {
		u, e := utils.ToUint32(item.Key)
		if e != nil {
			return nil, e
		}
		users[u] = time.Unix(int64(item.Score), 0)
	}
	return
}

//GetFollowTags获取某用户标记的用户的标签类型，不包括不喜欢（黑名单）
func GetFollowTags(me uint32, targets []uint32) (tags map[uint32]uint16, e error) {
	tags = make(map[uint32]uint16, len(targets))
	interests := make(map[interface{}]bool, len(targets))
	focus := make(map[interface{}]bool, len(targets))
	for _, uid := range targets {
		interests[uid] = false
		focus[uid] = false
		tags[uid] = common.FOLLOW_TAG_NONE
	}
	if e = rdb.ZMultiIsMember(redis_db.REDIS_FOLLOW, general.MakeKey("f", me), interests); e != nil {
		return nil, e
	}
	if e = rdb.ZMultiIsMember(redis_db.REDIS_FOLLOW, general.MakeKey("sf", me), focus); e != nil {
		return nil, e
	}
	for uid, _ := range tags {
		if focus[uid] {
			tags[uid] = common.FOLLOW_TAG_FOCUS
		} else if interests[uid] {
			tags[uid] = common.FOLLOW_TAG_INTEREST
		}
	}
	return
}

func GetFollowUids(isFollowing bool, uid uint32, cur int, ps int) (users []uint32, total int, e error) {
	return nil, 0, errors.New("deprecated")
}

//分页获取标记的用户信息，好友排在前面，不包括不喜欢的用户
func GetFollowUsers(uid uint32, cur, ps int) (users []User, total int, e error) {
	if e = mdb.QueryRow("select count(*) from follow where f_uid=?", uid).Scan(&total); e != nil {
		return nil, 0, e
	}
	sql := "select t_uid,friend,focus,tm from follow where f_uid=? order by friend desc,tm desc" + utils.BuildLimit(cur, ps)
	rows, e := mdb.Query(sql, uid)
	if e != nil {
		return nil, 0, e
	}
	users = make([]User, 0, ps)
	for rows.Next() {
		var user User
		var isFriend, isFocus int
		var tmStr string
		if e := rows.Scan(&user.Uid, &isFriend, &isFocus, &tmStr); e != nil {
			return nil, 0, e
		}
		user.Tm, _ = utils.ToTime(tmStr, format.TIME_LAYOUT_1)
		user.IsFriend = (isFriend == 1)
		if isFocus == 1 {
			user.Tag = common.FOLLOW_TAG_FOCUS
		} else {
			user.Tag = common.FOLLOW_TAG_INTEREST
		}
		users = append(users, user)
	}
	if e = makeFollowUsersInfo(users); e != nil {
		return nil, 0, e
	}
	return
}

//分页获取不喜欢（黑名单中）的用户列表
func GetBlacklist(uid uint32, cur, ps int) (users []BlackUser, total int, e error) {
	items, total, e := rdb.ZREVRangeWithScoresPS(redis_db.REDIS_BLACKLIST, uid, cur, ps)
	if e != nil {
		return nil, 0, e
	}
	if users, e = makeBlacklistUsersInfo(items); e != nil {
		return nil, 0, e
	}
	return
}

//添加到黑名单（不喜欢）
func AddToBlacklist(uid uint32, badUser uint32) (e error) {
	//从标记列表中删除
	if e = UnFollow(uid, badUser); e != nil {
		return
	}
	//删除好友关系
	if e = DelFriend(uid, badUser); e != nil {
		return
	}
	//删除认识一下请求
	if _, e = rdb.ZRem(redis_db.REDIS_SAYHELLO, general.MakeKey(common.SAYHELLO_TARGET_ME, uid), badUser); e != nil {
		return e
	}
	if e = rdb.ZAdd(redis_db.REDIS_BLACKLIST, uid, utils.Now.Unix(), badUser); e != nil {
		return
	}
	return nil
}

//从黑名单移除（不喜欢）
func DelFromBlacklist(uid uint32, badUser uint32) (e error) {
	_, e = rdb.ZRem(redis_db.REDIS_BLACKLIST, uid, badUser)
	return e
}

//---------------------Private Functions-----------------------//

func makeFollowUsersInfo(users []User) (e error) {
	//取出当前请求的uid集合
	uids := make([]uint32, 0, len(users))
	for _, u := range users {
		uids = append(uids, u.Uid)
	}
	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return e
	}
	for i, uid := range uids {
		if ui := uinfos[uid]; ui != nil {
			users[i].Nickname = ui.Nickname
			users[i].Avatar = ui.Avatar
		}
	}

	return nil
}
func makeBlacklistUsersInfo(items []redis.ItemScore) (users []BlackUser, e error) {
	//取出当前请求的uid集合
	uids := make([]uint32, 0, len(items))
	for _, u := range items {
		if uid, e := utils.ToUint32(u.Key); e != nil {
			return nil, e
		} else {
			uids = append(uids, uid)
		}
	}
	uinfos, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return nil, e
	}
	users = make([]BlackUser, 0, len(uids))
	for i, item := range items {
		if ui := uinfos[uids[i]]; ui != nil {
			users = append(users, BlackUser{uids[i], ui.Nickname, ui.Avatar, time.Unix(int64(item.Score), 0)})
		}
	}

	return users, nil
}
