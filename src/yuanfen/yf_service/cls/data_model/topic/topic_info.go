package topic

import (
	"math"
	"time"
	"yf_pkg/utils"
)

type TopicInfo struct {
	Uid           uint32        `json:"uid"`
	Tid           uint32        `json:"tid"`
	Nickname      string        `json:"nickname"`
	Gender        int           `json:"gender"`
	Age           int           `json:"age"`
	Avatar        string        `json:"avatar"`
	Province      string        `json:"province"`
	Title         string        `json:"title"`
	Tag           string        `json:"tag"`
	Pics          string        `json:"pics"`
	PicsLevel     int8          `json:"pics_level"`
	InRoom        bool          `json:"-"`
	IsAdmin       bool          `json:"admin"`
	OnlineTimeout time.Time     `json:"-"`
	Trend         string        `json:"trend"`
	TrendValue    int           `json:"-"`
	Capacity      uint32        `json:"capacity"`
	Online        uint32        `json:"online"`
	RealOnline    uint32        `json:"real_online"`
	Lat           float64       `json:"lat"`
	Lng           float64       `json:"lng"`
	Tm            time.Time     `json:"tm"`
	Birthday      time.Time     `json:"birthday"`
	Messages      []interface{} `json:"messages"`
	Priority      int           `json:"priority"`
	Score         int           `json:"score"` //越低越靠前
	ScoreDistence float64       `json:"score_distence"`
	ScoreAge      float64       `json:"score_age"`
	ScoreOnline   float64       `json:"score_online"`
	ScoreInromm   float64       `json:"score_inroom"`
	ScorePics     float64       `json:"score_pics"`
	ScoreTrend    float64       `json:"score_trend"`
}

func DiscovScore(i *TopicInfo, base *TopicInfo) {
	if i.Priority > 0 {
		i.Score = math.MinInt32/2 - i.Priority
		return
	}
	if i.Uid == base.Uid {
		i.Score = math.MinInt32
		return
	}
	//0 ~ 20
	distence := utils.Distence(utils.Coordinate{i.Lat, i.Lng}, utils.Coordinate{base.Lat, base.Lng})
	distence /= 10000
	//0 ~ -110
	//	ageDistence := -math.Abs(i.Birthday.Sub(base.Birthday).Hours()/24) / 20
	ageDistence := 0.0
	//0 | -100
	inroom := 0.0
	if i.InRoom {
		inroom = -100
	}
	//-40 ~ 0,1000
	onlineTime := utils.Now.Sub(i.OnlineTimeout).Minutes()
	if onlineTime > 0 {
		onlineTime = 1000
	}
	trend := -float64(i.TrendValue) * 10
	if trend < -200 {
		trend = -200
	}
	//图片质量: -200 ~ 0
	pics := float64(-int(i.PicsLevel) * 20)
	//fmt.Printf("tid=%v, distence=%v, ageDistence=%v, inroom=%v, onlineTime=%v, pics=%v\n", i.Tid, distence, ageDistence, inroom, onlineTime, pics)
	i.ScoreAge, i.ScoreInromm, i.ScoreOnline, i.ScoreDistence, i.ScorePics, i.ScoreTrend = ageDistence, inroom, onlineTime, distence, pics, trend
	i.Score = int(distence + ageDistence + inroom + onlineTime + pics + trend)
}

type TopicItems []TopicInfo

func (i TopicItems) Len() int {
	return len(i)
}

func (items TopicItems) Less(i, j int) bool {
	return items[i].Score < items[j].Score
}

func (items TopicItems) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
}
