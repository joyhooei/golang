package msgm

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"yf_pkg/net"
	"yf_pkg/utils"
)

//----------------------Message Format----------------------------
//
//	/---Toid----/---msgid---/---sender---/---len---/-type-/--tag--|-------content-------/
//   4Bytes       8Bytes      4Bytes      4Bytes     1B        len Bytes
//
//----------------------------------------------------------------
type Message struct {
	toid []byte
	head []byte
	tag  []byte
	body []byte
}

func New(toid uint32, msgType int8, msgid uint64, from uint32, content []byte, tag string) (m *Message) {
	m = &Message{make([]byte, 4, 4), make([]byte, 17, 17), nil, nil}
	m.SetTOID(toid)
	m.SetSender(from)
	m.SetID(msgid)
	m.SetType(msgType)
	m.SetContent(content, []byte(tag))
	return m
}

func ReadMessageM(conn *net.TCPConn) (Message, error) {
	msg := Message{make([]byte, 4, 4), make([]byte, 17, 17), nil, nil}
	err := conn.ReadSafe(msg.toid)
	if err != nil {
		return msg, err
	}
	err = conn.ReadSafe(msg.head)
	if err != nil {
		return msg, err
	}
	if msg.Length() >= 1024*1024*2 {
		return Message{}, errors.New(fmt.Sprintf("message too long %v", msg.Length()))
	}
	body := make([]byte, msg.Length(), msg.Length())
	err = conn.ReadSafe(body)
	if err != nil {
		return msg, err
	}
	for i, v := range body {
		if v == '|' {
			msg.body = body[i+1:]
			msg.tag = body[:i]
			return msg, nil
		}
	}
	return msg, errors.New("no | in message body")
}

func ReadMessage(conn *net.TCPConn) (Message, error) {
	msg := Message{make([]byte, 4, 4), make([]byte, 17, 17), nil, nil}
	err := conn.ReadSafe(msg.head)
	if err != nil {
		return msg, err
	}
	if msg.Length() >= 1024*1024*2 {
		return Message{}, errors.New(fmt.Sprintf("message too long %v", msg.Length()))
	}
	body := make([]byte, msg.Length(), msg.Length())
	err = conn.ReadSafe(body)
	if err != nil {
		return msg, err
	}
	for i, v := range body {
		if v == '|' {
			msg.body = body[i+1:]
			msg.tag = body[:i]
			return msg, nil
		}
	}
	return msg, errors.New("no | in message body")
}

//发送消息
func (m *Message) Send(conn *net.TCPConn) error {
	e := conn.WriteSafe(bytes.Join([][]byte{m.head, m.tag, []byte("|"), m.body}, nil))
	return e
}

//发送消息给客服
func (m *Message) SendM(conn *net.TCPConn) error {
	e := conn.WriteSafe(bytes.Join([][]byte{m.toid, m.head, m.tag, []byte("|"), m.body}, nil))
	return e
}

//消息ID
func (m *Message) ID() (id uint64) {
	b_buf := bytes.NewBuffer(m.head[0:8])
	binary.Read(b_buf, binary.BigEndian, &id)
	return
}

//消息ID
func (m *Message) ToID() (id uint64) {
	b_buf := bytes.NewBuffer(m.toid[0:4])
	binary.Read(b_buf, binary.BigEndian, &id)
	return
}

//给谁的
func (m *Message) SetTOID(id uint32) {
	b_buf := new(bytes.Buffer)
	binary.Write(b_buf, binary.BigEndian, id)
	copy(m.toid[0:4], b_buf.Bytes()[0:4])
}

func (m *Message) SetID(id uint64) {
	b_buf := new(bytes.Buffer)
	binary.Write(b_buf, binary.BigEndian, id)
	copy(m.head[0:8], b_buf.Bytes()[0:8])
}

//发送者ID
func (m *Message) Sender() (s uint32) {
	b_buf := bytes.NewBuffer(m.head[8:12])
	binary.Read(b_buf, binary.BigEndian, &s)
	return
}
func (m *Message) SetSender(s uint32) {
	b_buf := new(bytes.Buffer)
	binary.Write(b_buf, binary.BigEndian, s)
	copy(m.head[8:12], b_buf.Bytes()[0:4])
}

//PassWord，登陆秘钥，仅对登陆消息有效
func (m *Message) PassWord() string {
	var data map[string]interface{}
	e := json.Unmarshal(m.body, &data)
	if e != nil {
		fmt.Printf("parse key %v error : %v\n", string(m.body), e.Error())
		return ""
	}
	return utils.ToString(data["password"])
}

//登陆账号名，仅对登陆消息有效
func (m *Message) UserName() string {
	var data map[string]interface{}
	e := json.Unmarshal(m.body, &data)
	if e != nil {
		fmt.Printf("parse key %v error : %v\n", string(m.body), e.Error())
		return ""
	}
	return utils.ToString(data["username"])
}

//消息体的长度
func (m *Message) Length() (l uint32) {
	b_buf := bytes.NewBuffer(m.head[12:16])
	binary.Read(b_buf, binary.BigEndian, &l)
	return
}

//消息类型
func (m *Message) Type() (t int8) {
	return int8(m.head[16])
}

func (m *Message) SetType(t int8) {
	m.head[16] = byte(t)
}

func (m *Message) Content() []byte {
	return m.body
}

func (m *Message) Tag() []byte {
	return m.tag
}

func (m *Message) String() string {
	return fmt.Sprintf("[%v][%v][%v][%v][%v][%v|%v]", m.ToID(), m.ID(), m.Sender(), m.Length(), m.Type(), string(m.Tag()), string(m.Content()))
}

func (m *Message) SetContent(body []byte, tag []byte) {
	l := uint32(len(body) + len(tag) + 1)
	b_buf := new(bytes.Buffer)
	binary.Write(b_buf, binary.BigEndian, l)
	copy(m.head[12:16], b_buf.Bytes()[0:4])
	m.body = body
	m.tag = tag
}
