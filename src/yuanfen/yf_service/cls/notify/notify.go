package notify

import (
	"encoding/json"
	"errors"
	"yf_pkg/service"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

const (
	NOTIFY_KEY = "sys_notify" // 节点名称
)

// 默认按钮key
const (
	BUT_CLOSE  = "but_close"
	BUT_IGNORE = "but_ignore"
)

// 消息类型ID定义
const (
	NOTIFY_TYPE_RECOMMEND = 1  // 推荐消息
	NOTIFY_TYPE_FRIENDS   = 2  // 好友消息
	NOTIFY_TYPE_PURSUE    = 3  // 女神和追求消息
	NOTIFY_TYPE_CHAT      = 4  // 私聊消息
	NOTIFY_TYPE_TOPIC     = 5  // 圈子消息
	NOTIFY_TYPE_ACTIVITY  = 6  // 活动消息
	NOTIFY_TYPE_SYSTEM    = 7  //系统消息
	NOTIFY_TYPE_GAME      = 8  //游戏消息
	NOTIFY_TYPE_OTHER     = 9  //游戏消息
	NOTIFY_TYPE_INVITE    = 10 //邀请消息
)

// cmd 命令配置
const (
	CMD_OPEN_WEB          = "cmd_open_web"          // 自定义打开某web页面
	CMD_CLOSE             = "cmd_close"             // 关闭消息框
	CMD_OPEN_CHAT         = "cmd_open_chat"         // 打开私聊框
	CMD_OPEN_TOPIC        = "cmd_open_topic"        // 打开话题圈子
	CMD_ENTRY_GAME        = "cmd_entry_game"        // 进入游戏
	CMD_AWARD_LIST        = "cmd_award_list"        // 打开奖品列表
	CMD_PURSUE_INFO       = "cmd_pursue_info"       // 打开女神界
	CMD_PURSUE_LIST       = "cmd_pursue_list"       // 打开追求者页面
	CMD_USER_INFO         = "cmd_user_info"         // 打开个人资料页面
	CMD_OPEN_ACP_GIFT     = "cmd_open_accept_gift"  // 打开接受礼物面板
	CMD_OPEN_LUCKY        = "cmd_open_lucky"        // 打开幸运值面板
	CMD_OPEN_MYFANS       = "cmd_open_myfans"       // 打开我的粉丝界面
	CMD_OPEN_ACCOUNT      = "cmd_open_account"      // 打开我的帐号界面
	CMD_NOHAND_PURSUE     = "cmd_nohand_pursue"     // 打开未处理追求者弹框
	CMD_ENTRY_INVITE_GAME = "cmd_entry_invite_game" // 受邀请进入游戏
	CMD_OPEN_ACTLIST      = "cmd_open_actlist"      // 打开活动列表界面
	CMD_OPEN_ACT          = "cmd_open_act"          // 打开某个活动
	CMD_OPEN_ACTAWARD     = "cmd_open_actaward"     // 打开活动中奖清单
	CMD_ENTRY_GAMEROOM    = "cmd_entry_gameroom"    // 进入游戏大厅，不组队
	CMD_HONESTY           = "cmd_honesty"           // 查看诚信度页面
	CMD_PHONE_PRI         = "cmd_phone_pri"         // 查看手机认证特权页面
	CMD_VIDEO_PRI         = "cmd_video_pri"         // 查看视频认证特权页
	CMD_IDCARD_PRI        = "cmd_idcard_pri"        // 查看身份证特权页面
	CMD_OPEN_REQUIRE      = "cmd_open_require"      // 打开择友问答（uid ）

	CMD_NEAR_MSG   = "cmd_near_msg"  // 打开个人附近消息,参数（uid）
	CMD_ADVISE_MSG = "cmd_advise"    // 打开提意见界面
	CMD_MYCHARGE   = "cmd_mycharge"  // 到充值页面
	CMD_DATEPLACE  = "cmd_dateplace" // 到官方约会地点
)

const (
	NOTIFY_NEAR                = "notify_near"           // 附近10公里范围内的匹配用户发布了一个附近约会消息提醒
	NOTIFY_CREATE_TOPIC        = "notify_create_topic"   // 创建圈子消息
	NOTIFY_MESSAGE             = "notify_message"        // 发布了一个附近的约会消息
	NOTIFY_FOLLOW              = "notify_follow"         // 关注消息
	NOTIFY_NEW_PHOTO           = "notify_new_photo"      // 用户上传了新照片
	NOTIFY_PURSUE_SEND         = "notify_pursue_send"    // 发送追求消息
	NOTIFY_PURSUE_ACCEPT       = "notify_pursue_accept"  // 接受追求消息
	NOTIFY_PURSUE_SET          = "notify_pursue_set"     // 设置为了女神
	NOTIFY_PURSUE_BREAK        = "notify_pursue_break"   // 接受追求消息
	NOTIFY_PURSUE_MESSAGE      = "notify_pursue_message" // 女神发布约会消息
	NOTIFY_PURSUE_CREATE_TOPIC = "notify_pcreate_topic"  // 女神发布圈子
	NOTIFY_ADD_LUCKY           = "notify_add_lucky"      // 添加幸运
	NOTIFY_PNEW_PHOTO          = "notify_pnew_photo"     // 女神上传新照片
	NOTIFY_NEAR_TOPIC          = "notify_near_topic"     // 附近10公里内的匹配用户发布了一个圈子话题
	NOTIFY_TOPIC_MESSAGE       = "notify_topic_message"  // 我发布的圈子消息和我加入的圈子消息提醒
	NOTIFY_GIFT_ACCEPT         = "notify_gift_accept"    //谁接受了我的礼物
	NOTIFY_GIFT_SEND           = "notify_gift_send"      //发送礼物
	NOTIFY_VIEW                = "notify_view"           //谁看过我
	NOTIFY_GAME_INVITE         = "notify_game_invite"    //游戏邀请消息
	NOTIFY_GAME_ACCEPT         = "notify_game_accept"    //好友接受我的游戏邀请
	NOTIFY_GAME_REFUSE         = "notify_game_refuse"    //好友拒绝我的游戏邀请
	NOTIFY_CHAT                = "notify_chat"           // 用户聊天消息
	NOTIFY_COIN                = "notify_coin"           // 金币变化
	NOTIFY_PLANE_NUM           = "notify_plane_num"      // 飞行点
	NOTIFY_SAY_HELLO           = "notify_say_hello"      // 打招呼
	NOTIFY_HONESTY_CHANGE      = "notify_honesty_change" // 诚信度变化通知

	NOTIFY_ACTIVITY  = "notify_activity"  //好友拒绝我的游戏邀请
	NOTIFY_SYSTEM    = "notify_system"    //系统消息
	NOTIFY_ACT_AWARD = "notify_act_award" //活动奖品中奖通知
	NOTIFY_ACT_BEGIN = "notify_act_begin" //游戏活动开始
	NOTIFY_ACT_OVER  = "notify_act_over"  //游戏活动结束

	NOTIFY_FILL_FINISH = "notify_fill_finish" //资料完善完成
	NOTIFY_INVITE_FILL = "notify_invite_fill" //邀请完善资料

	NOTIFY_INV_PHOTO        = "notify_inv_photo"       //邀请上传照片
	NOTIFY_INV_CERTIFY      = "notify_certify"         //认证邀请(默认跳转到视频认证)
	NOTIFY_INV_FILL_REQUIRE = "notify_inv_fillrequire" //择友条件邀请

	NOTIFY_INV_PHOTO_FINISH          = "notify_inv_photo_finish"    //上传照片邀请反馈
	NOTIFY_INV_CERTIFY_VIDEO_FINISH  = "notify_inv_video_finish"    //视频认证邀请反馈
	NOTIFY_INV_CERTIFY_PHONE_FINISH  = "notify_inv_phone_finish"    //手机认证邀请反馈
	NOTIFY_INV_CERTIFY_IDCARD_FINISH = "notify_inv_idcard_finish"   //身份证认证邀请反馈
	NOTIFY_INV_FILL_REQUIRE_FINISH   = "notify_fill_require_finish" //择优填写邀请反馈

	NOTIFY_POP_CHAT       = "notify_pop_chat"       //聊天无权限返回弹窗
	NOTIFY_POP_CHAT_NOIMG = "notify_pop_chat_noimg" //无头像无法聊天

	NOTIFY_CERTIFY_VIDEO = "notify_certify_video" //视频认证通知
)

// 通知消息的button
type But struct {
	Tip  string                 `json:"tip"`  // 按钮提示文字信息
	Cmd  string                 `json:"cmd"`  // 按钮执行命令
	Def  bool                   `json:"def"`  // 是否为默认事件按钮
	Data map[string]interface{} `json:"data"` // cmd命令执行所需参数
}

/* 通知消息

notify示例：
	{
	"status": "ok",
	"sys_notify": {
		"id": 7,
		"title": "金币变化",
		"content": "每日上线奖励 100金币",
		"img": "http://image1.yuanfenba.net/uploads/oss/avatar_big/201510/11/13383142478.jpg",
		"uid": 5149483,
		"flag": 0,
		"showType": 7,
		"saveFlag": 1,
		"buts": [
		{
			"tip": "忽略",
			"cmd": "cmd_close",
			"def": true,
			"data": {}
		},
		{
			"tip": "查看",
			"cmd": "cmd_xx",
			"def": false,
			"data": {}
		}
		]
	},
	"tm": 1444579364,
	"unread": {
		"plane_free": {
			"num": 20,
			"show": ""
		}
	}
	}
*/
type Notify struct {
	Id       int    `json:"id"`       // 消息类型，详见消息类型ID定义
	Title    string `json:"title"`    // 消息标题
	Content  string `json:"content"`  // 通知消息内容
	Img      string `json:"img"`      // 消息图片
	Uid      uint32 `json:"uid"`      // 用户uid
	Flag     int    `json:"flag"`     // 标记是否需要处理
	ShowType int    `json:"showType"` // 显示类型
	SavaFlag int    `json:"saveFlag"` // 是否保存到通知中心列表中
	Buts     []But  `json:"buts"`
}

// 获取默认按钮(作为，非默认事件按钮)
func GetDefBut(t string) (but But) {
	but.Def = false
	but.Data = map[string]interface{}{}
	if t == BUT_CLOSE {
		but.Cmd = CMD_CLOSE
		but.Tip = "关闭"
	} else if t == BUT_IGNORE {
		but.Cmd = CMD_CLOSE
		but.Tip = "忽略"
	}
	return
}

//获取消息通知按钮
func GetBut(tip, cmd string, def bool, data map[string]interface{}) (but But) {
	but.Cmd = cmd
	but.Def = def
	but.Tip = tip
	if data == nil {
		data = make(map[string]interface{})
	}
	but.Data = data
	return
}

/* 获取通知消息对象title 标题，content 内容，img 图片，data 额外数据 ,nid 通知消息类型id，buts 对应按钮flag是否为待处理
uid img 对应的用户uid，默认为0 ,
save_flag : 该通知消息是否保存于通知中心中
*/
func GenNotify(title, content, img string, nid, flag, show_type int, uid uint32, save_flag int, buts ...But) (n Notify) {
	n.Id = nid
	n.Title = title
	n.Content = content
	n.Img = img
	n.Flag = flag
	n.ShowType = show_type
	n.Buts = buts
	n.Uid = uid
	n.SavaFlag = save_flag
	return n
}

// 根据key获取消息,管理所有通知栏消息中心，在这里控制消息结构和数据, to 接受方，判断性别，决定showType
func GetNotify(uid uint32, key string, data map[string]interface{}, title, content string, to uint32) (n Notify, e error) {
	var u *user_overview.UserViewItem
	var info_uid uint32
	if uid != common.USER_SYSTEM {
		u, e = user_overview.GetUserObjectByUid(uid)
		if e != nil {
			return n, e
		}
		info_uid = uid
	}

	if data == nil {
		data = make(map[string]interface{})
	}
	img := "http://image2.yuanfenba.net/oss/other/chat_notfify_game.png"
	nickname := ""
	if u != nil {
		img = u.Avatar
		nickname = u.Nickname
	}
	buts := make([]But, 0, 0)
	def_ignore_but := GetDefBut(BUT_IGNORE)
	def_close_but := GetDefBut(BUT_CLOSE)
	nid := 0
	flag := 1
	show_type := 7        // 默认全部显示  show_type 为二进制值，0000  应用内弹窗、锁屏、通知栏、应用内通知栏
	var save_flag int = 1 // 是否保存在通知栏中显示(默认1保存)

	ex_m := map[string]int{NOTIFY_ACT_BEGIN: 1, NOTIFY_ACT_OVER: 1, NOTIFY_ACT_AWARD: 1, NOTIFY_CHAT: 1, NOTIFY_SAY_HELLO: 1}
	// 所有通知除活动，私聊，打招呼外，其他女性用户锁屏界面都不显示
	if _, ok := ex_m[key]; to > 0 && !ok {
		to_user, e := user_overview.GetUserObjectByUid(to)
		if e != nil {
			return n, e
		}
		if to_user.Gender == common.GENDER_WOMAN {
			show_type = 3
		}
	}
	switch key {
	case NOTIFY_NEAR:
		title = "来自附近的约会消息"
		data["uid"] = uid
		content = nickname + "说:" + content
		but := GetBut("查看", CMD_NEAR_MSG, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_RECOMMEND

	case NOTIFY_CREATE_TOPIC, NOTIFY_PURSUE_CREATE_TOPIC:
		title = nickname + "创建了一个新话题"
		but := GetBut("聊一聊", CMD_OPEN_TOPIC, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_FRIENDS
		if key == NOTIFY_PURSUE_CREATE_TOPIC {
			nid = NOTIFY_TYPE_PURSUE
		}

	case NOTIFY_MESSAGE, NOTIFY_PURSUE_MESSAGE:
		title = nickname + "发布了一个约会消息"
		but := GetBut("聊一聊", CMD_OPEN_CHAT, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_FRIENDS
		if key == NOTIFY_PURSUE_CREATE_TOPIC {
			nid = NOTIFY_TYPE_PURSUE
		}

	case NOTIFY_FOLLOW:
		title = "关注消息"
		content = nickname + "关注了你"
		data["uid"] = uid
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_FRIENDS

	case NOTIFY_NEW_PHOTO, NOTIFY_PNEW_PHOTO:
		title = "好友动态"
		content = nickname + content
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_FRIENDS
		if key == NOTIFY_PURSUE_CREATE_TOPIC {
			nid = NOTIFY_TYPE_PURSUE
		}

	case NOTIFY_PURSUE_SEND:
		title = "新追求消息"
		content = nickname + "追求了你"
		but := GetBut("查看", CMD_NOHAND_PURSUE, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_PURSUE

	case NOTIFY_PURSUE_ACCEPT:
		title = "追求消息"
		content = nickname + "接受了你的追求"
		but := GetBut("查看", CMD_PURSUE_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_PURSUE

	case NOTIFY_PURSUE_SET:
		title = "女神消息"
		content = nickname + "将你设置为自己的女神"
		but := GetBut("查看", CMD_PURSUE_LIST, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_PURSUE

	case NOTIFY_PURSUE_BREAK:
		title = "解除追求消息"
		content = nickname + "解除了与你的追求关系"
		buts = append(buts, def_close_but)
		nid = NOTIFY_TYPE_PURSUE
		flag = 0

	case NOTIFY_ADD_LUCKY:
		title = "幸运值增加"
		content = nickname + "给你加了一点幸运"
		but := GetBut("查看", CMD_OPEN_LUCKY, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_PURSUE

	case NOTIFY_NEAR_TOPIC:
		title = "来自附近的话题推荐"
		but := GetBut("聊一聊", CMD_OPEN_TOPIC, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_RECOMMEND

	case NOTIFY_TOPIC_MESSAGE:
		title = "圈子消息"
		content = nickname + " :" + content
		but := GetBut("查看", CMD_OPEN_TOPIC, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_TOPIC

	case NOTIFY_GIFT_ACCEPT:
		title = "礼物送出反馈"
		content = nickname + " 接受了你的礼物"
		but := GetBut("查看", CMD_OPEN_CHAT, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_CHAT

	case NOTIFY_VIEW:
		title = "系统消息"
		content = nickname + "正在查看你的资料"
		data["uid"] = uid
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_SYSTEM

	case NOTIFY_GAME_INVITE:
		title = "收到一个游戏邀请"
		content = nickname + "邀请你玩" + content
		but := GetBut("接受", CMD_ENTRY_INVITE_GAME, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_GAME

	case NOTIFY_GAME_ACCEPT:
		title = "邀请成功"
		content = nickname + "接受了你的游戏邀请"
		buts = append(buts, def_close_but)
		nid = NOTIFY_TYPE_GAME
		flag = 0

	case NOTIFY_GAME_REFUSE:
		title = "邀请失败"
		content = nickname + "拒绝了你的游戏邀请"
		buts = append(buts, def_ignore_but)
		nid = NOTIFY_TYPE_GAME
		flag = 0

	case NOTIFY_CHAT:
		title = nickname
		but := GetBut("回复", CMD_OPEN_CHAT, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_CHAT

	case NOTIFY_PLANE_NUM:
		title = "飞行点变化"
		buts = append(buts, def_ignore_but)
		flag = 0
		nid = NOTIFY_TYPE_SYSTEM

	case NOTIFY_COIN:
		title = "金币变化"
		buts = append(buts, def_ignore_but)
		nid = NOTIFY_TYPE_SYSTEM
		flag = 0

	case NOTIFY_SAY_HELLO:
		title = "打招呼"
		content = nickname + "给你打了个招呼！"
		data["uid"] = uid
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_FRIENDS

	case NOTIFY_ACT_AWARD:
		title = "用户中奖通知"
		but := GetBut("我也去玩", CMD_ENTRY_GAME, true, data)
		buts = append(buts, def_ignore_but, but)

	case NOTIFY_ACT_BEGIN:
		but := GetBut("我要参加", CMD_ENTRY_GAME, true, data)
		img = "http://image2.yuanfenba.net/oss/other/chat_notfify_game.png"
		buts = append(buts, def_ignore_but, but)

	case NOTIFY_ACT_OVER:
		but := GetBut("查看", CMD_OPEN_ACTAWARD, true, data)
		img = "http://image2.yuanfenba.net/oss/other/chat_notfify_game.png"
		buts = append(buts, def_ignore_but, but)

	case NOTIFY_HONESTY_CHANGE:
		title = "诚信度变更通知"
		but := GetBut("查看", CMD_HONESTY, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_SYSTEM
		flag = 0
		show_type = 6

	case NOTIFY_INV_PHOTO:
		title = "邀请消息"
		content = nickname + "想要多看看你，邀请你多上传几张生活照!"
		data["uid"] = to
		but := GetBut("去上传", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_INVITE
		flag = 0

	case NOTIFY_INV_FILL_REQUIRE:
		title = "邀请消息"
		content = nickname + "希望能多了解了解你，邀请你填写择友问答"
		data["uid"] = to
		but := GetBut("去填写", CMD_OPEN_REQUIRE, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_INVITE
		flag = 0

	case NOTIFY_INV_CERTIFY:
		title = "邀请消息"
		content = nickname + "对你很感兴趣，邀请你进行认证增加诚信"
		data["uid"] = to
		but := GetBut("去认证", CMD_VIDEO_PRI, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_INVITE
		flag = 0

	case NOTIFY_INV_PHOTO_FINISH:
		title = "邀请消息反馈"
		content = nickname + "刚刚上传了照片"
		data["uid"] = uid
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_INVITE
		flag = 0

	case NOTIFY_INV_CERTIFY_PHONE_FINISH:
		title = "邀请消息反馈"
		content = nickname + "刚刚进行手机认证"
		data["uid"] = uid
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_INVITE
		flag = 0

	case NOTIFY_INV_CERTIFY_IDCARD_FINISH:
		title = "邀请消息反馈"
		content = nickname + "刚刚进行身份证认证"
		data["uid"] = uid
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_INVITE
		flag = 0

	case NOTIFY_INV_CERTIFY_VIDEO_FINISH:
		title = "邀请消息反馈"
		content = nickname + "刚刚进行视频认证"
		data["uid"] = uid
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_INVITE
		flag = 0

	case NOTIFY_INV_FILL_REQUIRE_FINISH:
		title = "邀请消息反馈"
		content = nickname + "刚刚填写了择友问答"
		data["uid"] = uid
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_INVITE
		flag = 0
	case NOTIFY_CERTIFY_VIDEO:
		title = "认证通知"
		but := GetBut("查看", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_SYSTEM
		flag = 0
		show_type = 3

	case NOTIFY_INVITE_FILL:

	case NOTIFY_FILL_FINISH:

	case NOTIFY_ACTIVITY:

	case NOTIFY_SYSTEM:
		buts = append(buts, def_ignore_but)
		nid = NOTIFY_TYPE_SYSTEM
		flag = 0

	default:
		e = errors.New("key [" + key + "] is not found ")
		//	mlog.AppendObj(e, "GetNotify is error")
		return n, e
	}
	n = GenNotify(title, content, img, nid, flag, show_type, info_uid, save_flag, buts...)
	//	mlog.AppendObj(e, "notify --> ", n)
	//	fmt.Println(n)
	return n, nil
}

//增加了能修改img
func GetNotify2(key string, data map[string]interface{}, title, content string, img string, to uint32) (n Notify, e error) {

	if data == nil {
		data = make(map[string]interface{})
	}

	buts := make([]But, 0, 2)
	def_ignore_but := GetDefBut(BUT_IGNORE)
	// def_close_but := GetDefBut(BUT_CLOSE)
	nid := 0
	flag := 1
	// show_type 为二进制值，000 第一位表示是否在软件内显示，第二位是否是在通知栏显示，第三位是否在锁屏弹窗显示
	show_type := 7 // 默认全部显示
	var save_flag int = 1
	ex_m := map[string]int{NOTIFY_ACT_BEGIN: 1, NOTIFY_ACT_OVER: 1, NOTIFY_ACT_AWARD: 1, NOTIFY_CHAT: 1, NOTIFY_SAY_HELLO: 1}
	// 所有通知除活动，私聊，打招呼外，其他女性用户锁屏界面都不显示
	if _, ok := ex_m[key]; to > 0 && !ok {
		to_user, e := user_overview.GetUserObjectByUid(to)
		if e != nil {
			return n, e
		}
		if to_user.Gender == common.GENDER_WOMAN {
			show_type = 3
		}
	}
	switch key {
	case NOTIFY_GIFT_SEND:
		title = "收到新的礼物"
		content = content
		but := GetBut("查看", CMD_OPEN_CHAT, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_CHAT

	case NOTIFY_GIFT_ACCEPT:
		title = "礼物送出反馈"
		content = content
		but := GetBut("查看", CMD_OPEN_CHAT, true, data)
		buts = append(buts, def_ignore_but, but)
		nid = NOTIFY_TYPE_CHAT

	case NOTIFY_PLANE_NUM:
		title = "飞行点变化"
		buts = append(buts, def_ignore_but)
		flag = 0
		nid = NOTIFY_TYPE_SYSTEM

	case NOTIFY_COIN:
		title = "金币变化"
		buts = append(buts, def_ignore_but)
		nid = NOTIFY_TYPE_SYSTEM
		flag = 0

	case NOTIFY_ACTIVITY:

	case NOTIFY_SYSTEM:
		buts = append(buts, def_ignore_but)
		nid = NOTIFY_TYPE_SYSTEM
		flag = 0
	default:
		e = errors.New("key [" + key + "] is not found ")
		//		mlog.AppendObj(e, "GetNotify is error")
		return n, e
	}
	n = GenNotify(title, content, img, nid, flag, show_type, 0, save_flag, buts...)
	//mlog.AppendObj(e, "notify --> ", n)
	return n, nil
}

func GetPopNotifyError(uid uint32, key string, data map[string]interface{}, title, content string, to uint32) (re service.Error) {
	n, e := GetPopNotify(uid, key, data, title, content, to)
	if e != nil {
		return service.NewError(service.ERR_INTERNAL, e.Error())
	}
	b, e := json.Marshal(n)
	if e != nil {
		return service.NewError(service.ERR_INTERNAL, "json 解析错误")
	}
	return service.NewError(service.ERR_POP_NOTIFY, string(b))
}

// 根据key获取消息,管理所有通知栏消息中心，在这里控制消息结构和数据, to 接受方，判断性别，决定showType
func GetPopNotify(uid uint32, key string, data map[string]interface{}, title, content string, to uint32) (n Notify, e error) {
	var u *user_overview.UserViewItem
	var info_uid uint32
	if uid != common.USER_SYSTEM {
		u, e = user_overview.GetUserObjectByUid(uid)
		if e != nil {
			return n, e
		}
		info_uid = uid
	}
	if data == nil {
		data = make(map[string]interface{})
	}
	img := "http://image2.yuanfenba.net/oss/other/chat_notfify_game.png"
	//	nickname := ""
	if u != nil {
		img = u.Avatar
		//		nickname = u.Nickname
	}
	buts := make([]But, 0, 0)
	def_ignore_but := GetDefBut(BUT_IGNORE)
	//	def_close_but := GetDefBut(BUT_CLOSE)
	nid := 0
	flag := 1
	// show_type 为二进制值，0000  应用内弹窗、锁屏、通知栏、应用内通知栏
	show_type := 8 // 默认应用内弹窗
	// 所有通知除活动，私聊，打招呼外，其他女性用户锁屏界面都不显示
	var save_flag int = 1
	switch key {
	case NOTIFY_POP_CHAT: // 聊天无权限时提示
		nid = NOTIFY_TYPE_SYSTEM
		flag = 0
		but := GetBut("去提升星级", CMD_HONESTY, true, data)
		buts = append(buts, def_ignore_but, but)
		save_flag = 0

	case NOTIFY_POP_CHAT_NOIMG: // 聊天无权限(无头像时)弹窗
		nid = NOTIFY_TYPE_SYSTEM
		flag = 0
		but := GetBut("去上传头像", CMD_USER_INFO, true, data)
		buts = append(buts, def_ignore_but, but)
		save_flag = 0

	default:
		return
	}
	n = GenNotify(title, content, img, nid, flag, show_type, info_uid, save_flag, buts...)
	//	fmt.Println("get pop norify: ", n)
	return n, nil
}
