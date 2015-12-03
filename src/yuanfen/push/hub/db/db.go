package db

import (
	"errors"
	"fmt"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yuanfen/redis_db"
)

const (
	MAX_TAG_MEMBERS = 300        //每个标签下的用户数上限
	TIMEOUT         = 10 * 86400 //redis里记录的用户所在的节点保存时间（秒）
)

type Node struct {
	PrivateAddress string
	PublicAddress  string
}

var mysqldb *mysql.MysqlDB
var redisdb *redis.RedisPool

func InitMysql(wdb string, rdbs []string) (e error) {
	mysqldb, e = mysql.New(wdb, rdbs)
	return e
}

func InitRedis(wdb string, rdbs []string, maxConn int) error {
	redisdb = redis.New(wdb, rdbs, maxConn)
	return nil
}

func GetNodePrivateAddr(uid uint32) (addr string, e error) {
	con := redisdb.GetReadConnection(redis_db.REDIS_USER_NODE)
	defer con.Close()
	addr, e = redis.String(con.Do("GET", uid))

	return
}

func SetUserNode(uid uint32, node string) (e error) {
	con := redisdb.GetWriteConnection(redis_db.REDIS_USER_NODE)
	defer con.Close()
	_, e = con.Do("SETEX", uid, TIMEOUT, node)
	return
}

func GetNodes() (nodes []Node, e error) {
	nodes = []Node{}
	sql := "select private_addr,public_addr from push_nodes order by private_addr"
	rows, e := mysqldb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var node Node
		if err := rows.Scan(&node.PrivateAddress, &node.PublicAddress); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func SaveMessages(from uint32, to []uint32, tag string, msgType string, content []byte) (msgid []uint64, e error) {
	sql := "insert into message(`from`,`to`,`tag`,`type`,`content`)values(?,?,?,?,?)"
	stmt, e := mysqldb.PrepareExec(sql)
	if e != nil {
		return
	}
	defer stmt.Close()
	msgid = make([]uint64, 0, len(to))
	for _, uid := range to {
		res, e := stmt.Exec(from, uid, tag, msgType, content)
		if e != nil {
			return msgid, e
		}
		lastId, e := res.LastInsertId()
		if e != nil {
			return msgid, e
		}
		msgid = append(msgid, uint64(lastId))
	}
	return
}

func SaveMessage(from uint32, to uint32, tag string, msgType string, content []byte) (msgid uint64, e error) {
	sql := "insert into message(`from`,`to`,`tag`,`type`,`content`)values(?,?,?,?,?)"
	stmt, e := mysqldb.PrepareExec(sql)
	if e != nil {
		return
	}
	defer stmt.Close()
	res, e := stmt.Exec(from, to, tag, msgType, content)
	if e != nil {
		return
	}
	lastId, e := res.LastInsertId()
	if e != nil {
		return
	}
	return uint64(lastId), nil
}

func SaveTagMessage(from uint32, tag string, msgType string, content []byte) (msgid uint64, e error) {
	/*
		sql := "insert into tag_message(`from`,`tag`,`type`,`content`)values(?,?,?,?)"
		stmt, e := mysqldb.PrepareExec(sql)
		if e != nil {
			return
		}
		defer stmt.Close()
		res, e := stmt.Exec(from, tag, msgType, content)
		if e != nil {
			return
		}
		lastId, e := res.LastInsertId()
		if e != nil {
			return
		}
		return uint64(lastId), nil
	*/
	return SaveMessage(from, 0, tag, msgType, content)
}

func InTag(uid uint32, tag string) (bool, error) {
	rds := redisdb.GetReadConnection(redis_db.REDIS_TAG_USERS)
	defer rds.Close()
	exists, e := redis.Bool(rds.Do("SISMEMBER", tag, uid))
	return exists, e
}

func AddTag(uid uint32, tag string) error {
	return redisdb.SMultiAdd(redis_db.REDIS_TAG_USERS, "t_"+tag, uid, fmt.Sprintf("u_%d", uid), tag)
}

func DelTag(uid uint32, tag string) error {
	return redisdb.SMultiRem(redis_db.REDIS_TAG_USERS, "t_"+tag, uid, fmt.Sprintf("u_%d", uid), tag)
}

func ClearTag(tag string) error {
	return errors.New("not implemented")
}

func GetUserTags(uid uint32) (tags []string, e error) {
	e = redisdb.SMembers(redis_db.REDIS_TAG_USERS, &tags, fmt.Sprintf("u_%d", uid))
	return
}
