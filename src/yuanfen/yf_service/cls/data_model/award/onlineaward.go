package award

import (
	// "errors"
	// "fmt"
	// "time"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/data_model/coin"
)

// var onlineAwardMap []map[string]interface{}

// func referchMap() {
// 	for {
// 		Now = time.Now().Round(time.Second)
// 		time.Sleep(1000 * 60 * 3 * time.Millisecond)
// 	}
// }
// func loadOnlineAwardMap() (rlist []map[string]interface{}, e error) {
// 	rows, err := mdb.Query("select id,img,info,d from award_package where `type`=1")
// 	if err != nil {
// 		return 0, nil, err
// 	}
// 	defer rows.Close()
// 	for rows.Next() {
// 		var d int
// 		var img, info string
// 		var id int
// 		if err := rows.Scan(&id, &img, &info, &d); err != nil {
// 			return 0, nil, err
// 		}
// 		item := make(map[string]interface{})
// 		item["img"] = img
// 		item["info"] = info
// 		if d == int(utils.Now.Weekday()) {
// 			item["stat"] = 1
// 		} else {
// 			item["stat"] = 0
// 		}
// 		items := make([]map[string]interface{}, 0, 0)
// 		rows2, err2 := mdb.Query("select award_config.id,img,`name`,type,price,virtualtype,virtualcount,info FROM award_package_relation LEFT JOIN award_config on award_config.id= award_package_relation.adard_id where award_package_relation.package_id=?", id)
// 		if err2 != nil {
// 			return 0, nil, err2
// 		}
// 		defer rows2.Close()
// 		for rows2.Next() {
// 			var aid, avirtualtype, atype, avirtualcount, aprice int
// 			var aimg, aname, ainfo string
// 			if err := rows2.Scan(&aid, &aimg, &aname, &atype, &aprice, &avirtualtype, &avirtualcount, &ainfo); err != nil {
// 				return 0, nil, err
// 			}
// 			aitem := make(map[string]interface{})
// 			aitem["img"] = img
// 			aitem["name"] = aname
// 			aitem["vtype"] = virtualtype
// 			items = append(items, aitem)
// 		}
// 		item["items"] = items
// 		result = append(result, item)
// 	}
// 	return
// }

func onlineAwardGeted(uid uint32) (b bool, e error) {
	var tm string
	// fmt.Println(fmt.Sprintf("%v", uid))
	err := mdb.QueryRow("SELECT tm from user_online_award_log where uid=? ORDER BY tm DESC LIMIT 1", uid).Scan(&tm)
	if err != nil {
		return false, nil
	}
	d, _ := utils.ToTime(tm)
	// fmt.Println(fmt.Sprintf("%v,%v", d, tm))
	return d.YearDay() == utils.Now.YearDay(), nil
}

func OnlineAwards(uid uint32) (stat int, result []map[string]interface{}, e error) {
	result = make([]map[string]interface{}, 0, 0)
	if b, err := onlineAwardGeted(uid); err != nil {
		return 0, nil, err
	} else {
		if b {
			return 0, result, nil
		}
	}
	stat = 1
	rows, err := mdb.Query("select id,img,info,d from award_package where `type`=1")
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d int
		var img, info string
		var id int
		if err := rows.Scan(&id, &img, &info, &d); err != nil {
			return 0, nil, err
		}
		item := make(map[string]interface{})
		item["img"] = img
		item["info"] = info
		// if d == int(utils.Now.Weekday()) {
		tt := utils.Now
		if d == int(tt.Weekday()) {
			item["stat"] = 1
		} else {
			item["stat"] = 0
		}
		items := make([]map[string]interface{}, 0, 0)
		rows2, err2 := mdb.Query("select img,`name`,virtualtype FROM award_package_relation LEFT JOIN award_config on award_config.id= award_package_relation.adard_id where award_package_relation.package_id=?", id)
		if err2 != nil {
			return 0, nil, err2
		}
		defer rows2.Close()
		for rows2.Next() {
			var aimg, aname string
			var virtualtype int
			if err := rows2.Scan(&aimg, &aname, &virtualtype); err != nil {
				return 0, nil, err
			}
			aitem := make(map[string]interface{})
			aitem["img"] = img
			aitem["name"] = aname
			aitem["vtype"] = virtualtype
			items = append(items, aitem)
		}
		item["items"] = items
		result = append(result, item)
	}
	return
}

func ReceiveOnlineAward(uid uint32) (icoin int, playcount int, e service.Error) {
	if b, err := onlineAwardGeted(uid); err != nil {
		return 0, 0, service.NewError(service.ERR_INTERNAL, err.Error())
	} else {
		if b {
			return 0, 0, service.NewError(service.ERR_INTERNAL, "你今天已经领取过奖品", "你今天已经领取过奖品")
		}
	}
	_, err := mdb.Exec("insert into user_online_award_log (uid,tm)values(?,?)", uid, utils.Now)
	if err != nil {
		return 0, 0, service.NewError(service.ERR_INTERNAL, err.Error())
	}
	// var gamenum int
	tt := utils.Now
	rows, err2 := mdb.Query("select award_config.type,award_config.price,award_config.virtualtype,award_config.virtualcount,award_config.info from award_package,award_package_relation left JOIN award_config on award_config.id =award_package_relation.adard_id where award_package.id=award_package_relation.package_id and award_package.d=?", int(tt.Weekday()))
	if err2 != nil {
		return 0, 0, service.NewError(service.ERR_INTERNAL, err2.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var itype, price, virtualtype, virtualcount int
		var info string
		if err := rows.Scan(&itype, &price, &virtualtype, &virtualcount, &info); err != nil {
			return 0, 0, service.NewError(service.ERR_INTERNAL, err.Error())
		}
		switch itype {
		case 1: //金币
			if err := coin.UserCoinChange(mdb, uid, 0, coin.EARN_ONLINEAWARD, 0, price, info); err.Code != service.ERR_NOERR {
				return 0, 0, service.NewError(service.ERR_INTERNAL, err.Error())
			}
			icoin = price
		case 4: //虚拟物品
			switch virtualtype {
			case 1: //飞机次数
				//	if err := service_game.UpdateGameNum(mdb, uid, virtualcount, true); err != nil {
				//		return 0, 0, service.NewError(service.ERR_INTERNAL, err.Error())
				//	}

				playcount = virtualcount
			}
		}
	}
	return icoin, playcount, service.NewError(service.ERR_NOERR, "")
}
