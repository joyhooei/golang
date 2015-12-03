package net

import (
	"net"
	"strings"
	"time"
	"yf_pkg/utils"
)

type TCPConn struct {
	conn         *net.TCPConn
	readTimeout  uint
	writeTimeout uint
}

type TCPListener net.TCPListener

var zeroTime time.Time

func Connect(ip string, port int) (conn *TCPConn, err error) {
	conn = new(TCPConn)
	addr := &net.TCPAddr{net.ParseIP(ip), port, ""}
	c, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	conn.conn = c
	conn.readTimeout = 0
	conn.writeTimeout = 0
	return
}

func Listen(ip string, port int) (listener *TCPListener, err error) {
	ln, err := net.ListenTCP("tcp", &net.TCPAddr{net.ParseIP(ip), port, ""})
	if err != nil {
		return nil, err
	}
	listener = (*TCPListener)(ln)
	return listener, err
}

func (l *TCPListener) Accept() (conn *TCPConn, err error) {
	c, err := (*net.TCPListener)(l).AcceptTCP()
	if err != nil {
		return nil, err
	}
	conn = new(TCPConn)
	conn.conn = c
	return
}

func (conn *TCPConn) SetTimeout(to uint) {
	conn.readTimeout = to
	conn.writeTimeout = to
	if to == 0 {
		conn.conn.SetDeadline(zeroTime)
	} else {
		conn.conn.SetDeadline(utils.Now.Add(time.Duration(to) * time.Second))
	}
}
func (conn *TCPConn) SetReadTimeout(to uint) {
	conn.readTimeout = to
	if to == 0 {
		conn.conn.SetReadDeadline(zeroTime)
	}
}
func (conn *TCPConn) SetWriteTimeout(to uint) {
	conn.writeTimeout = to
	if to == 0 {
		conn.conn.SetWriteDeadline(zeroTime)
	}
}

func (conn *TCPConn) WriteSafe(data []byte) (err error) {
	start := 0
	length := len(data)
	for length > 0 {
		if conn.writeTimeout > 0 {
			conn.conn.SetWriteDeadline(utils.Now.Add(time.Duration(conn.writeTimeout) * time.Second))
		}
		l, err := conn.conn.Write(data[start:])
		if err != nil {
			return err
		}
		length -= l
		start += l
	}
	return
}
func (conn *TCPConn) ReadSafe(data []byte) (err error) {
	start := 0
	length := len(data)
	for length > 0 {
		if conn.readTimeout > 0 {
			conn.conn.SetReadDeadline(utils.Now.Add(time.Duration(conn.readTimeout) * time.Second))
		}
		l, err := conn.conn.Read(data[start:])
		if err != nil {
			return err
		}
		length -= l
		start += l
	}
	return
}

func (conn *TCPConn) Close() (err error) {
	err = conn.conn.Close()
	return
}

func (conn *TCPConn) ClientIP() string {
	return strings.Split(conn.conn.RemoteAddr().String(), ":")[0]
}
func (conn *TCPConn) RemoteAddr() string {
	return conn.conn.RemoteAddr().String()
}
