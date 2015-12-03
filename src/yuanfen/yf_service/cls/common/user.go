package common

// 用户性别 GENDER_MAN 男性 GENDER_WOMAN 女性
const (
	GENDER_BOTH  = 0
	GENDER_MAN   = 1
	GENDER_WOMAN = 2
)

const (
	USER_STAT_NORMAL   = 0 //账号正常
	USER_STAT_RESTRICT = 5 //账号被封
)

//标记的类型
const (
	FOLLOW_TAG_NONE     = 0 //未标记
	FOLLOW_TAG_INTEREST = 1 //有好感
	FOLLOW_TAG_FOCUS    = 2 //特别关注
	FOLLOW_TAG_UNLIKE   = 3 //不喜欢（黑名单）
)

const (
	MAX_AGE    = 999 //年龄上限
	MAX_HEIGHT = 999 //身高上限
)

const (
	AVLEVEL_PENDING   = -2 //待审核
	AVLEVEL_INVALID   = -1 //审核不通过
	AVLEVEL_VALID     = 3  //合法但非优质
	AVLEVEL_RECOMMEND = 9  //优质用户
)
