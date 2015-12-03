package main

import (
	"fmt"
	"math/rand"
	"os"
	"pkg/yh_net"
	"runtime"
	"strconv"
	"time"
	"yuanfen/adapter/models"
)

const (
	addr = "127.0.0.1:8923"
)

var ip string
var port int
var content []byte = make([]byte, 1000, 1000)

var read_msg_count, write_msg_count uint64 = 0, 0

func init() {
	s := []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	for i := 0; i < 100; i++ {
		v := content[i*10:]
		copy(v, s)
	}
	//	fmt.Println(content)
}

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage : ", os.Args[0], " ip port from_id to_id")
		return
	}
	runtime.GOMAXPROCS(8)
	rand.Seed(time.Now().UnixNano())
	ip = os.Args[1]
	p, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	port = p
	from, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	to, err := strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	//	fmt.Printf("ip=%v, port=%v, count=%v\n", ip, port, count)
	i := from
	for ; i < to; i++ {
		go Client(i + 1)
		time.Sleep(10 * time.Millisecond)
		if i%100 == 0 {
			fmt.Printf("client number : %v\n", i)
		}
	}
	for {
		time.Sleep(10 * time.Second)
		fmt.Printf("write freq : %v/s\tread freq : %v/s\n", write_msg_count/10, read_msg_count/10)
		write_msg_count, read_msg_count = 0, 0
	}
}

func Client(uid int) {
	head := models.NewMsg(0)
	head.SetID(uint32(uid))
	conn, err := yh_net.Connect(ip, port)
	if err != nil {
		fmt.Println("连接服务端失败:", err.Error())
		return
	}
	go ReadMsgs(conn)
	//	fmt.Println("已连接服务器")
	defer conn.Close()
	head.SetLength(8)
	head.SetType(65535)
	err = conn.WriteSafe(head)
	if err != nil {
		fmt.Println("client %v write head error : %v", uid, err.Error())
		return
	}
	err = conn.WriteSafe(content[:head.Length()])
	if err != nil {
		fmt.Println("client %v write content error : %v", uid, err.Error())
		return
	}
	write_msg_count++
	for {
		head.SetLength(rand.Uint32()%1000 + 1)
		head.SetType(uint16(rand.Uint32() % 2))
		err := conn.WriteSafe(head)
		if err != nil {
			fmt.Println("client %v write head error : %v", uid, err.Error())
			break
		}
		err = conn.WriteSafe(content[:head.Length()])
		if err != nil {
			fmt.Println("client %v write content error : %v", uid, err.Error())
			break
		}
		write_msg_count++
		time.Sleep(5 * time.Second)
	}
}

func ReadMsgs(conn *yh_net.TCPConn) {
	head := models.NewMsg(0)
	var uid int
	rcontent := make([]byte, 1000)
	conn.SetReadTimeout(0)
	for {
		err := conn.ReadSafe(head)
		if err != nil {
			fmt.Println("client %v read head error :", uid, err.Error())
			break
		}
		err = conn.ReadSafe(rcontent[:head.Length()])
		if err != nil {
			fmt.Println("client %v read content error :", uid, err.Error())
			break
		}
		//		fmt.Printf("len=%v, id=%v, content=%v\n", head.Length(), head.ID(), string(rcontent[:head.Length()]))
		read_msg_count++
	}
}
