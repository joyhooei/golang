package comments

import (
	"database/sql"
	"errors"
	"yf_pkg/mysql"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

/*
获取某人在某条动态上的相关评论
source_id: 资源id
source_uid: 资源用户uid
uid: 涉及用户
source_type:资源类型，1，动态评论 2，扩展
stm: 查看开始时间，时间倒序，
id : 开始id
ps: 每次查询条数
isAll: 是否查询全部资源下的评论 (当为TRUE 时，忽略参数uid)
exId: 需要过滤掉的评论ID exId
*/
func GetCommentByIdAndUid(source_id, source_uid, uid uint32, source_type int, stm string, id uint32, ps int, is_all bool, exId uint32) (r []Comment, e error) {
	s := "select id,uid,source_id,type,ruid,content,status,tm from comment where source_id = ? and source_type =? and type !=1 and id != " + utils.ToString(exId)
	if !is_all { // 非查询全部，自己的和群主的...
		s += " and (uid = ? or (uid = ? and (ruid = 0 or ruid = ?) )) "
	}
	s += " and tm <= ? and id < ? and status = 0  order by tm desc,id desc limit ?"
	rows := new(sql.Rows)
	if !is_all {
		rows, e = mdb.Query(s, source_id, source_type, uid, source_uid, uid, stm, id, ps)
	} else {
		rows, e = mdb.Query(s, source_id, source_type, stm, id, ps)
	}
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var c Comment
		if e = rows.Scan(&c.Id, &c.Uid, &c.SourceId, &c.Type, &c.Ruid, &c.Content, &c.Status, &c.Tm); e != nil {
			return
		}
		r = append(r, c)
	}
	return
}

/*
获取拼图游戏首位用户
source_id 对应动态id ，获取第一名
*/
func GetPuzzleWinComment(source_id uint32) (c Comment, e error) {
	s := "select id,uid,source_id,type,ruid,content,status,tm  from comment where source_id =? and type =3 order by CONVERT(content,SIGNED) asc limit 1"
	rows, e := mdb.Query(s, source_id)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&c.Id, &c.Uid, &c.SourceId, &c.Type, &c.Ruid, &c.Content, &c.Status, &c.Tm); e != nil {
			return
		}
	}
	return
}

/*
获取某资源中的点赞用户
*/
func GetLikesById(source_id uint32) (uids []uint32, e error) {
	s := "select distinct(uid) from comment where source_id = ? and type = 1 and status = 0 order by tm desc"
	rows, e := mdb.Query(s, source_id)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var uid uint32
		if e = rows.Scan(&uid); e != nil {
			return
		}
		uids = append(uids, uid)
	}
	return
}

/*
拼接用户信息
*/
func GenCommentInfo(v []Comment, uid uint32) (res []map[string]interface{}, e error) {
	mlog.AppendObj(nil, "--v---", v)
	res = make([]map[string]interface{}, 0, len(v))
	if len(v) <= 0 {
		return
	}
	uids := make([]uint32, 0, len(v))
	for _, d := range v {
		uids = append(uids, d.Uid)
		if d.Ruid > 0 {
			uids = append(uids, d.Ruid)
		}
	}
	// 查询用户信
	m, e := user_overview.GetUserObjects(uids...)
	if e != nil {
		return
	}
	um, e := user_overview.CheckUserDynamicStatus(uids)
	if e != nil {
		return
	}
	mlog.AppendObj(nil, "--v---", um)
	um[uid] = true
	// 查询评论和点赞数
	for _, d := range v {
		if b, ok := um[d.Uid]; !ok || !b {
			mlog.AppendObj(nil, "--v---", d)
			continue
		}
		r := make(map[string]interface{})
		c := make(map[string]interface{})
		c["id"] = d.Id
		c["uid"] = d.Uid
		c["ruid"] = d.Ruid
		c["content"] = d.Content
		c["source_id"] = d.SourceId
		c["type"] = d.Type
		tm, er := utils.ToTime(d.Tm)
		if er != nil {
			mlog.AppendObj(er, "GenCommentInfo is error", d)
			continue
		}
		c["tm"] = tm

		u := make(map[string]interface{})
		user, ok := m[d.Uid]
		if !ok || user == nil {
			mlog.AppendObj(errors.New("get user is error"), " GenCommentInfo get user is error")
			continue
		}
		u["uid"] = user.Uid
		u["age"] = user.Age
		u["nickname"] = user.Nickname
		u["avatar"] = user.Avatar
		u["job"] = user.Job

		reu := make(map[string]interface{})

		ru := new(user_overview.UserViewItem)
		if d.Ruid > 0 {
			ruser, ok := m[d.Ruid]
			if !ok || user == nil {
				mlog.AppendObj(errors.New("get user is error"), " GenCommentInfo get user is error")
				continue
			}
			ru = ruser
		}
		reu["age"] = ru.Age
		reu["nickname"] = ru.Nickname
		reu["avatar"] = ru.Avatar
		reu["job"] = "UI设计师"

		r["comment"] = c
		r["user"] = u
		r["ruser"] = reu
		res = append(res, r)
	}
	return

}

/*
获取id 获取某条评论
*/
func GetCommentById(tx utils.SqlObj, id uint32) (c Comment, e error) {
	s := "select id,uid,source_id,type,ruid,content,status,tm from comment where id = ? "
	rows, e := tx.Query(s, id)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&c.Id, &c.Uid, &c.SourceId, &c.Type, &c.Ruid, &c.Content, &c.Status, &c.Tm); e != nil {
			return
		}
	}
	return
}

/*
获取某用户对某条资源的点赞评论
*/
func GetLikeComment(id, uid uint32) (c Comment, e error) {
	s := "select id,uid,source_id,type,ruid,content,status,tm from comment where source_id = ? and uid =? and type = 1 limit 1 "
	rows, e := mdb.Query(s, id, uid)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&c.Id, &c.Uid, &c.SourceId, &c.Type, &c.Ruid, &c.Content, &c.Status, &c.Tm); e != nil {
			return
		}
	}
	return
}

/*
获取某用户对某条资源的点赞评论
*/
func GetLikeComentInIds(ids []uint32, uid uint32) (m map[uint32]int, e error) {
	if len(ids) <= 0 {
		return
	}
	s := "select source_id from comment where source_id in (" + utils.Uint32ArrTostring(ids) + ")  and uid =? and type = 1 and status = 0"
	rows, e := mdb.Query(s, uid)
	if e != nil {
		return
	}
	defer rows.Close()
	m = make(map[uint32]int)
	for rows.Next() {
		var source_id uint32
		if e = rows.Scan(&source_id); e != nil {
			return
		}
		m[source_id] = 1
	}
	return
}

/*
添加评论
uid:用户 source_id 资源id ruid 回复uid， source_type 1 动态 t : 评论类型    1. 点赞   2. 评论 ，3.拼图游戏时间
*/
func AddComment(tx utils.SqlObj, uid, source_id, ruid uint32, source_type, t int, content string) (id uint32, e error) {
	s := "insert into comment(uid,source_id,source_type,type,ruid,content) values(?,?,?,?,?,?)"
	rs, e := tx.Exec(s, uid, source_id, source_type, t, ruid, content)
	if e != nil {
		return
	}
	if i, e := rs.LastInsertId(); e == nil {
		id = uint32(i)
	}
	return
}

/*
根据id，修改评论status字段
*/
func UpdateCommentStatus(tx utils.SqlObj, id uint32, status int) (e error) {
	s := "update comment set status = ? where id = ?"
	_, e = tx.Exec(s, status, id)
	return
}

/*
根据id，修改评论flag字段
*/
func UpdateCommentFlag(tx utils.SqlObj, id uint32, status int) (e error) {
	s := "update comment set flag = ? where id = ?"
	_, e = tx.Exec(s, status, id)
	return
}

/*
获取某某资源的全部点赞用户
*/
func GetLikeUsers(id uint32) (uids []uint32, e error) {
	s := "select uid from comment where source_id =? and type = 1 and status = 0 order by tm desc"
	rows, e := mdb.Query(s, id)
	if e != nil {
		return
	}
	defer rows.Close()
	uids = make([]uint32, 0, 10)
	for rows.Next() {
		var uid uint32
		if e = rows.Scan(&uid); e != nil {
			return
		}
		if b, e := user_overview.CheckUserDynamicStatusByUid(uid); e != nil || !b {
			continue
		}
		uids = append(uids, uid)
	}
	return
}

/*
获取某用户发布动态下的评论
t: source_type
ids: 评论id
*/
func GetUserCommentsByIds(t int, ids []uint32) (r []Comment, e error) {
	if len(ids) <= 0 {
		return
	}
	s := "select id,uid,source_id,type,ruid,content,status,tm from comment where id " + mysql.In(ids) + " and source_type = ? and status = 0 order by tm desc,id desc "
	rows, e := mdb.Query(s, t)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var c Comment
		if e = rows.Scan(&c.Id, &c.Uid, &c.SourceId, &c.Type, &c.Ruid, &c.Content, &c.Status, &c.Tm); e != nil {
			return
		}
		r = append(r, c)
	}
	return
}
