package model

import (
	"errors"
	"fmt"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/pkg/format"
)

type HKPOrder struct {
	detail    *Order
	Providers []*HKPOrderProvider
}

func NewHKPOrder() (h *HKPOrder) {
	h = &HKPOrder{}
	h.detail = NewOrder()
	h.detail.Role = common.ROLE_HKP
	h.Providers = make([]*HKPOrderProvider, 0, 6)
	return h
}

func (h *HKPOrder) OrderId() uint64 {
	return h.detail.Id
}
func (h *HKPOrder) SetOrderId(id uint64) (err error) {
	h.detail.Id = id
	for _, p := range h.Providers {
		p.OrderId = id
	}
	return
}

func (h *HKPOrder) Role() uint {
	return h.detail.Role
}

func (h *HKPOrder) Job() uint {
	return h.detail.Job
}
func (h *HKPOrder) SetJobStr(job string) (err error) {
	j, err := format.ParseUint(job)
	if err != nil {
		err = errors.New("convert [" + job + "] to uint error : " + err.Error())
		return
	}
	err = h.SetJobUint(uint(j))
	return
}
func (h *HKPOrder) SetJobUint(job uint) (err error) {
	if job > 3 {
		return errors.New(fmt.Sprintf("unkown job id : [%d]", job))
	}
	h.detail.Job = job
	return
}

func (h *HKPOrder) Customer() common.UIDType {
	return h.detail.Customer
}
func (h *HKPOrder) SetCustomer(c common.UIDType) (err error) {
	h.detail.Customer = c
	return
}

func (h *HKPOrder) Persons() uint8 {
	return h.detail.Persons
}
func (h *HKPOrder) SetPersonsStr(p string) (err error) {
	persons, err := format.ParseUint8(p)
	if err != nil {
		err = errors.New("convert [" + p + "] to uint8 error : " + err.Error())
		return
	}
	err = h.SetPersonsUint(uint8(persons))
	return
}
func (h *HKPOrder) SetPersonsUint(p uint8) (err error) {
	if p > 10 {
		return errors.New(fmt.Sprintf("too many persons : [%d]", p))
	}
	h.detail.Persons = p
	return
}

func (h *HKPOrder) Address() uint64 {
	return h.detail.Address
}
func (h *HKPOrder) SetAddressStr(a string) (err error) {
	addr, err := format.ParseUint64(a)
	if err != nil {
		err = errors.New("convert [" + a + "] to uint64 error : " + err.Error())
		return
	}
	err = h.SetAddressUint(addr)
	return
}
func (h *HKPOrder) SetAddressUint(a uint64) (err error) {
	h.detail.Address = a
	return
}
func (h *HKPOrder) StartTime() time.Time {
	return h.detail.StartTime
}
func (h *HKPOrder) SetStartTimeStr(s string) (err error) {
	st, err := time.ParseInLocation(format.TIME_LAYOUT_1, s, time.Local)
	if err != nil {
		errors.New("convert [" + s + "] to Time error : " + err.Error())
		return
	}
	err = h.SetStartTimeTime(st)
	return
}
func (h *HKPOrder) SetStartTimeTime(s time.Time) (err error) {
	h.detail.StartTime = s
	return
}

func (h *HKPOrder) Phase() uint8 {
	return h.detail.Phase
}
func (h *HKPOrder) SetPhase(p uint8) (err error) {
	if p > 8 || p == 0 {
		return errors.New(fmt.Sprintf("phase [%d] error : 1<=phase<=8", p))
	}
	h.detail.Phase = p
	return
}

func (h *HKPOrder) PhaseTime() time.Time {
	return h.detail.PhaseTime
}
func (h *HKPOrder) SetPhaseTime(p time.Time) (err error) {
	h.detail.PhaseTime = p
	return
}

func (h *HKPOrder) PhaseReason() string {
	return h.detail.PhaseReason
}
func (h *HKPOrder) SetPhaseReason(r string) (err error) {
	if len(r) > 255 {
		err = errors.New("reason string too long, must small than 256.")
		return
	}
	h.detail.PhaseReason = r
	return
}

func (h *HKPOrder) CSR() uint {
	return h.detail.CSR
}
func (h *HKPOrder) SetCSR(c uint) (err error) {
	h.detail.CSR = c
	return
}

func (h *HKPOrder) ServiceStart() time.Time {
	return h.detail.ServiceStart
}
func (h *HKPOrder) SetServiceStartStr(localTime string) (err error) {
	st, err := time.ParseInLocation(format.TIME_LAYOUT_1, localTime, time.Local)
	if err != nil {
		errors.New("convert [" + localTime + "] to Time error : " + err.Error())
		return
	}
	err = h.SetServiceStartTime(st)
	return
}
func (h *HKPOrder) SetServiceStartTime(s time.Time) (err error) {
	h.detail.ServiceStart = s
	return
}

func (h *HKPOrder) Duration() uint {
	return h.detail.Duration
}
func (h *HKPOrder) SetDurationStr(d string) (err error) {
	du, err := format.ParseUint(d)
	if err != nil {
		err = errors.New("convert [" + d + "] to uint error : " + err.Error())
		return
	}
	err = h.SetDurationUint(uint(du))
	return
}
func (h *HKPOrder) SetDurationUint(d uint) (err error) {
	if d < 30 || d > 600 {
		return errors.New(fmt.Sprintf("duration [%d] error : 30<=duration<=600", d))
	}
	h.detail.Duration = d
	return
}

func (h *HKPOrder) Community() uint {
	return h.detail.Community
}
func (h *HKPOrder) SetCommunityStr(d string) (err error) {
	du, err := format.ParseUint(d)
	if err != nil {
		err = errors.New("convert [" + d + "] to uint error : " + err.Error())
		return
	}
	err = h.SetCommunityUint(uint(du))
	return
}
func (h *HKPOrder) SetCommunityUint(d uint) (err error) {
	h.detail.Community = d
	return
}
func (h *HKPOrder) Description() string {
	return h.detail.Description
}
func (h *HKPOrder) SetDescription(d string) (err error) {
	if len(d) > 255 {
		err = errors.New("Description too long. must short than 255")
		return
	}
	h.detail.Description = d
	return
}

func (h *HKPOrder) AddProvider(job *HKPJob, priority uint8, isRec bool) (err error) {
	p := NewHKPOrderProvider()
	p.OrderId = h.OrderId()
	p.Job = *job
	p.Confirm = common.HKP_PROVIDER_CONFIRM_UNKNOWN
	p.Recommend = isRec
	p.AvailableTime = common.InitDate
	p.UserId = job.Uid
	p.Priority = priority
	h.Providers = append(h.Providers, p)
	return
}
