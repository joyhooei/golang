package model

import (
	"errors"
	"fmt"
	"time"

	"code.serveyou.cn/common"
)

type HKPOrderProvider struct {
	OrderId       uint64
	UserId        common.UIDType
	Job           HKPJob
	Recommend     bool      //是否是系统推荐的用户
	Priority      uint8     //预约优先级，越小越高
	Confirm       uint8     //是否确认，0-未知，1-是，2-否
	AvailableTime time.Time //可以提供服务的时间
	speed         uint8
	quality       uint8
	attitude      uint8
	comment       string
}

func NewHKPOrderProvider() (h *HKPOrderProvider) {
	return &HKPOrderProvider{0, 0, *NewHKPJob(), false, 0, 0, common.InitDate, 0, 0, 0, ""}
}

func (h *HKPOrderProvider) Speed() uint8 {
	return h.speed
}
func (h *HKPOrderProvider) SetSpeed(s uint8) (err error) {
	if s > common.MAX_RANK_VALUE {
		return errors.New(fmt.Sprintf("invalid value : 0<=speed<=%d", common.MAX_RANK_VALUE))
	}
	h.speed = s
	return
}

func (h *HKPOrderProvider) Quality() uint8 {
	return h.quality
}
func (h *HKPOrderProvider) SetQuality(q uint8) (err error) {
	if q > common.MAX_RANK_VALUE {
		return errors.New(fmt.Sprintf("invalid value : 0<=quality<=%d", common.MAX_RANK_VALUE))
	}
	h.quality = q
	return
}

func (h *HKPOrderProvider) Attitude() uint8 {
	return h.attitude
}
func (h *HKPOrderProvider) SetAttitude(a uint8) (err error) {
	if a > common.MAX_RANK_VALUE {
		return errors.New(fmt.Sprintf("invalid value : 0<=attitude<=%d", common.MAX_RANK_VALUE))
	}
	h.attitude = a
	return
}

func (h *HKPOrderProvider) SetRank(s uint8, q uint8, a uint8) (err error) {
	if err = h.SetSpeed(s); err != nil {
		return
	}
	if err = h.SetQuality(q); err != nil {
		return
	}
	if err = h.SetAttitude(a); err != nil {
		return
	}
	return
}

func (h *HKPOrderProvider) Comment() string {
	return h.comment
}
func (h *HKPOrderProvider) SetComment(c string) (err error) {
	if len(c) > common.MAX_COMMENT_LEN {
		return errors.New(fmt.Sprintf("invalid value : comment length must small than %v", common.MAX_COMMENT_LEN))
	}
	h.comment = c
	return
}
