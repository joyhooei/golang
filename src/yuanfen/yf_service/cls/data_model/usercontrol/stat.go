package usercontrol

import (
	"yuanfen/common/stat"
)

//聊天送礼
func StatGift(uid uint32) (e error) {
	//stat.Append(uid, stat.ACTION_GIFT_SEND, map[string]interface{}{})
	return
}

//女神送礼
func StatGiftPursue(uid uint32) (e error) {
	//stat.Append(uid, stat.ACTION_GIFT_PURSUE, map[string]interface{}{})
	return
}

func StatCharge(uid uint32, money int) (e error) {
	stat.Append(uid, stat.ACTION_CHARGE, map[string]interface{}{"money": money})
	return
}

func StatUserInfo(uid uint32) (e error) {
	// if count, err := getUidInfoGrade(uid); err == nil {

	// 	// stat.Append(uid, stat.ACTION_USER_INFO, map[string]interface{}{"count": count})
	// }
	return
}

func StatUserOnTop(uid uint32) (e error) {
	//stat.Append(uid, stat.ACTION_USER_ONTOP, map[string]interface{}{})
	return
}

func getUidInfoGrade(uid uint32) (count int, e error) {
	var aboutme, job, tag, interest, require, looking, contact string
	var height, star, birthdaystat int
	if err := mdb.QueryRow("select aboutme,height,star,job,tag,interest,`require`,looking,contact,birthdaystat from user_main,user_detail where user_main.uid=? and user_detail.uid=user_main.uid", uid).Scan(&aboutme, &height, &star, &job, &tag, &interest, &require, &looking, &contact, &birthdaystat); err != nil {
		return 0, err
	}

	var photocount int
	if err := mdb.QueryRow("select count(*) from user_photo_album where uid=?", uid).Scan(&photocount); err != nil {
		return 0, err
	}
	if height > 0 {
		count++
	}
	if star > 0 {
		count++
	}
	if birthdaystat == 1 {
		count++
	}

	if aboutme != "" {
		count++
	}
	if job != "" {
		count++
	}

	if tag != "" {
		count++
	}
	if interest != "" {
		count++
	}
	if require != "" {
		count++
	}
	if looking != "" {
		count++
	}
	if contact != "" {
		count++
	}

	if photocount > 3 {
		photocount = 3
	}
	count += photocount
	count += 4
	return
}
