package message

import (
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
)

func SendSysMessage(to uint32, msg string) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_TEXT
	content["content"] = msg
	return general.SendMsg(common.USER_SYSTEM, to, content, "")

}
