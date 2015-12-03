package mtcp

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
	"yuanfen/push/pusher/common"
)

const (
	MESSAGE_BUFFER_SIZE = 100 //每个用户消息队列的最大长度
)

type User struct {
	Uid  uint32
	Mid  uint32
	Msgs chan *msgm.Message
	conn *net.TCPConn
}

var users map[uint32]*User

var clientMutex sync.RWMutex

var SendToManager func(mid uint32, message *msgm.Message) (online bool, e error)

var logger *log.MLogger

func init() {
	users = make(map[uint32]*User)

}

func Init(conf *common.Config) error {

	log1, err := log.NewMLogger(conf.Log.Dir+"/usertcp", 10000, conf.Log.Level)
	if err != nil {
		return err
	}
	logger = log1
	return nil
}

//删除某个用户，同时关闭连接
func DelUser(uid uint32) {
	clientMutex.Lock()
	user, ok := users[uid]
	if ok {
		//		fmt.Printf("DelUser user %v disconnected\n", uid)
		delete(users, uid)
	}
	clientMutex.Unlock()
	if ok {
		user.Close()
	}
}

//获取某个客服号下的用户
func GetMyOnlineUids(mid uint32) (uids string, e error) {
	// uids = make([]uint32, 0, 0)

	clientMutex.Lock()
	for uid, us := range users {
		if us.Mid == mid {
			uids = uids + utils.ToString(uid) + ","
		}
	}
	clientMutex.Unlock()
	return
}

func PrintMap() string {
	clientMutex.Lock()
	defer clientMutex.Unlock()
	i := 0
	for _, _ = range users {
		i++
	}
	return "map count " + utils.ToString(i)
}

func AddUser(uid uint32, mid uint32, conn *net.TCPConn) {
	logger.AppendInfo(fmt.Sprintf("AddUser  uid %v,mid %v", uid, mid))
	DelUser(uid) //删除可能存在的上一个连接
	user := &User{uid, mid, make(chan *msgm.Message, MESSAGE_BUFFER_SIZE), conn}
	clientMutex.Lock()
	users[uid] = user
	clientMutex.Unlock()
	go user.Heartbeat()
	go user.ReadMsgs()
}

func DoLogin(uid uint32, sid string) (*net.TCPConn, error) {
	fmt.Println(fmt.Sprintf("DoLogin %v, %v", uid, sid))
	// host = fmt.Sprintf("%s:%v", ip, port)
	body, e := http.HttpSend("service.mumu123.cn", "s/user/GetEndpoint", map[string]string{"uid": utils.Uint32ToString(uid), "key": sid}, map[string]string{"uid": utils.Uint32ToString(uid), "key": sid}, nil)
	data := make(map[string]interface{})
	e = json.Unmarshal(body, &data)
	if e != nil {
		return nil, e
	}
	if utils.ToString(data["status"]) != "ok" {
		return nil, errors.New("Login Error " + string(body))
	}
	// fmt.Println("data=", data)
	res2 := data["res"].(map[string]interface{})
	addr := res2["address"].(string)
	p, _ := utils.ToInt(res2["port"])
	key := res2["key"]
	logger.AppendInfo(fmt.Sprintf("GetEndpoint uid %v,sid %v,addr %v,port %v", uid, sid, addr, p))
	// fmt.Printf("connect %v:%v\n", addr, p)

	conn, err := net.Connect(addr, p)
	if err != nil {
		return nil, err
	}
	j, e := json.Marshal(map[string]interface{}{"key": key})
	if e != nil {
		return conn, e
	}
	m := msgm.New(0, 1, 0, uid, j, "")
	// fmt.Println("Send :", m.String())
	m.Send(conn)

	reply, e := msgm.ReadMessage(conn)
	if e != nil {
		return conn, e
	}
	logger.AppendInfo(fmt.Sprintf("Connect Replay uid %v,sid %v ,key %v,content %v ", uid, sid, key, string(reply.Content())))
	// fmt.Println("receive message :", reply.String())
	res := make(map[string]string)
	e = json.Unmarshal(reply.Content(), &res)
	if e != nil {
		return conn, e
	}
	if res["status"] != "ok" {
		return conn, errors.New(fmt.Sprintf("<%v,%v>", res["code"], res["msg"]))
	}
	return conn, nil
}

func (u *User) DelMe() {
	clientMutex.Lock()
	user, ok := users[u.Uid]
	if ok && user == u {
		delete(users, u.Uid)
	}
	clientMutex.Unlock()
	if ok {
		user.Close()
		sendDisconnect(u.Mid, u.Uid)
	}
}

func (u *User) Close() {
	e := u.conn.Close()
	if e != nil {

	}
}

func (u *User) Heartbeat() {
	hb := msgm.New(0, 1, 0, u.Uid, []byte(`{"lat":12.23,"lng":33.11211}`), "")
	for {
		e := hb.Send(u.conn)
		if e != nil {
			logger.AppendInfo(fmt.Sprintf("Send Heartbeat error:uid %v,mid %v, error %v", u.Uid, u.Mid, e.Error()))
			// fmt.Println("send heartbeat error :", e.Error())
			u.DelMe()
			break
		}
		time.Sleep(30 * time.Second)
	}
}

func (u *User) ReadMsgs() {

	defer u.conn.Close()
	u.conn.SetTimeout(24 * 3600)
	for {
		msg, err := msgm.ReadMessage(u.conn)
		if err != nil {
			logger.AppendInfo(fmt.Sprintf("ReadMsgs error:uid %v,mid %v, error %v", u.Uid, u.Mid, err.Error()))
			fmt.Printf("%v user %v ReadMsgs error : %v\n", utils.Now, u.Uid, err.Error())
			u.DelMe()
			break
		}
		msg.SetTOID(u.Uid)
		// fmt.Println(fmt.Sprintf("ReadMsgs %v ", msg))
		_, e := SendToManager(u.Mid, &msg)
		if e != nil {
			logger.AppendInfo(fmt.Sprintf("SendToManager error:uid %v,mid %v, error %v", u.Uid, u.Mid, e.Error()))
		}
	}
}

func sendDisconnect(mid uint32, uid uint32) (e error) {
	logger.AppendInfo(fmt.Sprintf("sendDisconnect mid %v,uid %v", mid, uid))
	m := msgm.New(uid, -10, 0, uid, []byte(`{"status":"ok"}`), "")
	SendToManager(mid, m)
	return
}
