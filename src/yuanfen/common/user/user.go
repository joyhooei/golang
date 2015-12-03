package user

// 判断该用户是否是客服用户（客服用户uid< 5000000, 可以屏蔽一些不必要的操作）
func IsKfUser(uid uint32) (ok bool) {
	if uid < 5000000 {
		ok = true
	}
	return ok
}
