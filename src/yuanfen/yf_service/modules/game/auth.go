package game

import (
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/data_model/service_game"
)

/*
 游戏授权接口

 URL: s/game/Auth

参数:
	appid: [string] 游戏appid

返回值:
	{
		"res": {
			"auth": "e6fad26a9fb2a3bdcbdcbb20ae05a46c",  // 授权码
			"uid": 5000761	// 授权用户uid
		},
		"status": "ok",
		"tm": 1442636600
	}
*/
func (pm *GameModule) SecAuth(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var app_id string
	if e = req.Parse("appid", &app_id); e != nil {
		return
	}
	// 获取授权对象
	a, e := service_game.GetGameAuth(req.Uid, app_id)
	if e != nil {
		return
	}
	var is_exist bool
	if a.Uid > 0 && a.AppId != "" && a.Code != "" {
		is_exist = true
	}
	tx, e := pm.mdb.Begin()
	if e != nil {
		return
	}
	code := service_game.GetAuthCode(req.Uid, app_id)
	code_tm := utils.Now.AddDate(0, 2, 0)
	if is_exist {
		if e = service_game.UpdateAuthCode(tx, req.Uid, app_id, code, code_tm); e != nil {
			tx.Rollback()
			pm.log.AppendObj(e, "update auth is error", req.Uid)
			return
		}
	} else {
		if e = service_game.AddAuth(tx, req.Uid, app_id, code, code_tm); e != nil {
			tx.Rollback()
			pm.log.AppendObj(e, "addauth is error", req.Uid)
			return
		}
	}
	tx.Commit()
	res := make(map[string]interface{})
	res["auth"] = code
	res["uid"] = req.Uid
	result["res"] = res
	return
}
