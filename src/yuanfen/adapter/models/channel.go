package models

import "net"

import "pkg/yh_net"

type Channel struct {
	ID   uint32
	Ch   chan Msg
	IP   []byte
	conn *yh_net.TCPConn
	Key  uint32
}

func NewChannel(id uint32, conn *yh_net.TCPConn, bufSize uint, key uint32) (c *Channel) {
	c = new(Channel)
	c.Ch = make(chan Msg, bufSize)
	c.IP = make([]byte, 4)
	c.ID = id
	c.conn = conn
	c.Key = key
	if conn != nil {
		copy(c.IP, []byte(net.ParseIP(conn.ClientIP()))[12:])
	}
	return c
}

func (c *Channel) Conn() *yh_net.TCPConn {
	return c.conn
}

func (c *Channel) SetConn(conn *yh_net.TCPConn) {
	c.conn = conn
}

func (c *Channel) Close() {
	close(c.Ch)
	c.conn.Close()
}
