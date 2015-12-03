package db

import (
	"fmt"
	"time"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/push/pusher/common"
	"yuanfen/redis_db"
)

var mysqldb *mysql.MysqlDB
var onlinedb *mysql.MysqlDB
var redisdb *redis.RedisPool

const DELETE_USER_AFTER_DAYS = 30 //离线多久后从user_online删除

func Init(conf *common.Config) (e error) {
	fmt.Println("init main db...")
	if mysqldb, e = mysql.New(conf.Mysql.Main.Master, conf.Mysql.Main.Slave); e != nil {
		return e
	}
	fmt.Println("success")
	fmt.Println("init online db...")
	if onlinedb, e = mysql.New(conf.Mysql.Online.Master, conf.Mysql.Online.Slave); e != nil {
		return e
	}
	fmt.Println("success")
	fmt.Println("init redis...")
	redisdb = redis.New(conf.Redis.Main.Master.String(), conf.Redis.Main.Slave.StringSlice(), conf.Redis.Main.MaxConn)
	fmt.Println("success")
	go deleteInactiveUsers()
	return nil
}

func GetUserTags(uid uint32) (tags map[string]bool, e error) {
	t := []string{}
	if e = redisdb.SMembers(redis_db.REDIS_TAG_USERS, &t, fmt.Sprintf("u_%d", uid)); e != nil {
		return nil, e
	}
	tags = map[string]bool{}
	for _, tag := range t {
		tags[tag] = true
	}
	return tags, nil
}

func GetNotificationUrls() ([]string, error) {
	sql := "select * from push_notification"
	rows, e := mysqldb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	urls := make([]string, 0)
	for rows.Next() {
		var url string
		e := rows.Scan(&url)
		if e != nil {
			return nil, e
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func UserOffline(uid uint32) error {
	sql := "update user_online set  `tm`=? where uid=?"
	_, e := onlinedb.Exec(sql, utils.Now.Add(-3*time.Second), uid)
	return e
}

func UpdateOnline(uid uint32) error {
	t := utils.Now.Add(40 * time.Minute)
	sql := "insert into user_online(uid,tm)values(?,?) on duplicate key update `tm`=?"
	_, e := onlinedb.Exec(sql, uid, t, t)
	return e
}

func deleteInactiveUsers() {
	sql := "delete from user_online where tm<?"
	for {
		fmt.Println("delete deleteInactiveUsers")
		time.Sleep(10 * time.Minute)
		_, e := onlinedb.Exec(sql, utils.Now.AddDate(0, 0, -DELETE_USER_AFTER_DAYS))
		if e != nil {
			fmt.Println("deleteInactiveUsers error :", e.Error())
		}
	}
}
