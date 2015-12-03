package general

import (
	"encoding/json"
	"fmt"
	"yf_pkg/format"
	"yf_pkg/mysql"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
)

/*
GetMessageById根据消息ID获取消息内容

参数：
	ids: 消息ID集合
返回值：
	msgs: 存储消息内容的map，如果某个消息ID找不到对应的消息，则map中不出现这项，也不会抛出异常
*/
func GetMessageById(ids []uint64) (msgs map[uint64]interface{}, e error) {
	sql := "select id,`from`,`to`,tag,content,tm from message where id" + mysql.In(ids)
	fmt.Println("sql:", sql)
	rows, e := msgdb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	msgs = make(map[uint64]interface{}, len(ids))
	for rows.Next() {
		var msgid uint64
		var from, to uint32
		var content []byte
		var tmStr, tag string
		if e = rows.Scan(&msgid, &from, &to, &tag, &content, &tmStr); e != nil {
			return nil, e
		}
		fmt.Println("msgid=", msgid)
		tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
		if e != nil {
			return nil, e
		}
		var j map[string]interface{}
		if e := json.Unmarshal(content, &j); e != nil {
			return nil, e
		}
		msgs[msgid] = map[string]interface{}{"msgid": msgid, "from": from, "to": to, "tag": tag, "tm": tm, "content": j}
	}
	return
}

/*
SendSysNotice发送系统通知

参数：
	stype: 系统通知的子类型，参看common中SYS_NOTICE_开头的常量
	show: 是否显示在其它消息列表中，在消息体中的key为SYS_NOTICE_SHOW_KEY
	to: 发送对象uid，0表示tag消息
	tag: tag消息的名称，all是一个特殊值，代表所有用户
*/
func SendSysNotice(from uint32, to uint32, stype string, show bool, content map[string]interface{}, tag string) (msgid uint64, e error) {
	content["stype"] = stype
	content[common.SYS_NOTICE_SHOW_KEY] = show
	if to == 0 {
		msgid, _, e = SendTagMsgWithThirdParty(from, tag, content)
		return
	} else {
		return SendMsg(from, to, content, tag)
	}
}

//获取当前最后一条消息的ID
func GetLastMsgID() (msgid uint64, e error) {
	e = msgdb.QueryRow("select max(id) from message").Scan(&msgid)
	return
}
