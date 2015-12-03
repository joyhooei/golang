package user_overview

import (
	"time"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/data_model/general"
)

const (
	TIMEOUT_MAX = 72 //最长超时时间不超过72小时
)

type Localtag struct {
	Content string              `json:"content"` //本地标签内容
	Tm      time.Time           `json:"tm"`      //发布时间
	Timeout time.Time           `json:"timeout"` //超时时间
	Req     general.Requirement `json:"req"`     //用户限制条件
	HasReq  bool                `json:"has_req"` //是否有限制条件
}

func NewLocaltag() (lt Localtag) {
	lt.Req = general.NewRequirement()
	lt.Timeout = utils.Now.Add(-24 * time.Hour)
	lt.HasReq = false
	return
}
