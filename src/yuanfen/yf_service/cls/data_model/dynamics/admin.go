package dynamics

import "yf_pkg/mysql"

// 检测图片，如果通过则发布动态
func CheckPhtotos(uid uint32) (pics []string, e error) {
	/*
		tx, e := mdb.Begin()
		if e != nil {
			return
		}
		s := "update user_photo_album set status = ? where albumid " + mysql.In(albumIds)
			if _, e = tx.Exec(s, status); e != nil {
				mlog.AppendObj(e, "extx is eroor")
				tx.Rollback()
				return
			}
			// 查询是否还有未审核的的图片
			s2 := "select count(*) from user_photo_album where uid =? and status = 0 and albumid not " + mysql.In(albumIds)
			rows, e := mdb.Query(s2, uid)
			if e != nil {
				mlog.AppendObj(e, "quer is eroor")
				return
			}
			defer rows.Close()
			var n int
			if rows.Next() {
				if e = rows.Scan(&n); e != nil {
					mlog.AppendObj(e, "que2r is eroor")
					return e
				}
			}
			mlog.AppendObj(nil, "----1->", n)
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
			mlog.AppendObj(nil, "---2-->", pics)
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
	*/

	// 如果没有需要审核的图片，则需要查询是否有需要发送动态的图片..
	s3 := "select albumid,pic from user_photo_album where uid = ? and flag = 0 "
	pics = make([]string, 0, 5)
	pids := make([]uint32, 0, 5)
	rows2, e := mdb.Query(s3, uid)
	if e != nil {
		return
	}
	defer rows2.Close()
	for rows2.Next() {
		var pic string
		var pid uint32
		if e = rows2.Scan(&pid, &pic); e != nil {
			return
		}
		pics = append(pics, pic)
		pids = append(pids, pid)
	}
	if len(pics) <= 0 {
		return
	}
	s4 := "update user_photo_album set flag= 1 where albumid  " + mysql.In(pids)
	if _, e = mdb.Exec(s4); e != nil {
		return
	}
	return
	/*	// 构建新增形象照动态
		var dy Dynamic
		dy.Pic = strings.Join(pics, ",")
		dy.Stype = common.DYNAMIC_STYPE_AVATAR
		dy.Text = "上传形象照"
		dy.Type = common.DYNAMIC_TYPE_USER
		dy.Uid = uid
		did, e := AddDynamic(dy)
		if e != nil {
			return
		}
		// dopush msg
		go DoCheckPicAndPush(did, false)
		return
	*/
}
