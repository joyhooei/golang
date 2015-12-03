package user

/*
 用户认证方面模块
*/
import (
	"errors"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/certify"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/data_model/usercontrol"
)

/* 身份证认证接口 URI: /s/user/IdCardCertify
参数：
	{
		name:"张三",           	// 姓名
		id:"61232119908235011",	// 身份证号
		os:0//[opt]操作系统 不填默认0为安卓 1为苹果
	}

返回值:
	{
		status:"ok",
		msg:"认证成功",
		res:{
			certify_level:2 // 认证等级
			tip:xx
		}
	}
或者
	{
		status:"ok",
		res:{
			isPay:0, 	// 是否支付身份认证手续费（0 未支付，1 已支付）
			product:{ 	// 同充值接口商品列表
				coincost:200,id:15,img:xx,info:xx,money:xx,name:xx
			}
		}
	}
*/
func (sm *UserModule) SecIdCardCertify(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id, name, code, key string
	if e = req.Parse("id", &id, "name", &name); e != nil {
		return
	}
	os, err := utils.ToInt(req.Body["os"])
	if err != nil {
		os = 0
	}
	sm.log.AppendObj(nil, "code: ", code, "key: ", key, id, name)
	// 验证该身份证是否已经验证过
	if ok, e := usercontrol.CheckIdcardAvailable(id); !ok || e != nil {
		sm.log.AppendObj(e, "usercontrol.CheckIdcardAvailable is error ", id)
		return service.NewError(service.ERR_INTERNAL, "该身份证已被认证", "该身份证已被认证")
	}

	res := make(map[string]interface{})
	ismatch, ecode, e := usercontrol.DoIdCardCertify(req.Uid, id, name)
	if e != nil {
		sm.log.AppendObj(e, " doidcardcertify result is error ,match:  ", ismatch, " ,ecode: ", ecode, id, name)
		return e
	}
	sm.log.AppendObj(e, " doidcardcertify result ,ismatch: ", ismatch, ecode)
	res["match_status"] = 2
	res["isPay"] = 1
	res["tip"] = "认证失败"
	res["isMatch"] = ismatch
	if ecode == 10 { // 未充值
		// 添加一条认证记录
		lastId, e := usercontrol.AddIdCardRecord(req.Uid, id, name)
		if e != nil {
			return e
		}
		res["isPay"] = 0
		res["id"] = lastId
		// 获取身份证认证充值商品
		r, e := usercontrol.ProductList(4, 0, os)
		if e != nil {
			return e
		}
		if len(r) > 0 {
			res["product"] = r[0]
		}
	}
	if ismatch {
		res["tip"] = "认证成功"
	}
	var certify_level int
	if u, e := user_overview.GetUserObjectNoCache(req.Uid); e == nil {
		certify_level = u.CertifyLevel
	}
	sm.log.AppendObj(nil, "--直接返回结果-", certify_level, res)
	res["certify_level"] = certify_level
	result["res"] = res
	return
}

/* 视频认证接口 URI: /s/user/VideoCertify
参数：
	 {video_url:"xx"}		说明：video_url:视频图片（字符串）以逗号隔开
返回值：
 	{status:"ok",msg:"提交成功"}
*/
func (sm *UserModule) SecVideoCertify(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var video_url string
	if e = req.Parse("video_url", &video_url); e != nil {
		return
	}
	if video_url == "" {
		return errors.New("parm is empty")
	}
	//1.检测自身状态
	u, e := user_overview.GetUserObjectNoCache(req.Uid)
	if e != nil {
		return e
	}
	if u.CertifyVideo == 1 {
		return service.NewError(service.ERR_INTERNAL, "", "已视频认证，不能重复认证")
	}
	/*	s3 := "select count(*) from video_record where uid = ? and status = 0"
		var cnt int
		if e := sm.mdb.QueryRow(s3, req.Uid).Scan(&cnt); e != nil {
			return e
		}
		if cnt > 0 { // 已经存在记录，待审核,不能重复提交
			return
		}
		tx, e := sm.mdb.Begin()
		if e != nil {
			return e
		}
	*/
	tx, e := sm.mdb.Begin()
	if e != nil {
		return
	}
	// 重新提交
	s4 := "update video_record set status = -2 where uid = ? and status= 0"
	if _, e = tx.Exec(s4, req.Uid); e != nil {
		tx.Rollback()
		return
	}
	s := "insert into video_record(uid,photos) values(?,?)"
	if _, e = tx.Exec(s, req.Uid, video_url); e != nil {
		tx.Rollback()
		return e
	}
	tx.Commit()
	return
}

/*
获取用户特权接口

URI: /s/user/Privilege

返回值：
	{
	 "privilege": {
		  	"pri_bigimg":{
				can:fasle,	// 是否具有该特权
				bal:0,		// 权限余额
				msg:"xxx",	// 弹出框提示内容
				but:{		// 弹出框按钮执行操作
					tip:"查看",   		//按钮上的提示
					cmd:"cmd_idcard_pri",	//按钮执行cmd
					def:true              	//是否为默认操作
					data:{}			//cmd所需参数
				}
			},
		   "pri_chat": {...},    	// 每日聊天权限
		   "pri_contact": {...},  	// 查看联系方式
		   "pri_follow": {...},  	// 每日关注人数
		   "pri_nearmsg": {...},  	// 发附近消息
		   "pri_nearmsg_filter": {...},  // 附近消息高级定向功能
		   "pri_online_award": {...},    // 在线奖励
		   "pri_phonelogin": {...},      // 使用手机登录
		   "pri_private_photos": {...},  // 浏览私密照
		   "pri_pursue": {...},          // 每日追求数
		   "pri_sayhi": {...},         	 // 每日打招呼
		   "pri_search": {...},          // 使用高级筛选
		   "pri_see_require": {...}      // 查看用户的择友条件
		},
		"status": "ok",
		"tm": 1438569801
	}
	cmd 对应关系详见：http://120.131.64.91:8082/pkg/yuanfen/yf_service/cls/notify/#pkg-constants
*/
func (sm *UserModule) SecPrivilege(req *service.HttpRequest, result map[string]interface{}) (e error) {
	/*	if _, e := certify.CheckCretifyLevel(req.Uid); e != nil {
			return e
		}
		m, _, e := certify.GetPrivilege(req.Uid)
		if e != nil {
			return e
		}
		result[common.PRI_KEY] = m
	*/
	return
}

/*
SecInviteFill邀请用户完善信息

URI: s/user/InviteFill

参数:
		{
			"uid":123,	//用户ID
			"key":"certify", //邀请类型，certify-认证邀请，require-择友条件, photo-上传照片邀请
		}
返回值:
	{
		"status": "ok"，
	}

*/
func (sm *UserModule) SecInviteFill(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	var key string
	if err := req.Parse("uid", &uid, "key", &key); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if req.Uid == uid {
		return service.NewError(service.ERR_INVALID_REQUEST, "", "不能邀请自己")
	}
	e = usercontrol.InviteFill(req.Uid, uid, key)
	return
}

/*
SecInviteList查看当天邀请自己完善信息的用户列表

URI: s/user/InviteList

参数:
		{
			"key":"certify", //邀请类型，certify-认证邀请，require-择友条件，photo-上传照片邀请
		}
返回值:
	{
		"res": {
			"users": [
				{
					"uid": 1008391,
					"nickname": "小萝莉",
					"avatar": "http://image1.yuanfenba.net/uploads/oss/photo/201506/26/10445630138.jpg",
					"tm": "1992-06-28T00:00:00+08:00"	//邀请的时间
				}
			]
		}
		"status": "ok",
		"tm": 1438229879
	}


*/
func (sm *UserModule) SecInviteList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var key string
	if err := req.Parse("key", &key); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	users, e := usercontrol.InviteList(req.Uid, key)
	if e != nil {
		return e
	}
	res := make(map[string]interface{})
	res["users"] = users
	result["res"] = res
	return
}

/*
头像审核认证通知

URI: s/user/ICertifyAvatar

参数：
	uid: 用户uid
*/
func (sm *UserModule) SecICertifyAvatar(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	var uid uint32
	if e := req.Parse("uid", &uid); e != nil {
		return e
	}
	_, e = certify.CheckCretifyLevel(uid, common.PRI_GET_AVATAR)
	return
}

/*
放弃认证接口

URI:  s/user/CloseCertify

参数:
	key 关闭认证的key,包含以下值:
	PRI_GET_PHONE  = "pri_get_phone"  // 手机认证
	PRI_GET_VIDEO  = "pri_get_video"  // 视频认证
	PRI_GET_IDCARD = "pri_get_idcard" // 身份证认证
*/
func (sm *UserModule) SecCloseCertify(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var key string
	if e := req.Parse("key", &key); e != nil {
		return e
	}
	tx, e := sm.mdb.Begin()
	if e != nil {
		return e
	}
	u, e := user_overview.GetUserObject(req.Uid)
	if e != nil {
		return e
	}
	if key == common.PRI_GET_VIDEO {
		if e := usercontrol.UpdateVideoStatus(tx, req.Uid, 0); e != nil {
			tx.Rollback()
			return e
		}
		s := "update video_record set status = -2 where uid = ? and status in(0,1)"
		if _, e := tx.Exec(s, req.Uid); e != nil {
			tx.Rollback()
			return e
		}

	} else if key == common.PRI_GET_IDCARD {
		if u.CertifyIDcard == 0 {
			return errors.New("还未认证")
		}
		// 只需要修改下验证状态
		if e := usercontrol.UpdateIdcardStatus(tx, req.Uid, 0); e != nil {
			tx.Rollback()
			return e
		}
		s := "update idcertify_record set is_use = -1 where uid =? and is_use =1"
		if _, e := tx.Exec(s, req.Uid); e != nil {
			tx.Rollback()
			return e
		}
	}
	tx.Commit()
	// 新检测等级和诚信值变化
	//	_, e = certify.CheckCretifyLevel(req.Uid, key)
	return
}

/*
获取星级权限配置 SecGetAllHonesty
URI: user/GetAllHonesty
参数：空
返回值：
	{
		 "res": {
		 	 "honestylist": [
   							{
    							"level": 0,
    							"list": [
								{
								"level":1,
								"item":"pri_chat",
								"name":"每日主动私聊用户人数",
								"num":0,
								"flag":1,
								"tips":"人"
								},{}
		 						]
		 					}
			 ]
		 },
		 "status": "ok",
		 "tm": 1438744687
	}
*/
func (sm *UserModule) GetAllHonesty(req *service.HttpRequest, result map[string]interface{}) (e error) {
	res := make(map[string]interface{})
	m, e := certify.GetAllHonesty()
	if e != nil {
		return e
	}
	rmap := make([]map[string]interface{}, 0, 0)
	for k, v := range m {
		amap := make(map[string]interface{})
		amap["level"] = k
		amap["list"] = v
		rmap = append(rmap, amap)
	}
	res["honestylist"] = rmap
	result["res"] = res
	return
}

/*
获取诚信详情 SecGetMyHonestyInfo
URI: s/user/GetMyHonestyInfo
参数
{
	"uid":"112233"//[opt]要查看的用户ID 不填写表示查看自己的
}
返回值：
	{
		 "res": {
		 	  "honesty_level": 3,   //诚信等级
			  "pri_get_avatar": 0,    // 头像认证 1为已认证
			  "pri_get_photos":1,// 相册至少上传3张生活照
			  "pri_get_info": 1,// 基本必填项填写
			  "pri_get_phone":1,// 手机认证
			  "pri_get_video": 0,// 视频认证
			  "pri_get_idcard":0// 身份证认证
		 },
		 "status": "ok",
		 "tm": 1438744687
	}
*/
func (sm *UserModule) SecGetMyHonestyInfo(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var uid uint32
	_, ok := req.EnsureBody("uid")
	if ok {
		if i, err := utils.ToUint32(req.Body["uid"]); err != nil {
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		} else {
			uid = i
		}
	} else {
		uid = req.Uid
	}
	res, e := certify.GetMyHonestyInfo(uid)
	if e != nil {
		return e
	}
	result["res"] = res
	return
}

/*
轮询身份证认证结果接口

URI:  s/user/CheckIdCardResult

参数：
	id: 轮询id，由身份证返回的接口

返回值：
	{
		"res":{
			"match_status": 0,	// 匹配状态 0 待处理，1，已付费，2 ，已验证
			"isMatch": true/false,	// 认证是否成功
			"tip":"xx"		// 提示信息
			"certify_level":2		// 认证等级
		}
		"status": "ok",
		"tm": 1438744687
	}
*/
func (sm *UserModule) SecCheckIdCardResult(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var id, status int
	if e := req.Parse("id", &id); e != nil {
		return e
	}
	s := "select status from idcertify_record where id = ?"
	if e := sm.mdb.QueryRow(s, id).Scan(&status); e != nil {
		return e
	}
	res := make(map[string]interface{})
	res["match_status"] = status
	res["isMatch"] = false
	res["tip"] = "认证失败"
	u, e := user_overview.GetUserObjectNoCache(req.Uid)
	if e != nil {
		return e
	}
	if status == 2 && u.CertifyIDcard == 1 {
		res["isMatch"] = true
		res["tip"] = "认证成功"
		res["certify_level"] = u.CertifyLevel
	}
	result["res"] = res
	return
}
