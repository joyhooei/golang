package model

import (
	"errors"
	"fmt"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/pkg/format"
)

type VerifyCode struct {
	cellphone string
	code      string
	forWhat   uint8
	timeout   time.Time
}

func NewVerifyCode(cp string, code string, fw uint8, to time.Time) (v *VerifyCode, err error) {
	v = new(VerifyCode)
	if err = v.SetCellphone(cp); err != nil {
		return
	}
	if err = v.SetCode(code); err != nil {
		return
	}
	if err = v.SetForWhat(fw); err != nil {
		return
	}
	if err = v.SetTimeout(to); err != nil {
		return
	}
	return
}

func (v *VerifyCode) Cellphone() string {
	return v.cellphone
}
func (v *VerifyCode) SetCellphone(c string) (err error) {
	if !format.CheckCellphone(c) {
		err = errors.New("cellphone number [" + c + "] format error.")
		return
	}
	v.cellphone = c
	return
}

func (v *VerifyCode) Code() string {
	return v.code
}
func (v *VerifyCode) SetCode(c string) (err error) {
	if len(c) != 6 {
		err = errors.New("verify code length must be 6")
		return
	}
	v.code = c
	return
}

func (v *VerifyCode) ForWhat() uint8 {
	return v.forWhat
}
func (v *VerifyCode) SetForWhat(f uint8) (err error) {
	if f >= common.DB_VTYPE_MAX {
		err = errors.New(fmt.Sprintf("invalid verfiy type [%v]", f))
		return
	}
	v.forWhat = f
	return
}

func (v *VerifyCode) Timeout() time.Time {
	return v.timeout
}
func (v *VerifyCode) SetTimeout(to time.Time) (err error) {
	v.timeout = to
	return
}
