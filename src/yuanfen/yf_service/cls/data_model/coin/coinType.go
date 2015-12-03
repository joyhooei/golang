package coin

// 消费类型，对应类型 1为充值，2为赠送,3为打工获取,4为消费
//消费主类型
const (
	EARN_CHARGE      = 1   //充值
	EARN_GIVE        = 2   //赠送
	EARN_WORK        = 3   //打工获取
	EARN_GIFT        = 4   //礼物
	EARN_AWARD       = 5   //奖品
	EARN_ONLINEAWARD = 6   //上线奖励
	COST_MSG         = 101 //发消息消费
	COST_VIP         = 102 //VIP消费
	COST_GAME_PLANE  = 103 //游戏
	COST_PURSUE      = 104 //女神
	COST_ITEM        = 105 //购买道具
	COST_BUYGAME     = 106 //购买游戏次数
)

//消息子类型
const (
	// 游戏入场费
	PLANE_ENTRANCE_FEE = 1
	// 游戏道具使用费
	PLANE_TOOLS_FEE = 2
	// 解除组队，返币
	PLANE_BREAK_TEAM = 3
)

//消息子类型 ITEM
const (
	ITEM_RECOMMEND = 1 // 推荐展示
)
