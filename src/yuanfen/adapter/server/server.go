package main

import (
	"fmt"
	"os"
	"pkg/yh_net"
	"strconv"
	"time"
	"yuanfen/adapter/models"
)

var clientMsgChan chan models.Msg = make(chan models.Msg, 1000)

var connect *yh_net.TCPConn = nil

var rec_count, send_count uint64 = 0, 0

func ReadMsgs(conn *yh_net.TCPConn) {
	defer conn.Close()
	conn.SetReadTimeout(60)

	head := models.NewMsg(0)
	first := true
	ip := make([]byte, 4)

	for {
		rec_count++
		err := conn.ReadSafe(ip)
		if err != nil {
			fmt.Println("ReadSafe ip error:" + err.Error())
			break
		}
		fmt.Printf("read ip=%v\n", ip)
		err = conn.ReadSafe(head.Header())
		if err != nil {
			fmt.Println("ReadSafe head error:" + err.Error())
			break
		}
		data := models.NewMsg(head.Length())
		err = conn.ReadSafe(data.Content())
		if err != nil {
			fmt.Println("ReadSafe content error:" + err.Error())
			break
		}
		copy(data.Header(), head.Header())
		//		if first {
		fmt.Printf("receive ip=%v len=%d id=%d type=%v content=%v\n", ip, head.Length(), head.ID(), head.Type(), string(data.Content()))
		//		}
		if head.Length() > 0 {
			clientMsgChan <- data
		}

		if first {
			connect = conn
			go WriteMsg()
			first = false
		}
	}
}

func WriteMsg() {
	for {
		send_count++
		msg := <-clientMsgChan
		//fmt.Printf("send len=%d id=%d content=%s\n", msg.Length(), msg.ID(), string(msg.Content()))
		for {
			err := connect.WriteSafe([]byte(msg))
			if err != nil {
				fmt.Println("Write content error:" + err.Error())
				time.Sleep(3 * time.Second)
				continue
			}
			break
		}
	}
}

func stat() {
	var last_rec_count, last_send_count uint64 = 0, 0
	for {
		time.Sleep(10 * time.Second)
		fmt.Printf("send freq : %v,  rec freq : %v\n", (send_count-last_send_count)/10, (rec_count-last_rec_count)/10)
		last_rec_count, last_send_count = rec_count, send_count
	}
}

func main() {

	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s ip port\n", os.Args[0])
		return
	}

	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	ln, err := yh_net.Listen(os.Args[1], port)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	go stat()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		go ReadMsgs(conn)
	}
	time.Sleep(2 * time.Second)
}
