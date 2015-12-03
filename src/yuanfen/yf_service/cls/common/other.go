package common

import "yf_pkg/utils"

const (
	PUSH_TAG_FEMALE = "female"
	PUSH_TAG_MALE   = "male"
)

const (
	USER_SYSTEM       = 1    //系统管理员的UID
	USER_HONGNIANG    = 2    //红娘的UID
	ONLINE_TIMEOUT    = 1200 //用户在线超时时间(秒)
	MAX_RECENT_GUESTS = 30   //系统存储的最近访客的数量上限

	RECOMMENDED_USERS_KEY = "recommended_users" //已经推送过推荐用户的用户集合
)

// 常用result-> msg 返回信息
const (
	MAG_DEF                  = "系统繁忙"
	MAG_INVALID_USER         = "用户验证失败"
	MAG_INVALID_PARAM        = "参数错误"
	MAG_MYSQL_ERROR          = "数据获取失败"
	MAG_REDIS_ERROR          = "获取缓存数据失败"
	MAG_NOT_ENOUGH_MONRY     = "余额不足"
	MAG_NOT_ENOUGH_PLANE_NUM = "飞行点余额不足"
)

//频道前缀j
const (
	TAG_PREFIX_TOPIC = "topic_"
	TAG_PREFIX_GAME  = "game_"
)

//标签类别
const (
	TAG_TYPE_TOPIC    = "topic"    //话题标签所属类别
	TAG_TYPE_ADJACENT = "adjacent" //附近的人标签所属类别
	TAG_TYPE_USER     = "user"     //用户个人信息标签所属类别
)

// 用户状态type通用配置
const (
	STATUS_TYPE_INTOPIC = "in_topic"
	STATUS_TYPE_INGAME  = "in_game"
)

//供养类型
const (
	FEED_GIRL = "Send_Girl_Friend" //供养女神
	FEED_SELf = "Self_Play"        //为自己
)

//话题状态
const (
	TOPIC_STATUS_ACTIVE = 1
	TOPIC_STATUS_CLOSED = 2
)

//用户标签
const (
	UTAG_KEY = "utag"
)

// 用户特权key定义
const (
	PRI_KEY            = "privilege"          // 权限key
	PRI_SAYHI          = "pri_sayhi"          // 打招呼
	PRI_CHAT           = "pri_chat"           // 私聊
	PRI_PURSUE         = "pri_pursue"         //追求
	PRI_NEARMSG        = "pri_nearmsg"        // 附近消息
	PRI_FOLLOW         = "pri_follow"         //关注
	PRI_BIGIMG         = "pri_bigimg"         // 查看大图
	PRI_CONTACT        = "pri_contact"        // 查看用户联系方式
	PRI_PHONELOGIN     = "pri_phonelogin"     // 使用手机登录
	PRI_PRIVATE_PHOTOS = "pri_private_photos" // 查看私密照
	PRI_SEARCH         = "pri_search"         // 使用高级搜索
	PRI_NEARMSG_FILTER = "pri_nearmsg_filter" //附近消息高级定向推送功能
	PRI_SEE_REQUIRE    = "pri_see_require"    // 查看他人择友条件
	PRI_ONLINE_AWARD   = "pri_online_award"   // 专属上线奖励

	PRI_SEEINFO_NOTIFY  = "pri_seeinfo_notify"  //[特殊] 查看个人资料是否发送消息通知
	PRI_INVITE_NOTIFY   = "pri_invite_notify"   //[特殊] 是否发送邀请消息
	PRI_AWARD_PHONECARD = "pri_award_phonecard" //[特殊] 是否能够领取充值卡奖品（手机认证用户）
)

//获取途径配置
const (
	PRI_GET_AVATAR = "pri_get_avatar" // 头像为真实用户头像并审核通过
	PRI_GET_PHOTOS = "pri_get_photos" // 相册至少上传3张生活照
	PRI_GET_INFO   = "pri_get_info"   // 基本必填项填写
	PRI_GET_PHONE  = "pri_get_phone"  // 手机认证
	PRI_GET_VIDEO  = "pri_get_video"  // 视频认证
	PRI_GET_IDCARD = "pri_get_idcard" // 身份证认证
)

// CACHE_GAME 对应的游戏相关的缓存key值
const (
	CACHE_GAME_KEY_GAMELIST = "gamedata_list" // 游戏列表key
	CACHE_GAME_KEY_AWARD    = "award_config"  // 奖品配置表
	CACHE_GAME_KEY_CONFIG   = "com_config"    // 通用配置
)

// 非游戏相关cache配置
const (
	CACHE_KEY_HONESTY = "honesty_config" // 诚信特权配置
	CACHE_KEY_VERSION = "app_version"    // 版本信息配置
	CACHE_KEY_APPIMG  = "app_img"        // app注册图片
)

const (
	LAT_NO_VALUE = 1000 //没有经度
	LNG_NO_VALUE = 1000 //没有纬度
)

const PIC_NOALLOWN_MSG = "你的图片审核失败"

// 动态和评论相关常量定义
const (
	// 动态类型 1：用户动态 2：小游戏 3：文章
	DYNAMIC_TYPE_USER    = 1
	DYNAMIC_TYPE_GAME    = 2
	DYNAMIC_TYPE_ARTICLE = 3
	//用户动态类型 0 主动发送 1 交友寄语 2 形象照
	DYNAMIC_STYPE_USER    = 0
	DYNAMIC_STYPE_ABOUTME = 1
	DYNAMIC_STYPE_AVATAR  = 2

	// 评论类型1. 点赞 2. 评论 ，3.拼图游戏时间
	COMMENT_TYPE_LIKE    = 1
	COMMENT_TYPE_COMMENT = 2
	COMMENT_TYPE_GAME    = 3

	//评论资源类型，1 ： 动态
	COMMENT_SOURCE_TYPE_DYNAMIC = 1
)

//认识一下的消息状态
const (
	SAYHELLO_MSG_UNREAD  = 1     //未读
	SAYHELLO_MSG_READ    = 2     //已读
	SAYHELLO_MSG_REPLIED = 3     //已回复
	SAYHELLO_TARGET_ME   = "me"  //给我发的认识请求
	SAYHELLO_TARGET_HIM  = "him" //我发出的认识请求
)

//学历
const (
	EDU_NO_LIMIT           = 0 //没填或不限
	EDU_HIGH_SCHOOL        = 1 //高中
	EDU_THREE_YEAR_COLLEGE = 2 //专科
	EDU_BACHELOR           = 3 //本科
	EDU_POSTGRADUATE       = 4 //硕士
)
const MAX_FOLLOW_NUM = 2000 //标记人数的上限

//必填项和选填项列表
const ( //形象照、职业、年龄、身高、称呼、家乡、工作单位、工作地点、毕业学校
	USERCOMPLETE_LIST_MUST   = "avatar,job,age,height,nickname,homecity,workarea" //必填项
	USERCOMPLETE_LIST_CHOOSE = "aboutme,interest,tag,school,workunit"             //选填项
)

const (
	MSG_START_POS               = "s_msg"                                          //起始消息ID的key
	LAST_MSG_ID_PREFIX          = "l_msg"                                          //与某个用户的最后一条聊天记录的key的前缀
	LAST_SAYHELLO_PREFIX        = "l_shl"                                          //最后一条认识一下的消息ID的前缀
	LAST_SAYHELLO_TO_ME_PREFIX  = LAST_SAYHELLO_PREFIX + "_" + SAYHELLO_TARGET_ME  //给我发送的最后一条认识一下的消息ID的前缀
	LAST_SAYHELLO_TO_HIM_PREFIX = LAST_SAYHELLO_PREFIX + "_" + SAYHELLO_TARGET_HIM //我发送的最后一条认识一下的消息ID的前缀
)

//不参与同行推荐的职业
var JobNotRecommend = map[string]bool{"其它": true, "学生": true}

//搜索半径（公里）
var LOCATION_RADIUS float64 = utils.KmToLng(10)
var WORKPLACE_RADIUS float64 = utils.KmToLng(10)
var DATEPLACE_RADIUS float64 = utils.KmToLng(10)
