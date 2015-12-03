/*
主要包含:
	1.标记相关接口。
	2.认识一下的相关接口。
	3.最近聊天列表相关接口。
*/
package relation

import (
	"fmt"
	"time"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/dynamics"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/relation"
	"yuanfen/yf_service/cls/data_model/relation/base"
	"yuanfen/yf_service/cls/unread"
)

type RelationModule struct {
	log   *log.MLogger
	mdb   *mysql.MysqlDB
	rdb   *redis.RedisPool
	cache *redis.RedisPool
}

func (sm *RelationModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	return
}

/*
SecRecentUserList：获取最近聊过天的用户列表

URI: s/relation/RecentUserList

参数:
		{
			"cur":1,	//页码
			"ps":10,	//每页条数
		}
返回值:

	{
		"res": {
			"users": {
				"list": [
					{
						"uid": 5004006,
						"nickname": "平凡的蔷微花",
						"gender": 1,
						"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201507/01/17163176935.jpg",
						"tag":1 //标签，0-表示未标记，1-表示有好感，2-表示特别关注
						"last_msg":{
							//最后发送或接收的消息，参考消息的结构定义。
						}
						"tm": "2015-07-01T18:22:31+08:00"	//最后一条消息的发送时间
					}
				],
				"pages": {
					"cur": 1,
					"total": 3,
					"ps": 2,
					"pn": 2
				}
			},
			"last_msgid":123,	//最大消息ID
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecRecentUserList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	if err := req.Parse("cur", &cur, "ps", &ps); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	res := make(map[string]interface{})
	list, total, err := relation.GetRecentChatUserList(req.Uid, cur, ps, res)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	pages := utils.PageInfo(total, cur, ps)
	users := make(map[string]interface{})
	users["list"] = list
	users["pages"] = pages
	res["users"] = users
	result["res"] = res
	return
}

/*
SecDelRecentChatUser把用户从最近聊天聊表中删除

URI: s/relation/DelRecentChatUser

参数:
		{
			"uid":123,	//要删除的目标用户
		}
返回值:

	{
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecDelRecentChatUser(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	if err := req.Parse("uid", &uid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	return relation.DelRecentChatUser(req.Uid, uid)
}

/*
SecSayHello发送认识一下请求

URI: s/relation/SayHello

参数:
		{
			"uid":123,	//目标用户
			"content":"你好，我想认识你一下！"	//打招呼要说的话
		}
返回值:

	{
		"res":{
			"msgid":123123,	//发送给对方的消息ID
			"online":true	//对方是否在线
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecSayHello(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var content string
	if err := req.Parse("uid", &uid, "content", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	msgid, online, err := base.SayHello(req.Uid, uid, content)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	res["msgid"] = msgid
	res["online"] = online
	result["res"] = res
	return
}

/*
SecGetSayHelloStatus查看认识一下消息的状态

URI: s/relation/GetSayHelloStatus

参数:
		{

			"uid":123,	//目标用户
			"target":"me", //认识对象，me-想认识我的用户，him-我想认识的用户
		}
返回值:

	{
		"res":{
			"status":1,	//认识一下的消息状态，1-未读，2-已读，3-已回复，0-没有认识一下的消息
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecGetSayHelloStatus(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var target string
	if err := req.Parse("uid", &uid, "target", &target); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	status, err := relation.GetSayHelloStatus(target, req.Uid, uid)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	res["status"] = status
	result["res"] = res
	return
}

/*
SecReadSayHello把认识消息发送者标记为已读

URI: s/relation/ReadSayHello

参数:
		{
			"uid":123,	//认识消息发送者
		}
返回值:

	{
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecReadSayHello(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	if err := req.Parse("uid", &uid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}

	res := make(map[string]interface{})
	err := relation.ReadSayHello(req.Uid, uid)
	if err != nil {
		return err
	}
	result["res"] = res
	return
}

/*
SecDelSayHelloUser删除我想认识或想认识我的用户

URI: s/relation/DelSayHelloUser

参数:
		{
			"target":"me", //认识对象，me-想认识我的用户，him-我想认识的用户
			"uid":123,	//目标用户
		}
返回值:

	{
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecDelSayHelloUser(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var target string
	if err := req.Parse("uid", &uid, "target", &target); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}

	return relation.DelSayHelloUser(target, req.Uid, uid)
}

/*
SecSayHelloUsers获取认识一下用户列表

URI: s/relation/SayHelloUsers

参数:
		{
			"target":"me", //认识对象，me-想认识我的用户，him-我想认识的用户
			"cur":1,	//页码
			"ps":10,	//每页条数
		}
返回值:

	{
		"res": {
			"users": {
				"list": [
				{
					"id":1,	//消息ID
					"uid": 5004006,
					"nickname": "平凡的蔷微花",
					"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201507/01/17163176935.jpg",
					"content":"你好，能认识一下么？",	//最后一条消息的内容
					"status":1,	//最后一条消息的状态，1-未读，2-已读，3-已回复
					"tm": "2015-07-01T18:22:31+08:00"	//最后一条消息的发送时间
				}
				],
				"pages": {
					"cur": 1,
					"total": 3,
					"ps": 2,
					"pn": 2
				}
			}
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecSayHelloUsers(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	var target string
	if err := req.ParseOpt("cur", &cur, 1, "ps", &ps, 10, "target", &target, common.SAYHELLO_TARGET_ME); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, total, err := relation.SayHelloUsers(target, req.Uid, cur, ps)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	pages := utils.PageInfo(total, cur, ps)
	res := make(map[string]interface{})
	users := make(map[string]interface{})
	users["list"] = list
	users["pages"] = pages
	res["users"] = users
	result["res"] = res
	return
}

/*
SecSayHelloList获取某用户的认识消息列表，同时也会把消息变为已读。

URI: s/relation/SayHelloList

参数:
		{
			"target":"me", //认识对象，me-想认识我的人，him-我想认识的人
			"uid":123,	//目标用户
			"cur":1,	//页码
			"ps":10,	//每页条数
			"detail":false	//是否获取用户详情
		}
返回值:

	{
		"res": {
			"messages": {
				"list": [
				{
					"id":1,	//消息ID
					"content":"你好，能认识一下么？",	//消息的内容
					"status":1,	//消息的状态，1-未读，2-已读，3-已回复
					"tm": "2015-07-01T18:22:31+08:00"	//发送时间
				}
				],
				"pages": {
					"cur": 1,
					"total": 3,
					"ps": 2,
					"pn": 2
				}
			}
			"uinfo": {	//detail=true时显示这个节点
				"uid":5004006,
				"nickname":"凡的蔷微花",
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201507/01/17163176935.jpg",
				"age":32,
				"height":178,
				"workunit":"炬鑫网络",
				"province":"北京",
				"city":"海淀",
				"job":"设计师",
				"connection":"你们都是设计师",
				"dynum":34,	//动态总数
				"dynamics":[
					"http://image1.yuanfenba.net/uploads/oss/photo/201507/01/17163176935.jpg",
					"http://image1.yuanfenba.net/uploads/oss/photo/201507/01/17163176935.jpg",
					"http://image1.yuanfenba.net/uploads/oss/photo/201507/01/17163176935.jpg"
				],
				"distence":1344.11	//距离（米）
			}
			"connection":[		//detail=false时显示这个节点
				"你们都在翠微百货工作",
				"她温柔善良"
			],
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecSayHelloList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	var target string
	var uid uint32
	var detail bool
	if err := req.Parse("uid", &uid, "cur", &cur, "ps", &ps, "target", &target); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("detail", &detail, false); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, err := relation.SayHelloList(target, req.Uid, uid, cur, ps)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	if detail {
		uinfo, e := relation.GetSayHelloUserInfo(req.Uid, uid)
		if e != nil {
			return e
		}
		res["uinfo"] = uinfo
	} else {
		res["connection"], e = relation.GetConnection(req.Uid, uid)
		if e != nil {
			return e
		}
	}
	pages := utils.PageInfo(-1, cur, ps)
	messages := make(map[string]interface{})
	messages["list"] = list
	messages["pages"] = pages
	res["messages"] = messages
	result["res"] = res
	return
}

/*
SecFollow标记用户，tag自动设置为1（感兴趣）。

URI: s/relation/Follow

参数:
		{
			"uid":123,	//目标用户
		}
返回值:

	{
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecFollow(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	if err := req.Parse("uid", &uid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	err := relation.Follow(req.Uid, uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("process follow request error : uid [%v] : %v", uid, err.Error()))
	}
	res := make(map[string]interface{})
	result["res"] = res

	// 添加新标记用户时，获取下标记动态角标
	result[common.UNREAD_KEY] = dynamics.GetUnReadMarkDynamic(req.Uid)
	return
}

/*
SecUpdateFollowTag修改标记的标签

URI: s/relation/UpdateFollowTag

参数:
		{
			"uid":123,	//用户ID
			"tag":1 //标签，0-表示未标记，1-表示有好感，2-表示特别关注，3-表示不喜欢(拉黑)
		}
返回值:

	{
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecUpdateFollowTag(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var tag uint16
	if err := req.Parse("uid", &uid, "tag", &tag); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	err := relation.UpdateFollowTag(req.Uid, uid, tag)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("update follow request error : uid [%v] : %v", uid, err.Error()))
	}
	res := make(map[string]interface{})
	result["res"] = res

	// 添加新标记用户时，获取下标记动态角标
	if tag == 1 || tag == 2 {
		result[common.UNREAD_KEY] = dynamics.GetUnReadMarkDynamic(uid)
	} else if tag == 0 {
		un := map[string]interface{}{common.UNREAD_DYNAMIC_MARK: nil}
		unread.GetUnreadNum(req.Uid, un)
		result[common.UNREAD_KEY] = un
	}
	return
}

/*
SecFollowing查看标记的用户，好友排在前面

URI: s/relation/Following

参数:
		{
			"cur":1,	//页码
			"ps":10,	//每页条数
		}
返回值:

	{
		"res": {
			"users": {
				"list": [
				{
					"uid": 5009103,
					"nickname": "微笑是每个人旳坚强",
					"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201507/03/12091196457.jpg",
					"tag":1 //标签，1-表示有好感，2-表示特别关注
					"is_friend":true,	//是否是好友（即是否聊过天）
					"tm": "2015-07-04T10:59:03+08:00"
				}
				],
				"pages": {
					"cur": 1,
					"total": 2,
					"ps": 10,
					"pn": 1
				}
			}
		},
		"status": "ok",
		"tm": 1442491885,
	}

*/
func (sm *RelationModule) SecFollowing(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	if err := req.Parse("cur", &cur, "ps", &ps); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, total, err := relation.GetFollowUsers(req.Uid, cur, ps)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	unread.UpdateUnread(req.Uid, common.UNREAD_FANS, result)
	pages := utils.PageInfo(total, cur, ps)
	users := make(map[string]interface{})
	users["list"] = list
	users["pages"] = pages
	res["users"] = users
	result["res"] = res
	return
}

/*
SecFriends获取认识的人的列表

URI: s/relation/Friends

参数:
		{
			"cur":1,	//页码
			"ps":10,	//每页条数
		}
返回值:

	{
		"res": {
			"users": {
				"list": [
				{
					"uid": 5009103,
					"nickname": "微笑是每个人旳坚强",
					"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201507/03/12091196457.jpg",
					"height": 0,
					"tag":1 //标签，1-表示有好感，2-表示特别关注，3-表示不喜欢(拉黑)
					"following": false,
					"tm": "2015-07-04T10:59:03+08:00"
				}
				],
				"pages": {
					"cur": 1,
					"total": 2,
					"ps": 10,
					"pn": 1
				}
			}
		},
		"status": "ok",
		"tm": 1442491885,
	}

*/
func (sm *RelationModule) SecFriends(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	if err := req.Parse("cur", &cur, "ps", &ps); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, total, err := relation.Friends(req.Uid, cur, ps)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	pages := utils.PageInfo(total, cur, ps)
	res := make(map[string]interface{})
	users := make(map[string]interface{})
	users["list"] = list
	users["pages"] = pages
	res["users"] = users
	result["res"] = res
	return
}

/*
SecBlacklist查看不喜欢（黑名单中）的用户

URI: s/relation/Blacklist

参数:
		{
			"cur":1,	//页码
			"ps":10,	//每页条数
		}
返回值:

	{
		"res": {
			"users": {
				"list": [
				{
					"uid": 5009103,
					"nickname": "微笑是每个人旳坚强",
					"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201507/03/12091196457.jpg",
					"tm": "2015-07-04T10:59:03+08:00"
				}
				],
				"pages": {
					"cur": 1,
					"total": 2,
					"ps": 10,
					"pn": 1
				}
			}
		},
		"status": "ok",
		"tm": 1442491885,
	}

*/
func (sm *RelationModule) SecBlacklist(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	if err := req.Parse("cur", &cur, "ps", &ps); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, total, err := relation.GetBlacklist(req.Uid, cur, ps)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	pages := utils.PageInfo(total, cur, ps)
	users := make(map[string]interface{})
	users["list"] = list
	users["pages"] = pages
	res["users"] = users
	result["res"] = res
	return
}

/*
SecWantDate想要和对方约会

URI: s/relation/WantDate

参数:
		{
			"uid":123,	//约会对象ID
		}
返回值:

	{
		"res":{
			"available":true,	//是否可以发起约会
			"tip_msg": {
				"but": {
					"tip": "查看帮助",
					"cmd": "http://www.baidu.com",
					"def": false,
					"data": {}
				},
				"content": "如果\"时间现实表现\"也有意向见面，系统则会通知女方来选择约会地点和时间",
				"type": "hint"
			}
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecWantDate(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	if err := req.Parse("uid", &uid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	res := make(map[string]interface{})
	available, err := relation.WantDate(req.Uid, uid, res)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("WantDate error : uid [%v] : %v", uid, err.Error()))
	}
	res["available"] = available
	result["res"] = res
	return
}

/*
SecCancelDate取消想要和对方约会

URI: s/relation/CancelDate

参数:
		{
			"uid":123,	//约会对象ID
		}
返回值:

	{
		"res":{
			"tip_msg": {
				"content": "您已取消见面意向",
				"type": "hint"
			}
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecCancelDate(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	if err := req.Parse("uid", &uid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	res := make(map[string]interface{})
	err := relation.CancelDate(req.Uid, uid, res)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("CancelDate error : uid [%v] : %v", uid, err.Error()))
	}
	result["res"] = res
	return
}

/*
SecGetRecommendDatePlaces获取推荐约会地点

URI: s/relation/GetRecommendDatePlaces

参数:
		{
			"uid":123,	//约会对象ID
		}
返回值:

	{
		"res":{
			"place":[//备选的约会地点，都是官方认证的约会地点
			{
				"id":"xxdkj23kdxx",
				"name":"XXX星巴克",
				"address":"北京市海淀区清河二街",
				"pic":"http://image1.yuanfenba.net/uploads/oss/avatar/201511/03/14254456779.jpg",	//店铺图片
				"lat":32.222,
				"lng":42.281
			},
			{
				"id":"xxdkj23kdxx",
				"name":"XXX星巴克",
				"address":"北京市海淀区清河二街",
				"pic":"http://image1.yuanfenba.net/uploads/oss/avatar/201511/03/14254456779.jpg",	//店铺图片
				"lat":32.222,
				"lng":42.281
			}
			]
			"my_workplace":{	//我的工作地点经纬度
				"uid":123,
				"nickname":"justin",
				"avatar":"http://image1.yuanfenba.net/uploads/oss/avatar/201511/03/14254456779.jpg",
				"location":{
					"lat":22.1132,
					"lng":12.13354
				}
			}
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
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecGetRecommendDatePlaces(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	if err := req.Parse("uid", &uid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	places, mWorkPlace, hWorkPlace, err := relation.GetRecommendDatePlaces(req.Uid, uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("GetDatePlaces error : uid [%v] : %v", uid, err.Error()))
	}
	res := make(map[string]interface{})
	res["places"] = places
	res["my_workplace"] = mWorkPlace
	res["him_workplace"] = hWorkPlace
	result["res"] = res
	return
}

/*
SecGetDatePlaces获取约会地点

URI: s/relation/GetDatePlaces

参数:
		{
			"lat":32.2112,	//搜索范围中心的纬度
			"lng":42.12123	//搜索范围中心的经度
		}
返回值:

	{
		"res":{
			"place":[//备选的约会地点，都是官方认证的约会地点
			{
				"id":"xxdkj23kdxx",
				"name":"XXX星巴克",
				"address":"北京市海淀区清河二街",
				"pic":"http://image1.yuanfenba.net/uploads/oss/avatar/201511/03/14254456779.jpg",	//店铺图片
				"lat":32.222,
				"lng":42.281
			},
			{
				"id":"xxdkj23kdxx",
				"name":"XXX星巴克",
				"address":"北京市海淀区清河二街",
				"lat":32.222,
				"lng":42.281
			}
			]
		}
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecGetDatePlaces(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var lat, lng float64
	if err := req.Parse("lat", &lat, "lng", &lng); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	places, err := relation.GetDatePlaces(lat, lng)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	res["places"] = places
	result["res"] = res
	return
}

/*
SecMakeDate发起约会，必须双方都想要约会才能成功发起约会。

URI: s/relation/MakeDate

参数:
		{
			"uid":123,	//约会对象ID
			"firstTime":1,	//是否是女方第一次发起约会。1-是，0-不是
			"tm":123245, //约会时间（秒数）
			"place":"ckfkeu32kj3ka0" //约会地点
		}
返回值:

	{
		"res":{
			"msg": {
				"msgid": 106438,
				"date_time": "2015-11-12T13:59:55+08:00",
				"text":"已告知“女神”地点和时间",
				"place": {
					"id": "e308284843d29fa6b24c49f7",
					"name": "COSTA COFFEE(阜成门店)",
					"address": "北京市西城区阜成门外大街1号华联商厦一层F1-01",
					"pic": "",
					"lat": 39.929748,
					"lng": 116.360441,
					"distence": 0
				},
				"sender": {
					"avatar": "http://image2.yuanfenba.net/uploads/oss/photo/201506/23/14202423086.jpg",
					"nickname": "Cinderella",
					"uid": 5000339
				},
				"him_workplace": {
					"uid": 5000339,
					"nickname": "Cinderella",
					"gender":0,
					"avatar": "http://image2.yuanfenba.net/uploads/oss/photo/201506/23/14202423086.jpg",
					"location": {
						"lat": 40.03629,
						"lng": 116.352824
					}
				},
				"my_workplace": {
					"uid": 5001690,
					"nickname": "男神啊五",
					"gender":1,
					"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201510/31/13512782860.jpg",
					"location": {
						"lat": 40.073396,
						"lng": 116.35527
					}
				},
				"tm": "2015-11-07T14:11:04+08:00",
				"type": "date_request"
			}
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecMakeDate(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var tm int64
	var place string
	var firstTime int
	if err := req.Parse("uid", &uid, "tm", &tm, "place", &place); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("first_time", &firstTime, 0); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	fmt.Println("tm:", tm, time.Unix(tm, 0))
	msg, err := relation.MakeDate(req.Uid, uid, time.Unix(tm, 0), place, firstTime)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("MakeDate error : uid [%v] : %v", uid, err.Error()))
	}
	res := make(map[string]interface{})
	res["msg"] = msg
	result["res"] = res
	return
}

/*
SecGetDateStatus查看约会状态

URI: s/relation/GetDateStatus

参数:
		{
			"uid":123,	//约会对象ID
		}
返回值:

	{
		"res":{
			"status":1	//约会状态，0-未发起约会请求，1-已发起约会请求，2-双方都发起了约会请求
		},
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecGetDateStatus(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	if err := req.Parse("uid", &uid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	status, err := relation.GetDateStatus(req.Uid, uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("MakeDate error : uid [%v] : %v", uid, err.Error()))
	}
	res := make(map[string]interface{})
	res["status"] = status
	result["res"] = res
	return
}

/*
UpdateDatePlace更新约会地点

URI: s/relation/UpdateDatePlace

参数:
		{
			"provinces":["北京市","湖南省"],	//搜索的省市范围
			"keywords":["星巴克"]//实体关键字
		}
返回值:

	{
		"total":123,	//更新的数量
		"status": "ok",
		"tm": 1442486383
	}

*/
func (sm *RelationModule) SecUpdateDatePlace(req *service.HttpRequest, res map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return service.NewError(service.ERR_PERMISSION_DENIED, "permission denied")
	}
	var provinces, keywords []string
	if err := req.Parse("provinces", &provinces, "keywords", &keywords); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	go relation.UpdateDatePlace(provinces, keywords...)
	if e != nil {
		return e
	}
	return
}
