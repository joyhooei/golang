package models

import (
	"bytes"
	"encoding/binary"
)

/*******消息格式*******

|---消息体长度(4字节)---|---ClientID(4字节)---|---消息类型(2字节)---|---消息体（最长2M)---|

**********************/

const HEARTBEAT_MSG = 65535

type Msg []byte

func NewMsg(length uint32, capacity ...uint32) (m Msg) {
	if len(capacity) > 0 {
		m = make([]byte, length+10, capacity[0]+10)
	} else {
		m = make([]byte, length+10)
	}
	m.SetLength(length)
	return
}

//消息体的长度
func (m *Msg) Length() (l uint32) {
	b_buf := bytes.NewBuffer([]byte(*m)[0:4])
	binary.Read(b_buf, binary.BigEndian, &l)
	return
}
func (m *Msg) SetLength(length uint32) {
	b_buf := new(bytes.Buffer)
	binary.Write(b_buf, binary.BigEndian, length)
	copy([]byte(*m)[0:4], b_buf.Bytes()[0:4])
}

//客户ID
func (m *Msg) ID() (id uint32) {
	b_buf := bytes.NewBuffer([]byte(*m)[4:8])
	binary.Read(b_buf, binary.BigEndian, &id)
	return
}
func (m *Msg) SetID(id uint32) {
	b_buf := new(bytes.Buffer)
	binary.Write(b_buf, binary.BigEndian, uint32(id))
	copy([]byte(*m)[4:8], b_buf.Bytes()[0:4])
}

//获取心跳key
func (m *Msg) HeartBeatContent() (oldKey uint32, newKey uint32) {
	b_buf := bytes.NewBuffer([]byte(*m)[10:18])
	binary.Read(b_buf, binary.BigEndian, &oldKey)
	binary.Read(b_buf, binary.BigEndian, &newKey)
	return
}

//消息类型
func (m *Msg) Type() (t uint16) {
	b_buf := bytes.NewBuffer([]byte(*m)[8:10])
	binary.Read(b_buf, binary.BigEndian, &t)
	return
}
func (m *Msg) SetType(t uint16) {
	b_buf := new(bytes.Buffer)
	binary.Write(b_buf, binary.BigEndian, uint16(t))
	copy([]byte(*m)[8:10], b_buf.Bytes()[0:2])
}

//消息内容
func (m *Msg) Content() []byte {
	return []byte(*m)[10:]
}

//消息头
func (m *Msg) Header() []byte {
	return []byte(*m)[0:10]
}

//原始Slice
func (m *Msg) Append(b ...byte) {
	*m = append([]byte(*m), b...)
}
