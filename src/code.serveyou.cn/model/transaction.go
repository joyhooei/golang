package model

import (
	"time"

	"code.serveyou.cn/common"
)

type Transaction struct {
	Id         uint64
	Type       uint16
	Time       time.Time
	UserId     common.UIDType
	OrderId    uint64
	AddScore   int
	AddBalance int
	AddCoupons int
	Score      int
	Balance    float32
	Coupons    int
	Desc       string
}

func (t *Transaction) Title() (title string) {
	switch t.Type {
	case common.TRANS_REG:
		return "注册奖励"
	case common.TRANS_INVITE:
		return "邀请朋友注册奖励"
	case common.TRANS_FIRST_ORDER:
		return "首次下单成功奖励"
	case common.TRANS_EACH_ORDER:
		return "下单成功奖励"
	case common.TRANS_BUY_GOODS:
		return "购买商品"
	default:
		return "未知"
	}
}

type Goods struct {
	Id      uint
	Name    string
	Pic     string
	Price   float32
	Detail  string
	Explain string
}
