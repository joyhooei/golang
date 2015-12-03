package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
	"yf_pkg/log"
	"yf_pkg/net"
	"yf_pkg/net/http"
	"yf_pkg/utils"
	"yuanfen/manageragent/cls/msgm"
	"yuanfen/manageragent/cls/mtcp"
	"yuanfen/push/pusher/common"
	// "yuanfen/push/pusher/msg"
)

const (
	CLIENT_FIRST_READ_TIMEOUT = 10  //客户端连接后第一条消息的超时时间（秒）
	CLIENT_TIMEOUT            = 130 //客户端超时时间（秒）
	MESSAGE_BUFFER_SIZE       = 100 //每个用户消息队列的最大长度
)

var users map[uint32]*user
var ulock sync.RWMutex
var logger *log.MLogger

type user struct {
	Uid  uint32
	Msgs chan *msgm.Message
	conn *net.TCPConn
}

func init() {
	users = make(map[uint32]*user)
}

func Init(conf *common.Config) error {
	fmt.Println("conf.LogDir " + conf.Log.Dir)
	log1, err := log.NewMLogger(conf.Log.Dir+"/manager", 1000, conf.Log.Level)
	if err != nil {
		return err
	}
	logger = log1
	return nil
}

func PrintMap() string {
	ulock.Lock()
	defer ulock.Unlock()
	i := 0
	for _, _ = range users {
		i++
	}
	return "map count " + utils.ToString(i)
}

func (u *user) readHeartBeat() {
	u.conn.SetTimeout(CLIENT_TIMEOUT)
	for {
		_, err := msgm.ReadMessageM(u.conn)
		if err != nil {
			logger.AppendInfo(fmt.Sprintf("Read HEART-BEAT error:uid %v, error %v", u.Uid, err.Error()))
			// fmt.Printf("%v user %v read HEART-BEAT error : %v\n", utils.Now, u.Uid, err.Error())
			u.DelMe()
			break
		}
		m := msgm.New(0, -1, 0, 0, []byte(`{"status":"ok"}`), "")
		u.Msgs <- m
	}
}

func (u *user) writeMessages() {
	for msg := range u.Msgs {
		if msg == nil {
			break
		}
		// fmt.Printf("send Messages to Manager %v %v ", u.Uid, msg.String())
		logger.AppendInfo(fmt.Sprintf("send to :uid %v,msg %v", u.Uid, msg.String()))
		e := msg.SendM(u.conn)
		if e != nil {
			u.DelMe()
		}
		// logger.AppendInfo(fmt.Sprintf("send to :uid %v,msg %v, error %v", u.Uid, msg.String(), e.Error()))

	}
}

func (u *user) close() {
	fmt.Printf("%v close channel of user %v\n", utils.Now, u.Uid)
	logger.AppendInfo(fmt.Sprintf("user close :uid %v", u.Uid))
	u.Msgs <- nil
	e := u.conn.Close()
	if e != nil {
	}

}

//删除某个用户，同时关闭连接
func DelUser(uid uint32) {
	ulock.Lock()
	user, ok := users[uid]
	ulock.Unlock()
	if ok {
		// fmt.Printf("DelUser user %v disconnected\n", uid)
		delete(users, uid)
		user.close()
	}
}

func (u *user) DelMe() {
	ulock.Lock()
	user, ok := users[u.Uid]
	ulock.Unlock()
	if ok && user == u {
		// fmt.Printf("DelMe user %v disconnected\n", u.Uid)
		delete(users, u.Uid)
		user.close()
	}
}

func SendMessage(mid uint32, message *msgm.Message) (online bool, e error) {
	online = false
	ulock.Lock()
	user, ok := users[mid]
	ulock.Unlock()
	if ok {
		user.Msgs <- message
		online = true
	} else {
		// return false, errors.New("user not on this shard")
	}
	return online, nil
}

func AddManager(conn *net.TCPConn) {
	uid, valid, reason := validate(conn)
	if valid {
		DelUser(uid)
		user := &user{uid, make(chan *msgm.Message, MESSAGE_BUFFER_SIZE), conn}
		// fmt.Println(fmt.Sprintf("%v user %v is valid\n", utils.Now, uid))
		ulock.Lock()
		users[uid] = user
		ulock.Unlock()
		// fmt.Println("try GetMyOnlineUids")
		uids, err := mtcp.GetMyOnlineUids(uid)
		if err != nil {
			// fmt.Println("GetMyOnlineUids error " + err.Error())
			return
		}
		logger.AppendInfo(fmt.Sprintf("GetMyOnlineUids :mid %v,uids %v", uid, uids))
		j, e := json.Marshal(map[string]interface{}{"status": "ok", "uids": uids})
		if e == nil {
			m := msgm.New(0, common.USER_MSG, 0, 0, j, "")
			user.Msgs <- m
		} else {
			m := msgm.New(0, common.USER_MSG, 0, 0, []byte(`{"status":"ok"}`), "")
			user.Msgs <- m
		}
		// user.Msgs <- m
		go user.readHeartBeat()
		go user.writeMessages()
	} else {
		logger.AppendInfo(fmt.Sprintf("user invalid :mid %v", uid))
		fmt.Println(fmt.Sprintf("user %v invalid", uid))
		j, e := json.Marshal(map[string]interface{}{"status": "fail", "msg": reason})
		if e == nil {
			m := msgm.New(0, common.USER_MSG, 0, 0, j, "")
			m.Send(conn)
			time.Sleep(1 * time.Second)
		}
		conn.Close()
	}
}

func managerLogin(username string, sid string) (e error, aid uint32) {
	fmt.Println(fmt.Sprintf("ManagerLogin %v, %v", username, sid))
	content := make(map[string]interface{})
	content["username"] = username
	content["password"] = sid
	j, e := json.Marshal(content)
	if e != nil {
		return e, 0
	}
	body, e := http.HttpSend("service.mumu123.cn", "user/AdminLogin", nil, nil, j)
	logger.AppendInfo(fmt.Sprintf("managerLogin :username %v,password %v,result %v", username, sid, string(body)))

	data := make(map[string]interface{})
	e = json.Unmarshal(body, &data)
	if e != nil {
		return e, 0
	}
	if utils.ToString(data["status"]) != "ok" {
		return errors.New("Login Error " + utils.ToString(data["msg"])), 0
	}
	switch value := data["res"].(type) {
	case map[string]interface{}:
		aid, e = utils.ToUint32(value["uid"])
	default:
		return errors.New("Invalid return"), 0
	}
	return
}

func validate(conn *net.TCPConn) (uint32, bool, string) {
	conn.SetTimeout(CLIENT_FIRST_READ_TIMEOUT)
	msg, err := msgm.ReadMessageM(conn)
	if err != nil {
		return 0, false, fmt.Sprintf("read login message error : %v", err.Error())
	}
	sender := msg.Sender()
	username := msg.UserName()
	password := msg.PassWord()
	logger.AppendInfo(fmt.Sprintf("check manager :mid %v,username %v,password %v", sender, username, password))
	fmt.Printf("check user %v , username %v, password=%v\n", sender, username, password)
	err, aid := managerLogin(username, password)
	if err != nil {
		return sender, false, err.Error()
	}
	if aid == sender {
		return sender, true, ""
	} else {
		return sender, false, ""
	}
}
