package message

import "yf_pkg/message"

const (
	ONLINE           = 1
	OFFLINE          = 2
	REGISTER         = 3
	VISIT            = 4  //A用户访问B用户
	BIRTHDAY_CHANGE  = 5  //生日变化时触发
	LOCATION_CHANGE  = 6  //用户位置变化时触发
	MSG_DANGER       = 7  //出现提示风险消息时触发
	CLEAR_CACHE      = 8  //清除用户缓存
	CREATETOPIC      = 9  //创建话题
	ONTOP            = 10 //切换前台
	RECOMMEND_CHANGE = 11 //用户推荐状态发生变化
	SAMPLE           = 9999
)

func SendMessage(msgID int, data interface{}, result map[string]interface{}) {
	res := message.SendMessage(msgID, data)
	if result != nil {
		result["yf_plugins"] = res
	}
}
func RegisterCallback(msgID int, name string, callback message.Callback) (result interface{}) {
	return message.RegisterCallback(msgID, name, callback)
}

func RegisterNotification(msgID int, notification message.Notification) {
	message.RegisterNotification(msgID, notification)
}

func RemoveCallback(msgID int, name string) {
	message.RemoveCallback(msgID, name)
}

func RemoveNotification(msgID int, notification message.Notification) {
	message.RemoveNotification(msgID, notification)
}
