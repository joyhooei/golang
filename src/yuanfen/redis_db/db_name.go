package redis_db

//redis数据库的表名
const (
	REDIS_MISC               = 0 //用来存放一些零散的key
	REDIS_TOPIC_TREND        = 1
	REDIS_TOPIC_BLACKLIST    = 2
	REDIS_TOPIC_USERS        = 3
	REDIS_UNREAD_TIME        = 4
	REDIS_PHONE_CODE         = 5  //校验短信验证码
	REDIS_PLANE_INVITE       = 6  //飞机游戏邀请组队记录
	REDIS_USER_STATUS        = 7  //用户状态记录表
	REDIS_SAYHELLO           = 8  //打过招呼的用户集合
	REDIS_FOLLOW             = 9  //标记的用户集合,f_开头的key表示我标记的，t_开头的表示标记我的
	REDIS_FRIENDS            = 10 //聊过天的用户集合
	REDIS_LOCATION           = 11 //用户的位置
	REDIS_HONGNIANG          = 12 //分配给用户的红娘，key-用户ID，value-红娘ID
	REDIS_RECENT_CHAT_USERS  = 13 //最近聊天用户列表
	REDIS_LOGIN_TIME         = 14 //记录用户上线时间.用来记算等级
	REDIS_USER_TOPIC         = 15 //用户加入的话题id
	REDIS_GEO_CITY           = 16 //坐标(精确到小数点后两位)到城市的映射和城市到坐标的映射
	REDIS_BLACKLIST          = 17 //黑名单
	REDIS_TOPIC_RECENT_MSG   = 18 //话题最近N条消息
	REDIS_USER_TAG           = 19 //用户标签
	REDIS_REG_TIME           = 20 //用户注册时间
	REDIS_USER_PRI           = 21 //用户特权表
	REDIS_LOCALTAG_VIEWERS   = 22 //查看过用户本地标签的用户列表
	REDIS_INVITE_FILL        = 23 //邀请完善资料的用户列表
	REDIS_USER_PRI_CHAT      = 24 //用户聊天特权记录
	REDIS_DYNAMIC            = 25 //用户动态列表
	REDIS_USER_DATA          = 26 //用户数据表
	REDIS_RECOMMEND_USERS    = 27 //已推荐用户列表，保存最近1个月的
	REDIS_GAME               = 28 //游戏平台通用表
	REDIS_PHONE_CITY         = 29 //手机号到城市对应表
	REDIS_USER_MSG_START_POS = 30 //针对好友的聊天记录的起始位置
	REDIS_CHANGE_NICKAVATAR  = 31 //用户最后更改昵称和头像的时间

	//hub使用
	REDIS_TAG_USERS = 111 //tag下的user集合
	REDIS_USER_TAGS = 112 //user下的tag集合
	REDIS_USER_NODE = 113 //user所在的节点j
)

const (
	REDIS_GAME_KEY_TAGBASE = "game_tag_base" // GAME 平台下聊天室基数
)

//redis缓存的表名
const (
	CACHE_USER_ABOUT      = 1
	CACHE_USER_OVERVIEW   = 2
	CACHE_USER_PASSWORD   = 3
	CACHE_DISCOVERY       = 4
	CACHE_TOPIC           = 5
	CACHE_USER_SYSTEM     = 6
	CACHE_USER_VALID      = 7
	CACHE_GAME            = 8
	CACHE_TAG             = 9
	CACHE_USER_ALLOT      = 10 //用户是否最近分配过男客服
	CACHE_REGIP           = 11 //IP注册用户数
	CACHE_ONTOP           = 12 //切换到前台的次数
	CACHE_CAN_TEN         = 13 //是否今天能关注邀请游戏
	CACHE_INVITE_FILL     = 14 //上次给用户发送邀请完善资料通知的时间，24小时超时
	CACHE_VERSION         = 15 //版本信息缓存
	CACHE_SAYHELLO_TIMES  = 16 //认识一下消息的发送次数，缓存72小时
	CACHE_DYNAMIC         = 17 //动态缓存
	CACHE_RECOMMEND_USERS = 18 //已推荐用户数的缓存，有效期一天
	CACHE_SMS_RESEND      = 19 //上次发送短信时间，两次间隔最小不能

	CACHE_DB = 30
)
