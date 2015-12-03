package usercontrol

import (
	"math/rand"
	"time"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
)

var infomap map[string][]uint32  //男机器人
var infomap2 map[string][]uint32 //全部机器人
var infomap3 map[string][]uint32 //女机器人

//初始化
func InitAllot() (e error) {
	rows, err := mdb.Query("select user_detail.uid,province,city,user_main.gender from manager_users left join user_main on user_main.uid=manager_users.uid left join user_detail on user_detail.uid=manager_users.uid")
	if err != nil {
		return err
	}
	defer rows.Close()
	infomap = make(map[string][]uint32)
	infomap2 = make(map[string][]uint32)
	infomap3 = make(map[string][]uint32)
	for rows.Next() {
		var uid uint32
		var province, city string
		var gender int
		if err := rows.Scan(&uid, &province, &city, &gender); err != nil {
			return err
		}
		{
			var arr []uint32
			if a, ok := infomap2[province]; ok {
				arr = a
			} else {
				arr = make([]uint32, 0, 0)
			}
			arr = append(arr, uid)
			infomap2[province] = arr

			if a, ok := infomap2[city]; ok {
				arr = a
			} else {
				arr = make([]uint32, 0, 0)
			}
			arr = append(arr, uid)
			infomap2[city] = arr
		}
		switch gender {
		case 1:
			var arr []uint32
			if a, ok := infomap[province]; ok {
				arr = a
			} else {
				arr = make([]uint32, 0, 0)
			}
			arr = append(arr, uid)
			infomap[province] = arr

			if a, ok := infomap[city]; ok {
				arr = a
			} else {
				arr = make([]uint32, 0, 0)
			}
			arr = append(arr, uid)
			infomap[city] = arr
		case 2:
			var arr []uint32
			if a, ok := infomap3[province]; ok {
				arr = a
			} else {
				arr = make([]uint32, 0, 0)
			}
			arr = append(arr, uid)
			infomap3[province] = arr

			if a, ok := infomap3[city]; ok {
				arr = a
			} else {
				arr = make([]uint32, 0, 0)
			}
			arr = append(arr, uid)
			infomap3[city] = arr
		}
	}
	// InitAllot2()
	return
}

// func InitAllot2() (e error) {
// 	rows, err := mdb.Query("select user_detail.uid,province,city from manager_users left join user_main on user_main.uid=manager_users.uid left join user_detail on user_detail.uid=manager_users.uid")
// 	if err != nil {
// 		return err
// 	}
// 	defer rows.Close()
// 	infomap2 = make(map[string][]uint32)
// 	for rows.Next() {
// 		var uid uint32
// 		var province, city string
// 		if err := rows.Scan(&uid, &province, &city); err != nil {
// 			return err
// 		}
// 		var arr []uint32
// 		if a, ok := infomap2[province]; ok {
// 			arr = a
// 		} else {
// 			arr = make([]uint32, 0, 0)
// 		}
// 		arr = append(arr, uid)
// 		infomap2[province] = arr

// 		if a, ok := infomap2[city]; ok {
// 			arr = a
// 		} else {
// 			arr = make([]uint32, 0, 0)
// 		}
// 		arr = append(arr, uid)
// 		infomap2[city] = arr
// 	}
// 	return
// }

// func InitAllot3() (e error) {
// 	rows, err := mdb.Query("select user_detail.uid,province,city from manager_users left join user_main on user_main.uid=manager_users.uid left join user_detail on user_detail.uid=manager_users.uid")
// 	if err != nil {
// 		return err
// 	}
// 	defer rows.Close()
// 	infomap3 = make(map[string][]uint32)
// 	for rows.Next() {
// 		var uid uint32
// 		var province, city string
// 		if err := rows.Scan(&uid, &province, &city); err != nil {
// 			return err
// 		}
// 		var arr []uint32
// 		if a, ok := infomap3[province]; ok {
// 			arr = a
// 		} else {
// 			arr = make([]uint32, 0, 0)
// 		}
// 		arr = append(arr, uid)
// 		infomap3[province] = arr

// 		if a, ok := infomap3[city]; ok {
// 			arr = a
// 		} else {
// 			arr = make([]uint32, 0, 0)
// 		}
// 		arr = append(arr, uid)
// 		infomap3[city] = arr
// 	}
// 	return
// }

func getRanUid(ulist []uint32) uint32 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	i := r.Intn(len(ulist))
	return ulist[i]
}

//分配客服给女用户
func AllotUid(province, city string) (uid uint32) {
	if k, ok := infomap[city]; ok {
		return getRanUid(k)
	} else {
		if k2, ok := infomap[province]; ok {
			return getRanUid(k2)
		} else {
			if k3, ok := infomap["北京市"]; ok {
				return getRanUid(k3)
			}
		}
	}
	// fmt.Println(fmt.Sprintf("AllotUid %v,%v,%v", province, city, uid))
	return
}

//分配客服给一般用户
func AllotUid2(province, city string) (uid uint32) {
	if k, ok := infomap2[city]; ok {
		return getRanUid(k)
	} else {
		if k2, ok := infomap2[province]; ok {
			return getRanUid(k2)
		} else {
			if k3, ok := infomap2["北京市"]; ok {
				return getRanUid(k3)
			}
		}
	}
	// fmt.Println(fmt.Sprintf("AllotUid %v,%v,%v", province, city, uid))
	return
}

//分配客服给男用户
func AllotUid3(province, city string) (uid uint32) {
	if k, ok := infomap3[city]; ok {
		return getRanUid(k)
	} else {
		if k2, ok := infomap3[province]; ok {
			return getRanUid(k2)
		} else {
			if k3, ok := infomap3["北京市"]; ok {
				return getRanUid(k3)
			}
		}
	}
	// fmt.Println(fmt.Sprintf("AllotUid %v,%v,%v", province, city, uid))
	return
}

//是否已经分配过机器人
func IfUidAllot(uid uint32) bool {
	_, err := redis.String(cache.Get(redis_db.CACHE_USER_ALLOT, uid))
	switch err {
	case nil:
		return true
	case redis.ErrNil:
		cache.SetEx(redis_db.CACHE_USER_ALLOT, uid, 3600*72, utils.Now.Unix())
		return false
	default:
		return false
	}

}

//根据注册时间判断 是否要自动回复
func RegTimeCheck(uid uint32) int {
	i, err := redis.Int64(rdb.Get(redis_db.REDIS_REG_TIME, uid))
	switch err {
	case nil: //已注册
		t := time.Unix(i, 0)
		if t.YearDay() == utils.Now.YearDay() {
			if utils.Now.Hour()-t.Hour() >= 5 {
				rdb.Set(redis_db.REDIS_REG_TIME, uid, utils.Now.Unix())
				return 0
			} else {
				return -1
			}
		} else {
			return 1
		}
	case redis.ErrNil:
		return -1 //没注册
	default:
		return -1 //当天注册
	}
	return 0
}

//根据注册时间判断 是否要自动回复
func WriteRegTime(uid uint32) {
	rdb.SetEx(redis_db.REDIS_REG_TIME, uid, 3600*24*30, utils.Now.Unix())
}
