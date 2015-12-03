package stat

const (
	ACTION_TEST          = 0  //测试
	ACTION_REG           = 1  //注册
	ACTION_ONLINE        = 2  //上线
	ACTION_CHARGE        = 4  //充值
	ACTION_FINISH_GUIDE  = 5  //完成引导
	ACTION_GET_FRIEND    = 6  //获得一个好友
	ACTION_SAYHELLO_SUC  = 7  //认识一下发送成功
	ACTION_SAYHELLO      = 8  //认识一下
	ACTION_SEND_MSG      = 9  //发送私聊消息（图片、文字、语音）
	ACTION_FOLLOW        = 10 //标记成功
	ACTION_MUST_COMPLETE = 12 //完成资料必填项
	ACTION_DYNAMICS      = 13 //发布动态
	ACTION_CAFE          = 14 //咖啡交友成功
	ACTION_ACTIVE        = 15 //用户活跃

	DEV_ACTION_SEND_CODE   = 1001 //发送验证码
	DEV_ACTION_VERIFY_CODE = 1002 //成功输入验证码
	DEV_ACTION_REG         = 1003 //注册
	DEV_ACTION_FIRST_START = 1004 //首次启动
	DEV_ACTION_START       = 1005 //启动
)
