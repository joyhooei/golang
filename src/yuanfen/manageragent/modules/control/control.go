package control

import (
	"fmt"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/manageragent/cls/manager"
	"yuanfen/manageragent/cls/mtcp"
)

type Control struct {
}

func (r *Control) Init(env *service.Env) error {

	return nil
}

func (r *Control) SendInfo(req *service.HttpRequest, res map[string]interface{}) (e error) {
	// fmt.Println(fmt.Sprintf("%v", req.Body))
	var uid, mid uint32
	var sid string

	sid = req.GetParam("sid")
	uid, _ = utils.ToUint32(req.GetParam("uid"))
	mid, _ = utils.ToUint32(req.GetParam("mid"))
	conn, err := mtcp.DoLogin(uid, sid) //tcp, err :=
	if err != nil {
		fmt.Println("DoLogin error " + err.Error())
		return err
	}
	mtcp.AddUser(uid, mid, conn)
	return
}

func (r *Control) GetInfo(req *service.HttpRequest, res map[string]interface{}) (e error) {
	res["manager"] = manager.PrintMap()
	res["user"] = mtcp.PrintMap()
	return
}

func (r *Control) DisConnectUid(req *service.HttpRequest, res map[string]interface{}) (e error) {
	uid, _ := utils.ToUint32(req.GetParam("uid"))
	mtcp.DelUser(uid)
	return
}

func (r *Control) GetMyOnlineUids(req *service.HttpRequest, res map[string]interface{}) (e error) {
	mid, _ := utils.ToUint32(req.GetParam("mid"))
	uids, err := mtcp.GetMyOnlineUids(mid)
	if err != nil {
		return err
	}
	res["uids"] = uids
	return
}
