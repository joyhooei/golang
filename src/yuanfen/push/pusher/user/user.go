package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"yf_pkg/log"
	"yf_pkg/net"
	"yf_pkg/service"
	"yf_pkg/thread_safe/safe_map"
	"yf_pkg/utils"
	"yuanfen/push/pusher/common"
	"yuanfen/push/pusher/db"
	"yuanfen/push/pusher/msg"
	"yuanfen/push/pusher/notifier"
)

const (
	KEY_TIMEOUT               = 30  //登陆用的秘钥失效时间（秒）
	CLIENT_FIRST_READ_TIMEOUT = 10  //客户端连接后第一条消息的超时时间（秒）
	CLIENT_TIMEOUT            = 130 //客户端超时时间（秒）
	MESSAGE_BUFFER_SIZE       = 100 //每个用户消息队列的最大长度
)

const (
	MIN_DISTENCE   = 500  //需要更新坐标的最小距离(米)
	ONLINE_TIMEOUT = 1200 //用户在线状态超时时间(秒)
)

var tagUserMap *TagUserMap
var users *safe_map.SafeMap
var logger *log.Logger

type User struct {
	Uid  uint32
	Msgs chan *msg.Message
	Tags map[string]bool
	conn *net.TCPConn
}

func TO_USER_POINT(v interface{}, ok bool) (user *User, found bool) {
	if ok {
		switch u := v.(type) {
		case *User:
			return u, ok
		default:
			return nil, false
		}
	} else {
		return nil, false
	}
}

func init() {
	users = safe_map.New(1000)
	tagUserMap = NewTagUserMap(100)
	logger, _ = log.New("/dev/null", 10000, log.ERROR)
	go logStatus()
}

var heartBeatCount, writeCount int

func logStatus() {
	for {
		d := fmt.Sprintf("users: %v | heartbeat routines: %v | write routines: %v", users.Len(), heartBeatCount, writeCount)
		logger.Append(d, log.NOTICE)
		fmt.Println(d)
		time.Sleep(10 * time.Second)
	}
}

func sub(d *int) {
	(*d)--
}
func (u *User) readHeartBeat() {
	heartBeatCount++
	defer sub(&heartBeatCount)

	var lastOnlineUpdateTime, lastLocationUpdateTime int64 = 0, 0
	var lastCoordinate utils.Coordinate

	for {
		if utils.Now.Unix()-lastOnlineUpdateTime > ONLINE_TIMEOUT/2 {
			err := db.UpdateOnline(u.Uid)
			if err == nil {
				lastOnlineUpdateTime = utils.Now.Unix()
			} else {
				logger.Append(fmt.Sprintf("UpdateOnline to mysql of user %v error: %v", u.Uid, err.Error()), log.DEBUG)
			}
		}
		msg, err := msg.ReadMessage(u.conn)
		if err != nil {
			u.DelMe()
			logger.Append(fmt.Sprintf("user %v read HEART-BEAT error : %v", u.Uid, err.Error()), log.DEBUG)
			break
		}
		var c utils.Coordinate
		e := json.Unmarshal(msg.Content(), &c)
		if e != nil {
			continue
		}
		if utils.Distence(c, lastCoordinate) > MIN_DISTENCE || utils.Now.Unix()-lastLocationUpdateTime > ONLINE_TIMEOUT/2 {
			notifier.NotifyLocation(u.Uid, c.Lat, c.Lng)
			lastLocationUpdateTime = utils.Now.Unix()
			lastCoordinate = c
		}
	}

	logger.Append(fmt.Sprintf("user %v HeartBeat routine stop", u.Uid), log.DEBUG)
}

func (u *User) writeMessages() {
	writeCount++
	defer sub(&writeCount)
	for msg := range u.Msgs {
		if msg == nil {
			break
		}
		e := msg.Send(u.conn)
		if e != nil {
			u.DelMe()
			logger.Append(fmt.Sprintf("fail send msg [%v->%v] : %v (%v)", msg.Sender(), u.Uid, msg.String(), e.Error()), log.ERROR)
		} else {
			logger.Append(fmt.Sprintf("success send msg [%v->%v] : %v", msg.Sender(), u.Uid, msg.String()), log.NOTICE)
		}
	}
	logger.Append(fmt.Sprintf("user %v writeMessages routine stop", u.Uid), log.DEBUG)
}

func (u *User) close() {
	start := time.Now()
	defer utils.PrintDuration("close", start, time.Second)
	logger.Append(fmt.Sprintf("user %v offline", u.Uid), log.NOTICE)
	tagUserMap.DelUserTags(u.Uid, u.Tags)
	u.Msgs <- nil
	logger.Append(fmt.Sprintf("user %v put nil to channel. len(u.Msgs)=%v", u.Uid, len(u.Msgs)), log.DEBUG)
	e := u.conn.Close()
	if e != nil {
		logger.Append(fmt.Sprintf("close connection of user %v error : %v", u.Uid, e.Error()), log.ERROR)
	}
	logger.Append(fmt.Sprintf("user %v close connection", u.Uid), log.DEBUG)
	//通知订阅者该用户掉线
	notifier.NotifyOffline(u.Uid)
	//logger.Append(fmt.Sprintf("user %v notify", u.Uid), log.DEBUG)
	//更新mysql用户在线状态为离线
	e = db.UserOffline(u.Uid)
	if e != nil {
		logger.Append(fmt.Sprintf("delete user %v from user_online error : %v", u.Uid, e.Error()), log.ERROR)
	}
	logger.Append(fmt.Sprintf("user %v offline", u.Uid), log.DEBUG)
}

func validate(conn *net.TCPConn) (uint32, bool, string) {
	conn.SetTimeout(CLIENT_FIRST_READ_TIMEOUT)
	msg, err := msg.ReadMessage(conn)
	if err != nil {
		logger.Append(fmt.Sprintf("user %v validate failed : %v", 0, err.Error()), log.DEBUG)
		return 0, false, fmt.Sprintf("read login message error : %v", err.Error())
	}
	sender := msg.Sender()
	//fmt.Printf("check user %v key=%v\n", sender, msg.Key())
	isValid, reason := IsValid(sender, msg.Key())
	DelKey(sender)
	if isValid == false {
		logger.Append(fmt.Sprintf("user %v validate failed : %v", sender, reason), log.DEBUG)
		return sender, false, reason
	}
	conn.SetTimeout(CLIENT_TIMEOUT)
	return sender, true, ""
}

func OnlineUsers() int {
	start := time.Now()
	defer utils.PrintDuration("OnlineUsers", start, time.Second)

	return users.Len()
}

//删除某个用户，同时关闭连接
func DelUser(uid uint32) {

	start := time.Now()
	defer utils.PrintDuration(fmt.Sprintf("DelUser %v", uid), start, time.Second)

	user, ok := TO_USER_POINT(users.Get(uid))
	if ok {
		users.Del(uid)
		user.close()
	}
	logger.Append(fmt.Sprintf("user %v leave DelUser", uid), log.DEBUG)
}

func (u *User) DelMe() {
	start := time.Now()
	defer utils.PrintDuration(fmt.Sprintf("DelMe %v", u.Uid), start, time.Second)

	user, ok := TO_USER_POINT(users.Get(u.Uid))
	if ok && user == u {
		users.Del(u.Uid)
		user.close()
	}
	logger.Append(fmt.Sprintf("user %v leave DelMe", u.Uid), log.DEBUG)
}
func AddUser(conn *net.TCPConn) {
	uid, valid, reason := validate(conn)
	if valid {
		DelUser(uid) //删除可能存在的上一个连接
		tags, e := db.GetUserTags(uid)
		if e != nil {
			logger.Append(fmt.Sprintf("GetUserTags(%v) failed:%v", uid, e.Error()), log.ERROR)
			j, err := json.Marshal(map[string]interface{}{"status": "fail", "ecode": service.ERR_INTERNAL, "edesc": "内部错误"})
			if err == nil {
				m := msg.New(common.USER_MSG, 0, 0, j, "")
				m.Send(conn)
				time.Sleep(1 * time.Second)
			}
			conn.Close()
			return
		}
		user := &User{uid, make(chan *msg.Message, MESSAGE_BUFFER_SIZE), tags, conn}
		tagUserMap.AddUserTags(uid, tags)
		logger.Append(fmt.Sprintf("user %v online", uid), log.NOTICE)
		users.Set(uid, user)
		logger.Append(fmt.Sprintf("user %v added to map", uid), log.DEBUG)
		m := msg.New(common.USER_MSG, 0, 0, []byte(`{"status":"ok"}`), "")
		user.Msgs <- m
		go user.readHeartBeat()
		go user.writeMessages()
	} else {
		logger.Append(fmt.Sprintf("user %v invalid", uid), log.NOTICE)
		j, e := json.Marshal(map[string]interface{}{"status": "fail", "ecode": service.ERR_INVALID_USER, "edesc": reason})
		if e == nil {
			m := msg.New(common.USER_MSG, 0, 0, j, "")
			m.Send(conn)
			time.Sleep(1 * time.Second)
		}
		conn.Close()
	}
}

func SendMessage(uid uint32, message *msg.Message) (online bool, e error) {
	if uid == message.Sender() {
		return false, errors.New("cannot send to self")
	}
	start := time.Now()
	defer utils.PrintDuration(fmt.Sprintf("SendMessage %v", uid), start, time.Millisecond*10)

	online = false
	user, ok := TO_USER_POINT(users.Get(uid))
	if ok {
		logger.Append(fmt.Sprintf("begin send msg [%v->%v] : %v", message.Sender(), uid, message.String()), log.NOTICE)
		if len(user.Msgs) > MESSAGE_BUFFER_SIZE/2 {
			return false, errors.New(fmt.Sprintf("user %v message buffer too long %v", uid, e.Error()))
		} else {
			user.Msgs <- message
			online = true
		}
	}
	return online, nil
}

func SendTagMessage(tagid string, message *msg.Message) {
	if tagid == "all" {
		users.Iterate(func(key interface{}, value interface{}) error {
			user := (value).(*User)
			uid := key.(uint32)
			if uid != message.Sender() {
				if len(user.Msgs) > MESSAGE_BUFFER_SIZE/2 {
					logger.Append(fmt.Sprintf("user %v message buffer too long", uid))
				} else {
					user.Msgs <- message
				}
			}
			return nil
		})
	} else {
		tagUserMap.Iterate(tagid, func(uid uint32) error {
			user, ok := TO_USER_POINT(users.Get(uid))
			if ok {
				if uid != message.Sender() {
					if len(user.Msgs) > MESSAGE_BUFFER_SIZE/2 {
						logger.Append(fmt.Sprintf("user %v message buffer too long", uid))
					} else {
						user.Msgs <- message
					}
				}
			}
			return nil
		})
	}
}

func AddTag(uid uint32, tag string) {
	tagUserMap.AddUserTag(uid, tag)
	users.GetAndDo(uid, func(v interface{}, ok bool) {
		if ok {
			user := (v).(*User)
			user.Tags[tag] = true
		}
	})
}

func DelTag(uid uint32, tag string) {
	tagUserMap.DelUserTag(uid, tag)
	users.GetAndDo(uid, func(v interface{}, ok bool) {
		if ok {
			user := (v).(*User)
			delete(user.Tags, tag)
		}
	})
}
func batchClose(users []*User) {
	for _, user := range users {
		user.close()
	}
}

func SetLog(log *log.Logger) {
	logger = log
}
