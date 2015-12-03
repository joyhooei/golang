package model

import (
	"time"

	"code.serveyou.cn/common"
)

type Order struct {
	Id           uint64
	Role         uint
	Job          uint
	Customer     common.UIDType
	Address      uint64
	Community    uint
	Persons      uint8
	StartTime    time.Time
	Phase        uint8
	PhaseTime    time.Time
	PhaseReason  string
	CSR          uint
	ServiceStart time.Time
	Duration     uint
	Description  string
}

func NewOrder() (o *Order) {
	return &Order{0, 0, 0, 0, 0, 0, 0, common.InitDate, 0, common.InitDate, "", 0, common.InitDate, 0, ""}
}
