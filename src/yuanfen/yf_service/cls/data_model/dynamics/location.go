package dynamics

import (
	"errors"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/data_model/general"
)

// 获取用户在于uid用户的距离
func GetUserDistance(uid uint32, uids []uint32) (m map[uint32]float64, e error) {
	if len(uids) <= 0 {
		return
	}
	m = make(map[uint32]float64)
	suids := uids
	suids = append(suids, uid)
	rm, e := general.MUserLocation(suids)
	if e != nil {
		return
	}
	myco, ok := rm[uid]
	if !ok {
		e = errors.New("get my Coordinate is error")
		return
	}
	for _, id := range uids {
		var d float64
		if c, ok := rm[id]; ok {
			d = utils.Distence(myco, c)
		}
		m[id] = d
	}
	return
}
