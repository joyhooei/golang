package common

//redis数据库的表名
const (
	REDIS_TOPIC_TREND      = 1
	REDIS_TOPIC_BLACKLIST  = 2
	REDIS_TOPIC_USERS      = 3
	REDIS_UNREAD_TIME      = 4
	REDIS_PHONE_CODE       = 5  //校验短信验证码
	REDIS_PLANE_INVITE     = 6  //飞机游戏邀请组队记录
	REDIS_USER_STATUS      = 7  //用户状态记录表
	REDIS_SAYHELLO         = 8  //打过招呼的用户集合
	REDIS_FOLLOW           = 9  //关注的用户集合,f_开头的key表示我关注的，t_开头的表示关注我的
	REDIS_FRIENDS          = 10 //聊过天的用户集合
	REDIS_LOCATION         = 11 //用户的位置
	REDIS_HONGNIANG        = 12 //分配给用户的红娘，key-用户ID，value-红娘ID
	REDIS_RECENT_GUESTS    = 13 //最近访客，每个用户最多存储30条
	REDIS_LOGIN_TIME       = 14 //记录用户上线时间.用来记算等级
	REDIS_USER_TOPIC       = 15 //用户加入的话题id
	REDIS_GEO_CITY         = 16 //坐标(精确到小数点后两位)到城市的映射
	REDIS_BLACKLIST        = 17 //黑名单
	REDIS_TOPIC_RECENT_MSG = 18 //话题最近N条消息
	REDIS_USER_TAG         = 19 //用户标签
	REDIS_REG_TIME         = 20 //用户注册时间

)

//redis缓存的表名
const (
	CACHE_USER_ABOUT    = 1
	CACHE_USER_OVERVIEW = 2
	CACHE_USER_PASSWORD = 3
	CACHE_DISCOVERY     = 4
	CACHE_TOPIC         = 5
	CACHE_USER_SYSTEM   = 6
	CACHE_USER_VALID    = 7
	CACHE_GAME          = 8
	CACHE_TAG           = 9
	CACHE_USER_ALLOT    = 10 //用户是否最近分配过男客服
	CACHE_REGIP         = 11 //IP注册用户数
	CACHE_CAN_LOCALTAG  = 12 //是否能发本地标签判断
	CACHE_CAN_TEN       = 13 //是否今天能关注邀请游戏

	CACHE_DB = 30
)
