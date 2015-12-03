package general

import (
	"errors"
	"yf_pkg/cachedb"
	"yf_pkg/mysql"
	"yf_pkg/utils"
)

type UserProtect struct {
	Uid               uint32 //uid
	DoNotFindMe       int    //允许附近的人找到我,1表示不允许,0表示允许,0为默认值
	ChatNotNotify     int    //私聊提醒,1表示不允许,0表示允许,0为默认值
	SayHelloNotNotify int    //陌生人新消息提醒,1表示不允许,0表示允许,0为默认值
	LikeNotNotify     int    //被点赞提醒,1表示不允许,0表示允许,0为默认值
	CommentNotNotify  int    //被评论提醒,1表示不允许,0表示允许,0为默认值
	MsgNotRing        int    //消息无提示音,0为有提示音,1为关闭提示音(默认值)
	MsgNotShake       int    //消息无震动,0为有震动(默认值),1为关闭震动
	NightRing         int    //晚上是否响铃震动,0半夜不响铃(默认值),1为半夜仍响铃
}

//获取用户防骚扰配置
func GetUserProtect(uid uint32) (result *UserProtect, e error) {
	result = &UserProtect{}
	e = cachedb2.Get(uid, result)
	return
}

//批量获取用户防骚扰配置
func GetUserProtects(uidlist ...uint32) (obj map[uint32]*UserProtect, e error) {
	if len(uidlist) == 0 {
		return
	}
	users := make(map[interface{}]cachedb.DBObject)
	for _, v := range uidlist {
		users[utils.Uint32ToString(v)] = new(UserProtect)
	}
	e = cachedb2.GetMap(users, NewUserProtect)
	obj1 := make(map[uint32]*UserProtect)
	if e != nil {
		return nil, e
	} else {
		for id, user := range users {
			uid, e := utils.ToUint32(id)
			if e != nil {
				return nil, e
			}
			// fmt.Println(fmt.Printf("GetUserProtectsr %v ,%v", id, user))
			if user != nil {
				obj1[uid] = user.(*UserProtect)
			}
		}
	}
	return obj1, nil
}

//清除用户防骚扰配置
func ClearUserProtect(uid uint32) (e error) {
	return cachedb2.ClearCache(NewUserProtect(uid))
}

func NewUserProtect(uid interface{}) cachedb.DBObject {
	user := &UserProtect{}
	switch v := uid.(type) {
	case uint32:
		user.Uid = v
	}
	return user
}

func (u *UserProtect) ID() (id interface{}, ok bool) {
	return u.Uid, u.Uid != 0
}

func (u *UserProtect) Save(mysqldb *mysql.MysqlDB) (id interface{}, e error) {
	return nil, errors.New("not implemented")
}

func (u *UserProtect) Get(id interface{}, mysqldb *mysql.MysqlDB) (e error) {
	switch v := id.(type) {
	case uint32:
		u.Uid = v
	}
	e = mdb.QueryRow("select canfind,chatremind,stranger,praise,commit,msgnotring,msgnotshake,nightring from user_protect where uid=?", id).Scan(&u.DoNotFindMe, &u.ChatNotNotify, &u.SayHelloNotNotify, &u.LikeNotNotify, &u.CommentNotNotify, &u.MsgNotRing, &u.MsgNotShake, &u.NightRing)
	if e != nil {
		if e.Error() == "sql: no rows in result set" { //没查到取默认值
			return nil
		}
		return e
	}
	return nil
}

func (u *UserProtect) GetMap(ids []interface{}, mysqldb *mysql.MysqlDB) (objs map[interface{}]cachedb.DBObject, e error) {
	in := mysql.In(ids)
	sql := "select uid,canfind,chatremind,stranger,praise,commit,msgnotring,msgnotshake,nightring from user_protect where uid" + in
	rows, e := mdb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()

	obj := make(map[interface{}]cachedb.DBObject)
	for rows.Next() {
		var user UserProtect
		if e = rows.Scan(&user.Uid, &user.DoNotFindMe, &user.ChatNotNotify, &user.SayHelloNotNotify, &user.LikeNotNotify, &user.CommentNotNotify, &user.MsgNotRing, &user.MsgNotShake, &user.NightRing); e != nil {
			return nil, e
		}
		obj[user.Uid] = &user
	}
	return obj, nil
}

func (u *UserProtect) Expire() int {
	return 600
}
