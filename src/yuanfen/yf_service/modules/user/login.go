package user

import (
	"fmt"
	"strings"
	"time"
	// "yf_pkg/lbs/baidu"
	"yf_pkg/push"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/dynamics"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/login"
	"yuanfen/yf_service/cls/data_model/relation/base"
	"yuanfen/yf_service/cls/data_model/topic"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/data_model/usercontrol"
	"yuanfen/yf_service/cls/message"
	"yuanfen/yf_service/cls/unread"
)

const (
	SMS_CODECHECK_FORMAT = "你的短信验证码是: %v ."
)

const (
	key_WX_OPENID = "openid"
)

func getport(suri string) (host string, port uint) {
	r := strings.Split(suri, ":")
	if len(r) == 2 {
		host = r[0]
		port, _ = utils.StringToUint(r[1])
	}
	return
}

func (sm *UserModule) TestRelogin(req *service.HttpRequest, result map[string]interface{}) (e error) {
	// uid, _ := utils.ToUint32(req.GetParam("uid"))
	base.AddFriend(5001686, 5001687)
	// usercontrol.SetAvatar(uid, "http://image1.yuanfenba.net/uploads/oss/avatar/201510/17/10425329626.jjg")
	// info, e := user_overview.GetUserPhotos(uid, 5000366)
	// res := make(map[string]interface{})
	// for k, v := range info {
	// 	res[utils.ToString(k)] = v
	// }
	// result["res"] = res
	return
}

func (sm *UserModule) GetUidCode(req *service.HttpRequest, result map[string]interface{}) (e error) {
	phone := req.GetParam("phone")
	code, e := login.GetPhoneCode(phone)
	if e != nil {
		return
	}
	res := make(map[string]interface{})
	res["code"] = code
	result["res"] = res

	return
}

func (sm *UserModule) DelMyPhone(req *service.HttpRequest, result map[string]interface{}) (e error) {
	phone := req.GetParam("phone")
	sm.mdb.Exec("update user_main set phone='' where phone=?", phone)
	return
}

func (sm *UserModule) GetNick(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	res := make(map[string]interface{})
	res["list"], res["list_female"] = usercontrol.GetRandomNick()
	result["res"] = res
	return
}

/*
三方登陆

请求URL：user/ThirdLogin

参数:

{
	"tp":1,  //快捷登陆类型 1 微信 2 QQ 3 微博
	"code":"112233"//[opt]用户换取access_token的code 只在微信有用
	"openid":"sasds121212"//[opt]用户ID 只在 QQ  微博 有用
	"ver":"1.01" //[opt] 版本号
	"channel":"abc"//[opt] 渠道号
	"channel_sid":"abc"//[opt]子渠道号
	"imei":""//[opt]imei
	"imsi":""//[opt]imsi
	"mac":""//[opt]mac
	"sysver":"4.42"//[opt]操作系统版本
	"factory":"HUAWEI"//[opt]生产厂商
	"model":"hwsdddd"//[opt]手机型号
	"devid":123123,//设备ID，32位整数
	"lat":32.22,
	"lng":123.442
}

返回结果：

{
	"status": "ok",
	"res":{
			"guidecomplete":1  //是否完成引导 1为已完成 0为未完成
	       	"uid":10001,  //uid
			"sid":"ea34343434bbccaaaaf20f883e"
	}
}
*/
func (sm *UserModule) ThirdLogin(req *service.HttpRequest, result map[string]interface{}) (e error) {

	var openid, sid string
	var uid uint32
	var ifnew bool = false
	var tp int
	var lat, lng float64
	err := req.Parse("tp", &tp, "lat", &lat, "lng", &lng)
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	var fieldname string
	switch tp {
	case 1: //微信
		if err = req.ParseOpt(key_WX_OPENID, &openid, ""); err != nil {

		}
		if openid == "" {
			abs, ok := req.EnsureBody("code")
			if !ok {
				return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
			}
			code := utils.ToString(req.Body["code"])

			vmap, err := login.GetWeixin(code)
			if err != nil {
				return service.NewError(service.ERR_INTERNAL, err.Error())
			}
			if v, ok := vmap[key_WX_OPENID]; !ok {
				return service.NewError(service.ERR_UNKNOWN, "微信ID转换错误")
			} else {
				openid = utils.ToString(v)
			}
		}

		fieldname = "wx_username"

	case 2: //qq
		abs, ok := req.EnsureBody(key_WX_OPENID)
		if !ok {
			return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
		}
		openid = utils.ToString(req.Body[key_WX_OPENID])
		fieldname = "qq_username"
	case 3: //微博
		abs, ok := req.EnsureBody(key_WX_OPENID)
		if !ok {
			return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
		}
		openid = utils.ToString(req.Body[key_WX_OPENID])
		// token := utils.ToString(req.Body["token"])
		fieldname = "wb_username"
	}
	var guidecomplete int
	ifnew, uid, sid, guidecomplete, err = login.CheckThirdID(openid, fieldname)
	if err != nil {
		if err.Error() == "此账号因违规已被封禁。" {
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		}
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	var ver, imei, imsi, mac string
	var devid uint32
	if e := req.Parse("ver", &ver, "imei", &imei, "imsi", &imsi, "mac", &mac, "devid", &devid); e != nil {
		return e
	}

	var channel, channel_sid string
	req.Parse("channel", &channel, "channel_sid", &channel_sid)
	var sysver, factory, model string
	req.Parse("sysver", &sysver, "factory", &factory, "model", &model)

	if ifnew {
		uid, sid, err = login.RegMain(tp, openid, "", "", common.GENDER_BOTH, ver, imei, imsi, mac, sysver, factory, model, channel, channel_sid, req.IP(), "", devid)
		if err != nil {
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
		imodel := login.CheckPhoneModel(model, factory)
		user_overview.SetSystemInfo(uid, imodel)
	} else {
		login.PostVer(uid, imei, imsi, mac) //ver,
		login.PostModel(uid, sysver, factory, model)
		imodel := login.CheckPhoneModel(model, factory)
		user_overview.SetSystemInfo(uid, imodel)
	}

	if !ifnew {
		if b, err := login.CheckRelogin(uid); err == nil {
			if b {
				sm.log.AppendInfo(fmt.Sprintf("PushRelogin uid %v,imei %v", uid, imei))
				imei = utils.ToString(req.Body["imei"])
				usercontrol.PushRelogin(uid, imei)
				time.Sleep(1 * time.Second)
				// push.Kick(uid)
			}
		}
	}
	go login.UpdateUidArea(uid, lat, lng, req.IP(), common.GENDER_BOTH)
	res := make(map[string]interface{})

	res["guidecomplete"] = guidecomplete
	res["uid"] = uid
	res["sid"] = sid
	result["res"] = res
	return
}

//查询慕慕ID是否已存在
func (sm *UserModule) CheckUser(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {

	abs, ok := req.EnsureBody("uname")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	username := utils.ToString(req.Body["uname"])
	iexists, err := login.CheckUser(username)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["exists"] = iexists
	result["res"] = res
	return
}

/*
Reg2 手机号直接注册
请求URL：user/Reg2
参数:
{
   	"sid":,//[opt]密码 ,需要把慕慕密码做一次MD5
   	"phone":"131XXXXXXXX",//[opt]用户手机号 只有类型6包含此字段
   	"code":"112233",//[opt]短信验证码 只有类型6包含此字段
	"ver":"1.01", //[opt] 版本号
	"channel":"abc",//[opt] 渠道号
	"channel_sid":"abc",//[opt]子渠道号
	"imei":"",//[opt]imei
	"imsi":"",//[opt]imsi
	"mac":"",//[opt]mac
	"lat":"1.1111",//[opt]lat
	"lng":"2.2222",//[opt]lng
	"sysver":"4.42",//[opt]操作系统版本
	"factory":"HUAWEI",//[opt]生产厂商
	"model":"hwsdddd",//[opt]手机型号
	"devid":123123,//设备ID，32位整数
}
返回结果：
{
	"status": "ok",
	"code":0   //返回错误时，2009表示短信验证码错误 只会发生在类型6，7
	"res": {
		"uid": 10099     // 用户id
		"sid":"ea34343434bbccaaaaf20f883e"  // sid
	}
}
*/
func (sm *UserModule) Reg2(req *service.HttpRequest, result map[string]interface{}) (e error) {
	res := make(map[string]interface{})
	var ver, imei, imsi, mac string
	var devid uint32
	if e := req.Parse("ver", &ver, "imei", &imei, "imsi", &imsi, "mac", &mac, "devid", &devid); e != nil {
		return e
	}
	canreg, err := login.CheckImei(imei, imsi, mac)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if !canreg {
		return service.NewError(service.ERR_INTERNAL, "无法注册", "无法注册")
	}

	var spass, phone, code string
	abs, ok := req.EnsureBody("phone", "sid", "code")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	phone = utils.ToString(req.Body["phone"])
	spass = utils.ToString(req.Body["sid"])
	code = utils.ToString(req.Body["code"])
	r, err := login.CheckPhoneCode(phone, code)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if r != 1 {
		return service.NewError(service.ERR_VCODE_ERROR, "验证码输入错误", "验证码输入错误")
	}

	var sysver, factory, model string
	if _, ok := req.EnsureBody("sysver", "factory", "model"); ok {
		sysver = login.SubstrByByte(utils.ToString(req.Body["sysver"]), 16)
		factory = login.SubstrByByte(utils.ToString(req.Body["factory"]), 16)
		model = login.SubstrByByte(utils.ToString(req.Body["model"]), 16)
	}
	var channel, channel_sid string
	if _, ok := req.EnsureBody("channel", "channel_sid"); ok {
		channel = login.SubstrByByte(utils.ToString(req.Body["channel"]), 30)
		channel_sid = login.SubstrByByte(utils.ToString(req.Body["channel_sid"]), 30)
	}
	if sm.mode == cls.MODE_PRODUCTION {
		if canreg, err := login.IpCanReg(req.IP()); err != nil {
			return service.NewError(service.ERR_INTERNAL, err.Error())
		} else {
			if !canreg {
				return service.NewError(service.ERR_INTERNAL, "注册失败，次数过多", "注册失败，次数过多")
			}
		}
	}

	uid, nsid, err := login.RegMain(7, "", "", spass, common.GENDER_BOTH, ver, imei, imsi, mac, sysver, factory, model, channel, channel_sid, req.IP(), phone, devid)
	if err != nil {
		switch err.Error() {
		case "用户名已存在", "绑定ID已存在":
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		default:
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
	}
	var lat, lng float64
	if _, ok := req.EnsureBody("lat", "lng"); ok {
		lat, _ = utils.ToFloat64(req.Body["lat"])
		lng, _ = utils.ToFloat64(req.Body["lng"])
	}

	go login.UpdateUidArea(uid, lat, lng, req.IP(), common.GENDER_BOTH)
	imodel := login.CheckPhoneModel(model, factory)
	sm.log.AppendInfo(fmt.Sprintf("CheckPhoneModel model %v,factory %v, sysinfo %v", model, factory, imodel))
	user_overview.SetSystemInfo(uid, imodel)

	unread.UpdateReadTime(uid, common.UNREAD_BINDPHONE, utils.Now.AddDate(-1, 0, 0))
	unread.UpdateReadTime(uid, common.UNREAD_WALLET, utils.Now.AddDate(-1, 0, 0))
	unread.UpdateReadTime(uid, common.UNREAD_LOCALTAG, utils.Now.AddDate(-1, 0, 0))
	//	unread.UpdateReadTime(uid, common.UNREAD_PROVID)
	unread.UpdateReadTime(uid, common.UNREAD_MYAWARD)
	unread.UpdateReadTime(uid, common.UNREAD_GIFT)
	// fmt.Println("22")
	login.IpRegDec(req.IP())
	res["uid"] = uid
	res["sid"] = nsid
	result["res"] = res
	// message.SendMessage(message.REGISTER, message.Register{uid, gender, birthday}, result)
	//设置OnnTop
	login.UserOnTop(uid, 1)
	message.SendMessage(message.ONTOP, message.OnTop{uid, 1}, result)
	usercontrol.StatUserOnTop(uid)
	return
}

// RegCheckPhone 注册时 校验手机号 是否注册过
//
// URI: user/RegCheckPhone
//
// 参数
// {
//		"phone":"137XXXXXXXX"//用户手机号
// }
//
// 返回值
// {
// 		"status":"ok"
// 		"code": 0,//code 返回错误 且为9001时 表示改手机号已被绑定
// 		"res":
// 		{
// 		}
// }
func (sm *UserModule) RegCheckPhone(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("phone")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	phone := utils.ToString(req.Body["phone"])
	if in, err2 := login.PhoneInDb(phone); err2 != nil {
		return service.NewError(service.ERR_INTERNAL, err2.Error())
	} else {
		if in {
			return service.NewError(service.ERR_CELLPHONE_EXISTS, "该手机号不可用或已被注册", "该手机号不可用或已被注册")
		}
	}
	return
}

// RegSendCode 注册时 发送验证码
//
// URI: user/RegSendCode
//
// 参数
// {
//		"phone":"137XXXXXXXX"//用户手机号
// }
//
// 返回值
// {
// 		"status":"ok"
// 		"code": 0,
// 		"res":
// 		{
// 		}
// }
func (sm *UserModule) RegSendCode(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var phone string
	if err := req.Parse("phone", &phone); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	scode := login.GetCodeRandom2()
	if err := login.AddPhoneCode(phone, scode); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	smsg := fmt.Sprintf(SMS_CODECHECK_FORMAT, scode)
	if err := general.SendSmsDelay(phone, smsg); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	sm.log.AppendInfo(fmt.Sprintf("RegSendCode %v,%v", phone, smsg))
	return
}

// RegCheckCode 注册时 验证手机验证码
//
// URI: user/RegCheckCode
//
// 参数
// {
//		"phone":"137XXXXXXXX"//用户手机号
// 		"code":"3344"//手机验证码
// }
//
// 返回值
// {
// 		"status":"ok"
// 		"code": 0,//code 返回错误 且为2009时 表示验证码输入错误
// 		"res":
// 		{
// 		}
// }
func (sm *UserModule) RegCheckCode(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var phone, code string
	if err := req.Parse("phone", &phone, "code", &code); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	r, err := login.CheckPhoneCode(phone, code)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if r != 1 {
		return service.NewError(service.ERR_VCODE_ERROR, "验证码输入错误", "验证码输入错误")
	}
	return
}

//绑定慕慕ID
func (sm *UserModule) SecBindMumu(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("username", "sid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	username := utils.ToString(req.Body["username"])
	spass := utils.ToString(req.Body["sid"])
	err := login.BindMumu(req.Uid, username, spass)
	if err != nil {
		switch err.Error() {
		case "此账号已存在":
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		default:
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
	}
	// err = user_overview.SetUserPassCache(req.Uid, sid)
	// if err != nil {
	// 	return service.NewError(service.ERR_INTERNAL, err.Error())
	// }
	res := make(map[string]interface{})
	// res["sid"] = sid
	result["res"] = res
	unread.UpdateReadTime(req.Uid, common.UNREAD_MUMUID)
	un := make(map[string]interface{})
	un[common.UNREAD_MUMUID] = 0
	unread.GetUnreadNum(req.Uid, un)
	result[common.UNREAD_KEY] = un
	return
}

//密码找回-发送验证短信
func (sm *UserModule) SendPhoneCode(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("phone")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	phone := utils.ToString(req.Body["phone"])
	if in, err2 := login.PhoneInDb(phone); err2 != nil {
		return service.NewError(service.ERR_INTERNAL, err2.Error())
	} else {
		if !in {
			return service.NewError(service.ERR_USER_NOT_FOUND, "此账号不存在！", "此账号不存在！")
		}
	}

	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// code := r.Intn(1000000)
	// scode := strconv.Itoa(code)
	// scode = strings.Repeat("0", 6-len(scode)) + scode
	scode := login.GetCodeRandom2()
	if err := login.AddPhoneCode(phone, scode); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	smsg := fmt.Sprintf(SMS_CODECHECK_FORMAT, scode)
	if err := general.SendSmsDelay(phone, smsg); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	return
}

//密码找回-验证短信码
func (sm *UserModule) CheckPhoneCode(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("phone", "code")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	phone := utils.ToString(req.Body["phone"])
	code := utils.ToString(req.Body["code"])
	r, err := login.CheckPhoneCode(phone, code)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["exists"] = r
	result["res"] = res
	return
}

//密码找回-设置密码
func (sm *UserModule) SetPhonePwd(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("phone", "sid", "code")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	phone := utils.ToString(req.Body["phone"])
	sid := utils.ToString(req.Body["sid"])
	code := utils.ToString(req.Body["code"])
	v, err := login.CheckPhoneCode(phone, code)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if v != 1 {
		return service.NewError(service.ERR_INTERNAL, "验证短信失败", "验证短信失败")
	}
	login.DelPhone(phone)
	uid, rsid, err := login.SetPhonePwd(phone, sid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
	}
	err = user_overview.SetUserPassCache(uid, rsid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	return
}

//发送验证短信（手机绑定,解除绑定）
func (sm *UserModule) SecSendPhoneCode(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("phone")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}

	phone := utils.ToString(req.Body["phone"])
	if can, err := login.MyPhoneUser(req.Uid, phone); err != nil {
		switch err.Error() {
		case "不存在此手机号":
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		default:
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}

	} else {
		if !can {
			if in, err2 := login.PhoneInDb(phone); err2 != nil {
				return service.NewError(service.ERR_INTERNAL, err2.Error())
			} else {
				if in {
					return service.NewError(service.ERR_CELLPHONE_EXISTS, "手机号已存在！", "手机号已存在！")
				}
			}
		}
	}

	scode := login.GetCodeRandom2()
	if err := login.AddPhoneCode(phone, scode); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	smsg := fmt.Sprintf(SMS_CODECHECK_FORMAT, scode)
	if err := general.SendSmsDelay(phone, smsg); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	return
}

// SecCertifyPhoneSend 手机认证发送验证短信
//
// URI: s/user/CertifyPhoneSend
//
// 参数
// {
//		"phone":“13111111111”,//[opt]用户手机号 解除认证时可不填
// }
//
// 返回值
// {
// 		"status":"ok"
// 		"res":
// 		{
// 		}
// }
func (sm *UserModule) SecCertifyPhoneSend(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var phone string
	_, ok := req.EnsureBody("phone")
	if ok {
		phone = utils.ToString(req.Body["phone"])
	}
	sp, err := login.GetPhone(req.Uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if phone != "" {
		if sp != "" && sp != phone {
			return service.NewError(service.ERR_INTERNAL, "先解除绑定手机号", "先解除绑定手机号")
		}
		if in, err2 := login.PhoneInDb(phone); err2 != nil {
			return service.NewError(service.ERR_INTERNAL, err2.Error())
		} else {
			if in {
				return service.NewError(service.ERR_CELLPHONE_EXISTS, "手机号已存在！", "手机号已存在！")
			}
		}
	} else {
		if sp == "" {
			return service.NewError(service.ERR_INTERNAL, "无绑定手机号", "无绑定手机号")
		}
		phone = sp
	}

	scode := login.GetCodeRandom2()
	if err := login.AddPhoneCode(phone, scode); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	smsg := fmt.Sprintf(SMS_CODECHECK_FORMAT, scode)
	if err := general.SendSmsDelay(phone, smsg); err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	return
}

// SecCertifyPhone 手机认证 完成验证
//
// URI: s/user/CertifyPhone
//
// 参数
// {
//		"phone":"13111111111",//[opt]用户手机号 解除认证时可以不填
//		"btype":1,//1 绑定手机 2 解除绑定
//		"code":"3344"//短信验证码
// }
//
// 返回值
// {
// 		"status":"ok"
// 		"res":
// 		{
//			"certify_level" :2  // 认证等级
// 		}
// }
func (sm *UserModule) SecCertifyPhone(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("btype", "code")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	btype, err := utils.ToInt(req.Body["btype"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "btype")
	}
	code := utils.ToString(req.Body["code"])
	var phone string
	if v, ok := req.Body["phone"]; ok {
		phone = utils.ToString(v)
	}

	if btype == 2 {
		phone2, err := login.GetPhone(req.Uid)
		if err != nil {
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
		phone = phone2
	} else {
		if phone == "" {
			return service.NewError(service.ERR_INVALID_PARAM, "need [phone]")
		}
	}

	// fmt.Printf("phone %v, code %v", phone, code)
	v, err := login.CheckPhoneCode(phone, code)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if v != 1 {
		return service.NewError(service.ERR_INTERNAL, "验证短信失败", "验证短信失败")
	}
	if err := login.BandPhone(req.Uid, phone, btype); err != nil {
		switch err.Error() {
		case "填入的手机号已存在", "解除绑定失败,手机号不同":
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		default:
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
	}
	certify_level := 1
	//	certify_level, err := certify.CheckCretifyLevel(req.Uid, common.PRI_GET_PHONE)
	//	if err != nil {
	//		return service.NewError(service.ERR_INTERNAL, err.Error())
	//	}
	usercontrol.NotifyAndDelInviteList(req.Uid, usercontrol.INVITE_KEY_CERTIFY_PHONE)
	usercontrol.RefStar(req.Uid)
	res := make(map[string]interface{})
	res["certify_level"] = certify_level
	sm.log.AppendObj(nil, "certify_phone is ", certify_level)
	result["res"] = res
	return
}

/*
手机绑定 SecBindPhone

请求URL：s/user/BindPhone

参数:

{
	"phone ":"13701232332",  	//手机号
	"code":"315432"				//验证码
	"newpass":"aabbcc112233"	//新密码的MD5
}

返回结果：

{
	"status": "ok",
}
*/
func (sm *UserModule) SecBindPhone(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var phone, code, newpass string
	if e = req.Parse("phone", &phone, "code", &code, "newpass", &newpass); e != nil {
		return
	}
	// fmt.Printf("phone %v, code %v", phone, code)
	v, err := login.CheckPhoneCode(phone, code)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if v != 1 {
		return service.NewError(service.ERR_INTERNAL, "验证短信失败", "验证短信失败")
	}

	if err := login.BindPhone(req.Uid, phone, newpass); err != nil {
		switch err.Error() {
		case "填入的手机号已存在":
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		default:
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
	}

	unread.UpdateReadTime(req.Uid, common.UNREAD_BINDPHONE)
	un := make(map[string]interface{})
	un[common.UNREAD_BINDPHONE] = 0
	unread.GetUnreadNum(req.Uid, un)
	result[common.UNREAD_KEY] = un

	return
}

/*
修改绑定手机号 SecChangeBindPhone

请求URL：s/user/ChangeBindPhone

参数:

{
	"oldphone":"13701232332",	//之前的手机号
	"oldcode":"3154",			//之前的手机号的验证码
	"newphone":"13701232332",	//新的手机号
	"newcode":"3154",			//新的手机号的验证码
	"newpass":"aabbcc112233"	//新密码的MD5
}

返回结果：

{
	"status": "ok",
}
*/
func (sm *UserModule) SecChangeBindPhone(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var oldphone, newphone, oldcode, newcode, newpass string
	if e = req.Parse("oldphone", &oldphone, "newphone", &newphone, "oldcode", &oldcode, "newcode", &newcode, "newpass", &newpass); e != nil {
		return
	}

	// fmt.Printf("phone %v, code %v", phone, code)
	v, err := login.CheckPhoneCode(oldphone, oldcode)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if v != 1 {
		return service.NewError(service.ERR_INTERNAL, "验证短信失败", "验证短信失败")
	}

	v, err = login.CheckPhoneCode(newphone, newcode)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	if v != 1 {
		return service.NewError(service.ERR_INTERNAL, "验证短信失败", "验证短信失败")
	}

	if err := login.ChangePhone(req.Uid, oldphone, newphone, newpass); err != nil {
		switch err.Error() {
		case "填入的手机号已存在":
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		default:
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
	}

	return
}

// 修改密码
// 请求URL：s/user/ChangePwd
// 参数: {
// 		"oldsid":"210adc3949ba59abbe56232323383e",//用户旧的sid
// 		"sid":"e10adc3949ba59abbe56e057f20f883e" //用户新设置的sid
// 	}
// 返回结果：
// {
// 	"status": "ok",
// 	"res":{
// 		"sid":"ea34343434bbccaaaaf20f883e"
// 	}
// }
func (sm *UserModule) SecChangePwd(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("sid", "oldsid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	upass := utils.ToString(req.Body["sid"])
	oldpass := utils.ToString(req.Body["oldsid"])
	sid, err := login.ChangePwd(req.Uid, upass, oldpass)
	if err != nil {
		switch err.Error() {
		case "修改失败，密码不对":
			return service.NewError(service.ERR_INTERNAL, err.Error(), err.Error())
		default:
			return service.NewError(service.ERR_INTERNAL, err.Error())
		}
	}
	err = user_overview.SetUserPassCache(req.Uid, sid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	return
}

/*
用户获取连接点

请求URL：s/user/GetEndpoint

参数说明：

{
	"sysinfo":1//操作系统，2为小米，1为苹果，其他为0
}

返回结果：

{
	"status": "ok",
	"res": {
		"address":"12.0.34.1",//用户TCP长连接地址
		"port":10131,// 长连接端口号  i
		"key":"xdkrekdsxx"  //TCP长连接使用的KEY
	}
}
*/
func (sm *UserModule) SecGetEndpoint(req *service.HttpRequest, result map[string]interface{}) (e error) {
	// var sysinfo string
	// if req.Uid >= 5000000 {
	// 	if v, ok := req.Body["sysinfo"]; ok {
	// 		sysinfo, err := utils.ToInt(v)
	// 		if err != nil {
	// 			return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	// 		}
	// 		user_overview.SetSystemInfo(req.Uid, sysinfo)
	// 		// _, err := sm.mdb.Exec("update user_detail set sysinfo=? where uid=?", sysinfo, req.Uid)
	// 		// if err != nil {
	// 		// 	return service.NewError(service.ERR_UNKNOWN, err.Error())
	// 		// }
	// 	}
	// }

	res := make(map[string]interface{})
	address, key, err := push.GetEndpoint(req.Uid)
	if err != nil {
		// fmt.Println(" GetEndpoint ERROR " + err.Error())
		return err
	}
	host, port := getport(address)
	res["address"] = host
	res["port"] = port
	res["key"] = key
	result["res"] = res
	// data := make(map[string]interface{})
	// data["uid"] = req.Uid
	// message.SendMessage(message.LOGIN, data, result)

	message.SendMessage(message.ONLINE, message.Online{req.Uid}, result)
	usercontrol.SetLoginTime(req.Uid)
	go login.UpdateUidGps(req.Uid)
	return
}

// 用户在前端状态更新
// 请求URL：s/user/OnTopStat
// {
// 	"stat":1 //1为在前端 0为在后端
// }
// 返回结果：
// {
// 	"status": "ok”
// }
func (sm *UserModule) SecOnTopStat(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("stat")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	stat, err := utils.ToInt(req.Body["stat"]) //1为在前端 0为在后端
	if err != nil {
		return
	}
	login.UserOnTop(req.Uid, stat)
	usercontrol.StatUserOnTop(req.Uid)
	message.SendMessage(message.ONTOP, message.OnTop{req.Uid, stat}, result)
	return
}

// 用户登录 Login
// 请求URL：user/Login
// 参数说明：
//
// {
// 	username:  //用户手机号
//  sid:  // sid 慕慕密码的 MD5
// 	"ver":"1.01" // 版本号
// 	"channel":"abc"// 渠道号
// 	"channel_sid":"abc"//子渠道号
// 	"imei":""//imei
// 	"imsi":""//imsi
// 	"mac":""//mac
// 	"sysver":"4.42"//操作系统版本
// 	"factory":"HUAWEI"//生产厂商
// 	"model":"hwsdddd"//手机型号
// }
// 注意：慕慕密码 即为用户密码
//
// 返回结果：
//
// {
// 	"status": "ok",
// 	"msg": "success"
//     "code": 0,
// 	"res": {
// 		"uid": 10099     // 用户id
// 		"sid": "e3ceb5881a0a1fdaad01296d7554868d"  //sid
// 		"guidecomplete":1  //是否完成引导 1为已完成 0为未完成
// 	}
// }
func (sm *UserModule) Login(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var guidecomplete int
	abs, ok := req.EnsureBody("sid", "username")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	username := utils.ToString(req.Body["username"])
	password := utils.ToString(req.Body["sid"])
	// var uid uint32
	// var pwd string
	uid, sid, guidecomplete, err := login.UserLogin(username, password)
	if err != nil {
		switch err.Error() {
		case "用户名不存在", "密码错误":
			return service.NewError(service.ERR_USER_NOT_FOUND, err.Error(), err.Error())
		default:
			return service.NewError(service.ERR_USER_NOT_FOUND, err.Error())
		}

	}
	var imei string
	res := make(map[string]interface{})
	res["uid"] = uid
	res["sid"] = sid
	res["guidecomplete"] = guidecomplete
	result["res"] = res
	// fmt.Println(fmt.Sprintf("Login %v", req))
	if _, ok := req.EnsureBody("imei", "imsi", "mac"); ok { //"ver",
		// ver := utils.ToString(req.Body["ver"])
		imei := utils.ToString(req.Body["imei"])
		imsi := utils.ToString(req.Body["imsi"])
		mac := utils.ToString(req.Body["mac"])
		login.PostVer(uid, imei, imsi, mac) // ver,
	}
	if _, ok := req.EnsureBody("sysver", "factory", "model"); ok {
		sysver := utils.ToString(req.Body["sysver"])
		factory := utils.ToString(req.Body["factory"])
		model := utils.ToString(req.Body["model"])
		login.PostModel(uid, sysver, factory, model)
		imodel := login.CheckPhoneModel(model, factory)
		user_overview.SetSystemInfo(uid, imodel)
	}
	// fmt.Println(fmt.Sprintf("Login IMEI %v", req.Body["imei"]))
	if b, err := login.CheckRelogin(uid); err == nil {
		if b {
			sm.log.AppendInfo(fmt.Sprintf("PushRelogin uid %v,imei %v", uid, imei))
			imei = utils.ToString(req.Body["imei"])
			usercontrol.PushRelogin(uid, imei)
			time.Sleep(1 * time.Second)
			// fmt.Println(fmt.Sprintf("SysSleep %v", utils.Now))
			// push.Kick(uid)
		}
	}
	return
}

func (sm *UserModule) Disconnect(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var uid uint32
	abs, ok := req.EnsureBody("uid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	if v, err := utils.ToUint32(req.Body["uid"]); err == nil {
		uid = v
	}
	data := make(map[string]interface{})
	data["uid"] = uid
	message.SendMessage(message.OFFLINE, data, result)
	usercontrol.CountExp(uid)
	return
}

func (sm *UserModule) AreaUpdate(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var uid uint32
	var lat, lng float64
	abs, ok := req.EnsureBody("uid", "lat", "lng")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	if v, err := utils.ToUint32(req.Body["uid"]); err == nil {
		uid = v
	}
	if v, err := utils.ToFloat64(req.Body["lat"]); err == nil {
		lat = v
	}
	if v, err := utils.ToFloat64(req.Body["lng"]); err == nil {
		lng = v
	}
	data := make(map[string]interface{})
	data["uid"] = uid
	data["lat"] = lat
	data["lng"] = lng
	message.SendMessage(message.LOCATION_CHANGE, data, result)
	return
}

/*
封禁用户

URL : /user/BanUser

参数：
	uid:[uint32] 封禁用户uid

返回值：
	{
		"status": "ok",
	}

*/
func (sm *UserModule) BanUser(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("uid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	uid, err := utils.ToUint32(req.Body["uid"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "uid")
	}
	if uid < 5000000 {
		return service.NewError(service.ERR_INVALID_PARAM, "不能封停客服号 ", "不能封停客服号 ")
	}
	err = login.BanUser(uid)
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "uid")
	}
	// login.ChangeSid(uid)
	push.Kick(uid)
	dynamics.CloseUserDynamic(uid)
	topic.CloseTopic(0, uid)
	return
}

//解封
func (sm *UserModule) UnBanUser(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("uid")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	uid, err := utils.ToUint32(req.Body["uid"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "uid")
	}
	err = login.UnBanUser(uid)
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "uid")
	}
	return
}

//测试用 获得所有
func (sm *UserModule) GetJQUser(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {

	uid, err := utils.ToUint32(req.GetParam("uid"))

	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "uid")
	}
	index, err := utils.ToUint32(req.GetParam("index"))
	list, err := login.GetJQUser(uid, index)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["list"] = list
	result["res"] = res
	return
}

func (sm *UserModule) AdminLogin(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("username", "password")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	username := utils.ToString(req.Body["username"])
	password := utils.ToString(req.Body["password"])
	admin_id, err := login.AdminLogin(username, password)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["uid"] = admin_id
	result["res"] = res
	return
}

func (sm *UserModule) SecSetUidArea(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	abs, ok := req.EnsureBody("x", "y")
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "need ["+abs+"]")
	}
	lng, err := utils.ToFloat64(req.Body["x"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "x")
	}
	lat, err := utils.ToFloat64(req.Body["y"])
	if err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, "y")
	}
	sm.log.AppendInfo("SecSetUidArea", fmt.Sprintf("%v", req.Body))
	err = login.SetUidArea(req.Uid, lng, lat)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	//发送消息设置
	// data := make(map[string]interface{})
	// data["uid"] = req.Uid
	// data["lat"] = y
	// data["lng"] = x
	// message.LocationChange
	message.SendMessage(message.LOCATION_CHANGE, message.LocationChange{req.Uid, lat, lng}, result)
	sm.log.AppendInfo("LOCATION_CHANGE", fmt.Sprintf("%v", message.LocationChange{req.Uid, lat, lng}))

	lat1, lng1, _ := general.UserLocation(req.Uid)
	sm.log.AppendInfo(fmt.Sprintf("LOCATION UID %v, %v,%v", req.Uid, lat1, lng1))
	return
}

func (sm *UserModule) KickUser(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	uid, _ := utils.ToUint32(req.GetParam("uid"))
	if uid != 0 {
		push.Kick(uid)
	}
	return
}

func (sm *UserModule) ClearIpRegR(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	login.ClearIpReg(req.IP())
	return
}

/*
PhoneArea 手机归属地查询

URI: user/PhoneArea

参数

{
		"phone":"137XXXXXXXX"//手机号
}

返回值

{
		"status":"ok"
		"res":
		{
			"province":"河北",		//省
			"city":"保定",			//市
			"supplier":"移动",		//运营商
		}
}
*/
func (sm *UserModule) PhoneArea(req *service.HttpRequest, result map[string]interface{}) (e service.Error) {
	var phone string
	if err := req.Parse("phone", &phone); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	province, city, supplier, err := general.GetCityByPhone(phone)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	res["province"] = province
	res["city"] = city
	res["supplier"] = supplier
	result["res"] = res
	return
}
