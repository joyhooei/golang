package comconfig

import (
	"errors"
	"strings"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/common/stat"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/unread"
)

// 一些不涉及到业务逻辑的对外接口
type CommonModule struct {
	log   *log.MLogger
	mdb   *mysql.MysqlDB
	rdb   *redis.RedisPool
	cache *redis.RedisPool
	mode  string
}

func (co *CommonModule) Init(env *service.Env) (err error) {
	co.log = env.Log
	co.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	co.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	co.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
	co.mode = env.ModuleEnv.(*cls.CustomEnv).Mode
	return
}

// 获取通用配置
func (co *CommonModule) Config(req *service.HttpRequest, result map[string]interface{}) (e error) {
	c_arr, e := general.GetComConfig()
	if e != nil {
		return e
	}
	res := general.FormatConfig(c_arr)
	result["res"] = res
	return
}

// 清楚游戏相关缓存key
func (co *CommonModule) ClearCache(req *service.HttpRequest, result map[string]interface{}) (e error) {
	conn := co.cache.GetWriteConnection(redis_db.CACHE_GAME)
	defer conn.Close()
	key_arr := []string{"game_list", "gamedata_list", "award_config", "area_mem", "plane_type_info", "com_config", "award_ratio", "plane_props", "game_info", "game_area", "plane_award_record", "plane_award_rank", "game_acts", "game_actaward_list1", "game_actaward_list2", "game_newactaward_list1", "game_newactaward_list2", "game_user_cnt_1", "honesty_config", "app_img"}
	for _, key := range key_arr {
		if e := conn.Send("DEL", key); e != nil {
			return e
		}
	}
	conn.Flush()
	for _, _ = range key_arr {
		if _, err := conn.Receive(); err != nil {
			return err
		}
	}
	return
}

// 清楚动态相关缓存key
func (co *CommonModule) ClearCacheByDb(req *service.HttpRequest, result map[string]interface{}) (e error) {
	db_str := req.GetParam("db")
	db, e := utils.ToInt(db_str)
	if e != nil {
		return
	}
	conn := co.cache.GetWriteConnection(db)
	defer conn.Close()
	_, e = conn.Do("flushdb")
	return
}

// 清楚动态相关缓存key
func (co *CommonModule) ClearCacheByDbKey(req *service.HttpRequest, result map[string]interface{}) (e error) {
	db_str := req.GetParam("db")
	key := req.GetParam("key")
	db, e := utils.ToInt(db_str)
	if e != nil {
		return
	}
	e = co.cache.Del(db, key)
	return
}

// 【编辑后台】清除升级版本缓存
func (co *CommonModule) SecClearVersionCache(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	conn := co.cache.GetWriteConnection(redis_db.CACHE_VERSION)
	defer conn.Close()
	_, e = conn.Do("FLUSHDB")
	return
}

// 重新加载管理员集合
func (co *CommonModule) SecReloadAdmin(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	return general.UpdateAdmin()
}

// 发送日志
func (co *CommonModule) AppendLog(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var platform, action int
	var devid uint32
	var channel, sub_channel, version string
	var data map[string]interface{}
	if e := req.Parse("action", &action, "ver", &version, "platform", &platform, "devid", &devid); e != nil {
		return e
	}
	if e := req.ParseOpt("channel", &channel, "未知", "sub_channel", &sub_channel, "未知", "data", &data, map[string]interface{}{}); e != nil {
		return e
	}
	province, city, e := general.QueryIpInfo(req.IP())
	if e != nil {
		co.log.AppendObj(nil, "------", action, e.Error())
		return
	}
	co.log.AppendObj(nil, "---appednloag---", action, devid)
	stat.AppendDev(devid, action, data, platform, version, province, city, channel, sub_channel)
	return
}

/*
图片检测（色情+是否人物）

URL: /common/CheckPic

参数：
	url: 待检测图片
	type: 自动检测组合类型，0 未检测任何类型，1 色情+广告 2 色情+是否人物 待扩展
返回值：
	{
	 "res": {
		   "url": "http://image2.yuanfenba.net/uploads/oss/avatar/201502/14/16161586502.jpg",
		   "status": 0  //图片检查状态 -1 待处理,0 正常 1 不正常 2 待扩展
	   },
	 "status": "ok",
	 "tm": 1440490482
	}

php上传的接口：

URL：http://test.upload.imswing.cn:10080/Index  (测试) http://upload.imswing.cn/Index （线上）

提交方式：post

参数：
	 file_name 文件字段名
	 type：
	 	图片类：avatar 头像  avatar_big 大头像  photo 相册图片  chat 聊天图片  video_certify 视屏认证图片 dynamic 动态图片  其他：other
		语音类：voice 语音
	 uid: 用户uid
返回值：
	{
		status: "ok",
		msg: "上传成功",
		code: "00",  // 00 表示成功，否则失败
		res: {
			file_s_path: "oss/avatar/201508/25/15241246891.png",
			file_http_path: "http://image1.yuanfenba.net/uploads/oss/avatar/201508/25/15241246891.png"  // 图片地址
		}
	}
*/
func (co *CommonModule) CheckPic(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var url string
	var t int
	if e := req.Parse("url", &url, "type", &t); e != nil {
		return e
	}
	if url == "" {
		return service.NewError(service.ERR_INVALID_PARAM, "param is error", "参数错误")
	}
	if !(t == general.IMGCHECK_SEXY_AND_HUMAN || t == general.IMGCHECK_SEXY_AND_AD) {
		return service.NewError(service.ERR_INVALID_PARAM, "param is error", "参数错误")
	}
	m, e := general.CheckImg(t, url)
	if e != nil {
		return e
	}
	if v, ok := m[url]; ok {
		result["res"] = v
	}
	return
}

/*
版本升级接口

URL: /common/IosVersion

参数:
	ver: 当前软件版本

返回值:
	{
		"res":{
			"is_force": "1",  // 是否强制升级  1 强制 0 非强制
			"ver": "13"        // 升级包版本
	 	},
		"status": "ok",
		"tm": 1441598919
	}
*/
func (co *CommonModule) IosVersion(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var ver string
	if e := req.Parse("ver", &ver); e != nil {
		return e
	}
	c_uid := "10000"
	c_sid := "10000"
	// 如果是测试环境，则不设置升级
	if co.mode == cls.MODE_TEST {
		//		result["res"] = make(map[string]interface{})
		//		return
	}
	version, e := general.CheckUpdate(c_uid, c_sid, ver, 1)
	if e != nil {
		return e
	}
	// 无可升级
	rs := make(map[string]interface{})
	if version.Ver <= 0 {
		result["res"] = rs
		return
	}
	rs["ver"] = version.Ver
	rs["is_force"] = version.IsForce
	result["res"] = rs
	return
}

/*
版本升级接口

URL: /common/Version

参数:
	ver: 当前软件版本
	c_uid: 当前用户渠道（主）
	c_sid: 当前用户渠道（子）
	update_ver: 用户修改版本

返回值:
	{
		"res":{
			"is_force": "1",  // 是否强制升级  1 强制 0 非强制
			"summary":[       // 升级描述
		            "版本：1.012",
					"1、修复了已知bug",
					"2、服务器部分功能调整",
					"3、注册功能优化以及启动页面更换"
		 	],
			"title": "v1.012", // 版本描述
			"url": "http://down.xingyuan01.cn/mumu/MuMu_2_888_13_1.012.apk",  // 安装包下载地址
			"ver": "13",        // 升级包版本
			"size": "5.8M",     // 包的大小
	 	},
		"status": "ok",
		"tm": 1441598919
	}
*/
func (co *CommonModule) Version(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var c_uid, c_sid, ver, update_ver string
	if e := req.Parse("ver", &ver, "c_sid", &c_sid, "c_uid", &c_uid); e != nil {
		return e
	}
	update_ver = utils.ToString(req.Body["update_ver"])
	// 如果是测试环境，则不设置升级
	if co.mode == cls.MODE_TEST {
		result["res"] = make(map[string]interface{})
		return
	}
	version, e := general.CheckUpdate(c_uid, c_sid, ver, 0)
	if e != nil {
		return e
	}
	// 无可升级
	rs := make(map[string]interface{})
	if version.Ver <= 0 {
		result["res"] = rs
		return
	}
	down_base_url := "http://imswing.xingyuan01.cn/imswing/"
	arr := strings.Split(version.Summary, "\n")
	url := down_base_url + "/Swing_" + c_uid + "_" + c_sid + "_" + utils.ToString(version.Ver) + "_" + version.Title + ".apk"
	rs["summary"] = arr
	rs["title"] = "v" + version.Title
	rs["url"] = url
	rs["ver"] = version.Ver
	rs["size"] = version.Size
	rs["is_force"] = version.IsForce
	uidc, _ := req.Cookie("uid")
	var uid uint32
	if uidc != nil {
		uid, _ = utils.StringToUint32(uidc.Value)
	}
	// 更新用户的当前版本
	if update_ver != "" && uid > 0 {
		s := "update user_main set ver_current = ? where uid = ? "
		co.mdb.Exec(s, update_ver, uid)
	}
	if rs != nil {
		// 更新未读消息
		if uid > 0 {
			unread.UpdateReadTime(uid, common.UNREAD_VERSION)
		}
		unread_m := map[string]interface{}{"num": 0, "show": ""}
		result["unread"] = map[string]interface{}{common.UNREAD_VERSION: unread_m}
	}
	result["res"] = rs
	return
}

// 主动推送版本更新消息
func (co *CommonModule) PushUpdate(req *service.HttpRequest, result map[string]interface{}) (e error) {
	msg := make(map[string]interface{})
	msg["type"] = common.MSG_TYPE_VERSION

	s := "select uid from user_online where tm > ? and uid > 5000000"
	rows, e := co.mdb.Query(s, utils.Now)
	if e != nil {
		return e
	}
	defer rows.Close()
	uids := make([]uint32, 0, 100)
	for rows.Next() {
		var id uint32
		if e := rows.Scan(&id); e != nil {
			return e
		}
		uids = append(uids, id)
	}
	for _, id := range uids {
		msgid, e := general.SendMsg(common.USER_SYSTEM, id, msg, "")
		co.log.AppendObj(e, "--push update --", id, msgid)
	}
	return
}

// 发送日志(注册统计日志)
func (co *CommonModule) AppendErrLog(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var devid uint32
	var channel, sub_channel, version, content string
	devid_s := ""
	if c, e := req.Cookie("devid"); e == nil {
		devid_s = c.Value
	}
	if v, e := utils.ToUint32(devid_s); e == nil {
		devid = v
	}
	if e := req.Parse("ver", &version, "content", &content); e != nil {
		co.log.AppendObj(e, "AppendErrLog----1-")
	}
	if e := req.ParseOpt("channel", &channel, "未知", "sub_channel", &sub_channel, "未知"); e != nil {
		co.log.AppendObj(e, "AppendErrLog----2-")
	}
	co.log.AppendObj(errors.New(""), "devid_s", devid_s, "devid: ", devid, "channel: ", channel, "sub_channel: ", sub_channel, "version:", version, "content", content)
	return
}

/*
渠道通用下载包获取地址，主要为了获取最新下载包

参数：
	c_uid：主渠道
	c_sid：子渠道
*/
func (co *CommonModule) Download(req *service.HttpRequest, result map[string]interface{}) (e error) {
	down_base_url := "http://imswing.xingyuan01.cn/imswing/"
	var c_uid, c_sid string
	c_uid = req.GetParam("c_uid")
	c_sid = req.GetParam("c_sid")
	url := down_base_url + "Swing_2_888.apk"
	if c_uid != "" && c_sid != "" {
		version, e := general.CheckUpdate(c_uid, c_sid, "1", 0)
		if e == nil {
			url = down_base_url + "Swing_" + c_uid + "_" + c_sid + "_" + utils.ToString(version.Ver) + "_" + version.Title + ".apk"
		} else {
			co.log.AppendObj(e, version)
		}
	}
	co.log.AppendObj(e, "Download : ", url, c_uid, c_sid)
	result[service.SERVER_REDIRECT_KEY] = url
	return service.NewError(service.SERVER_REDIRECT, "跳转302")
}

/*
获取app背景图片

URL: common/AppImg

返回值：
	{
		"res":{
			"reg_img": "http://image2.yuanfenba.net/uploads/oss/photo/201506/01/11135012964.jpg",  // 注册界面图片URL
			"screen_img": "http://image2.yuanfenba.net/uploads/oss/photo/201506/01/11135012964.jpg"  //闪屏界面图片URL
			}
		"status": "ok",
		"tm": 1440490482
	}
*/
func (co *CommonModule) AppImg(req *service.HttpRequest, result map[string]interface{}) (e error) {
	arr, e := general.GetAppImgs()
	if e != nil {
		return e
	}
	var reg_img, screen_img string
	for _, ai := range arr {
		if reg_img != "" && screen_img != "" {
			break
		}
		if ai.Type == 1 {
			reg_img = ai.Url
		} else if ai.Type == 2 {
			screen_img = ai.Url
		}
	}
	res := make(map[string]interface{})
	res["reg_img"] = reg_img
	res["screen_img"] = screen_img
	result["res"] = res
	return
}

/*
获取列表项

URL: common/ListSet

参数：
	{
		"ver":0	//版本号 初始填写0
	}

返回值：
	{
		"res": [
		{
			"newver": 1001,//新版本号 如果为 0 表示无更新
			"trades":[
				{
					"name":"IT业"
					"jobs"["程序员","设计师"]
				},{...}
			],
			"provinces":[
				{
					"name":"湖南"
					"citys"["长沙","衡阳"]
				},{...}
			],
			"mentags":["进取型","浪漫型"],
			"womentags":["白富美","温柔型"]
		}
		"status": "ok",
	}
*/
func (co *CommonModule) ListSet(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var ver uint32
	if e := req.Parse("ver", &ver); e != nil {
		return e
	}
	res, e := general.GetListSet(ver)
	if e != nil {
		return e
	}
	result["res"] = res
	return
}

func (co *CommonModule) QueryCode(req *service.HttpRequest, result map[string]interface{}) (e error) {
	phone := req.GetParam("phone")
	v, e := redis.String(co.rdb.Get(redis_db.REDIS_PHONE_CODE, phone))
	result["res"] = map[string]interface{}{"code": v}
	return
}

func (co *CommonModule) InsertSort(req *service.HttpRequest, result map[string]interface{}) (e error) {
	list := []int{4, 1, 2, 5, 3}
	var temp int
	var i int
	var j int
	// 第1个数肯定是有序的，从第2个数开始遍历，依次插入有序序列
	for i = 1; i < len(list); i++ {
		temp = list[i] // 取出第i个数，和前i-1个数比较后，插入合适位置
		// 因为前i-1个数都是从小到大的有序序列，所以只要当前比较的数(list[j])比temp大，就把这个数后移一位
		for j = i - 1; j >= 0 && temp < list[j]; j-- {
			co.log.AppendObj(nil, "===i:", i, "j: ", j, "temp:", temp, "list: ", list)
			list[j+1] = list[j]
			co.log.AppendObj(nil, "i:", i, "j: ", j, "temp:", temp, "list: ", list)
		}
		list[j+1] = temp
		co.log.AppendObj(nil, "--end---i:", i, "j: ", j, "temp:", temp, "list: ", list)
	}
	return
}
