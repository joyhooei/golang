package model

import (
	"errors"
	"fmt"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/pkg/format"
)

type AppClient struct {
	userId     common.UIDType
	password   string
	deviceName string
	deviceId   string
	platform   uint8
	LastUpdate time.Time
}

func NewAppClient(uid common.UIDType) (a *AppClient) {
	a = &AppClient{uid, "", "", "", 0, common.InitDate}
	return
}

func (a *AppClient) UserId() common.UIDType {
	return a.userId
}

func (a *AppClient) Password() string {
	return a.password
}
func (a *AppClient) SetPassword(p string) (err error) {
	if !format.CheckPassword(p) {
		err = errors.New("password [" + p + "] invalid.")
		return
	}
	a.password = p
	return
}

func (a *AppClient) DeviceName() string {
	return a.deviceName
}
func (a *AppClient) SetDeviceName(d string) (err error) {
	if len(d) > 64 {
		err = errors.New("device name [" + d + "] too long.")
		return
	}
	a.deviceName = d
	return
}

func (a *AppClient) DeviceId() string {
	return a.deviceId
}
func (a *AppClient) SetDeviceId(d string) (err error) {
	if len(d) > 128 {
		err = errors.New("device id [" + "] too long.")
		return
	}
	a.deviceId = d
	return
}

func (a *AppClient) Platform() uint8 {
	return a.platform
}
func (a *AppClient) SetPlatform(p uint8) (err error) {
	if p >= common.PLATFORM_MAX {
		err = errors.New(fmt.Sprintf("invalid platform id [%v]", p))
		return
	}
	a.platform = p
	return
}
func (a *AppClient) SetPlatformStr(p string) (err error) {
	tmp, err := format.ParseUint8(p)
	if err != nil {
		return
	}
	err = a.SetPlatform(tmp)
	return
}
