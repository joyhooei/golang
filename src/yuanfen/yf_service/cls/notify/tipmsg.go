package notify

import (
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
)

/*

添加提示返回消息

参数列表：

	res :需要插入的HTTP RES：
	text:插入tip显示的文本，可包含换行符，表示换行
	tip:超链接文字
	cmd:超链接效果
	data:超链接点击的data


本消息在http返回中的结构为
{
	"status":"ok"
	"res"{
		"result":-1,
		"tip_msg":{
			"type":"hint",
			"content":"余额不足",
			................
		}
		.....
	}
}
*/
func AddTipMsg(res map[string]interface{}, text string, tip, cmd string, data map[string]interface{}) {

	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_HINT
	content["isSave"] = true
	content["content"] = text
	if cmd != "" {
		but := GetBut(tip, cmd, false, data)
		content["but"] = but
	}
	content["tm"] = utils.Now
	res[common.TIP_MSG] = content
	return
}
