package admin

import (
	"errors"
	"yf_pkg/service"
	"yf_pkg/utils"
)

/*
用于web统计

URI: admin/WebStat   示例：http://120.131.64.91:8181/admin/WebStat?cuid=8000&csid=1017&action=11

参数：
	cuid:[string] 8000   // 主渠道
	csid:[string] 1001   // 子渠道
	action:[int] 10      // 统计项（定义action）

action 统计项定义：
	1  包下载点击
	2  通用曝光
	10  pc官网曝光  www.imswing.cn
	11  wap官网曝光

返回值:
	{
		"status": "ok",
		"tm": 1438489368
	}
*/
func (co *AdminModule) WebStat(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if co.mode == "test" {
		return errors.New("can only run in test mode")
	}
	cuid := req.GetParam("cuid")
	if cuid == "" {
		cuid = "2"
	}
	csid := req.GetParam("csid")
	if csid == "" {
		csid = "888"
	}
	action, e := utils.ToInt(req.GetParam("action"))
	if e != nil {
		action = 0
	}
	s := "insert into web_stat(`cid`,`sid`,`action`,`ip`)  values(?,?,?,?)"
	_, e = co.statDb.Exec(s, cuid, csid, action, req.IP())
	return
}
