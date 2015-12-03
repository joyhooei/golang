package relation

import (
	"time"
	"yf_pkg/utils"
)

type UserScore struct {
	Uid   uint32
	Score int64
}

type User struct {
	Uid      uint32    `json:"uid"`
	Nickname string    `json:"nickname"`
	Avatar   string    `json:"avatar"`
	Tag      uint16    `json:"tag"`       //标记的用户类型
	IsFriend bool      `json:"is_friend"` //是否是认识的人
	Tm       time.Time `json:"tm"`
}

type FollowTarget struct {
	Uid uint32    `json:"uid"`
	Tm  time.Time `json:"tm"`  //标记的时间
	Tag uint16    `json:"tag"` //标记的类型
}

type BlackUser struct {
	Uid      uint32    `json:"uid"`
	Nickname string    `json:"nickname"`
	Avatar   string    `json:"avatar"`
	Tm       time.Time `json:"tm"`
}

type Friend struct {
	Uid      uint32    `json:"uid"`
	Nickname string    `json:"nickname"`
	Avatar   string    `json:"avatar"`
	Tag      uint16    `json:"tag"` //标记的用户类型
	Tm       time.Time `json:"tm"`
}

type SayHelloUser struct {
	Uid        uint32   `json:"uid"`
	Nickname   string   `json:"nickname"`
	Avatar     string   `json:"avatar"`
	Age        int      `json:"age"`
	Height     int      `json:"height"`
	WorkUnit   string   `json:"workunit"` //工作单位
	DyNum      uint64   `json:"dynum"`    //动态总数
	Dynamics   []string `json:"dynamics"` //动态
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Connection []string `json:"connection"` //与自己的关系
	Distence   float64  `json:"distence"`   //距离（米）
}

type UserWorkPlace struct {
	Uid      uint32           `json:"uid"`
	Nickname string           `json:"nickname"`
	Gender   int              `json:"gender"`
	Avatar   string           `json:"avatar"`
	Location utils.Coordinate `json:"location"`
}

type Place struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Pic      string  `json:"pic"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Distence float64 `json:"distence"`
}

type PlaceItems []Place

func (i PlaceItems) Len() int {
	return len(i)
}

func (items PlaceItems) Less(i, j int) bool {
	return items[i].Distence < items[j].Distence
}

func (items PlaceItems) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
}

func (h *PlaceItems) Push(x interface{}) {
	*h = append(*h, x.(Place))
}

func (h *PlaceItems) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
