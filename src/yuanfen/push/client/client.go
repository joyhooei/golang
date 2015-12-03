package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"yf_pkg/net"
	"yf_pkg/net/http"
	"yf_pkg/utils"
	"yuanfen/push/pusher/common"
	"yuanfen/push/pusher/msg"
)

var conn *net.TCPConn
var uid uint32
var host string

func SendMessages() {
	for {
		fmt.Println("-------------------------------------------")
		fmt.Println("message : msg|uid|content")
		fmt.Println("tag message : tmsg|tag|content")
		fmt.Println("add tag : atag|uid|tag")
		fmt.Println("del tag : dtag|uid|tag")
		fmt.Println("clear tag : ctag|tag")
		fmt.Printf("enter your message : ")
		d := make([]byte, 100)
		_, err := fmt.Scan(&d)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		items := strings.Split(string(d), "|")
		switch items[0] {
		case "msg":
			body, e := http.HttpSend(host, "push/Send", map[string]string{"from": utils.Uint32ToString(uid), "to": "u_" + items[1]}, nil, []byte(items[2]))
			data := make(map[string]interface{})
			e = json.Unmarshal(body, &data)
			if e != nil {
				fmt.Println(e.Error())
				continue
			} else {
				fmt.Println(data)
			}
		case "tmsg":
			body, e := http.HttpSend(host, "push/Send", map[string]string{"from": utils.Uint32ToString(uid), "to": "t_" + items[1]}, nil, []byte(items[2]))
			fmt.Println("body=", string(body))
			data := make(map[string]interface{})
			e = json.Unmarshal(body, &data)
			if e != nil {
				fmt.Println(e.Error())
				continue
			} else {
				fmt.Println(data)
			}
		case "atag":
			body, e := http.HttpSend(host, "push/AddTag", map[string]string{"uid": items[1], "tag": items[2]}, nil, nil)
			data := make(map[string]interface{})
			e = json.Unmarshal(body, &data)
			if e != nil {
				fmt.Println(e.Error())
				continue
			} else {
				fmt.Println(data)
			}
		case "dtag":
			body, e := http.HttpSend(host, "push/DelTag", map[string]string{"uid": items[1], "tag": items[2]}, nil, nil)
			data := make(map[string]interface{})
			e = json.Unmarshal(body, &data)
			if e != nil {
				fmt.Println(e.Error())
				continue
			} else {
				fmt.Println(data)
			}
		case "ctag":
			body, e := http.HttpSend(host, "push/ClearTag", map[string]string{"tag": items[1]}, nil, nil)
			data := make(map[string]interface{})
			e = json.Unmarshal(body, &data)
			if e != nil {
				fmt.Println(e.Error())
				continue
			} else {
				fmt.Println(data)
			}
		default:
			fmt.Println("unknown message type : ", items[0])
			continue
		}
	}
}

func ReadMessages() {
	for {
		m, e := msg.ReadMessage(conn)
		if e != nil {
			fmt.Println("read message erorr :", e.Error())
			break
		} else {
			fmt.Println("receive", m.String())
			fmt.Printf("enter your message : ")
		}
	}
}

func Heartbeat() {
	hb := msg.New(common.USER_MSG, 0, uid, []byte(`{"lat":12.23,"lng":33.11211}`), "")
	for {
		e := hb.Send(conn)
		if e != nil {
			fmt.Println("send heart beat error :", e.Error())
			break
		}
		time.Sleep(30 * time.Second)
	}
}

func Login(ip string, port int) (*net.TCPConn, error) {
	host = fmt.Sprintf("%s:%v", ip, port)
	body, e := http.HttpSend(host, "push/GetEndpoint", map[string]string{"uid": utils.Uint32ToString(uid)}, nil, nil)
	data := make(map[string]interface{})
	e = json.Unmarshal(body, &data)
	if e != nil {
		return nil, e
	}
	fmt.Println("data=", data)
	addr := strings.Split(data["address"].(string), ":")
	p, _ := utils.StringToInt(addr[1])
	fmt.Printf("connect %v:%v\n", addr[0], p)
	conn, err := net.Connect(addr[0], p)
	if err != nil {
		return nil, err
	}
	j, e := json.Marshal(map[string]interface{}{"key": data["key"]})
	if e != nil {
		return conn, e
	}
	m := msg.New(common.USER_MSG, 0, uid, j, "")
	fmt.Println("Send :", m.String())
	m.Send(conn)
	reply, e := msg.ReadMessage(conn)
	if e != nil {
		return conn, e
	}
	fmt.Println("receive message :", reply.String())
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

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage : ", os.Args[0], "ip port uid")
		return
	}

	t, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	uid = uint32(t)

	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	conn, err = Login(os.Args[1], port)
	if err != nil {
		fmt.Println("login failed :", err.Error())
		if conn != nil {
			conn.Close()
		}
		return
	}
	defer conn.Close()
	fmt.Println("已连接服务器")
	go Heartbeat()
	go ReadMessages()
	SendMessages()
}
