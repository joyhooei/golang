package common

/*
消息类型

MSG_TYPE_TEXT: 文本消息

消息格式：
	{
		"type":"text",				//消息类型
		"content": "xxxx",   			//消息内容
		"img": ["url1","url2"],  		//附带图片
		"tm": "1992-06-28T00:00:00+08:00"	//发送时间，发送时不填此项
	}

MSG_TYPE_VOICE: 语音消息

消息格式：
	{
		"type":"voice",		//消息类型
		"f_uid":5016734,	//发送者UID
		"length":2,		//时长（秒）
		"url":"http://dl.yuanfenba.net/uploads/oss/voice/201508/05/18002931436.amr",	//声音文件
		"tm":"2015-08-05T18:00:30+08:00"	//发送时间，发送时不填此项
	}

MSG_TYPE_PIC: 图片消息

消息格式：
	{
		"type":"pic",		//消息类型
		"f_uid":5049372,	//发送者UID
		"img":"http://image1.yuanfenba.net/uploads/oss/photo/201508/05/17033241871.png",	//图片地址
		"tm":"2015-08-05T17:03:34+08:00"	//发送时间，发送时不填此项
	}

MSG_TYPE_LOCATION: 位置分享消息

消息格式：
	{
		"type":"location",		//消息类型
		"img": "http://image1.yuanfenba.net/uploads/oss/photo/201508/05/17033241871.png",
		"place": {
			"id": "e308284843d29fa6b24c49f7",
			"name": "COSTA COFFEE(阜成门店)",
			"address": "北京市西城区阜成门外大街1号华联商厦一层F1-01",
			"pic": "",
			"lat": 39.929748,
			"lng": 116.360441,
			"distence": 0
		}
	}


MSG_TYPE_DEL_MSG: 删除消息的消息

消息格式：
	{
		"type":"del_msg",	//消息类型
		"msgid":2823163,	//被删除的消息ID
		"tm":"2015-08-02T16:01:32+08:00"	//发送时间，发送时不填此项
	}

MSG_TYPE_JOIN_TOPIC: 加入话题消息

消息格式：
	{
		"type":"join_topic",
		"uid":5011028,	//加入者UID
		"tid":3629,		//话题ID
		"top_users":[
			{
				"uid":5011028,
				"nickname":"体贴の小马驹",
				"gender":1,
				"age":26,
				"avatar":"http://image2.yuanfenba.net/uploads/oss/photo/201507/04/02281863592.jpg"
			}
		],	//活跃用户列表
		"total":1,	//话题总人数
		"tm":"2015-07-19T20:04:53+08:00"
	}

MSG_TYPE_LEAVE_TOPIC: 离开话题消息

消息格式：
	{
		"type":"leave_topic",
		"uid":5003148,		//离开话题的用户ID
		"mode":"self",	//离开话题的方式。self-自行离开，kick-被踢出话题
		"tid":3784,	//话题ID
		"top_users":[
			{
				"uid":5000010,
				"nickname":"纯真滴小虾滑",
				"gender":2,
				"age":21,
				"avatar":"http://image2.yuanfenba.net/uploads/oss/photo/201506/19/16085560380.jpg"
			}
		],
		"total":15,
		"tm":"2015-07-20T12:30:39+08:00"
	}

MSG_TYPE_TOPIC_TOP_USERS: 话题活跃用户变化通知

消息格式：
	{
		"type":"topic_top_users",
		"tid":3057,
		"top_users":[
			{
				"uid":5025405,
				"nickname":"ち心动o变痛",
				"gender":1,
				"age":27,
				"avatar":"http://image1.yuanfenba.net/uploads/oss/photo/201507/11/10091435129.jpg"
			}
		],
		"total":2,
		"tm":"2015-08-05T18:17:25+08:00"
	}

MSG_TYPE_SAYHELLO: 打招呼消息

消息格式：
	{
		"type":"say_hello"
		"f_nickname":"qk234399126",	//打招呼用户昵称
		"f_uid":5045733,	//打招呼用户ID
		"t_uid":1006620,	//收到打招呼消息的用户ID
		"tm":"2015-08-05T18:34:35+08:00"
	}

MSG_TYPE_FOLLOW: 关注消息

消息格式：
	{
		"type":"follow",
		"f_uid":5050101,	//关注者
		"t_uid":5047452,	//被关注者
		"tm":"2015-08-05T18:18:58+08:00"
	}

MSG_TYPE_UNREAD: 未读消息

消息格式：
	{
		"type":"unread",
		"unread":{
			"visit":{
				"num":31,	//未读数
				"show":""	//显示的内容，如果为空客户端直接显示数字，如果为"[红点]"，客户端显示一个红点
			}	//未读项，具体类型参考cls.common下UNREAD_前缀的常量
		}
		"tm":"2015-08-05T15:38:10+08:00"
	}

MSG_TYPE_INVITE_FILL: 邀请完善资料消息

消息格式：
	{
		"type":"invite_fill",				//消息类型
		"notify":{},		//参见通用的notify格式
		"tm": "1992-06-28T00:00:00+08:00"		//发送时间，发送时没有此项，接收时会有
	}

MSG_TYPE_FILL_FINISH: 资料完善后的通知

消息格式：
	{
		"type":"fill_finish",				//消息类型
		"notify":{},		//参见通用的notify格式
		"tm": "1992-06-28T00:00:00+08:00"		//发送时间，发送时没有此项，接收时会有
	}

MSG_TYPE_SYS_NOTICE:系统通知消息，客户端在获取离线消息时会处理此类型的消息，而不是直接扔掉

消息格式：
	{
		"type":"sys_notice",			//消息类型
		"stype":"game",					//消息子类型，具体有哪些子类型可以参看SYS_NOTICE_开头的常量
		"content":{},					//消息内容，不同子类型的消息content的格式会不同
		"tm": "1992-06-28T00:00:00+08:00"	//发送时间，发送时不填此项
	}


MSG_TYPE_PIC_INVALID:图片审核失败通知（私聊，群聊面板显示）

消息格式：
	{
		"type":"pic_invalid",			//消息类型
		"content": "xxxx",   			//消息内容
		"msgid": 50001,   			    //发送失败的消息id
		"but":{  						//超链接以及点击效果，如没有此字段则无超链接以及点击特效
			"tip":string		// 按钮提示文字信息
			"cmd":string		// 按钮执行命令
			"def":true      	// 是否为默认事件按钮 消息中无作用
			"Data":{} 			// cmd命令执行所需参数
		}
		"tm": "1992-06-28T00:00:00+08:00"	//发送时间，发送时不填此项
	}

MSG_TYPE_MYDYNAMIC_MSG:我的动态新消息 &  MSG_TYPE_DYNAMIC_MSG:参与动态新消息

消息格式：
	{
		"type": "my_dynamic_msg | dynamic_msg",
		"tm": "2015-09-30T16:56:18+08:00",
		"comment_info": {   // 评论信息，详细见http://120.131.64.91:8182/pkg/yuanfen/yf_service/modules/dynamics/#DynamicsModule.SecCommentList
			"comment": {
				"content": "哈哈哈哈",
				"id": 36,
				"ruid": 0,
				"source_id": 61,
				"tm": "2015-09-30T16:56:17+08:00",
				"uid": 5000761
			},
			"ruser": {
				"age": 0,
				"avatar": "",
				"job": "UI设计师",
				"nickname": ""
			},
			"user": {
				"age": 24,
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
				"job": "演艺人员",
				"nickname": "小气&豪猪"
			}
		},
		"dynamic_info": { // 动态信息，详见http://120.131.64.91:8182/pkg/yuanfen/yf_service/modules/dynamics/#DynamicsModule.SecMarkNew
			"dynamic": {
				"comments": 3,
				"gameinit": "",
				"gamekey": 0,
				"id": 61,
				"isLike": 1,
				"likes": 1, "location": "",
				"pic": [
				"http://image1.yuanfenba.net/uploads/oss/photo/201509/29/16311040627.jpg"
				],
				"sign": 2,
				"stype": 0,
				"text": "fffff fa yi tiao",
				"tm": "2015-09-29T16:31:18+08:00",
				"type": 1,
				"uid": 5001009,
				"url": ""
			},
			"user": {
				"age": 0,
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201509/09/18185488803.jpg",
				"job": "航空公司",
				"nickname": "打酱油得犰狳"
			}
		}
	}


MSG_TYPE_DYNAMIC_MARKUSER: 标记用户新动态通知

消息格式：
	{
		"type":"dynamic_markuser",
		"uid":5000762,      // 动态发布人
		"dynamic_id":1200    // 新动态id
	}

MSG_TYPE_DATE_NOTIFY: 通知用户可以发起约会

消息格式：
	{
		"type":"date_notify",
		"sender":{
			"uid":123,
			"gender":1,
			"nickname":"justin",
			"avatar":"http://image1.yuanfenba.net/uploads/oss/photo/201509/09/18185488803.jpg"
		},
		"uid":5000762      //约会对象
	}

MSG_TYPE_DATE_REQUEST: 向对方发送的约会邀请

消息格式：
	{
		"type":"date_request",
		"date_time": "2015-09-30T16:56:17+08:00",	//约会时间
		"text":"“asdf”选择了见面的地点和时间",
		"place":{		//约会地点
			"id":"xfjdsl12",
			"name":"清河翠微",
			"address":"...",
			"lat":123.11,
			"lng":11.123
		}
		"sender":{
			"uid":123,
			"gender":1,
			"nickname":"justin",
			"avatar":"http://image1.yuanfenba.net/uploads/oss/photo/201509/09/18185488803.jpg"
		},
		"my_workplace":{	//我的工作地点经纬度
			"uid":123,
			"nickname":"justin",
			"avatar":"http://image1.yuanfenba.net/uploads/oss/avatar/201511/03/14254456779.jpg",
			"location":{
				"lat":22.1132,
				"lng":12.13354
			}
		},
		"him_workplace":{	//对方的工作地点经纬度
			"uid":123,
			"nickname":"justin",
			"avatar":"http://image1.yuanfenba.net/uploads/oss/avatar/201511/03/14254456779.jpg",
			"location":{
				"lat":22.1132,
				"lng":12.13354
			}
		}
	}
*/
const (
	MSG_TYPE_TEXT             = "text"     //文本消息
	MSG_TYPE_VOICE            = "voice"    //语音消息
	MSG_TYPE_PIC              = "pic"      //图片消息
	MSG_TYPE_LOCATION         = "location" //位置分享消息
	MSG_TYPE_DEL_MSG          = "del_msg"  //删除消息
	MSG_TYPE_RECEIVED         = "received" //消息已送达
	MSG_TYPE_READ             = "read"     //消息已读
	MSG_TYPE_JOIN_TOPIC       = "join_topic"
	MSG_TYPE_SYS_NOTICE       = "sys_notice" //系统通知
	MSG_TYPE_LEAVE_TOPIC      = "leave_topic"
	MSG_TYPE_TOPIC_TOP_USERS  = "topic_top_users"  //话题活跃用户变化通知
	MSG_TYPE_SAYHELLO         = "say_hello"        //打招呼
	MSG_TYPE_FOLLOW           = "follow"           //关注
	MSG_TYPE_UNREAD           = "unread"           //未读更新消息
	MSG_TYPE_OTHER            = "other"            //其它自定义类型的消息
	MSG_TYPE_INVITE_FILL      = "invite_fill"      //邀请完善资料消息
	MSG_TYPE_FILL_FINISH      = "fill_finish"      //资料完善后的通知
	MSG_TYPE_PIC_INVALID      = "pic_invalid"      //图片审核不通过
	MSG_TYPE_MYDYNAMIC_MSG    = "my_dynamic_msg"   //我的动态消息（新评论，新点赞，新游戏结果）
	MSG_TYPE_DYNAMIC_MSG      = "dynamic_msg"      //参与他人动态互动消息（评论回复）
	MSG_TYPE_DYNAMIC_MARKUSER = "dynamic_markuser" //我标记的用户发布行动态，角标通知
	MSG_TYPE_DATE_NOTIFY      = "date_notify"      //可以约会的通知
	MSG_TYPE_DATE_REQUEST     = "date_request"     //约会邀请
)

const (
	RICHTEXT_TYPE_TEXT   = "text"   //文本消息项
	RICHTEXT_TYPE_BUTTON = "button" //按钮项
	RICHTEXT_TYPE_IMG    = "img"    //大图片消息项
	RICHTEXT_TYPE_ICON   = "icon"   //小图消息项
)

/*
MSG_TYPE_RICHTEXT 图文消息
{
		"type":"richtext",			//消息类型
		"folder":"sys_notice",		//消息目录 若包含此字段则为系统消息 否则为用户聊天消息
		"tip":"登陆奖励金币"   //消息简略信息 可用于在列表中显示和消息提醒
		"msgs":[   //消息节点列表 可包含多个不同类型消息项
			{
				"type":"img"  //消息项 类型 列表在 http://120.131.64.91:8082/pkg/yuanfen/yf_service/cls/common/#RICHTEXT_TYPE_TEXT
				"img":"http://image2.yuanfenba.net/uploads/oss/photo/201507/13/19361219824.png"//大图片
				"text":"登陆奖励金币"//文字
				"but":{  //[opt]点击效果 同NOTIFY中的but 只有cmd,Data有效 可选 没有此项表示没有点击效果
					"cmd":string		// 按钮执行命令
					"Data":{} 			// cmd命令执行所需参数
				}
			},
			{
				"type":"icon"//消息项 类型
				"img":"http://image2.yuanfenba.net/uploads/oss/photo/201507/13/19361219824.png"//小图片
				"text":"登陆奖励金币"//文字
				"but":{  //点击效果 可选
					"cmd":string		// 按钮执行命令
					"Data":{} 			// cmd命令执行所需参数
				}
			}，
			{
				"type":"text"//消息项 类型
				"text":"登陆奖励金币"//文字
				"but":{  //点击效果 可选
					"cmd":string		// 按钮执行命令
					"Data":{} 			// cmd命令执行所需参数
				}
			},
			{
				"type":"button"//消息项 类型
				"but":{  //点击效果 同NOTIFY中的but 只有cmd,Data有效
					"tip":string		// 按钮提示文字信息
					"cmd":string		// 按钮执行命令
					"def":true      	// 是否为默认事件按钮 消息中无作用
					"Data":{} 			// cmd命令执行所需参数
				}
			},{...}
		],
		"tm": "1992-06-28T00:00:00+08:00"	//发送时间，发送时不填此项
}
*/
const (
	MSG_TYPE_RICHTEXT = "richtext" //图文消息
)

//用户聊天消息的类型集合
var ChatMessageTypes = []string{MSG_TYPE_TEXT, MSG_TYPE_PIC, MSG_TYPE_VOICE, MSG_TYPE_GIVE_PRESENT, MSG_TYPE_RICHTEXT}

/*
IsChatMessage判断消息的类型是否是用户发送的聊天消息
*/
func IsChatMessage(mtype string) bool {
	for _, t := range ChatMessageTypes {
		if t == mtype {
			return true
		}
	}
	return false
}

/*
MSG_TYPE_BIGFACE:大表情消息

消息格式：
	{
		"type":"bigface",			//消息类型
		"id":111,				//表情ID
		"name":"在不在",		//表情名称
		"ico":"http://...",	//表情图标url·
		"pic":"http://...",		//表情效果图片
		"tm": "1992-06-28T00:00:00+08:00"	//发送时间，发送时不填此项
	}

MSG_TYPE_MINI_GAME:小游戏消息

消息格式：
	{
		"type":"mini_game",			//消息类型
		"f_uid":5016734,	//发送者UID
		"game_id": 1,   			//小游戏ID 1为猜拳 2为骰子
		"game_name": "猜拳",   			//小游戏ID 1为猜拳 2为骰子
		"data": {   			    //小游戏的结果 根据游戏类型返回 目前的两款游戏只有 result 一个参数
			"ico":"http://...",		//小游戏图标url·
			"pic":"http://...",		//小游戏效果图片
			"resultimg":"http://",			//小游戏停止图
			"result":1
 		}
		"tm": "1992-06-28T00:00:00+08:00"	//发送时间，发送时不填此项
	}
*/
const (
	MSG_TYPE_BIGFACE   = "bigface"   //表情消息
	MSG_TYPE_MINI_GAME = "mini_game" //小游戏消息
)

/*
MSG_TYPE_GIVE_PRESENT:送礼消息

消息格式：
	{
		"type":"give_present",
		"f_uid":"1001",  			//发送者UID
		"t_uid":"1601",				//接收者UID
		"gid":"23",					//礼物ID
		"gift_record_id":"12222",	//礼物记录ID
		"gift_name":"玫瑰花",		//礼物名称
		"gift_info":"一朵玫瑰花",	//礼物描述
		"gift_img":"/aaa/bbb.jpg"，	//礼物略缩图
		"gift_res":"http://res1.zip"，//礼物资源图片包
		"f_nickname":"票子"，		//送礼者的昵称
		"t_nickname":"小白",			//收礼者的昵称
		"tm": "1992-06-28T00:00:00+08:00"	//发送时间，发送时不填此项
	}
*/
const (
	MSG_TYPE_GIVE_PRESENT    = "give_present"    //送礼
	MSG_TYPE_THX_PRESENT     = "thx_present"     //收礼答谢
	MSG_TYPE_SURPRISE_THX    = "surprise_thank"  //收惊喜答谢
	MSG_TYPE_SURPRISE_NOTIFY = "surprise_notify" //惊喜通知
	MSG_TYPE_LOGIN           = "login"           //用户上线消息 暂时用于给客服后台推送女用户上线
	MSG_TYPE_MEN_REG         = "menreg"          //男用户注册消息 暂时用于给客服后台推送
	MSG_TYPE_WOMEN_REG       = "nvreg"           //男用户注册消息 暂时用于给客服后台推送
	MSG_TYPE_CREATE_TOPIC    = "create_topic"    //用户创建话题 暂时用于给客服后台推送
)

/*
MSG_TYPE_CERTIFY_IDCARD:身份证认证结果通知 |
MSG_TYPE_CERTIFY_PHONE:手机认证结果通知  |
MSG_TYPE_CERTIFY_VIDEO:视频认证结果通知

消息格式：
	{
		"type":"certify_idcard [certify_phone |  certify_video]",		//消息类型
		"flag":true/false,	//认证是否成功,true:成功,false:失败
		"tip":"认证成功",		//认证提示
		"certify_level":2      //认证等级
		"tm":"2015-08-05T18:00:30+08:00"	//发送时间，发送时不填此项
	}


MSG_TYPE_HONESTY_CHANGE:诚信等级变化通知

消息格式：
	{
		type:"honesty_change"
		now_level:3
		honesty:{
			pri_chat:{xxx}  该结构见http://120.131.64.91:8082/pkg/yuanfen/yf_service/cls/data_model/certify/#HonestyPri
			pri_pursue:{xxx}
		}
	}

*/
const (
	MSG_TYPE_PURSUE_RELIEVE = "pursue_relieve" //解除追求关系
	MSG_TYPE_ADDLUCKY       = "lucky_notify"   //幸运通知
	MSG_TYPE_PURSUE_NOTIFY  = "pursue_notify"  //追求通知
	MSG_TYPE_GAME_ACT_BEGIN = "game_act_begin" //活动开始（游戏活动）
	MSG_TYPE_HONESTY_CHANGE = "honesty_change" //诚信等级变化通知
	MSG_TYPE_CERTIFY_IDCARD = "certify_idcard" //身份证认证结果通知
	MSG_TYPE_CERTIFY_PHONE  = "certify_phone"  //手机认证结果通知
	MSG_TYPE_CERTIFY_VIDEO  = "certify_video"  //视频认证结果通知

)

/*
MSG_TYPE_VERSION://版本更新消息(用于主动发送版本更新消息)

消息格式：
	{
		type:"update_version"
		tm:"2015-08-05T18:00:30+08:00"	//发送时间，发送时不填此项
	}
*/
const (
	MSG_TYPE_VERSION = "update_version" //版本更新消息(用于主动发送版本更新消息)
)

// 游戏相关消息类型定义
const (
	MSG_TYPE_GAME_INVITE           = "game_invite"           //邀请游戏
	MSG_TYPE_GAME_ACCEPT           = "game_accept"           //接受游戏邀请
	MSG_TYPE_PLANE_TEAM_INV        = "plane_team_inv"        //邀请组队
	MSG_TYPE_PLANE_TEAM_NOTIFY     = "plane_team_notify"     //组队成功全局通知
	MSG_TYPE_PLANE_TEAM_SUC        = "plane_team_suc"        //组队成功队友通知
	MSG_TYPE_PLANE_TEAM_REFUSE     = "plane_team_refuse"     //拒绝组队邀请
	MSG_TYPE_PLANE_TEAM_EXIT       = "plane_team_exit"       //被邀请人退出房间
	MSG_TYPE_PLANE_BTEAM_SUC       = "plane_bteam_suc"       //解除组队队友通知
	MSG_TYPE_PLANE_BTEAM_NOTIFY    = "plane_bteam_notify"    //解除组队全局通知
	MSG_TYPE_CONFIRM_TEAM_OK       = "plane_cteam_ok"        //继续组队确认消息
	MSG_TYPE_CONFIRM_TEAM_FALSE    = "plane_cteam_false"     //取消继续组队
	MSG_TYPE_CHANGE_PLANE          = "plane_change_plane"    //组队期间换飞机通知
	MSG_TYPE_PLANE_ENTRY           = "plane_entry"           //进入游戏聊天室通知
	MSG_TYPE_PLANE_EXIT            = "plane_exit"            //退出游戏聊天室通知
	MSG_TYPE_PLANE_OFFLINE         = "plane_offline"         //用户掉线
	MSG_TYPE_PLANE_BEGIN           = "plane_begin"           //开始游戏
	MSG_TYPE_PLANE_BEGIN_NOTIFY    = "plane_begin_notify"    //开始游戏全局通知
	MSG_TYPE_PLANE_AWARD           = "plane_award"           //中奖通知
	MSG_TYPE_PLANE_PAY             = "plane_pay"             //付费通知
	MSG_TYPE_PLANE_PAY_FAILED      = "plane_pay_failed"      //付费失败通知
	MSG_TYPE_PLANE_GLOBAL_TEAM_INV = "plane_global_team_inv" //全局邀请组队
	MSG_TYPE_PLANE_MATCH_TEAM_SUC  = "plane_match_team_suc"  //匹配组队成功通知队友
	MSG_TYPE_PLANE_USER_STATUS     = "plane_user_status"     //用户状态变更通知
	MSG_TYPE_PLANE_GAMEOVER        = "plane_game_over"       //飞机大战游戏结束全局通知
	MSG_TYPE_PLANE_READY_STATUS    = "plane_ready_status"    //飞机大战组队准备通知
)

/*
MSG_TYPE_HINT:小提示，在消息框中为灰色小字

消息格式：
	{
		"type":"hint",			//消息类型
		"isSave":true,			//是否存储该消息
		"content": "xxxx",   			//消息内容,可包含换行符(表示换行)
		"msgid": 50001,   			    //发送失败的消息id
		"but":{  						//超链接以及点击效果，如没有此字段则无超链接以及点击特效
			"tip":string		// 按钮提示文字信息
			"cmd":string		// 按钮执行命令
			"def":true      	// 是否为默认事件按钮 消息中无作用
			"Data":{} 			// cmd命令执行所需参数
		}
		"tm": "1992-06-28T00:00:00+08:00"	//发送时间，发送时不填此项
	}
*/
const (
	MSG_TYPE_HINT = "hint" //小提示消息，在消息框中为灰色小字
)

/*
MSG_TYPE_KICK:重复登陆,T下线消息

消息格式：
	{
		"type":"kick",			//消息类型
		"imei": "AABBCCDD",   	//上线用户的IMEI,客户端发现和自己IMEI一样就不下线，防止误T,
		"tm": "1992-06-28T00:00:00+08:00"	//发送时间，发送时不填此项
	}
*/
const (
	MSG_TYPE_KICK = "kick" //uid重复登陆,你将被T下线
)

//系统消息类型
const (
	SYS_MSG_PAY     = "pay"     //充值的系统消息
	SYS_MSG_FEND    = "fend"    //女神收到的供养系统消息
	SYS_MSG_WORK    = "work"    //打工挣钱的系统消息
	SYS_MSG_Relogin = "relogin" //重复登陆
	SYS_MSG_OTHER   = "other"   //其他消息
)

const (
	SYS_NOTICE_SHOW_KEY = "show_in_list" //系统通知是否显示在其它消息的列表中
)

//其他消息里的系统UID
const (
	UID_SYSTEM           = 1001    //系统消息
	UID_EVENT            = 1002    //活动
	UID_PROBLEM          = 1003    //问题反馈
	UID_COMMENT          = 1004    //评论别人的动态有新回复
	UID_AWARD            = 1005    //中奖通知
	UID_COMMENT_TOME     = 1101    //我的动态有新评论
	UID_MARK_NEW_DYNAMIC = 1102    //我标记用户有新动态（角标）
	UID_COMMON_HIDE      = 1100    //通用的隐藏系统消息（不同存消息列表，一般用于推送不需要在客户端消息记录中出现的包含某种特效的系统消息）
	UID_DATE_MSG         = 1103    //约会消息
	UID_MAX_SYSTEM       = 1000000 //用户最大系统UID，小于此值为系统UID，大于为用户UID
)

const (
	FOLDER_OTHER = "sys_notice" //其他消息的folder名称
	FOLDER_HIDE  = "hide"       //隐藏消息的folder名称
	FOLDER_KEY   = "folder"
)
