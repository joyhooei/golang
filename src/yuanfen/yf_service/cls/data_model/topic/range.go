package topic

import (
	"fmt"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/data_model/general"
)

type RangeLevel struct {
	Id     int
	Name   string
	Radius uint32 //半径，公里
}

type RangeLevels map[int]RangeLevel

func (rs *RangeLevels) NextLevel(rid int, direction string, lat, lng float64) (rl RangeLevel) {
	var nrid int
	var tmpMax = RANGE_SECOND_MAX
	switch direction {
	case "+":
		nrid = rid + 1
		tmpMax = RANGE_MAX
	case "-":
		nrid = rid - 1
	default:
		nrid = rid
	}
	rl, ok := (*rs)[nrid]
	if !ok {
		if nrid < RANGE_MAX && nrid > RANGE_SECOND_MAX {
			rl = (*rs)[tmpMax]
		} else if nrid > RANGE_MAX {
			rl = (*rs)[RANGE_MAX]
		} else if nrid < RANGE_MIN {
			rl = (*rs)[RANGE_MIN]
		} else {
			//默认返回城市半径
			rl = (*rs)[RANGE_DEFAULT]
		}
	}
	rl = rs.Name(rl, lat, lng)
	return
}

func (rs *RangeLevels) Name(rl RangeLevel, lat, lng float64) RangeLevel {
	rl.Name = "当前正在查看"
	if rl.Id == RANGE_MAX {
		rl.Name += "【全国】范围的话题"
		return rl
	}
	rcon := rdb.GetReadConnection(redis_db.REDIS_GEO_CITY)
	defer rcon.Close()
	city, province, err := general.City(lat, lng)
	if err != nil {
		return rl
	}
	switch rl.Id {
	case RANGE_CITY:
		rl.Name += fmt.Sprintf("【%v】范围的话题", rl.Radius, city)
	case RANGE_PROVINCE:
		rl.Name += fmt.Sprintf("【%v】范围的话题", rl.Radius, province)
	default:
		rl.Name += fmt.Sprintf("【%v】范围的话题", province)
	}
	return rl
}

func NewRangeLevels() (RangeLevels, error) {
	sql := "select id,name,radius from range_level"
	rows, e := mdb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	r := make(map[int]RangeLevel)
	for rows.Next() {
		var rl RangeLevel
		if e = rows.Scan(&rl.Id, &rl.Name, &rl.Radius); e != nil {
			return nil, e
		}
		r[rl.Id] = rl
	}
	return r, nil
}
