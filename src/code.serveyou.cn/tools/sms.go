package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/pkg/format"
)

func writeBackErr(r *http.Request, w http.ResponseWriter, err common.Error) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"
	result["ecode"] = err.Code
	result["edesc"] = err.Desc
	writeBack(r, w, string(format.GenerateJSON(result)))
}

func writeBack(r *http.Request, w http.ResponseWriter, result string) {
	w.Write([]byte(result))
}

func push(w http.ResponseWriter, r *http.Request) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"
	var err common.Error
	var uid common.UIDType
	var title, content string
	var foundUid, foundTitle, foundContent bool = false, false, false
	extras := make(map[string]interface{})
	for key, value := range r.URL.Query() {
		switch key {
		case "uid":
			tmp, e := format.ParseUint64(value[0])
			if e != nil {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [uid] error: %v", e.Error()))
				return
			}
			uid = common.UIDType(tmp)
			foundUid = true
		case "title":
			title = value[0]
			foundTitle = true
		case "content":
			content = value[0]
			foundContent = true
		default:
			extras[key] = value[0]
		}
	}

	if !foundUid {
		err = common.NewError(common.ERR_INVALID_PARAM, "no [uid] provided.")
		return
	}
	if !foundContent {
		err = common.NewError(common.ERR_INVALID_PARAM, "no [content] provided.")
		return
	}
	if !foundTitle {
		err = common.NewError(common.ERR_INVALID_PARAM, "no [title] provided.")
		return
	}
	uids := make([]common.UIDType, 0, 1)
	uids = append(uids, uid)
	e := common.PushNotification(uids, nil, title, content, extras)
	if e != nil {
		err.Code = common.ERR_INTERNAL
		err.Desc = e.Error()
	}
	if err.Code == common.ERR_NOERR {
		result["status"] = "ok"
		writeBack(r, w, string(format.GenerateJSON(result)))
	} else {
		writeBackErr(r, w, err)
	}
}

func sms(w http.ResponseWriter, r *http.Request) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"
	var err common.Error
	phone, found := r.URL.Query()["phone"]
	if !found {
		err = common.NewError(common.ERR_INVALID_PARAM, "no [phone] provided.")
		return
	}
	content, found := r.URL.Query()["content"]
	if !found {
		err = common.NewError(common.ERR_INVALID_PARAM, "no [content] provided.")
		return
	}
	e := common.SendSMS(phone[0], content[0])
	if e != nil {
		err.Code = common.ERR_INTERNAL
		err.Desc = e.Error()
	}
	if err.Code == common.ERR_NOERR {
		result["status"] = "ok"
		writeBack(r, w, string(format.GenerateJSON(result)))
	} else {
		writeBackErr(r, w, err)
	}
}
func main() {
	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s [ip:port]\n", os.Args[0])
		return
	}
	http.HandleFunc("/sms", sms)
	http.HandleFunc("/push", push)
	err := http.ListenAndServe(os.Args[1], nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	time.Sleep(2 * time.Second)
}
