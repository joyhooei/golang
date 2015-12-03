package model

import (
	"database/sql"
	"errors"
	"time"

	"code.serveyou.cn/common"
)

type Province struct {
	Id     uint8
	Name   string
	Domain string
}

type City struct {
	Id       uint
	Name     string
	Lat      float32
	Lng      float32
	EName    string
	Province uint8
}

type Community struct {
	Id      uint
	lat     float32
	lng     float32
	name    string
	address string
	City    uint
}

func NewCommunity() (c *Community) {
	c = &Community{0, 0, 0, "", "", 0}
	return
}

func (c *Community) Latitude() float32 {
	return c.lat
}
func (c *Community) SetLatitude(lat float32) (err error) {
	if lat > 90 || (lat < -90) {
		err = errors.New("latitude out of range -90 and 90")
		return
	}
	c.lat = lat
	return
}

func (c *Community) Longitude() float32 {
	return c.lng
}
func (c *Community) SetLongitude(lng float32) (err error) {
	if lng > 180 || (lng < -180) {
		err = errors.New("longitude out of range -180 and 180")
		return
	}
	c.lng = lng
	return
}

func (c *Community) Name() string {
	return c.name
}
func (c *Community) SetName(n string) (err error) {
	if len(n) > 255 {
		err = errors.New("name [" + n + "] too long.")
		return
	}
	c.name = n
	return
}

func (c *Community) Address() string {
	return c.address
}
func (c *Community) SetAddress(a string) (err error) {
	if len(a) > 255 {
		err = errors.New("address [" + a + "] too long.")
		return
	}
	c.address = a
	return
}

type Address struct {
	userId    common.UIDType
	AddrId    uint64
	Community uint
	addr      string
	status    uint8
	LastUse   time.Time
}

func NewAddress(uid common.UIDType) (a *Address) {
	a = &Address{uid, 0, 0, "", 0, common.InitDate}
	return
}

func (a *Address) UserId() common.UIDType {
	return a.userId
}

func (a *Address) Addr() string {
	return a.addr
}
func (a *Address) SetAddr(address string) (err error) {
	if len(address) > 255 {
		err = errors.New("address [" + address + "] too long.")
		return
	}
	a.addr = address
	return
}

func (a *Address) Status() uint8 {
	return a.status
}
func (a *Address) SetStatus(s uint8) (err error) {
	if s > 1 {
		err = errors.New("status must be 0 or 1.")
		return
	}
	a.status = s
	return
}

type CityMap map[uint]City
type CommunityMap map[uint]Community

func (c *CityMap) Init(db *sql.DB) (err error) {
	stmt, err := db.Prepare(common.SQL_InitCityMap)
	if err != nil {
		return
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		return
	}
	defer rows.Close()

	var city City
	for rows.Next() {
		if err = rows.Scan(&city.Id, &city.Name); err != nil {
			return
		}
		(*c)[city.Id] = city
	}
	return
}

func (c *CommunityMap) Init(db *sql.DB) (err error) {
	stmt, err := db.Prepare(common.SQL_InitCommunityMap)
	if err != nil {
		return
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		return
	}
	defer rows.Close()

	var comm Community
	var lat, lng float32
	var name, addr string
	for rows.Next() {
		if err = rows.Scan(&comm.Id, &comm.City, &lat, &lng, &name, &addr); err != nil {
			return
		}
		if err = comm.SetLatitude(lat); err != nil {
			return
		}
		if err = comm.SetLongitude(lng); err != nil {
			return
		}
		if err = comm.SetName(name); err != nil {
			return
		}
		if err = comm.SetAddress(addr); err != nil {
			return
		}
		(*c)[comm.Id] = comm
	}
	return
}
