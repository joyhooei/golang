package usercontrol

import "yuanfen/yf_service/cls/data_model/user_overview"

func ISetAvatarLevel(uid uint32, istat int) (e error) {
	if istat == -1 {
		if _, e = mdb.Exec("update user_detail set avatarlevel=?,avatar=? where uid=?", istat, "http://image2.yuanfenba.net/oss/other/weishenghe4.png", uid); e != nil {
			return
		}
	} else {
		if _, e = mdb.Exec("update user_detail set avatarlevel=? where uid=?", istat, uid); e != nil {
			return
		}
	}
	user_overview.ClearUserObjects(uid)
	return
}
