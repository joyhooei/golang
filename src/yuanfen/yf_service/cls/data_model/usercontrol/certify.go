package usercontrol

import (
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/certify"
	"yuanfen/yf_service/cls/data_model/coin"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/notify"
)

// 进行身份证验证
func DoIdCardCertify(uid uint32, id, name string) (ismatch bool, code int, e error) {
	//1.查询是否有付费
	s := "select id,name,idcard from idcertify_record where uid =? and status =1 limit 1"
	rows, e := mdb.QueryFromMain(s, uid)
	if e != nil {
		return false, 0, e
	}
	defer rows.Close()
	if rows.Next() {
		// 有付费
		var name, idcard string
		var id int
		if e := rows.Scan(&id, &name, &idcard); e != nil {
			return false, 0, e
		}
		//2.有付费，直接执行验证接口
		ismatch, info, e := general.IsMatch(idcard, name)
		if e != nil {
			mainlog.AppendObj(e, "IsMatch is error ", uid, idcard, name)
			return false, 0, e
		}
		mainlog.AppendObj(e, "--", info)
		tx, e := mdb.Begin()
		if e != nil {
			return false, 0, e
		}
		var is_use int
		if ismatch {
			// 认证成功，需要修改状态值
			if e := UpdateIdcardStatus(tx, uid, 1); e != nil {
				tx.Rollback()
				return false, 0, e
			}
			is_use = 1
			tm, e := utils.ToTime(info["birthday"])
			if e != nil {
				mainlog.AppendObj(e, "tm is error ", uid)
			}
			// 发送认证邀请人消息
			if e := SetBirthday(uid, tm); e != nil {
				tx.Rollback()
				return false, 0, e
			}
			// 检测是否为第一次认证，如果是，则加币
			doAddIdCardAward(tx, uid)
			// 发送认证邀请人消息
			NotifyAndDelInviteList(uid, INVITE_KEY_CERTIFY_IDCARD)
		}
		mainlog.AppendObj(nil, "do idcard Match ", uid, idcard, name, ismatch)
		//3.只要执行无报错，无论结果是否匹配成功，都需要修改付费表状态
		if e := updateIdCardRecordWhenEnd(tx, id, 2, is_use); e != nil {
			mainlog.AppendObj(e, "IsMatch updateIdCardRecordWhenEnd is error ", uid, id, idcard, name)
			tx.Rollback()
			return false, 0, e
		}
		tx.Commit()
		//4.发送推送消息
		mainlog.AppendObj(nil, "do idcard Match push end ", uid, idcard, name, ismatch)
		//5.检测等级变化
		if _, e := certify.CheckCretifyLevel(uid, common.PRI_GET_IDCARD); e != nil {
			mainlog.AppendObj(e, "检测等级失败 key", common.PRI_GET_IDCARD, uid)
		}
		return ismatch, 0, nil
	} else {
		return false, 10, nil
	}
	return
}

// 认证提示
func DoCertifyPush(uid uint32, certify_type string, r bool) {
	msg := make(map[string]interface{})
	msg["type"] = common.MSG_TYPE_RICHTEXT
	msg["tip"] = "视频认证消息"
	msg[common.FOLDER_KEY] = common.FOLDER_OTHER

	msgs := make([]map[string]interface{}, 0, 2)
	if certify_type != common.PRI_GET_VIDEO {
		return
	}
	if r { // 认证成功
		var but notify.But
		ok_msg := map[string]interface{}{"type": common.RICHTEXT_TYPE_TEXT, "text": "你提交的视频认证已审核通过", "but": but}
		msgs = append(msgs, ok_msg)
	} else { // 认证失败
		var empty_but notify.But
		but := notify.GetBut("重新认证", notify.CMD_VIDEO_PRI, false, nil)
		fail_msg := map[string]interface{}{"type": common.RICHTEXT_TYPE_TEXT, "text": "你提交的视频认证审核未通过，原因：视频与形象照不符", "but": empty_but}
		but_msg := map[string]interface{}{"type": common.RICHTEXT_TYPE_BUTTON, "but": but}
		msgs = append(msgs, fail_msg, but_msg)
	}
	msg["msgs"] = msgs
	msid, e := general.SendMsg(common.UID_SYSTEM, uid, msg, "")
	mainlog.AppendObj(e, "doCertifyPush ", uid, msid, msg)
}

// 身份证认证付费成功回掉，提供给充值回掉使用...
func IdCardPayCallback(uid uint32) (e error) {
	tx, e := mdb.Begin()
	if e != nil {
		return e
	}
	s := "select id,name,idcard from  idcertify_record where uid =? and status =0 order by id desc limit 1"
	var id int
	var name, idcard string
	if e := tx.QueryRow(s, uid).Scan(&id, &name, &idcard); e != nil {
		tx.Rollback()
		return e
	}
	if e := updateIdCardRecord2(tx, id, 1); e != nil {
		tx.Rollback()
		return e
	}
	tx.Commit()
	mainlog.AppendObj(nil, "-IdCardPayCallback---", id, name, idcard)
	if _, _, e := DoIdCardCertify(uid, idcard, name); e != nil {
		return e
	}
	return
}

func AddIdCardRecord(uid uint32, id, name string) (lastId int, e error) {
	s := "insert into idcertify_record(uid,idcard,name) values(?,?,?)"
	r, e := mdb.Exec(s, uid, id, name)
	if e != nil {
		return 0, e
	}
	v, _ := r.LastInsertId()
	lastId, e = utils.ToInt(v)
	return
}

func updateIdCardRecordWhenEnd(db utils.SqlObj, id int, status int, is_use int) (e error) {
	s := "update idcertify_record set status=?,use_tm=?,is_use=? where id = ?"
	_, e = db.Exec(s, status, utils.Now, is_use, id)
	return
}

func updateIdCardRecord2(db utils.SqlObj, id int, status int) (e error) {
	s := "update idcertify_record set status=? where id = ?"
	_, e = db.Exec(s, status, id)
	return
}

// 修改用户身份证认证状态
func UpdateIdcardStatus(db utils.SqlObj, uid uint32, status int) (e error) {
	s := "update user_main set certify_idcard=? where uid = ?"
	_, e = db.Exec(s, status, uid)
	return
}

// 修改视频认证等状态 0 待处理，1 通过
func UpdateVideoStatus(db utils.SqlObj, uid uint32, status int) (e error) {
	s := "update user_main set certify_video=? where uid = ?"
	_, e = db.Exec(s, status, uid)
	return
}

// 查询用户视频认证状态
func GetVideoStatus(uid uint32) (res map[string]interface{}, e error) {
	res = make(map[string]interface{})
	s := "select tm,status from video_record where uid =? and status in(0,1) limit 1"
	rows, e := mdb.Query(s, uid)
	if e != nil {
		return res, e
	}
	defer rows.Close()
	if rows.Next() {
		var tm string
		var status int
		if e := rows.Scan(&tm, &status); e != nil {
			return res, e
		}
		res["isdo"] = true
		t, e := utils.ToTime(tm)
		if e != nil {
			return res, e
		}
		res["tm"] = t
		res["status"] = status
	} else {
		res["isdo"] = false
	}
	return res, nil
}

// 查询身份证是否可以使用
func CheckIdcardAvailable(id string) (ok bool, e error) {
	s := "select count(*) from idcertify_record where is_use = 1 and idcard = ?"
	var cnt int
	if e := mdb.QueryRow(s, id).Scan(&cnt); e != nil {
		mainlog.AppendObj(e, "CheckIdcardAvailable---id  is used", id)
		return false, e
	}
	if cnt > 0 {
		return false, nil
	}
	return true, nil
}

//检测是否需要为身份证认证用户奖励金币
func doAddIdCardAward(tx utils.SqlObj, uid uint32) (e error) {
	award_coin := 3000
	s := "select count(*) from idcertify_record where is_use in(1,-1) and uid = ?"
	var cnt int
	if e := mdb.QueryRow(s, uid).Scan(&cnt); e != nil {
		mainlog.AppendObj(e, "CheckIdcardAvailable---id  is used", uid)
		return e
	}
	// cnt 大于0 则表示已经领取过金币奖励
	if cnt > 0 {
		mainlog.AppendObj(nil, "doAddIdCardAward is got idcard award, uid : ", uid)
		return
	}
	e = coin.UserCoinChange(tx, uid, 0, 2, 0, award_coin, "身份证认证奖励")
	mainlog.AppendObj(e, "coin do add coin is do ", uid, award_coin)
	return
}

//用户更新头像，重置用户视频认证状态
func updateUserVideoCertify(uid uint32) (e error) {
	u, e := user_overview.GetUserObject(uid)
	if e != nil {
		return
	}
	if u.CertifyVideo == 0 {
		return
	}
	if e := UpdateVideoStatus(mdb, uid, 0); e != nil {
		return e
	}
	s := "update video_record set status = 0 where uid = ? and status = 1"
	_, e = mdb.Exec(s, uid)
	return e
}
