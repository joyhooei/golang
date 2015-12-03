package topic

import (
	"yf_pkg/cachedb"
	"yf_pkg/mysql"
)

type TopicDBObject map[string]interface{}

func (u *TopicDBObject) ID() (id interface{}, ok bool) {
	id, ok = (*u)["id"]
	return
}

func (u *TopicDBObject) Save(mysqldb *mysql.MysqlDB) (id interface{}, e error) {
	id, ok := (*u)["id"]
	if ok {
		sql := "update topic set uid=?,title=?,capacity=?,tag=?,tm=?,status=?,pics=?,picslevel=? where id=?"
		_, e := mysqldb.Exec(sql, (*u)["uid"], (*u)["title"], (*u)["capacity"], (*u)["tag"], (*u)["tm"], (*u)["status"], (*u)["pics"], (*u)["picslevel"], id)
		if e != nil {
			return id, e
		}
		return id, nil
	} else {
		sql := "insert into topic(uid,title,capacity,tag,tm,status,pics,picslevel)values(?,?,?,?,?,?,?,?)"
		res, e := mysqldb.Exec(sql, (*u)["uid"], (*u)["title"], (*u)["capacity"], (*u)["tag"], (*u)["tm"], (*u)["status"], (*u)["pics"], (*u)["picslevel"])
		if e != nil {
			return id, e
		}
		return res.LastInsertId()
	}
}

func NewTopicDBObject(id interface{}) cachedb.DBObject {
	topic := &TopicDBObject{"id": id}
	return topic
}

func (u *TopicDBObject) Get(id interface{}, mysqldb *mysql.MysqlDB) (e error) {
	sql := "select uid,title,capacity,tag,tm,status,pics,picslevel from topic where id = ?"
	var uid, capacity uint32
	var status uint8
	var picslevel int8
	var title, tag, pics, tm string
	e = mysqldb.QueryRow(sql, id).Scan(&uid, &title, &capacity, &tag, &tm, &status, &pics, &picslevel)
	(*u)["id"] = id
	(*u)["uid"] = uid
	(*u)["title"] = title
	(*u)["capacity"] = capacity
	(*u)["tag"] = tag
	(*u)["tm"] = tm
	(*u)["status"] = status
	(*u)["pics"] = pics
	(*u)["picslevel"] = picslevel
	return
}
func (u *TopicDBObject) GetMap(ids []interface{}, mysqldb *mysql.MysqlDB) (objs map[interface{}]cachedb.DBObject, e error) {
	sql := "select id,uid,title,capacity,tag,tm,status,pics,picslevel from topic where id" + mysql.In(ids)
	var id, uid, capacity uint32
	var status uint8
	var picslevel int8
	var title, pics, tag, tm string
	rows, e := mysqldb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	objs = make(map[interface{}]cachedb.DBObject)
	for rows.Next() {
		if e = rows.Scan(&id, &uid, &title, &capacity, &tag, &tm, &status, &pics, &picslevel); e != nil {
			return nil, e
		}
		t := NewTopicDBObject(id).(*TopicDBObject)
		(*t)["id"] = id
		(*t)["uid"] = uid
		(*t)["title"] = title
		(*t)["capacity"] = capacity
		(*t)["tag"] = tag
		(*t)["tm"] = tm
		(*t)["status"] = status
		(*t)["pics"] = pics
		(*t)["picslevel"] = picslevel
		objs[id] = t
	}
	return
}

func (u *TopicDBObject) Expire() int {
	return 400
}
