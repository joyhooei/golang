package discovery

import (
	"time"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

type ReasonObj struct {
	Type int    `json:"type"`
	Text string `json:"text"`
}

type RecUser struct {
	Uid       uint32                     `json:"uid"`
	Nickname  string                     `json:"nickname"`
	Age       int                        `json:"age"`
	Avatar    string                     `json:"avatar"`
	PhotoList []*user_overview.PhotoItem `json:"photos"`
	Height    int                        `json:"height"`
	Job       string                     `json:"job"`
	WorkUnit  string                     `json:"workunit"`
	Follow    bool                       `json:"follow"` //是否已标记
	Reason    ReasonObj                  `json:"reason"`
}

type SearchUser struct {
	Uid           uint32    `json:"uid"`
	Nickname      string    `json:"nickname"`
	Gender        int       `json:"gender"`
	Age           int       `json:"age"`
	Height        int       `json:"height"`
	Avatar        string    `json:"avatar"`
	City          string    `json:"city"`
	Job           string    `json:"job"`
	AboutMe       string    `json:"aboutme"`
	OnlineTimeout time.Time `json:"online_timeout"`
}

type AdjUser struct {
	Id            uint32                     `json:"uid"`
	Nickname      string                     `json:"nickname"`
	Gender        int                        `json:"gender"`
	Age           int                        `json:"age"`
	Avatar        string                     `json:"avatar"`
	PhotoList     []*user_overview.PhotoItem `json:"photos"`
	Height        int                        `json:"height"`
	Building      string                     `json:"building"`
	OnlineTimeout time.Time                  `json:"online_timeout"`
	Lat           float64                    `json:"lat"`
	Lng           float64                    `json:"lng"`
	AboutMe       string                     `json:"aboutme"`
	Distence      float64                    `json:"distence"`
}

type UserItems []AdjUser

func (i UserItems) Len() int {
	return len(i)
}

func (items UserItems) Less(i, j int) bool {
	return items[i].Distence < items[j].Distence
}

func (items UserItems) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
}
