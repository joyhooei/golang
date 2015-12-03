package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"pkg/yh_config"
	"pkg/yh_log"
	"pkg/yh_net"
	"pkg/yh_utils"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"yuanfen/adapter/models"
)

const (
	CLIENT_CHANNEL_SIZE       = 10      //客户端Channel的容量
	SERVER_CHANNEL_SIZE       = 10000   //服务端Channel的容量
	MAX_MSG_CONTENT_LEN       = 2097152 //消息体的最大长度
	LOG_INTERVAL              = 10      //状态日志输出的间隔
	CLIENT_FIRST_READ_TIMEOUT = 10      //客户端连接后第一条消息发送到服务器的超时时间
	SERVER_READ_TIMEOUT       = 300     //服务器idle的最长时间
	//SERVER_MSG_CONSUME_INTERVAL = 5       //服务器检查Channel中消息是否过多的时间间隔
	WAIT_LAST_CONN_TIMEOUT = 60 //等待同一用户上一个连接彻底关闭的超时时间
)

var levelStr map[string]int = map[string]int{"error": yh_log.ERROR, "debug": yh_log.DEBUG, "notice": yh_log.NOTICE, "warn": yh_log.WARN}

var rts, rt, wt, rlock, wlock int32
var waiting int32

//日志
var glog *yh_log.Logger

//配置
var conf *yh_config.Config

//心跳消息的间隔
var clientReadTimeout uint //客户端idle的最长时间

//丢弃的消息数量
var dropMsgs uint = 0
var dropServerMsgs uint = 0

//用户上行消息通道
var serverReceiveMsgChans map[uint16][]*models.Channel

//服务器下行消息通道表
var clientReceiveMsgChans map[uint32](*models.Channel) = make(map[uint32](*models.Channel))

var clientMutex sync.RWMutex

//是否需要向Server发送断开连接消息
var needSendCloseMsg map[uint32]bool = make(map[uint32]bool)

//配置文件中的必要项
var keywords = map[string]bool{
	"ips":       true,
	"port":      true,
	"log":       true,
	"log_level": true,
	"procs":     true,
	"servers":   true,
	"timeout":   true,
}

//检查配置文件是否合法
func checkConfig(conf *yh_config.Config) error {
	for key := range keywords {
		if _, found := conf.Items[key]; found == false {
			return errors.New("not found key [" + key + "]")
		}
	}
	return nil
}

//输出系统状态
func logStatus() {
	for {
		time.Sleep(LOG_INTERVAL * time.Second)
		log := ""
		for t, servers := range serverReceiveMsgChans {
			for id, ch := range servers {
				log += fmt.Sprintf("%v-%v:%v\t", t, id, len(ch.Ch))
			}
		}
		glog.Append("Messages in server channel : "+log, yh_log.NOTICE)
		glog.Append(fmt.Sprintf("online clients : %v\n\tread from client routings : %v\n\twrite to client routings : %v\n\twrite to server routings : %v | waiting : %v rlock: %v wlock: %v", len(clientReceiveMsgChans), rt, wt, rts, waiting, rlock, wlock), yh_log.NOTICE)
		glog.Append(fmt.Sprintf("drop messages : from clients %v, from servers %v", dropMsgs, dropServerMsgs), yh_log.NOTICE)
		dropMsgs, dropServerMsgs = 0, 0
	}
}

func closeClient(id uint32) {
	clientMutex.Lock()
	wlock++
	defer func() { wlock--; clientMutex.Unlock() }()
	c, ok := clientReceiveMsgChans[id]
	if ok {
		c.Close()
		delete(clientReceiveMsgChans, id)
	}
	v, ok := needSendCloseMsg[id]
	if !ok || v {
		e := models.NewMsg(0)
		e.SetID(id)
		e.SetLength(0)
		serverReceiveMsgChans[0][id%uint32(len(serverReceiveMsgChans[0]))].Ch <- e
	}
	delete(needSendCloseMsg, id)
}

func ReadMsgs(conn *yh_net.TCPConn) {
	atomic.AddInt32(&rt, 1)
	defer atomic.AddInt32(&rt, -1)
	defer conn.Close()
	//第一条消息要在10秒内收到，否则直接断开，避免攻击
	conn.SetReadTimeout(CLIENT_FIRST_READ_TIMEOUT)
	head := models.NewMsg(0)
	first := true
	second := true
	var newKey uint32 = 0
	var uid uint32

	for {
		err := conn.ReadSafe(head.Header())
		if err != nil {
			glog.Append(fmt.Sprintf("user %v ReadSafe head error: %v", uid, err.Error()))
			break
		}
		if head.Length() > MAX_MSG_CONTENT_LEN {
			glog.Append(fmt.Sprintf("user %v ReadSafe head error: msg too long len=%v", uid, head.Length()))
			glog.Append(fmt.Sprintf("from client len=%d id=%d type=%d", head.Length(), head.ID(), head.Type()), yh_log.DEBUG)
			break
		}

		data := models.NewMsg(head.Length())
		err = conn.ReadSafe(data.Content())
		if err != nil {
			glog.Append(fmt.Sprintf("user %v ReadSafe content error: %v", uid, err.Error()))
			break
		}
		copy(data.Header(), head.Header())
		if data.Type() == models.HEARTBEAT_MSG {
			glog.Append(fmt.Sprintf("receive from client len=%d id=%d type=HEART-BEAT", head.Length(), head.ID()), yh_log.DEBUG)
		} else {
			glog.Append(fmt.Sprintf("receive from client len=%d id=%d type=%d content=%s", head.Length(), head.ID(), head.Type(), string(data.Content())), yh_log.DEBUG)
		}
		if second {
			if first {
				if data.Type() != models.HEARTBEAT_MSG {
					glog.Append(fmt.Sprintf("user %v connected, but first message is not heart beat, abort.", head.ID()))
					break
				}
				oldKey, newKeyTmp := data.HeartBeatContent()
				newKey = newKeyTmp
				glog.Append(fmt.Sprintf("user %v oldKey=%v,newKey=%v", head.ID(), oldKey, newKey), yh_log.DEBUG)
				clientMutex.Lock() //R
				rlock++
				ch, ok := clientReceiveMsgChans[head.ID()]
				rlock--
				clientMutex.Unlock() /*R*/
				if ok {
					if data.ID() == ch.ID && ch.Key == oldKey {
						closeClient(data.ID())
					} else {
						//非法客户端连接，断开
						glog.Append(fmt.Sprintf("invalid user %v has connected, abort.", head.ID()))
						break
					}
				}
				first = false
				//忽略第一条心跳消息
				continue
			}
			i := 0
			for ; i < WAIT_LAST_CONN_TIMEOUT; i++ {
				clientMutex.Lock() //R
				rlock++
				_, ok := clientReceiveMsgChans[head.ID()]
				rlock--
				clientMutex.Unlock() /*R*/
				if ok {
					glog.Append(fmt.Sprintf("waiting last connection closed of user %v, %v seconds.", head.ID(), i), yh_log.DEBUG)
					time.Sleep(1 * time.Second)
				} else {
					break
				}
			}
			if i == WAIT_LAST_CONN_TIMEOUT {
				glog.Append(fmt.Sprintf("waiting last connection closed of user %v timeout.", head.ID()))
				break
			}
			uid = head.ID()
			ch := models.NewChannel(head.ID(), conn, CLIENT_CHANNEL_SIZE, newKey)
			clientMutex.Lock()
			wlock++
			clientReceiveMsgChans[head.ID()] = ch
			wlock--
			clientMutex.Unlock()
			conn.SetReadTimeout(clientReadTimeout)
			go WriteMsg(ch, conn)
			defer closeClient(head.ID())
			second = false
		}
		if head.ID() != uid {
			//非法消息
			glog.Append(fmt.Sprintf("invalid message, uid not match. expect %v, but is %v", uid, head.ID()))
			break
		}

		if data.Type() != models.HEARTBEAT_MSG {
			servers, ok := serverReceiveMsgChans[data.Type()]
			if !ok {
				glog.Append(fmt.Sprintf("invalid message type %v", data.Type()))
				break
			}

			id := data.ID() % uint32(len(servers))
			//glog.Append(fmt.Sprintf("send to server %v-%v ...", data.Type(), id), yh_log.DEBUG)
			select {
			case servers[id].Ch <- data:
			default:
				glog.Append(fmt.Sprintf("server channel %v full : %v\n", id, len(servers[id].Ch)))
				dropServerMsgs++
			}
		}
	}
}

func WriteMsg(ch *models.Channel, conn *yh_net.TCPConn) {
	defer conn.Close()
	atomic.AddInt32(&wt, 1)
	for {
		glog.Append(fmt.Sprintf("waiting to write to client %v ...", ch.ID), yh_log.DEBUG)
		atomic.AddInt32(&waiting, 1)
		msg, ok := <-ch.Ch
		atomic.AddInt32(&waiting, -1)
		//glog.Append(fmt.Sprintf("got message"), yh_log.DEBUG)
		if !ok {
			glog.Append(fmt.Sprintf("user %v channel closed", ch.ID))
			break
		}

		if msg.Length() == 0 {
			glog.Append(fmt.Sprintf("user %v connection closed by server", msg.ID()))
			needSendCloseMsg[msg.ID()] = false
			break
		}

		err := conn.WriteSafe([]byte(msg))
		if err != nil {
			glog.Append(fmt.Sprintf("user %v WriteSafe content error: %v", msg.ID(), err.Error()))
			break
		}
		glog.Append(fmt.Sprintf("send to client ip=%v len=%d id=%d type=%d content=%s\n", ch.IP, msg.Length(), msg.ID(), msg.Type(), string(msg.Content())), yh_log.DEBUG)
	}
	atomic.AddInt32(&wt, -1)
}

func ReadServerMsgs(conn *yh_net.TCPConn, msgType uint16, id uint32) {
	defer conn.Close()
	conn.SetReadTimeout(SERVER_READ_TIMEOUT)

	head := models.NewMsg(0)

	for {
		err := conn.ReadSafe(head.Header())
		if err != nil {
			glog.Append(fmt.Sprintf("server %v-%v ReadSafe head error: %v", msgType, id, err.Error()))
			break
		}
		data := models.NewMsg(head.Length())
		err = conn.ReadSafe(data.Content())
		if err != nil {
			glog.Append(fmt.Sprintf("server %v-%v ReadSafe content error: %v", msgType, id, err.Error()))
			break
		}
		copy(data.Header(), head.Header())
		glog.Append(fmt.Sprintf("from server %v-%v len=%d id=%d type=%d content=%s", msgType, id, data.Length(), data.ID(), data.Type(), string(data.Content())), yh_log.DEBUG)
		if head.ID() > 0 {
			func() {
				clientMutex.Lock() //R
				rlock++
				defer func() { rlock--; clientMutex.Unlock() /*R*/ }()
				ch, ok := clientReceiveMsgChans[head.ID()]
				if ok {
					//普通消息或断开连接消息
					select {
					case ch.Ch <- data:
					default:
						//不能阻塞协程，只能丢弃较早的消息
						dropMsgs++
					}
				} else {
					//该用户并未连接
					glog.Append(fmt.Sprintf("user %v not exists.", head.ID()))
					e := models.NewMsg(0)
					e.SetID(head.ID())
					e.SetType(0)
					e.SetLength(0)
					serverReceiveMsgChans[0][head.ID()%uint32(len(serverReceiveMsgChans[0]))].Ch <- e
				}
			}()
		}
	}
}

func InitServerMsg() (msg models.Msg) {
	clientMutex.Lock() //R
	rlock++
	defer func() { rlock--; clientMutex.Unlock() /*R*/ }()
	msg = models.NewMsg(uint32(len(clientReceiveMsgChans) * 8))
	index := 0
	content := msg.Content()
	for _, ch := range clientReceiveMsgChans {
		copy(content[index:index+4], yh_utils.Uint32ToBytes(ch.ID))
		index += 4
		copy(content[index:index+4], ch.IP)
		index += 4
	}
	return
}

func WriteServerMsgs(msgType uint16, serverReceiveMsgChan *models.Channel) {
	ip := make([]byte, 4)
	var msg, msgBuffer models.Msg
	sent := true
	var conn *yh_net.TCPConn = nil
	atomic.AddInt32(&rts, 1)
	for {
		for {
			conn = serverReceiveMsgChan.Conn()
			if sent == false {
				msg = msgBuffer
				glog.Append(fmt.Sprintf("server %v-%v use last msg %v", msgType, serverReceiveMsgChan.ID, msg.Content()), yh_log.DEBUG)
			} else {
				msg = <-serverReceiveMsgChan.Ch
				msgBuffer = msg
				sent = false
				clientMutex.Lock() //R
				rlock++
				ch, ok := clientReceiveMsgChans[msg.ID()]
				rlock--
				clientMutex.Unlock() /*R*/
				if ok {
					copy(ip, ch.IP)
				}
			}
			err := conn.WriteSafe(ip)
			if err != nil {
				glog.Append(fmt.Sprintf("server %v-%v Write IP error: %v", msgType, serverReceiveMsgChan.ID, err.Error()))
				time.Sleep(1 * time.Second)
				continue
			}

			err = conn.WriteSafe([]byte(msg))
			if err != nil {
				glog.Append(fmt.Sprintf("server %v-%v Write msg error: %v", msgType, serverReceiveMsgChan.ID, err.Error()))
				continue
			}
			sent = true
			glog.Append(fmt.Sprintf("send to server %v-%v ip=%v len=%d id=%d type=%d content=%s", msgType, serverReceiveMsgChan.ID, ip, msg.Length(), msg.ID(), msg.Type(), string(msg.Content())), yh_log.DEBUG)
			break
		}
	}
	atomic.AddInt32(&rts, -1)
}

func handleServer(ip string, port int, msgType uint16, serverReceiveMsgChan *models.Channel) {
	first := true
	for {
		glog.Append(fmt.Sprintf("connecting server %v-%v(%v:%v)....", msgType, serverReceiveMsgChan.ID, ip, port), yh_log.NOTICE)
		conn, err := yh_net.Connect(ip, port)
		if err != nil {
			glog.Append(fmt.Sprintf("connect %v-%v(%v:%v) failed : %v", msgType, serverReceiveMsgChan.ID, ip, port, err.Error()))
			time.Sleep(5 * time.Second)
		} else {
			glog.Append(fmt.Sprintf("server %v-%v(%v:%v) connected.", msgType, serverReceiveMsgChan.ID, ip, port), yh_log.NOTICE)
			if msgType == 0 {
				msg := InitServerMsg()
				err := conn.WriteSafe(make([]byte, 4))
				if err != nil {
					glog.Append(fmt.Sprintf("server %v-%v Write IP error: %v", msgType, serverReceiveMsgChan.ID, err.Error()))
					continue
				}

				err = conn.WriteSafe([]byte(msg))
				if err != nil {
					glog.Append(fmt.Sprintf("server %v-%v Write msg error: %v", msgType, serverReceiveMsgChan.ID, err.Error()))
					continue
				}
				b_buf := bytes.NewBuffer(msg.Content())
				ids := make([]uint32, 0, 5)
				for i := 0; i < len(msg.Content()); i += 4 {
					var id uint32
					e := binary.Read(b_buf, binary.BigEndian, &id)
					if e != nil {
						glog.Append(e.Error())
						break
					}
					ids = append(ids, id)
				}
				glog.Append(fmt.Sprintf("send to server %v-%v ip=0.0.0.0 len=%d id=%d type=%d content=%v", msgType, serverReceiveMsgChan.ID, msg.Length(), msg.ID(), msg.Type(), ids), yh_log.DEBUG)
			}
			serverReceiveMsgChan.SetConn(conn)

			if first {
				first = false
				go WriteServerMsgs(msgType, serverReceiveMsgChan)
			}
			ReadServerMsgs(conn, msgType, serverReceiveMsgChan.ID)
		}
	}
}

func InitServers(fileName string) (err error) {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	serverReceiveMsgChans = make(map[uint16][]*models.Channel)
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		} else {
			fields := strings.Split(line, "\t")
			if len(fields) != 3 {
				err = errors.New(fmt.Sprintf("invalid line in server config : %s", line))
				return err
			}
			t, err := strconv.Atoi(strings.Trim(fields[2], " \t\n"))
			if err != nil {
				fmt.Println(err.Error())
				return err
			}
			msgType := uint16(t)
			port, err := strconv.Atoi(strings.Trim(fields[1], " \t"))
			if err != nil {
				fmt.Println(err.Error())
				return err
			}
			servers, ok := serverReceiveMsgChans[msgType]
			if !ok {
				servers = make([]*models.Channel, 0, 4)
				serverReceiveMsgChans[msgType] = servers
			}
			serverReceiveMsgChan := models.NewChannel(uint32(len(servers)), nil, SERVER_CHANNEL_SIZE, 0)
			serverReceiveMsgChans[msgType] = append(servers, serverReceiveMsgChan)

			go handleServer(strings.Trim(fields[0], " \t"), port, msgType, serverReceiveMsgChan)
		}
	}
	return
}

func Accept(ip string, port int) {
	ln, err := yh_net.Listen(ip, port)
	if err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}

	glog.Append("Accept ...")
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err.Error())
			glog.Append(err.Error())
			return
		}
		glog.Append(fmt.Sprintf("new connection from %v", conn.RemoteAddr()), yh_log.DEBUG)
		go ReadMsgs(conn)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s [config]\n", os.Args[0])
		return
	}

	rand.Seed(time.Now().UnixNano())
	conf, err := yh_config.NewConfig(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = checkConfig(&conf)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	ps, err := strconv.Atoi(conf.Items["procs"])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	clientReadTimeout, err = yh_utils.StringToUint(conf.Items["timeout"])
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	runtime.GOMAXPROCS(ps)
	log_level, ok := levelStr[conf.Items["log_level"]]
	if !ok {
		fmt.Println("invalid log level : ", conf.Items["log_level"])
		return
	}

	glog, err = yh_log.New(conf.Items["log"], 10000, log_level)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer glog.Close()

	go logStatus()

	err = InitServers(conf.Items["servers"])
	if err != nil {
		fmt.Println("init servers error : ", err.Error())
		glog.Append("init servers error : " + err.Error())
		return
	}

	glog.Append("start service " + os.Args[0])
	port, err := strconv.Atoi(conf.Items["port"])
	if err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}
	ips := strings.Split(conf.Items["ips"], ",")
	for _, ip := range ips {
		go Accept(ip, port)
	}
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
		}
	}()
	for {
		time.Sleep(100 * time.Second)
	}
}
