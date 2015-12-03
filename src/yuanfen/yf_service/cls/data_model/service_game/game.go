package service_game

import (
	"errors"
	"strings"
	"yf_pkg/push"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/common/user"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"

	sys_message "yuanfen/yf_service/cls/message"
	_ "github.com/go-sql-driver/mysql"
)

// 获取游戏列表
func GetGameDataList() (gl []GameData, e error) {
	// 先读取cache数据
	exists, games, e := readGameDataCache()
	if exists && len(games) > 0 {
		return games, e
	}
	sql := "select id,name,info,img,package,url,size,class,appid,app_secret,isHot,isNew from  game_config where status = 1 "
	rows, e := mdb.Query(sql)
	if e != nil {
		return
	}
	defer rows.Close()
	gl = make([]GameData, 0, 5)
	for rows.Next() {
		var g GameData
		if e = rows.Scan(&g.Id, &g.Name, &g.Info, &g.Img, &g.Pack, &g.Url, &g.Size, &g.Class, &g.AppId, &g.Secret, &g.IsHot, &g.IsNew); e != nil {
			return
		}
		gl = append(gl, g)
	}
	// 写入cache
	e = writeGameDataCache(gl)
	return
}

/*
根据appid 获取游戏对象
*/
func GetGameByAppId(appid string) (g GameData, e error) {
	gl, e := GetGameDataList()
	if e != nil {
		return
	}
	if len(gl) <= 0 {
		return
	}
	for _, ga := range gl {
		if ga.AppId == appid {
			g = ga
			break
		}
	}
	return
}

/*
根据appId获取私钥
*/
func CheckAppValid(appid string) (isValid bool, g GameData, e error) {
	g, e = GetGameByAppId(appid)
	if e != nil {
		return
	}
	if g.AppId != "" && g.Secret != "" {
		return true, g, nil
	}
	return false, g, errors.New("appid is valid")
}

/*
根据appid获取游戏奖品配置
*/
func GetGameAwardConf(tx utils.SqlObj, appid string) (v []GameAwardConf, e error) {
	s := "select appid,award_id,num,balance from game_award_config where appid = ?"
	rows, e := tx.Query(s, appid)
	if e != nil {
		return
	}
	defer rows.Close()
	v = make([]GameAwardConf, 0, 10)
	for rows.Next() {
		var gc GameAwardConf
		if e = rows.Scan(&gc.AppId, &gc.AwardId, &gc.Num, &gc.Balance); e != nil {
			return
		}
		v = append(v, gc)
	}
	return
}

/////////////////////////大厅聊天/////////////////////////////////

// 退出游戏大厅， 删除用户tag
func ExitDeleteTag(uid uint32) (e error) {
	game_tags, e := getUserGameTags(uid)
	if e != nil {
		return
	}
	if len(game_tags) <= 0 {
		return
	}
	for _, tag := range game_tags {
		if e = push.DelTag(uid, tag); e != nil {
			return
		}
	}
	return
}

// 获取更具用户uid获取对应游戏聊天室tag
func GetUserGameTag(uid uint32, num int) (tag string, e error) {
	// 更具uid获取该用户所有的tag，并选出第最个tag
	game_tags, e := getUserGameTags(uid)
	if e != nil {
		return
	}
	if len(game_tags) > 0 {
		mlog.AppendObj(nil, "game tag is exist ", uid, game_tags)
		return game_tags[len(game_tags)-1], nil
	}
	// 无有效的tag，则此时，需要为用户添加tag标签
	// 获取当前所有聊天室总人数
	max_num := 500
	// 获取当前房间数
	n, e := getRoomBaseNum()
	if e != nil {
		return
	}
	new_n := n
	// 分配聊天室tag
	// 计算总人数占比
	ratio := float64(num) / float64(max_num*n) * 100
	mlog.AppendObj(errors.New(""), "query--base: ", uid, n, "ratio", ratio, "num:", num, "tags: ", game_tags)
	if ratio > 80 {
		new_n++
	} else if ratio < 40 {
		base_ratio := 70
		if (num*100)%(max_num*base_ratio) == 0 {
			new_n = num * 100 / (max_num * base_ratio)
		} else {
			new_n = num*100/(max_num*base_ratio) + 1
		}
	}
	if new_n <= 0 {
		new_n = 1
	}
	ns := utils.ToString(new_n)
	if new_n != n {
		if e = rdb.Set(redis_db.REDIS_GAME, redis_db.REDIS_GAME_KEY_TAGBASE, ns); e != nil {
			mlog.AppendObj(e, "do update base num is error", uid, new_n)
			return
		}
		mlog.AppendObj(e, "do update ", n, new_n)
	}
	// 根据uid取模，然后确定其tag标签
	tag = common.TAG_PREFIX_GAME + utils.ToString(uid%uint32(new_n))
	e = push.AddTag(uid, tag)
	mlog.AppendObj(e, "update--base: ", ns, "ratio", ratio, "num:", num, "new tag: ", tag)
	return
}

func GetGameTag(uid uint32) (tag string, e error) {
	n, e := getRoomBaseNum()
	if e != nil {
		return
	}
	tag = common.TAG_PREFIX_GAME + utils.ToString(uid%uint32(n))
	return
}

//*******************************private function**************************************/

//redis获取游戏平台聊天室数量
func getRoomBaseNum() (n int, e error) {
	exist, e := rdb.Exists(redis_db.REDIS_GAME, redis_db.REDIS_GAME_KEY_TAGBASE)
	if e != nil {
		return
	}
	if !exist {
		if e = rdb.Set(redis_db.REDIS_GAME, redis_db.REDIS_GAME_KEY_TAGBASE, "1"); e != nil {
			return
		}
		return 1, nil
	}
	b, e := redis.String(rdb.Get(redis_db.REDIS_GAME, redis_db.REDIS_GAME_KEY_TAGBASE))
	if e != nil {
		return
	}
	if b == "" {
		return 1, nil
	}
	n, e = utils.ToInt(b)
	return
}

// redis 获取用户聊天室tag
func getUserGameTags(uid uint32) (tags []string, e error) {
	all_tags, e := push.GetUserTags(uid)
	if e != nil {
		return
	}
	tags = make([]string, 0, 5)
	for _, v := range all_tags {
		if strings.Index(v, common.TAG_PREFIX_GAME) == 0 {
			tags = append(tags, v)
		}
	}
	return
}

//用户下线处理
func userOffline(msgid int, data interface{}) {
	switch v := data.(type) {
	case sys_message.Offline:
		if user.IsKfUser(v.Uid) {
			return
		}
		ExitDeleteTag(v.Uid)
	}
}
