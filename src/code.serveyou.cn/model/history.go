package model

import (
	"time"

	"code.serveyou.cn/common"
)

type History struct {
	customer        common.UIDType
	serviceProvider common.UIDType
	role            uint
	job             uint
	LastTime        time.Time
}

func NewHistory() *History {
	return &History{0, 0, 0, 0, common.InitDate}
}
func (h *History) Customer() common.UIDType {
	return h.customer
}
func (h *History) SetCustomer(cid common.UIDType) (err error) {
	h.customer = cid
	return
}
func (h *History) ServiceProvider() common.UIDType {
	return h.serviceProvider
}
func (h *History) SetServiceProvider(p common.UIDType) (err error) {
	err = common.SetUid(p, &h.serviceProvider)
	return
}
func (h *History) SetServiceProviderStr(p string) (err error) {
	err = common.SetUidStr(p, &h.serviceProvider)
	return
}

func (h *History) Role() uint {
	return h.role
}
func (h *History) SetRole(r uint) (err error) {
	err = common.SetRole(r, &h.role)
	return
}
func (h *History) SetRoleStr(r string) (err error) {
	err = common.SetRoleStr(r, &h.role)
	return
}

func (h *History) Job() uint {
	return h.job
}
func (h *History) SetJob(j uint) (err error) {
	err = common.SetJob(h.role, j, &h.job)
	return
}
func (h *History) SetJobStr(j string) (err error) {
	err = common.SetJobStr(h.role, j, &h.job)
	return
}
