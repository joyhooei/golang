package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"yuanfen/models"
)

var uid uint32
var t uint16

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage : ", os.Args[0], " address type uid")
		return
	}
	addr := os.Args[1]
	u, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	uid = uint32(u)
	tp, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	t = uint16(tp)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println("连接服务端失败:", err.Error())
		return
	}
	fmt.Println("已连接服务器")
	defer conn.Close()
	Client(conn)
}

func Client(conn net.Conn) {
	for {
		sms := models.NewMsg(0, 100)
		fmt.Print("请输入要发送的消息:")
		d := make([]byte, 100)
		_, err := fmt.Scan(&d)
		if err != nil {
			fmt.Println("数据输入异常:", err.Error())
		}

		fmt.Println(string(d))
		if string(d) == "close" {
			fmt.Println("bye")
			conn.Close()
			return
		}
		sms.Append(d...)
		sms.SetID(uid)
		sms.SetLength(uint32(len(sms.Content())))
		sms.SetType(t)
		conn.Write(sms)
		head := models.NewMsg(0)
		c, err := conn.Read(head)
		if err != nil {
			fmt.Println("读取服务器数据异常:", err.Error())
		}
		result := make([]byte, head.Length())
		c, err = conn.Read(result)
		if err != nil {
			fmt.Println("读取服务器数据异常:", err.Error())
		}
		fmt.Printf("read %v bytes\n", c)
		fmt.Printf("len=%v, id=%v, content=%v\n", head.Length(), head.ID(), result)
	}

}
