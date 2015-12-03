package dynamics

import (
	"crypto/md5"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"yf_pkg/format"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/comments"
	"yuanfen/yf_service/cls/data_model/dynamics"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/unread"
	"yuanfen/yf_service/cls/word_filter"
)

// 动态相关
type DynamicsModule struct {
	log   *log.MLogger
	mdb   *mysql.MysqlDB
	rdb   *redis.RedisPool
	cache *redis.RedisPool
	mode  string
}

func (dm *DynamicsModule) Init(env *service.Env) (err error) {
	dm.log = env.Log
	dm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	dm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	dm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	dm.mode = env.ModuleEnv.(*cls.CustomEnv).Mode
	return
}

/*
获取历史动态列表

URL: s/dynamics/List

参数：
	ps: 请求数量
	id: [uint32] 表示从该时间id后取ps条数据(最后一条动态id),客户端解析数据时，请务必注意
	gender: [int]查询 0 查询全部 1.查询男性用户动态 2.查询女性用户动态
	ex_ids:[可选参数] [string]已推荐动态id,已英文逗号隔开

返回值：

	{
		"res":{
		 "dy_list":[
		   {
			"dynamic": { // 字段含义详见 http://120.131.64.91:8182/pkg/yuanfen/yf_service/cls/data_model/dynamics/#Dynamic
				"comments": 10,
				"gameinit": "",
				"gamekey": 0,
				"id": 100011, // int64
				"likes": 12,
				"location": "北京市 海淀区翠微百货",
				"pic": 	[],   // 图片url数组
				"stype": 1,
				"text": "test",
				"tm": "2015-09-16T14:12:11+08:00",
				"type": 1,
				"uid": 5000761,
				"url": "",
				"isLike":1         // 是否点赞
				"is_join":false    //  是否已经参与游戏，当type=2时，该值有意义
			},
			"user": {  // 动态用户信息
				"uid": 5000761,
				"age": 25,
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
				"job": "UI设计师",
				"nickname": "小气&豪猪",
				"isOnline": true,  // 是否在线 true 在线 false 不在线
				"distance": "2000.0"  // 米
				"city": "长沙"  // 用户所在省
			}
			}
	 	  }
		],
		"ex_ids": [347,329,327]  // 推荐动态
		"gender": 0   // 查询性别
     	}
		"status": "ok",
		"tm": 1442398732
	}

*/
func (dm *DynamicsModule) SecList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	var ps, gender int
	var ex_ids string
	if e = req.Parse("id", &id, "ps", &ps, "gender", &gender); e != nil {
		return
	}
	if e = req.ParseOpt("ex_ids", &ex_ids, ""); e != nil {
		return
	}
	if gender < 0 { // 默认全部
		gender = 0
	}
	u, e := user_overview.GetUserObject(req.Uid)
	if e != nil {
		return
	}
	list, e := dynamics.GetDynamicList(dynamics.MakeProvinceDyanmicKey(u.Province), id, gender, ps, utils.StringToUint32Arr(ex_ids, ","), req.Uid)
	if e != nil {
		return
	}
	res, e := dynamics.GenDynamicsRes(list, req.Uid, 14)
	result["res"] = map[string]interface{}{"dy_list": res, "gender": gender}
	return
}

/*
获取最新动态消息

URL: s/dynamics/New

参数：
	ps: 请求数量
	gender: [int]查询 0 查询全部 1.查询男性用户动态 2.查询女性用户动态
	ex_ids:[可选参数] [string]已推荐动态id,已英文逗号隔开

返回值:

	{
		"res":{
		 "dy_list":	[  // 字段注释同s/dynamics/List 接口，http://120.131.64.91:8182/pkg/yuanfen/yf_service/modules/dynamics/#DynamicsModule.SecList
			{
			"dynamic": {
				"comments": 10,
				"gameinit": "",
				"gamekey": 0,
				"id": 100021,
				"likes": 12,
				"location": "北京市 海淀区翠微百货",
				"pic": [],
				"stype": 1,
				"text": "test",
				"tm": "2015-09-16T14:12:11+08:00",
				"type": 1,
				"uid": 5000761,
				"url": ""
			},
			"user": {
				"age": 25,
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
				"job": "UI设计师",
				"nickname": "小气&豪猪",
				"isOnline": true,  // 是否在线 true 在线 false 不在线,
				"distance": "2000.00",  // 米
				"city": "长沙"  // 用户所在省
			}
		},
		],
		"ex_ids": [347,329,327]  // 推荐动态
		"gender": 0   // 查询性别
	}
		"status": "ok",
		"tm": 1442409500
	}

*/
func (dm *DynamicsModule) SecNew(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var gender, ps int
	var ex_ids string
	if e = req.Parse("gender", &gender, "ps", &ps); e != nil {
		return
	}
	if e = req.ParseOpt("ex_ids", &ex_ids, ""); e != nil {
		return
	}
	if gender < 0 { // 默认查询全部
		gender = 0
	}
	u, e := user_overview.GetUserObject(req.Uid)
	if e != nil {
		return
	}
	list := make([]dynamics.Dynamic, 0, ps+3)
	newex_ids := make([]uint32, 0, 3)
	ex_dynamics, newex_ids, e := dynamics.GetExDynamics(dynamics.MakeExProvinceDyanmicKey(u.Province), gender, 2)
	if e != nil {
		return e
	}
	ex_ids_arr := utils.StringToUint32Arr(ex_ids, ",")
	if len(ex_dynamics) > 0 {
		ex_ids_arr = append(ex_ids_arr, newex_ids...)
	}
	dm.log.AppendObj(nil, "_______get GetExDynamics_____", newex_ids)

	dylist, e := dynamics.GetDynamicList(dynamics.MakeProvinceDyanmicKey(u.Province), 0, gender, ps, ex_ids_arr, req.Uid)
	if e != nil {
		return
	}
	if len(ex_dynamics) > 0 {
		list = append(list, ex_dynamics...)
		list = append(list, dylist...)
		/*	if len(dylist) > 5 {
				for k, v := range dylist {
					if k == 2 {
						list = append(list, ex_dynamics...)
					}
					list = append(list, v)
				}
			} else {
				list = append(list, dylist...)
				list = append(list, ex_dynamics...)
			}
		*/
	} else {
		list = dylist
	}
	// 获取最新文章
	dy, e := dynamics.GetArticle()
	if e != nil {
		return
	}
	dm.log.AppendObj(nil, "---GetArticle----", dy)
	// 获取最新文章
	dys := make([]dynamics.Dynamic, 0, len(list)+1)
	if dy.Id > 0 {
		if len(list) > 3 {
			for k, v := range list {
				if k == 2 {
					dys = append(dys, dy)
				}
				dys = append(dys, v)
			}
		} else {
			dys = append(dys, list...)
			dys = append(dys, dy)
		}
	} else {
		dys = list
	}
	rlist, e := dynamics.GenDynamicsRes(dys, req.Uid, 14)
	res := make(map[string]interface{})
	res["dy_list"] = rlist
	res["ex_ids"] = newex_ids
	res["gender"] = gender
	result["res"] = res
	return
}

/*
动态详情(需要判断是否已经玩过游戏，如果已经玩过了该游戏，则直接返回动态评论和点赞详情)

URL: s/dynamics/Detail

参数：
	id:动态id
返回值:
	{
	"res": {
		"comments_list": [  // 品论列表
		{
			"comment": {
				"content": "我的第一条评论",
				"id": 1,
				"ruid": 0,
				"tm": "2015-09-23T11:55:53+08:00",
				"uid": 5000761
			},
			"ruser": {
				"age": 0,
				"avatar": "",
				"job": "UI设计师",
				"nickname": ""
			},
			"user": {
				"age": 25,
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
				"job": "UI设计师",
				"nickname": "小气\u0026豪猪"
			}
		}
		],
		"dynamic": {     // 动态详情
			"dynamic": {
				"comments": 0,
				"gameinit": "",
				"gamekey": 0,
				"id": 25,
				"isLike": 1,
				"likes": 3,
				"location": "北京市海淀区清河中街",
				"pic": [],
				"sign": 1,
				"stype": 0,
				"text": "hahhahh,都去爱你了",
				"tm": "2015-09-22T16:27:47+08:00",
				"type": 1,
				"uid": 5000761,
				"url": ""
			},
			"user": {
				"age": 25,
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
				"job": "UI设计师",
				"nickname": "小气\u0026豪猪"
			}
		},
		"like_users": [  // 点赞用户列表
		{
			"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
			"nickname": "小气\u0026豪猪",
			"uid": 5000761
		}
		],
		},
	"status": "ok",
	"tm": 1443015904
}
*/
func (dm *DynamicsModule) SecDetail(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	if e = req.Parse("id", &id); e != nil {
		return
	}
	dy, e := dynamics.GetDynamicById(id)
	if e != nil {
		return
	}
	if er := dynamics.CheckDynamicValid(dy); er.Code != service.ERR_NOERR {
		return er
	}

	list := []dynamics.Dynamic{dy}
	res, e := dynamics.GenDynamicsRes(list, req.Uid, 2)
	if e != nil {
		return
	}
	if len(res) <= 0 {
		return service.NewError(service.ERR_INTERNAL, "动态不存在", "动态不存在")
	}

	var is_all bool
	if dy.Type == common.DYNAMIC_TYPE_ARTICLE || dy.Uid == req.Uid {
		is_all = true
	}
	var wc comments.Comment
	// 如果为拼图游戏，需要查询第一名用户
	if dy.Type == common.DYNAMIC_TYPE_GAME && is_all {
		wc, e = comments.GetPuzzleWinComment(dy.Id)
		if e != nil {
			return e
		}
	}

	stm := utils.Now.Format(format.TIME_LAYOUT_1)
	carrs, e := comments.GetCommentByIdAndUid(id, dy.Uid, req.Uid, 1, stm, math.MaxUint32, 40, is_all, wc.Id)
	if e != nil {
		return
	}
	carr := make([]comments.Comment, 0, len(carrs)+1)
	if wc.Id > 0 {
		carr = append(carr, wc)
	}
	carr = append(carr, carrs...)
	com_info, e := comments.GenCommentInfo(carr, req.Uid)
	if e != nil {
		return
	}

	dy_info := make(map[string]interface{})
	like_users := make([]map[string]interface{}, 0, 50)
	// 判断是否需要查询点赞用户
	if is_all {
		like_uids, e := comments.GetLikeUsers(id)
		if e != nil {
			return e
		}
		like_users, e = user_overview.GenUserInfo(like_uids, 0)
		if e != nil {
			return e
		}
	}

	dy_info["like_users"] = like_users
	dy_info["dynamic"] = res[0]
	dy_info["comments_list"] = com_info
	result["res"] = dy_info
	return
}

/*
获取动态评论列表

URL: s/dynamics/CommentList

参数：
	id:动态id
	start: 最后一次请求评论id，如为第一次请求，该值传0
	ps: 请求条数
返回值:
	{
		"res": [
		{
			"comment": {  //评论信息,详见: http://120.131.64.91:8182/pkg/yuanfen/yf_service/cls/data_model/comments/#Comment
				"content": "我的第一条评论",
				"id": 1,
				"ruid": 0,
				"tm": "2015-09-23T11:55:53+08:00",
				"uid": 5000761
				"source_id": 5000761
				"type":1
			},
			"ruser": { // 被回复用户信息，对应ruid
				"age": 0,
				"avatar": "",
				"job": "UI设计师",
				"nickname": ""
			},
			"user": {  // 评论用户信息，对应uid
				"age": 25,
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
				"job": "UI设计师",
				"nickname": "小气\u0026豪猪"
			}
		}
		],
		"status": "ok",
		"tm": 1442981501
	}
*/
func (dm *DynamicsModule) SecCommentList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id, start uint32
	var ps int
	if e = req.Parse("id", &id, "start", &start, "ps", &ps); e != nil {
		return
	}
	// 需要根据动态的类型，获取评论列表
	dy, e := dynamics.GetDynamicById(id)
	if e != nil {
		return
	}
	if dy.Id <= 0 {
		return service.NewError(service.ERR_INVALID_PARAM, "该动态不存在", "改动不存在")
	}
	var is_all bool
	if dy.Type == common.DYNAMIC_TYPE_ARTICLE || dy.Uid == req.Uid {
		is_all = true
	}

	stm := utils.Now.Format(format.TIME_LAYOUT_1)
	if start > 0 {
		c, er := comments.GetCommentById(dm.mdb, start)
		if er != nil {
			return er
		}
		stm = c.Tm
	} else {
		start = math.MaxUint32
	}
	var wc comments.Comment
	// 如果为拼图游戏，需要查询第一名用户
	if dy.Type == common.DYNAMIC_TYPE_GAME && is_all {
		if wc, e = comments.GetPuzzleWinComment(dy.Id); e != nil {
			return e
		}
	}
	//source_id, source_uid, uid uint32, source_type, stm string, id uint32, ps int, is_all bool
	carrs, e := comments.GetCommentByIdAndUid(id, dy.Uid, req.Uid, 1, stm, start, ps, is_all, wc.Id)
	if e != nil {
		return
	}
	carr := make([]comments.Comment, 0, len(carrs)+1)
	if wc.Id > 0 {
		carr = append(carr, wc)
	}
	carr = append(carr, carrs...)
	res, e := comments.GenCommentInfo(carr, req.Uid)
	if e != nil {
		return
	}
	result["res"] = res
	return
}

/*
动态删除

URL：s/dynamics/Delete
参数：
	id:[uint32] 动态id
返回值：
	{
		"status": "ok",
		"tm": 1442472031
	}
*/
func (dm *DynamicsModule) SecDelete(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	if e = req.Parse("id", &id); e != nil {
		return
	}
	dy, e := dynamics.GetDynamicById(id)
	if e != nil {
		return
	}
	if dy.Id <= 0 {
		return service.NewError(service.ERR_MYSQL, "改动态不存在", "改动态不存在")
	}
	if dy.Uid != req.Uid {
		return service.NewError(service.ERR_INVALID_USER, "无权限删除", "无权限删除")
	}

	u, e := user_overview.GetUserObject(req.Uid)
	if e != nil {
		return
	}
	tx, e := dm.mdb.Begin()
	if e != nil {
		return
	}
	if e = dynamics.UpdateDynamicStatus(tx, id, dynamics.DYNAMIC_STATUS_DELETE); e != nil {
		tx.Rollback()
		dm.log.AppendObj(e, "delete dynamic is error update status", req.Uid, id)
		return
	}
	if _, e = dm.rdb.ZRem(redis_db.REDIS_DYNAMIC, dynamics.MakeProvinceDyanmicKey(u.Province), dynamics.MakeDynamicKey(id, req.Uid)); e != nil {
		tx.Rollback()
		dm.log.AppendObj(e, "delete dynamic is error,zrem1 ", req.Uid, id)
		return
	}
	if _, e = dm.rdb.ZRem(redis_db.REDIS_DYNAMIC, dynamics.GetUserDynamicKey(req.Uid), dynamics.MakeDynamicKey(id, req.Uid)); e != nil {
		tx.Rollback()
		dm.log.AppendObj(e, "delete dynamic is error,zrem2", req.Uid, id)
		return
	}
	tx.Commit()
	return
}

/*
动态添加

URL：s/dynamics/Add

参数：
	type : [int]动态类型,1 普通动态 2 拼图游戏
	pic: [string]动态包含图片，多张已英文逗号分隔
	text: [string]动态文本内容
	location:[string]动态发布位置
	gameinit:[string]拼图游戏初始化序列，以逗号分割的数字
	注意：gameinit参数当type=2 时必须有值，其他情况可传对应类型默认值
返回值：
	{
		"res": {
			"code": 0
			"dynamic": {
				"comments": 0,
				"gameinit": "",
				"gamekey": 0,
				"id": 159,
				"isLike": 0,
				"is_join": false,
				"likes": 0,
				"location": "北京市海淀区清河中街",
				"pic": [
				"http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg",
				"http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
				"http://image1.yuanfenba.net/uploads/oss/avatar_big/201509/14/13272828068.jpg"
				],
				"sign": 0,
				"stype": 0,
				"text": "我的心头想爱你个",
				"tm": "2015-10-23T15:18:12+08:00",
				"type": 1,
				"uid": 5000761,
				"url": ""
			},
			"user": {
				"age": 25,
				"avatar": "http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg",
				"distance": 0,
				"isOnline": false,
				"job": "演艺人员",
				"nickname": "小气\u0026豪猪",
				"uid": 5000761
			}
		},
		"status": "ok",
		"tm": 1445584692
	}
	// 有图片审核未通过,返回未审核通过的图片列表，当code==2013 表示有图片未审核通过
	{
		"res": {
			"invalid_pic": {
				"code": 2013,
				"error_pic": [
					"http://image1.yuanfenba.net/uploads/oss/chat/201508/31/00470858095.jpg"
				],
				"error_reason": "图片涉黄"
			}
		},
		"status": "ok",
		"tm": 1446603996
	}

*/
func (dm *DynamicsModule) SecAdd(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var t, gamekey int
	var pic, text, location, gameinit string
	needCheck := true
	if e = req.Parse("type", &t, "pic", &pic, "text", &text, "location", &location); e != nil {
		return
	}
	pic = strings.TrimSpace(pic)
	filter := 0
	// 文字黄发,过滤
	if strings.TrimSpace(text) != "" {
		filter, text = word_filter.Replace(text)
	}
	if t == common.DYNAMIC_TYPE_GAME {
		if e = req.Parse("gameinit", &gameinit); e != nil {
			return
		}
	}
	// 文字黄发,过滤
	if text == "" && pic == "" {
		return service.NewError(service.ERR_INVALID_PARAM, "text and pic is both empty", common.MAG_INVALID_PARAM)
	}
	// 如果为拼图游戏，需要做任务和色情图片检测，并且需要做同步检测
	if t == common.DYNAMIC_TYPE_GAME {
		if pic == "" {
			return service.NewError(service.ERR_INVALID_PARAM, "game must have pic param", common.MAG_INVALID_PARAM)
		}
		// 如果为拼图游戏图片检测由异步转化同步
		needCheck = false
		ir, e := general.CheckImgByUrl(general.IMGCHECK_SEXY_AND_HUMAN, pic)
		if e != nil {
			return e
		}
		dm.log.AppendObj(e, "dynamic check ", req.Uid, pic, ir)
		if ir.Status != 0 {
			return service.NewError(service.ERR_INVALID_PARAM, "game dynamic pic is invalid", "拼图需要为人物图片")
		}
		gamekey = rand.Intn(950)
	}
	if t == common.DYNAMIC_TYPE_GAME && text == "" {
		text = "拼图游戏："
	}
	// 调用新增动态接口
	var dy dynamics.Dynamic
	dy.Uid = req.Uid
	dy.Type = t
	dy.Text = text
	dy.Pic = pic
	dy.Location = location
	dy.GamgeKey = gamekey
	dy.GamgeInit = gameinit
	// 动态文字被过滤了
	if filter > 0 {
		dy.Status = dynamics.DYNAMIC_STATUS_TXTINVALID
	}
	id, e := dynamics.AddDynamic(dy)
	if e != nil {
		return
	}
	dy.Id = id
	dy.Tm = utils.Now.Format(format.TIME_LAYOUT_1)
	dres, e := dynamics.GenDynamicsRes([]dynamics.Dynamic{dy}, req.Uid, 12)
	if e != nil {
		return
	}
	if len(dres) <= 0 {
		return
	}
	dres[0]["invalid_pic"] = []string{}
	dres[0]["code"] = service.ERR_NOERR
	result["res"] = dres[0]

	go dynamics.DoCheckPicAndPush(id, needCheck)

	return
}

/*
动态举报

URL：s/dynamics/Report

参数：
	id:[uint32]动态id
	reason_id:[int] 举报原因，1 广告 2 色情
返回值：
	{
		"status": "ok",
		"tm": 1442472031
	}
*/
func (dm *DynamicsModule) SecReport(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	if e = req.Parse("id", &id); e != nil {
		return
	}
	dy, e := dynamics.GetDynamicById(id)
	if e != nil {
		return
	}
	if er := dynamics.CheckDynamicValid(dy); er.Code != service.ERR_NOERR {
		return er
	}
	e = dynamics.UpdateComDynamic(dm.mdb, dy.Id, 0, 0, 1)
	return
}

/*
动态点赞

URL：s/dynamics/Like

参数：
	id:[uint32]动态id
返回值：
	{
		"res": {
			"comments": 9, // 动态最新评论数
			"likes": 0 // 动态最新点赞数
		},
		"status": "ok",
		"tm": 1444895681
	}

*/
func (dm *DynamicsModule) SecLike(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	if e = req.Parse("id", &id); e != nil {
		return
	}
	dy, e := dynamics.GetDynamicByIdNoCache(dm.mdb, id)
	if e != nil {
		return
	}
	if er := dynamics.CheckDynamicValid(dy); er.Code != service.ERR_NOERR {
		return er
	}
	c, e := comments.GetLikeComment(id, req.Uid)
	if e != nil {
		return
	}

	tx, e := dm.mdb.Begin()
	if e != nil {
		return
	}
	var nc comments.Comment
	if c.Id <= 0 {
		newid, e := comments.AddComment(tx, req.Uid, id, 0, common.COMMENT_SOURCE_TYPE_DYNAMIC, common.COMMENT_TYPE_LIKE, "")
		if e != nil {
			return e
		}
		fmt.Println("----add---", newid)
		if nc, e = comments.GetCommentById(tx, newid); e != nil {
			return e
		}

	} else if c.Status != comments.COMENT_STATUS_OK {
		e = comments.UpdateCommentStatus(tx, c.Id, comments.COMENT_STATUS_OK)
	} else {
		tx.Rollback()
		return
	}
	if e != nil {
		tx.Rollback()
		return
	}
	if e = dynamics.UpdateComDynamic(tx, dy.Id, 1, 0, 0); e != nil {
		tx.Rollback()
		return
	}

	tx.Commit()
	// 之前点过赞，重复点赞不推送消息
	if c.Id <= 0 && nc.Id > 0 {
		if isCompelete, e := user_overview.CompleteMust(req.Uid); e == nil && isCompelete {
			go dynamics.PushCommentMsg(nc, dy)
		}
	}
	res := make(map[string]interface{})
	res["comments"] = dy.Comments
	res["likes"] = dy.Likes + 1
	result["res"] = res
	return
}

/*
取消点赞

URL：s/dynamics/UnLike

参数：
	id:[uint32]动态id
返回值：
	{
		"res": {
			"comments": 9, // 动态最新评论数
			"likes": 0 // 动态最新点赞数
		},
		"status": "ok",
		"tm": 1444895681
	}

*/
func (dm *DynamicsModule) SecUnLike(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	if e = req.Parse("id", &id); e != nil {
		return
	}
	dy, e := dynamics.GetDynamicByIdNoCache(dm.mdb, id)
	if e != nil {
		return
	}
	if er := dynamics.CheckDynamicValid(dy); er.Code != service.ERR_NOERR {
		return er
	}
	c, e := comments.GetLikeComment(id, req.Uid)
	if e != nil {
		return
	}
	if c.Id <= 0 || c.Status != comments.COMENT_STATUS_OK {
		return
	}
	tx, e := dm.mdb.Begin()
	if e != nil {
		return
	}
	if e = comments.UpdateCommentStatus(tx, c.Id, comments.COMENT_STATUS_DELETE); e != nil {
		tx.Rollback()
		return
	}
	if e = dynamics.UpdateComDynamic(tx, dy.Id, -1, 0, 0); e != nil {
		tx.Rollback()
		return
	}
	tx.Commit()

	res := make(map[string]interface{})
	res["comments"] = dy.Comments
	likes := dy.Likes - 1
	if likes < 0 {
		likes = 0
	}
	res["likes"] = likes
	result["res"] = res
	return
}

/*
动态评论

URL：s/dynamics/Comment

参数：
	id:[uint32]动态id
	ruid:[uint32]回复用户uid，如果非回复，该参数传0
	content: 评论内容
返回值：
	{
		"res": {
			"comment": {
				"comment": {
					"content": "回复哈哈哈哈",
					"id": 305,
					"ruid": 5000761,
					"source_id": 91,
					"tm": "2015-10-23T15:27:36+08:00",
					"type": 2,
					"uid": 5000762
				},
				"ruser": {
					"age": 25,
					"avatar": "http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg",
					"job": "UI设计师",
					"nickname": "小气\u0026豪猪"
				},
				"user": {
					"age": 25,
					"avatar": "http://image2.yuanfenba.net/uploads/oss/photo/201507/01/17250723085.jpg",
					"job": "工程师",
					"nickname": "宽容的小野鸭",
					"uid": 5000762
				}
			},
			"comments": 21,
			"likes": 1
		},
		"status": "ok",
		"tm": 1445585256
	}
*/
func (dm *DynamicsModule) SecComment(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id, ruid uint32
	var content string
	if e = req.Parse("id", &id, "ruid", &ruid, "content", &content); e != nil {
		return
	}
	// 如果用户资料必填项未完全填写完毕,则不能发送评论
	if addRedis, e := user_overview.CompleteMust(req.Uid); e != nil || !addRedis {
		dm.log.AppendObj(e, "CompleteMust is false no comment", req.Uid, addRedis)
		return service.NewError(service.ERR_INTERNAL, "CompleteMust is false or e", "资料不完整，不能评论")
	}
	// 文字黄发,过滤
	if strings.TrimSpace(content) != "" {
		_, content = word_filter.Replace(content)
	}

	d, e := dynamics.GetDynamicByIdNoCache(dm.mdb, id)
	if e != nil {
		return e
	}
	if er := dynamics.CheckDynamicValid(d); er.Code != service.ERR_NOERR {
		return er
	}
	tx, e := dm.mdb.Begin()
	if e != nil {
		return
	}
	cid, e := comments.AddComment(tx, req.Uid, id, ruid, common.COMMENT_SOURCE_TYPE_DYNAMIC, common.COMMENT_TYPE_COMMENT, content)
	if e != nil || cid <= 0 {
		dm.log.AppendObj(e, "SecComment is error", cid)
		tx.Rollback()
		return service.NewError(service.ERR_MYSQL, "添加评论失败", "添加评论失败")
	}
	if e = dynamics.UpdateComDynamic(tx, id, 0, 1, 0); e != nil {
		tx.Rollback()
		return
	}
	c, e := comments.GetCommentById(tx, cid)
	if e != nil {
		tx.Rollback()
		return
	}
	tx.Commit()

	go dynamics.PushCommentMsg(c, d)

	r_arr, e := comments.GenCommentInfo([]comments.Comment{c}, req.Uid)
	if e != nil {
		return
	}
	if len(r_arr) <= 0 {
		return
	}
	res := make(map[string]interface{})
	res["comment"] = r_arr[0]

	res["comments"] = d.Comments + 1
	res["likes"] = d.Likes
	result["res"] = res
	return
}

/*
获取标记用户动态

URL：s/dynamics/MarkList

参数：
	ps: 请求数量
	id: [int64] 表示从该时间id后取ps条数据(最后一条动态id),客户端解析数据时，请务必注意

返回值：
	{
		"res":{
		"dy_list":[
		   {
			"dynamic": { // 字段含义详见 http://120.131.64.91:8182/pkg/yuanfen/yf_service/cls/data_model/dynamics/#Dynamic
				"comments": 10,
				"gameinit": "",
				"gamekey": 0,
				"id": 100011, // int64
				"likes": 12,
				"location": "北京市 海淀区翠微百货",
				"pic": 	[],   // 图片url数组
				"stype": 1,
				"text": "test",
				"tm": "2015-09-16T14:12:11+08:00",
				"type": 1,
				"uid": 5000761,
				"url": "",
				"is_join":false    //  是否已经参与游戏，当type=2时，该值有意义
			},
			"user": {
				"age": 25,
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
				"job": "UI设计师",
				"nickname": "小气&豪猪"
				"city": "长沙"  // 用户所在省
			}
	 	  }
		]
	}
		"status": "ok",
		"tm": 1442398732
	}
*/
func (dm *DynamicsModule) SecMarkList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	var ps int
	if e = req.Parse("id", &id, "ps", &ps); e != nil {
		return
	}
	// 获取我的标记用户列表
	m, e := general.GetInterestUids(req.Uid)
	if e != nil {
		return
	}
	res := make([]map[string]interface{}, 0, ps)
	if len(m) <= 0 {
		result["res"] = map[string]interface{}{"dy_list": res}
		return
	}
	u, e := user_overview.GetUserObject(req.Uid)
	if e != nil {
		return
	}
	list, e := dynamics.GetMarkDynamicList(dynamics.MakeProvinceDyanmicKey(u.Province), req.Uid, id, ps, m)
	if e != nil {
		return
	}
	res, e = dynamics.GenDynamicsRes(list, req.Uid, 2)

	result["res"] = map[string]interface{}{"dy_list": res}
	return
}

/*
获取标记用户的最新动态消息

URL: s/dynamics/MarkNew

参数：
	ps: 请求数量

返回值:

	{
		"res":{
			"dy_list":[  // 字段注释同s/dynamics/List 接口，http://120.131.64.91:8182/pkg/yuanfen/yf_service/modules/dynamics/#DynamicsModule.SecList
		{
			"dynamic": {
				"comments": 10,
				"gameinit": "",
				"gamekey": 0,
				"id": 100021,
				"likes": 12,
				"location": "北京市 海淀区翠微百货",
				"pic": [],
				"stype": 1,
				"text": "test",
				"tm": "2015-09-16T14:12:11+08:00",
				"type": 1,
				"uid": 5000761,
				"url": ""
			},
			"user": {
				"age": 25,
				"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
				"job": "UI设计师",
				"nickname": "小气&豪猪"
				"city": "长沙"  // 用户所在省
			}
			}
		},
		],
		"flag":"noMark"  // noMark 提示为标记用户  noDyanmic 提示没有动态
	   }
		"status": "ok",
		"tm": 1442409500
	}

*/
func (dm *DynamicsModule) SecMarkNew(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var ps int
	if e = req.Parse("ps", &ps); e != nil {
		return
	}
	// 获取我的标记用户列表
	m, e := general.GetInterestUids(req.Uid)
	if e != nil {
		return
	}
	fmt.Println("标记用户： ", m)
	res := make([]map[string]interface{}, 0, ps)
	if len(m) <= 0 {
		result["res"] = map[string]interface{}{"flag": "noMark", "dy_list": res}
		return
	}
	u, e := user_overview.GetUserObject(req.Uid)
	if e != nil {
		return
	}
	list, e := dynamics.GetMarkDynamicList(dynamics.MakeProvinceDyanmicKey(u.Province), req.Uid, 0, ps, m)
	if e != nil {
		return
	}
	res, e = dynamics.GenDynamicsRes(list, req.Uid, 2)
	ur := map[string]interface{}{common.UNREAD_DYNAMIC_MARK: unread.Item{Num: 0, Show: ""}}
	result[common.UNREAD_KEY] = ur
	result["res"] = map[string]interface{}{"flag": "noDynamic", "dy_list": res}
	// 更新标记动态未读消息
	go unread.UpdateReadTime(req.Uid, common.UNREAD_DYNAMIC_MARK, utils.Now)
	return
}

/*
拼图游戏成功通知

URL：s/dynamics/Result

参数：
	id:[uint32]动态id
	tm:[int]拼图成绩秒值
	gameanswer:游戏结果 (uid+id+tm+结果序列 然后md5) 结果序列：0-9序列加 gamekey二进制位

返回值：
	{
		"res": {
			"ratio": 66, // 击败了66%的人
			"tm": 8      // 当前用户拼图时间
		},
		"status": "ok",
		"tm": 1446199920
	}
*/
func (dm *DynamicsModule) SecResult(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	var tm int
	var gameanswer string
	if e = req.Parse("id", &id, "tm", &tm, "gameanswer", &gameanswer); e != nil {
		return
	}
	if tm <= 0 {
		return service.NewError(service.ERR_INVALID_PARAM, "tm is less than zero", "参数不合法")
	}

	dy, e := dynamics.GetDynamicById(id)
	if e != nil {
		return
	}
	if er := dynamics.CheckDynamicValid(dy); er.Code != service.ERR_NOERR {
		return er
	}
	// 验证是否结果正确   根据key
	var a = []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	for k, v := range a {
		fmt.Println(v, math.Exp2(float64(v)))
		n := int(math.Exp2(float64(v)))
		if dy.GamgeKey&n == n {
			fmt.Println(v, true)
			a[k] = a[k] + 1
		}
	}
	r := utils.ArrTostring(a, "")
	ms := utils.ToString(req.Uid) + utils.ToString(id) + utils.ToString(tm) + r
	md5_str := fmt.Sprintf("%x", md5.Sum([]byte(ms)))
	fmt.Println(ms, md5_str)
	if md5_str != gameanswer {
		return service.NewError(service.ERR_INTERNAL, "结果错误", "结果错误")
	}
	cid, e := dynamics.CheckIsJoinDynamicGame(id, req.Uid)
	if e != nil {
		return
	}
	if cid > 0 { // 已经玩过了游戏
		s := "update comment set  content =?,tm = ?,status=0 where id = ? "
		_, e = dm.mdb.Exec(s, tm, utils.Now, cid)
	} else {
		// 添加一条游戏结果评分
		if cid, e = comments.AddComment(dm.mdb, req.Uid, id, 0, common.COMMENT_SOURCE_TYPE_DYNAMIC, common.COMMENT_TYPE_GAME, utils.ToString(tm)); e != nil || cid <= 0 {
			dm.log.AppendObj(e, "result is error ", cid, req.Uid, id)
			return service.NewError(service.ERR_INTERNAL, "添加记录失败", "添加记录失败")
		}
		if e = dynamics.UpdateComDynamic(dm.mdb, id, 0, 1, 0); e != nil {
			return
		}
	}
	dynamics.ClearDynamicCache(cid)
	s := "select uid,content from comment where source_id = ? and type = 3 and status = 0 and id !=?"
	rows, e := dm.mdb.Query(s, id, cid)
	if e != nil {
		return
	}
	tm_arr := make([]dynamics.DynamicGameTm, 0, 10)
	defer rows.Close()
	for rows.Next() {
		var dgt dynamics.DynamicGameTm
		if e = rows.Scan(&dgt.Uid, &dgt.Tm); e != nil {
			return
		}
		tm_arr = append(tm_arr, dgt)
	}
	total := len(tm_arr)
	ratio := 100
	if total > 0 {
		beat_num := 0
		for _, gt := range tm_arr {
			if tm <= gt.Tm {
				beat_num++
			}
		}
		ratio = 100 * beat_num / total
	}
	res := make(map[string]interface{})
	res["ratio"] = ratio
	res["tm"] = tm
	result["res"] = res
	return
}

/*
删除动态评论

URL：s/dynamics/DeleteComment

参数：
	id:[uint32]评论ID

返回值：
	{
		"res": {
			"comments": 9, // 动态最新评论数
			"likes": 0 // 动态最新点赞数
		},
		"status": "ok",
		"tm": 1444895681
	}
*/
func (dm *DynamicsModule) SecDeleteComment(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	if e = req.Parse("id", &id); e != nil {
		return
	}
	c, e := comments.GetCommentById(dm.mdb, id)
	if e != nil {
		return
	}
	if c.Id <= 0 {
		return service.NewError(service.ERR_INTERNAL, "该评论不存在", "该评论不存")
	}
	if c.Uid != req.Uid {
		return service.NewError(service.ERR_INTERNAL, "无权限删除", "无权限删除")
	}
	if c.Status != comments.COMENT_STATUS_OK {
		return
	}
	tx, e := dm.mdb.Begin()
	if e != nil {
		return
	}
	if e = comments.UpdateCommentStatus(tx, id, comments.COMENT_STATUS_DELETE); e != nil {
		tx.Rollback()
		return
	}
	if e = dynamics.UpdateComDynamic(tx, c.SourceId, 0, -1, 0); e != nil {
		tx.Rollback()
		return
	}
	d, e := dynamics.GetDynamicByIdNoCache(tx, c.SourceId)
	if e != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
	res := make(map[string]interface{})
	res["comments"] = d.Comments
	res["likes"] = d.Likes
	result["res"] = res
	return
}

/*
个人中心，获取我的动态

URL：s/dynamics/OtherDynamic

参数：
	id:起始id
	uid:用户uid，查询用户

返回值：
	{
	 "res":{
		"dynamic_img": "http://image2.yuanfenba.net/uploads/oss/photo/201506/01/11020685200.jpg",  // 第一页返回
		"hasNext": true
		"dynamics":[
			{
			"date": "2015-09-24",  // 日期
			"list": [              // 当天内发表的动态列表
			{
				"dynamic": {  // 字段详见http://120.131.64.91:8182/pkg/yuanfen/yf_service/cls/data_model/dynamics/#Dynamic
					"comments": 0,
					"gameinit": "",
					"gamekey": 0,
					"id": 28,
					"isLike": 0,
					"likes": 0,
					"location": "北京市海淀区清河中街",
					"pic": [
					"http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg",
					"http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
					"http://image1.yuanfenba.net/uploads/oss/avatar_big/201509/14/13272828068.jpg"
					],
					"sign": 0,
					"stype": 0,
					"text": "我的心头想爱你个",
					"tm": "2015-09-24T20:44:42+08:00",
					"type": 1,
					"uid": 5000761,
					"url": ""
				},
				"user": {
					"age": 25,
					"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
					"job": "演艺人员",
					"nickname": "小气\u0026豪猪",
					"uid": 5000761
				}
			}
			]
		}
		],
		"status": "ok",
		"tm": 1444807375
	}
*/
func (dm *DynamicsModule) SecOtherDynamic(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id, uid uint32
	ps := 50
	if e = req.ParseOpt("id", &id, 0, "uid", &uid, 0); e != nil {
		return
	}
	date_m := make(map[string][]dynamics.Dynamic)
	date_arr := make([]string, 0, 10)
	reslist := make(map[string]interface{})
	if id == 0 {
		u, e := user_overview.GetUserObject(uid)
		if e != nil {
			return e
		}
		reslist["dynamic_img"] = u.Dynamic_img
	}

	hasNext := false
doSeach:
	list, e := dynamics.GetMydynamicList(id, uid, ps)
	if e != nil {
		return
	}
	// 将动态按日期分组
	for _, dy := range list {
		tm, er := utils.ToTime(dy.Tm, format.TIME_LAYOUT_1)
		if er != nil {
			return er
		}
		stm := tm.Format(format.TIME_LAYOUT_2)
		arr := make([]dynamics.Dynamic, 0, 5)
		if v, ok := date_m[stm]; ok {
			arr = v
		} else {
			date_arr = append(date_arr, stm)
		}
		arr = append(arr, dy)
		date_m[stm] = arr
	}

	// len(list)<20 则取全部，否则需判断舍去最后一天
	end := len(date_arr)
	dm.log.AppendObj(nil, "seach res :", len(list), ps, len(date_m))
	if len(list) >= ps {
		if len(date_m) < 2 {
			a := date_m[date_arr[len(date_arr)-1]]
			id = a[len(a)-1].Id
			goto doSeach
		} else {
			end = len(date_arr) - 1
		}
		hasNext = true
	}
	res := make([]map[string]interface{}, 0, len(date_arr))
	for i := 0; i < end; i++ {
		item := make(map[string]interface{})
		date := date_arr[i]
		item["date"] = date
		dl, ok := date_m[date]
		if !ok {
			continue
		}
		r, e := dynamics.GenDynamicsRes(dl, req.Uid, 2)
		if e != nil {
			return e
		}
		item["list"] = r
		res = append(res, item)
	}
	reslist["dynamics"] = res
	reslist["hasNext"] = hasNext
	result["res"] = reslist
	return
}

/*
个人中心，获取我的动态

URL：s/dynamics/MyDynamic

参数：
	id:起始id

返回值：
	{
	  "res":{
		"dynamic_img":"xxxxx",
		"hasNext":true,   // 是否有下一页
		"dynamics":[
		{
			"date": "2015-09-24",  // 日期
			"list": [              // 当天内发表的动态列表
			{
				"dynamic": {  // 字段详见http://120.131.64.91:8182/pkg/yuanfen/yf_service/cls/data_model/dynamics/#Dynamic
					"comments": 0,
					"gameinit": "",
					"gamekey": 0,
					"id": 28,
					"isLike": 0,
					"likes": 0,
					"location": "北京市海淀区清河中街",
					"pic": [
					"http://image2.yuanfenba.net/uploads/oss/photo/201506/19/11580056241.jpg",
					"http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
					"http://image1.yuanfenba.net/uploads/oss/avatar_big/201509/14/13272828068.jpg"
					],
					"sign": 0,
					"stype": 0,
					"text": "我的心头想爱你个",
					"tm": "2015-09-24T20:44:42+08:00",
					"type": 1,
					"uid": 5000761,
					"url": ""
				},
				"user": {
					"age": 25,
					"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201508/25/18142081295.jpg",
					"job": "演艺人员",
					"nickname": "小气\u0026豪猪",
					"uid": 5000761
				}
			}
			]
		}
		]
		}
		"status": "ok",
		"tm": 1444807375
	}
*/
func (dm *DynamicsModule) SecMyDynamic(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id uint32
	uid := req.Uid
	ps := 50
	if e = req.ParseOpt("id", &id, 0); e != nil {
		return
	}
	if e = req.Parse("id", &id); e != nil {
		return
	}
	reslist := make(map[string]interface{})
	if id == 0 {
		u, e := user_overview.GetUserObject(uid)
		if e != nil {
			return e
		}
		reslist["dynamic_img"] = u.Dynamic_img
	}

	date_m := make(map[string][]dynamics.Dynamic)
	date_arr := make([]string, 0, 10)
	hasNext := false
doSeach:
	list, e := dynamics.GetMydynamicList(id, req.Uid, ps)
	if e != nil {
		return
	}
	// 将动态按日期分组
	for _, dy := range list {
		tm, er := utils.ToTime(dy.Tm, format.TIME_LAYOUT_1)
		if er != nil {
			return er
		}
		stm := tm.Format(format.TIME_LAYOUT_2)
		arr := make([]dynamics.Dynamic, 0, 5)
		if v, ok := date_m[stm]; ok {
			arr = v
		} else {
			date_arr = append(date_arr, stm)
		}
		arr = append(arr, dy)
		date_m[stm] = arr
	}

	// len(list)<20 则取全部，否则需判断舍去最后一天
	end := len(date_arr)
	dm.log.AppendObj(nil, "seach res :", len(list), ps, len(date_m))
	if len(list) >= ps {
		if len(date_m) < 2 {
			a := date_m[date_arr[len(date_arr)-1]]
			id = a[len(a)-1].Id
			dm.log.AppendObj(nil, "循环查询，id ", id)
			goto doSeach
		} else {
			end = len(date_arr) - 1
		}
		hasNext = true
	}
	res := make([]map[string]interface{}, 0, len(date_arr))
	for i := 0; i < end; i++ {
		item := make(map[string]interface{})
		date := date_arr[i]
		item["date"] = date
		dl, ok := date_m[date]
		if !ok {
			continue
		}
		r, e := dynamics.GenDynamicsRes(dl, req.Uid, 2)
		if e != nil {
			return e
		}
		item["list"] = r
		res = append(res, item)
	}
	reslist["dynamics"] = res
	reslist["hasNext"] = hasNext
	result["res"] = reslist
	return
}

/*
获取标记我的用户数量(暂时未用)

URL: s/dynamics/MarkMeNum

返回值：
	{
		"res": {
			"markMeNum": 10,  // 标记我的用户数
		},
		"status": "ok",
		"tm": 1446089200
	}

*/
func (dm *DynamicsModule) SecMarkMeNum(req *service.HttpRequest, result map[string]interface{}) (e error) {
	m, e := general.GetInterestedUids(req.Uid)
	if e != nil {
		return
	}
	res := make(map[string]interface{})
	res["markMeNum"] = len(m)
	result["res"] = res
	return
}
