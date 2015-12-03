package model

import (
	"errors"
	"fmt"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/pkg/format"
)

type SearchCondition struct {
	community   uint
	role        uint
	job         uint
	birthplace  []string
	maxBirthday time.Time
	minBirthday time.Time
	start       time.Time
	orderby     string
	desc        bool
	pn          uint
	rn          uint
}

func NewSearchCondition() (s *SearchCondition) {
	s = &SearchCondition{0, 0, 0, make([]string, 0, 3), common.InitDate, common.InitDate, common.InitDate, "", false, 0, 0}
	return
}

func (s *SearchCondition) Community() uint {
	return s.community
}
func (s *SearchCondition) SetCommunityStr(c string) (err error) {
	comm, err := format.ParseUint(c)
	if err != nil {
		return
	}
	s.community = comm
	return
}
func (s *SearchCondition) Role() uint {
	return s.role
}
func (s *SearchCondition) SetRoleStr(r string) (err error) {
	err = common.SetRoleStr(r, &s.role)
	return
}
func (s *SearchCondition) Job() uint {
	return s.job
}
func (s *SearchCondition) SetJobStr(j string) (err error) {
	err = common.SetJobStr(s.role, j, &s.job)
	return
}
func (s *SearchCondition) Desc() bool {
	return s.desc
}
func (s *SearchCondition) SetDescStr(d string) (err error) {
	switch d {
	case "true":
		s.desc = true
	case "false":
		s.desc = false
	default:
		err = errors.New(fmt.Sprintf("invalid desc [%v]", d))
	}
	return
}
func (s *SearchCondition) Orderby() string {
	return s.orderby
}
func (s *SearchCondition) SetOrderby(o string) (err error) {
	s.orderby = o
	return
}
func (s *SearchCondition) Start() time.Time {
	return s.start
}
func (s *SearchCondition) SetStartStr(start string) (err error) {
	s.start, err = time.ParseInLocation(format.TIME_LAYOUT_1, start, time.Local)
	if err != nil {
		errors.New("convert [" + start + "] to Time error : " + err.Error())
		return
	}
	return
}
func (s *SearchCondition) Pn() uint {
	return s.pn
}
func (s *SearchCondition) SetPnStr(pn string) (err error) {
	tmp, err := format.ParseUint(pn)
	if err != nil {
		err = errors.New("parse pn error : " + err.Error())
		return
	}
	s.pn = tmp
	return
}
func (s *SearchCondition) Rn() uint {
	return s.rn
}
func (s *SearchCondition) SetRnStr(rn string) (err error) {
	tmp, err := format.ParseUint(rn)
	if err != nil {
		err = errors.New("parse rn error : " + err.Error())
		return
	}
	s.rn = tmp
	return
}
