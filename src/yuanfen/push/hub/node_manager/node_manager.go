package node_manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
	"yf_pkg/log"
	"yf_pkg/net/http"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/push/hub/db"
)

var ErrNotOnline error = errors.New("user not online")

var selected db.Node //当前在线人数最少的节点，如果为""表示没有任何可用的节点
var nodes []string
var userLog *log.MLogger

func Init(ulog *log.MLogger) error {
	userLog = ulog
	go refreshNode()
	return nil
}

func refreshNode() {
	for {
		ns, e := db.GetNodes()
		tmpNodes := []string{}
		min := math.MaxInt32
		var n db.Node
		nodeStatus := "node users: "
		if e == nil {
			for _, node := range ns {
				result, e := SendToNode(node.PrivateAddress, "OnlineUsers", nil, nil)
				if e.Code != service.ERR_NOERR {
					fmt.Println("node ", node.PrivateAddress, " offline:", e.Error())
				} else {
					users, _ := utils.ToInt(result["users"])
					nodeStatus += fmt.Sprintf("%v[%v] ", node.PrivateAddress, users)
					tmpNodes = append(tmpNodes, node.PrivateAddress)
					if users < min {
						n, min = node, users
					}
				}
			}
			selected = n
		} else {
			fmt.Println("refreshNodes error:" + e.Error())
		}
		fmt.Println(nodeStatus)
		nodes = tmpNodes
		userLog.Append(nodeStatus, log.NOTICE)
		time.Sleep(10 * time.Second)
	}
}

//获取用户所在node
func GetNodePrivateAddr(uid uint32) (string, error) {
	privateAddr, e := db.GetNodePrivateAddr(uid)
	switch e {
	case redis.ErrNil:
		return "", ErrNotOnline
	}
	return privateAddr, e
}

func GetEndpoint(uid uint32) (result map[string]interface{}, err service.Error) {
	n := selected
	if n.PrivateAddress == "" {
		return nil, service.NewError(service.ERR_INTERNAL, "no available node", "")
	}
	result, err = SendToNodeSec(n.PrivateAddress, "GetEndpoint", nil, uid, nil)
	if err.Code != service.ERR_NOERR {
		return nil, err
	}
	e := db.SetUserNode(uid, n.PrivateAddress)
	if e != nil {
		return nil, service.NewError(service.ERR_INTERNAL, e.Error(), "")
	}
	result["address"] = n.PublicAddress
	return
}

func Send(host string, path string, params map[string]string, cookies map[string]string, data []byte) (result map[string]interface{}, err service.Error) {
	bodyBytes, e := http.HttpSend(host, path, params, cookies, data)
	if e != nil {
		return nil, service.NewError(service.ERR_INTERNAL, fmt.Sprintf("request http://%v/%v error : %v", host, path, e.Error()))
	}
	e = json.Unmarshal(bodyBytes, &result)
	if e != nil {
		return nil, service.NewError(service.ERR_INTERNAL, fmt.Sprintf("request http://%v/%v error : %v", host, path, e.Error()))
	}
	if result["status"] != "ok" {
		return nil, service.NewError(uint(result["code"].(float64)), result["msg"].(string))
	}
	return
}

func SendToUser(uid uint32, method string, params map[string]string, data []byte) (result map[string]interface{}, err service.Error) {
	n, e := GetNodePrivateAddr(uid)
	switch e {
	case ErrNotOnline:
		result["online"] = false
		return
	case nil:
		return Send(n, "s/push/"+method, params, map[string]string{"uid": utils.ToString(uid)}, data)
	default:
		return nil, service.NewError(service.ERR_INTERNAL, e.Error(), "")
	}
}

func SendToNode(node string, method string, params map[string]string, data []byte) (result map[string]interface{}, err service.Error) {
	return Send(node, "push/"+method, params, nil, data)
}

func SendToNodeSec(node string, method string, params map[string]string, uid uint32, data []byte) (result map[string]interface{}, err service.Error) {
	return Send(node, "s/push/"+method, params, map[string]string{"uid": utils.ToString(uid)}, data)
}

func SendToNodes(method string, params map[string]string, data []byte) (result map[string]interface{}, err service.Error) {
	result = make(map[string]interface{})
	for _, address := range nodes {
		r, e := Send(address, "push/"+method, params, nil, data)
		if e.Code != service.ERR_NOERR {
			result[address] = map[string]interface{}{"code": e.Code, "msg": e.Desc}
		} else {
			result[address] = r
		}
	}
	fmt.Println("result=", result)
	return
}

func PrepareSendTagMsg(from uint32, tag string, typ string, data []byte, result map[string]interface{}) (err service.Error) {
	msgid, e := db.SaveTagMessage(from, tag, typ, data)
	if e != nil {
		return service.NewError(service.ERR_MYSQL, e.Error())
	}
	result["msgid"] = msgid
	return
}

func ExecSendTagMsg(msgid uint64, from uint32, tag string, typ string, data []byte, result map[string]interface{}) (err service.Error) {
	offline := make([]uint32, 0)
	params := make(map[string]string)
	params["to"] = "t_" + tag
	params["msgid"] = utils.ToString(msgid)

	for _, address := range nodes {
		r, e := Send(address, "s/push/Send", params, map[string]string{"uid": utils.ToString(from)}, data)
		if e.Code != service.ERR_NOERR {
			return service.NewError(e.Code, fmt.Sprintf("%v error : %v", address, e.Desc))
		}
		o, ok := r["offline"].([]interface{})
		if ok {
			for _, uid := range o {
				offline = append(offline, uint32(uid.(float64)))
			}
		}
	}
	result["offline"] = offline
	return
}

func SendTagMsg(from uint32, tag string, typ string, data []byte, result map[string]interface{}) (err service.Error) {
	offline := make([]uint32, 0)
	msgid, e := db.SaveTagMessage(from, tag, typ, data)
	if e != nil {
		return service.NewError(service.ERR_MYSQL, e.Error())
	}
	params := make(map[string]string)
	params["to"] = "t_" + tag
	params["msgid"] = utils.ToString(msgid)

	for _, address := range nodes {
		r, e := Send(address, "s/push/Send", params, map[string]string{"uid": utils.ToString(from)}, data)
		if e.Code != service.ERR_NOERR {
			return service.NewError(e.Code, fmt.Sprintf("%v error : %v", address, e.Desc))
		}
		o, ok := r["offline"].([]interface{})
		if ok {
			for _, uid := range o {
				offline = append(offline, uint32(uid.(float64)))
			}
		}
	}
	result["msgid"] = msgid
	result["offline"] = offline
	return
}

//准备发送消息，预先得到消息ID
func PrepareSendUserMsg(from uint32, to uint32, tag string, typ string, data []byte, result map[string]interface{}) (err service.Error) {
	msgid, e := db.SaveMessage(from, to, tag, typ, data)
	if e != nil {
		return service.NewError(service.ERR_MYSQL, e.Error())
	}
	result["msgid"] = msgid
	return
}

//实际发送消息，需要传消息ID
func ExecSendUserMsg(msgid uint64, from uint32, to uint32, tag string, typ string, data []byte, result map[string]interface{}) (err service.Error) {
	params := make(map[string]string)
	params["to"] = "u_" + utils.ToString(to)
	params["tag"] = tag
	params["msgid"] = utils.ToString(msgid)
	n, e := GetNodePrivateAddr(to)
	switch e {
	case ErrNotOnline:
		result["online"] = false
		return
	case nil:
		res, err := Send(n, "s/push/Send", params, map[string]string{"uid": utils.ToString(from)}, data)
		result["online"] = res["online"]
		return err
	default:
		return service.NewError(service.ERR_INTERNAL, e.Error(), "")
	}
}

func SendUserMsg(from uint32, to uint32, tag string, typ string, data []byte, result map[string]interface{}) (err service.Error) {
	msgid, e := db.SaveMessage(from, to, tag, typ, data)
	if e != nil {
		return service.NewError(service.ERR_MYSQL, e.Error())
	}
	params := make(map[string]string)
	params["to"] = "u_" + utils.ToString(to)
	params["tag"] = tag
	params["msgid"] = utils.ToString(msgid)
	result["msgid"] = msgid
	n, e := GetNodePrivateAddr(to)
	switch e {
	case ErrNotOnline:
		result["online"] = false
		return
	case nil:
		res, err := Send(n, "s/push/Send", params, map[string]string{"uid": utils.ToString(from)}, data)
		result["online"] = res["online"]
		return err
	default:
		return service.NewError(service.ERR_INTERNAL, e.Error(), "")
	}
}

func SendUsersMsg(from uint32, to []uint32, tag string, typ string, data []byte, result map[string]interface{}) (err service.Error) {
	res := map[string]interface{}{}
	msgids, e := db.SaveMessages(from, to, tag, typ, data)
	if e != nil {
		return service.NewError(service.ERR_MYSQL, e.Error())
	}
	gUids := map[string][]uint32{}
	gMids := map[string][]uint64{}
	for i, uid := range to {
		gid, e := GetNodePrivateAddr(uid)
		switch e {
		case ErrNotOnline:
			res[utils.ToString(uid)] = map[string]interface{}{"msgid": msgids[i], "online": false}
		case nil:
			gUid, ok := gUids[gid]
			gMid := gMids[gid]
			if !ok {
				gUid = []uint32{}
				gMid = []uint64{}
			}
			gUids[gid] = append(gUid, uid)
			gMids[gid] = append(gMid, msgids[i])
		default:
			return service.NewError(service.ERR_INTERNAL, e.Error(), "")
		}
	}
	for name, gUid := range gUids {
		params := make(map[string]string)
		v, e := utils.Join(gUid, ",")
		if e != nil {
			return service.NewError(service.ERR_MYSQL, e.Error())
		}
		params["to"] = v
		params["tag"] = tag
		v, e = utils.Join(gMids[name], ",")
		if e != nil {
			return service.NewError(service.ERR_MYSQL, e.Error())
		}
		params["msgid"] = v
		r, er := Send(name, "s/push/SendM", params, map[string]string{"uid": utils.ToString(from)}, data)
		if er.Code != service.ERR_NOERR {
			return er
		} else {
			switch users := r["res"].(type) {
			case map[string]interface{}:
				for uidStr, v := range users {
					res[uidStr] = v
				}
			}
		}
	}
	result["res"] = res
	return err
}

func AddTag(uid uint32, tag string) error {
	if err := db.AddTag(uid, tag); err != nil {
		desc := fmt.Sprintf("add tag %v of uid %v failed : %v", tag, uid, err.Error())
		return service.NewError(service.ERR_REDIS, desc)
	}
	_, e := SendToUser(uid, "AddTag", map[string]string{"tag": tag}, nil)
	return e
}

func DelTag(uid uint32, tag string) error {
	if err := db.DelTag(uid, tag); err != nil {
		desc := fmt.Sprintf("del tag %v of uid %v failed : %v", tag, uid, err.Error())
		return service.NewError(service.ERR_REDIS, desc)
	}
	_, e := SendToUser(uid, "DelTag", map[string]string{"tag": tag}, nil)
	return e
}
