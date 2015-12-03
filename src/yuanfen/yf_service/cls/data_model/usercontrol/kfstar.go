package usercontrol

import (
	"errors"
	"fmt"
	"yf_pkg/utils"
)

func RefStar(uid uint32) (e error) {
	fmt.Println(fmt.Sprintf("RefStar %v", uid))
	var ErrNoRows = errors.New("sql: no rows in result set")
	var level, certifyphone, certifyvideo int
	var tm string
	e = mdb.QueryRowFromMain("select level,tm,certify_video,phonestat from user_star_level,user_main where user_star_level.uid=? and user_main.uid=?", uid, uid).Scan(&level, &tm, &certifyvideo, &certifyphone)
	if e != nil {
		if e.Error() == ErrNoRows.Error() {
			return nil
		}
		return e
	}
	if level == 0 {
		return
	}
	tm2 := utils.Now.AddDate(0, 0, -7)
	var count int
	e = mdb.QueryRowFromMain("select count(*) from user_online_award_log where uid=? and tm>?", uid, tm2).Scan(&count)
	if e != nil {
		return e
	}
	var rlevel int
	fmt.Println(fmt.Sprintf("RefStar level %v,count %v,certifyphone %v,certifyvideo %v", level, count, certifyphone, certifyvideo))
	if (count >= 5) && ((certifyphone == 1) && (certifyvideo == 1)) {
		rlevel = 3
	} else {
		if (count >= 4) && ((certifyphone == 1) || (certifyvideo == 1)) {
			rlevel = 2
		} else {
			rlevel = 1
		}
	}

	fmt.Println(fmt.Sprintf("RefStar level %v,rlevel %v,count %v,certifyphone %v,certifyvideo %v", level, rlevel, count, certifyphone, certifyvideo))
	if rlevel != level {
		var changes int
		if rlevel > level {
			changes = 1
		} else {
			changes = -1
		}
		_, e = mdb.Exec("update user_star_level set level=?,changes=? where uid=?", rlevel, changes, uid)
	}
	return
}
