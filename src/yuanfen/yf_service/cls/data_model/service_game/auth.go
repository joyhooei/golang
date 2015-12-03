package service_game

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"time"
	"yf_pkg/service"
	"yf_pkg/utils"
)

/*
根据uid和appid获取该用户在该游戏的授权对象
*/
func GetGameAuth(uid uint32, appid string) (a GameAuth, e error) {
	s := "select uid,appid,code,code_tm,token,token_tm from game_auth where uid =? and appid = ?"
	rows, e := mdb.Query(s, uid, appid)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&a.Uid, &a.AppId, &a.Code, &a.CodeTm, &a.Token, &a.TokenTm); e != nil {
			return
		}
	}
	return
}

func GetAuthCode(uid uint32, appid string) (code string) {
	s := utils.ToString(uid) + appid + utils.ToString(utils.Now.Second()) + utils.ToString(rand.Intn(100))
	code = fmt.Sprintf("%x", md5.Sum([]byte(s)))
	return
}

/*
根据uid和token 获取auth
*/
func GetGameAuthByToken(uid uint32, token string) (a GameAuth, e error) {
	s := "select uid,appid,code,code_tm,token,token_tm from game_auth where uid =? and token = ?"
	rows, e := mdb.Query(s, uid, token)
	if e != nil {
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&a.Uid, &a.AppId, &a.Code, &a.CodeTm, &a.Token, &a.TokenTm); e != nil {
			return
		}
	}
	return
}

/*
检测授权状态
*/
func CheckGameAuthIsValid(a GameAuth) (int, service.Error) {
	now := utils.Now
	// 检测是否为空
	if a.Uid <= 0 || a.Code == "" || a.AppId == "" {
		return GAMEAUTH_STATUS_NOAUTH, service.NewError(GAMEAUTH_STATUS_NOAUTH, "未授权")
	}
	codeTm, e := utils.ToTime(a.CodeTm)
	if e != nil {
		return GAMEAUTH_STATUS_NOAUTH, service.NewError(service.ERR_INTERNAL, e.Error())
	}
	if now.After(codeTm) { //授权过期
		return GAMEAUTH_STATUS_CODE_TIMEOUT, service.NewError(GAMEAUTH_STATUS_CODE_TIMEOUT, "授权过期")
	}
	tokenTm, e := utils.ToTime(a.TokenTm)
	if e != nil {
		return GAMEAUTH_STATUS_NOAUTH, service.NewError(service.ERR_INTERNAL, e.Error())
	}
	if now.After(tokenTm) { //授权过期
		return GAMEAUTH_STATUS_TOKEN_TIMEOUT, service.NewError(GAMEAUTH_STATUS_TOKEN_TIMEOUT, "TOKEN授权过期")
	}
	return GAMEAUTH_STATUS_OK, service.NewError(service.ERR_NOERR, "")
}

func CheckAuthToken(uid uint32, token string) (a GameAuth, g GameData, er service.Error) {
	// 验证token是否存在
	a, e := GetGameAuthByToken(uid, token)
	if e != nil {
		return a, g, service.NewError(service.ERR_INTERNAL, e.Error())
	}
	// 验证是否过期
	status, er := CheckGameAuthIsValid(a)
	if status != GAMEAUTH_STATUS_OK {
		return a, g, er
	}
	is_ok, g, e := CheckAppValid(a.AppId)
	if e != nil || !is_ok {
		return a, g, service.NewError(service.ERR_INTERNAL, "appid is valid")
	}
	return a, g, service.NewError(service.ERR_NOERR, "")
}

/*
添加新的授权状态
*/
func AddAuth(tx utils.SqlObj, uid uint32, appid, code string, code_tm time.Time) (e error) {
	s := "insert into game_auth(uid,appid,code,code_tm) values(?,?,?,?)"
	_, e = tx.Exec(s, uid, appid, code, code_tm)
	return
}

/*
更新的授权状态
*/
func UpdateAuthCode(tx utils.SqlObj, uid uint32, appid, code string, code_tm time.Time) (e error) {
	s := "update game_auth set code = ?,code_tm =? where uid = ? and appid = ?"
	_, e = tx.Exec(s, code, code_tm, uid, appid)
	return
}

/*
更新的token
*/
func UpdateAuthToken(tx utils.SqlObj, uid uint32, appid, token string, token_tm time.Time) (e error) {
	s := "update game_auth set token = ?,token_tm =? where uid = ? and appid = ?"
	_, e = tx.Exec(s, token, token_tm, uid, appid)
	return
}
