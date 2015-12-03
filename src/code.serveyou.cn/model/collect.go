package model

import (
	"errors"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/pkg/format"
)

type Collection struct {
	customer        common.UIDType
	serviceProvider common.UIDType
	role            uint
	job             uint
	CollectTime     time.Time
}

func NewCollection() *Collection {
	return &Collection{0, 0, 0, 0, common.InitDate}
}

func (c *Collection) Customer() common.UIDType {
	return c.customer
}
func (c *Collection) SetCustomer(cid common.UIDType) (err error) {
	c.customer = cid
	return
}
func (c *Collection) SetCustomerStr(cid string) (err error) {
	co, err := format.ParseUint64(cid)
	if err != nil {
		err = errors.New("convert [" + cid + "] to uint64 error : " + err.Error())
		return
	}
	err = c.SetCustomer(common.UIDType(co))
	return
}

func (c *Collection) ServiceProvider() common.UIDType {
	return c.serviceProvider
}
func (c *Collection) SetServiceProvider(p common.UIDType) (err error) {
	err = common.SetUid(p, &c.serviceProvider)
	return
}
func (c *Collection) SetServiceProviderStr(p string) (err error) {
	err = common.SetUidStr(p, &c.serviceProvider)
	return
}

func (c *Collection) Role() uint {
	return c.role
}
func (c *Collection) SetRole(r uint) (err error) {
	err = common.SetRole(r, &c.role)
	return
}
func (c *Collection) SetRoleStr(r string) (err error) {
	err = common.SetRoleStr(r, &c.role)
	return
}

func (c *Collection) Job() uint {
	return c.job
}
func (c *Collection) SetJob(j uint) (err error) {
	err = common.SetJob(c.role, j, &c.job)
	return
}
func (c *Collection) SetJobStr(j string) (err error) {
	err = common.SetJobStr(c.role, j, &c.job)
	return
}
