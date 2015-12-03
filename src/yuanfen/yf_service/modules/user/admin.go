package user

// 编辑后台接口
import (
	"errors"
	"strings"
	"time"
	"yf_pkg/push"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/dynamics"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/login"
	"yuanfen/yf_service/cls/data_model/relation/base"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/data_model/usercontrol"
	"yuanfen/yf_service/cls/message"
	"yuanfen/yf_service/cls/notify"
)

/*
用户资料复审核

URI: s/user/IUserInfoVerify

参数：
	id:[uint32] 审核记录id
	uid:[uint32] 用户uid
	level:[int] 审核等级 (9 优质， 3 非优质 -1 不通过 -5 封号)
*/
func (sm *UserModule) SecIUserInfoVerify(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	var id, uid uint32
	var level int
	if e = req.Parse("uid", &uid, "level", &level, "id", &id); e != nil {
		return
	}
	if level > 0 { // 如果level >0
		base.TryPendingSayHello(uid)
		sm.log.AppendObj(nil, "TryPendingSayHello is do ", uid)
	}
	sm.log.AppendObj(e, "parm --->", uid, level)

	sql := "select uid,reason from verify_user where id =? "
	var suid uint32
	var reason string
	if e = sm.mdb.QueryRow(sql, id).Scan(&suid, &reason); e != nil {
		return
	}

	if uid != suid {
		return service.NewError(service.ERR_INTERNAL, "用户uid不对应", "用户uid不对应")
	}
	tx, e := sm.mdb.Begin()

	// 在修改之前查询结果
	item_m := make(map[string]int)
	if level > 0 {
		s3 := "select item from update_record where uid=? and status = 1 "
		rows3, e := tx.Query(s3, uid)
		if e != nil {
			return e
		}
		for rows3.Next() {
			var item string
			if e = rows3.Scan(&item); e != nil {
				return e
			}
			item_m[item] = 1
		}
	}

	// 删除verify_user 表状态
	s := "delete from update_record where uid = ? and status = 1"
	if _, e = tx.Exec(s, uid); e != nil {
		tx.Rollback()
		sm.log.AppendObj(e, "SecIUserInfoVerify-> delete update_record is wrong")
		return
	}
	//删除到复审表
	s2 := "update verify_user set flag = 1 , status2 =? , aid2=? , tm2=?  where id = ?"
	if _, e = tx.Exec(s2, level, req.Uid, utils.Now, id); e != nil {
		tx.Rollback()
		sm.log.AppendObj(e, "SecIUserInfoVerify-> update verify_user is wrong")
		return
	}
	if level == -5 { // 封号的话，要特殊处理
		e = login.BanUser(uid)
		if e != nil {
			return
		}
		push.Kick(uid)
		dynamics.CloseUserDynamic(uid)
	} else {
		if _, e = tx.Exec("update user_detail set avatarlevel=? where uid=?", level, uid); e != nil {
			tx.Rollback()
			return
		}
	}
	tx.Commit()
	user_overview.ClearUserObjects(uid)
	// 成功的时候发动态
	if level >= 0 { //审核通过并且发送交友寄语动态
		// 通知推荐模块
		time.Sleep(1000 * time.Millisecond)
		u, _ := user_overview.GetUserObject(uid)
		sm.log.AppendObj(e, " sleep -----u:  ", u)
		message.SendMessage(message.RECOMMEND_CHANGE, message.RecommendChange{uid}, map[string]interface{}{})
		// 如果同时填写了交友寄语和上传头像，则要合并一条动态
		var sendAvatar, sendAboutme bool
		if _, ok := item_m["all"]; ok {
			sendAvatar = true
			sendAboutme = true
		} else {
			if _, ok := item_m["avatar"]; ok {
				sendAvatar = true
			}
			if _, ok := item_m["aboutme"]; ok {
				sendAboutme = true
			}
		}
		sm.log.AppendObj(e, "send_aboutme ", uid, item_m, "sendAvatar: ", sendAvatar, " sendAboutme: ", sendAboutme, level)
		if !sendAvatar && !sendAboutme {
			return
		}
		// 分别获取pic 和 text
		var pic string
		if sendAvatar {
			if pic_arr, e := dynamics.CheckPhtotos(uid); e == nil && len(pic_arr) > 0 {
				pic = strings.Join(pic_arr, ",")
			}
		}
		text := "上传形象照"
		if sendAboutme {
			um, e := usercontrol.GetUserInfo(uid)
			if e != nil {
				return e
			}
			if um["aboutme"] != "" {
				text = um["aboutme"]
			}
		}
		// 查询用户交友寄语
		var dy dynamics.Dynamic
		dy.Stype = common.DYNAMIC_STYPE_ABOUTME
		dy.Text = text
		dy.Pic = pic
		dy.Type = common.DYNAMIC_TYPE_USER
		dy.Uid = uid
		id, e = dynamics.AddDynamic(dy)
		if e != nil {
			return e
		}
		go dynamics.DoCheckPicAndPush(id, false)
	} else if level == -1 {
		// 发布审核失败系统消息
		msg := make(map[string]interface{})
		msg["type"] = common.MSG_TYPE_RICHTEXT
		msg["folder"] = common.FOLDER_OTHER
		msg["tip"] = "资料审核未通过"

		text := "资料审核未通过，请认真填写相关资料，资料完整真实才有利于你在秋千交友。"
		if reason != "" {
			text += " 原因：" + reason
		}

		content := make([]map[string]interface{}, 0, 3)
		text_msg := make(map[string]interface{})
		text_msg["type"] = common.RICHTEXT_TYPE_TEXT
		text_msg["img"] = ""
		text_msg["text"] = text
		text_msg["but"] = notify.GetDefBut("")

		button_msg := make(map[string]interface{})
		button_msg["type"] = common.RICHTEXT_TYPE_BUTTON
		button_msg["img"] = ""
		button_msg["text"] = ""
		but := notify.GetBut("重新上传", notify.CMD_USER_INFO, false, map[string]interface{}{"uid": uid})
		button_msg["but"] = but
		content = append(content, text_msg, button_msg)
		msg["msgs"] = content
		mid, e := general.SendMsg(common.UID_SYSTEM, uid, msg, "")
		if e != nil {
			sm.log.AppendObj(e, "send msg is eroor ", uid)
		}
		sm.log.AppendObj(nil, "send msg is ok", uid, mid, msg)
	}
	return
}

/*
用户形象照一审核

URI: s/user/IUserPhotoVerify

参数：
	uid:[uint32]用户uid
	status:[int]审核状态状态，0为未审核，1为正常，2为冻结
	albumIds:[string]相册id字符串，英文逗号连接(主要用于冻结)
*/
func (sm *UserModule) SecIUserPhotoVerify(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	var status int
	var uid uint32
	var albumids string
	if e = req.Parse("uid", &uid, "status", &status); e != nil {
		return
	}
	if e = req.ParseOpt("albumIds", &albumids, ""); e != nil {
		return
	}
	sm.log.AppendObj(e, "---SecIUserPhotoVerify-->", uid, albumids)
	return

	/*	tx, e := sm.mdb.Begin()
		if e != nil {
			return
		}
		s := "update user_photo_album set status = ? where albumid = ?"
		if _, e = tx.Exec(s, status, id); e != nil {
			sm.log.AppendObj(e, "extx is eroor")
			tx.Rollback()
			return
		}
		// 查询是否还有未审核的的图片
		s2 := "select count(*) from user_photo_album where uid =? and status = 0 and albumid!=? limit 1"
		rows, e := sm.mdb.Query(s2, uid, id)
		if e != nil {
			sm.log.AppendObj(e, "quer is eroor")
			return
		}
		defer rows.Close()
		var n int
		if rows.Next() {
			if e = rows.Scan(&n); e != nil {
				sm.log.AppendObj(e, "que2r is eroor")
				return e
			}
		}
		sm.log.AppendObj(nil, "----1->", n)
		if n > 0 {
			tx.Commit()
			return
		}
		// 如果没有需要审核的图片，则需要查询是否有需要发送动态的图片..
		s3 := "select albumid,pic from user_photo_album where uid = ? and flag = 0 and status =1"
		pics := make([]string, 0, 5)
		pids := make([]uint32, 0, 5)
		rows2, e := tx.Query(s3, uid)
		if e != nil {
			tx.Rollback()
			return
		}
		defer rows2.Close()
		for rows2.Next() {
			var pic string
			var pid uint32
			if e = rows2.Scan(&pid, &pic); e != nil {
				tx.Rollback()
				return
			}
			pics = append(pics, pic)
			pids = append(pids, pid)
		}
		sm.log.AppendObj(nil, "---2-->", pics)
		if len(pics) <= 0 {
			tx.Commit()
			return
		}
		s4 := "update user_photo_album set flag= 1 where albumid  " + mysql.In(pids)
		if _, e = tx.Exec(s4); e != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
		// 构建新增形象照动态
		var dy dynamics.Dynamic
		dy.Pic = strings.Join(pics, ",")
		dy.Stype = common.DYNAMIC_STYPE_AVATAR
		dy.Text = "上传形象照"
		dy.Type = common.DYNAMIC_TYPE_USER
		dy.Uid = uid
		did, e := dynamics.AddDynamic(dy)
		if e != nil {
			return
		}
		// dopush msg
		go dynamics.DoCheckPicAndPush(did, false)
		return
	*/
}

/*
编辑后台视频认证结果接口

URI: s/user/ICertifyVideo

参数：
	id: 审核记录id
	status: 审核状态，[-1 认证失败,1 认证通过]
*/
func (sm *UserModule) SecICertifyVideo(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	var id int
	var status int
	if err := req.Parse("id", &id, "status", &status); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if !(status == 1 || status == -1) {
		return service.NewError(service.ERR_INVALID_PARAM, "status must be -1 or 1")
	}
	tx, e := sm.mdb.Begin()
	if e != nil {
		return e
	}
	s := "update video_record set status = ? where id =?"
	if _, e = tx.Exec(s, status, id); e != nil {
		return e
		tx.Rollback()
	}
	s2 := "select uid from video_record where id = ?"
	var uid uint32
	if e := tx.QueryRow(s2, id).Scan(&uid); e != nil {
		tx.Rollback()
		return e
	}
	video_status := 0
	if status == 1 {
		video_status = 1
	}
	if e := usercontrol.UpdateVideoStatus(tx, uid, video_status); e != nil {
		tx.Rollback()
		return e
	}
	tx.Commit()
	var r bool
	if video_status == 1 {
		r = true
	}
	go usercontrol.DoCertifyPush(uid, common.PRI_GET_VIDEO, r)
	usercontrol.RefStar(uid)

	return
}
