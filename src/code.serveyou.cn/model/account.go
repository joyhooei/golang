package model

import "code.serveyou.cn/common"

type Account struct {
	UserId  common.UIDType
	Balance float64
	Score   uint
	Coupons uint
}

func NewAccount(uid common.UIDType) (a *Account) {
	a = &Account{uid, 0, 0, 0}
	return
}
