package model

import "time"

type Notification struct {
	Id        uint64
	Province  uint
	City      uint
	Community uint
	Title     string
	Pic       string
	Content   string
	Url       string
	Time      time.Time
}
