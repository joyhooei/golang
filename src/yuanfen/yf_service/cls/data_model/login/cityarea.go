package login

import (
// "database/sql"
// "errors"
// "fmt"
// "math/rand"
// "time"
// "yf_pkg/encrypt"
// "yf_pkg/mysql"
// "yf_pkg/redis"
// "yf_pkg/service"
// "yf_pkg/utils"
// "yuanfen/yf_service/cls"
// "yuanfen/yf_service/cls/data_model/user_overview"
// "yuanfen/yf_service/cls/data_model/usercontrol"
)

type area struct {
	x, y float64
}

var areamap map[string]*area

func GetArea(province, city string) (x, y float64) {
	if k, ok := areamap[city]; ok {
		x = k.x
		y = k.y
	} else {
		if k2, ok := areamap[province]; ok {
			x = k2.x
			y = k2.y
		} else {
			if k3, ok := areamap["北京市"]; ok {
				x = k3.x
				y = k3.y
			}
		}
	}
	// fmt.Println(fmt.Sprintf("Getarea %v,%v", x, y))
	return
}

func InitCityArea() (e error) {
	rows, err := mdb.Query("select x,y,province,city from user_province_area")
	if err != nil {
		return err
	}
	defer rows.Close()
	areamap = make(map[string]*area)
	for rows.Next() {
		var x, y float64
		var province, city string
		if err := rows.Scan(&x, &y, &province, &city); err != nil {
			return err
		}
		var p *area
		p = new(area)
		p.x = x
		p.y = y
		areamap[city] = p
		areamap[province] = p
	}
	return
}
