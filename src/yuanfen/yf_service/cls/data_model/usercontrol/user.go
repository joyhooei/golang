package usercontrol

import "time"

type User struct {
	Uid      uint32    `json:"uid"`
	Nickname string    `json:"nickname"`
	Avatar   string    `json:"avatar"`
	Tm       time.Time `json:"tm"`
}
