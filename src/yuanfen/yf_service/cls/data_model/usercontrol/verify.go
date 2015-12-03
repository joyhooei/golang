package usercontrol

// 用户审核相关代码

/*
保存用户资料修改记录，用于审核标记修改项

	uid:修改用户uid
	key:以前审核过，现在修改资料项（key为对应修改项字段名如: nickname ），如果用户首次审核（key传all）

*/
func AddVerifyRecord(uid uint32, key string) (e error) {
	s := "insert into update_record(uid,item) values(?,?)"
	_, e = mdb.Exec(s, uid, key)
	return
}
