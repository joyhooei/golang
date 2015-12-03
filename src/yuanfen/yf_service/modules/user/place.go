package user

import (
	"yf_pkg/lbs/baidu"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/data_model/discovery"
	"yuanfen/yf_service/cls/data_model/user_overview"
	"yuanfen/yf_service/cls/data_model/usercontrol"
)

/*
默认相关位置 (进入页面时调用)

请求URL：user/AutoPlace

参数:

{
	"lat":1.1,//纬度
	"lng":1.1,//经度
	"type":1//[opt]类型 1为工作单位 2为不限制类型 0为写字楼 默认为0
	"cur":1,	//页码
	"ps":10,	//每页条数
}

返回结果：

{
	"status": "ok",
	"res":{
		"places":{
			"list":[
				{
					"name":"翠微大厦",//位置名
					"uid":"11122233",//位置id
					"Address":"海淀区八达岭高速"//地址
					"location":{
						"lat":1.1,//地点纬度
						"lng":1.1//地点经度
					}
				},{...}
			],
			"pages": {
				"cur": 1,
				"total": 1,
				"ps": 2,
				"pn": 1
			}
		}
	}
}
*/
func (sm *UserModule) AutoPlace(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var lat, lng float64
	var cur, ps int
	searchstr := []string{}
	if e = req.Parse("lat", &lat, "lng", &lng, "cur", &cur, "ps", &ps); e != nil {
		return
	}
	var tp int
	if e = req.Parse("type", &tp); e != nil {
		tp = 0
	}
	switch tp {
	case 0:
		searchstr = append(searchstr, "写字楼")
	case 1:
		searchstr = append(searchstr, "公司")
		searchstr = append(searchstr, "单位")
	case 2:
		searchstr = append(searchstr, "写字楼")
		searchstr = append(searchstr, "公司")
		searchstr = append(searchstr, "工厂")
		searchstr = append(searchstr, "商铺")
		searchstr = append(searchstr, "小区")
		searchstr = append(searchstr, "饭店")
		searchstr = append(searchstr, "景点")
		searchstr = append(searchstr, "学校")
	}
	list, total, e := baidu.SearchPlace(lat, lng, 1000, cur, ps, false, searchstr...)
	if e != nil {
		return e
	}
	pages := utils.PageInfo(total, cur, ps)
	res := make(map[string]interface{})
	places := make(map[string]interface{})
	places["list"] = list
	places["pages"] = pages
	res["places"] = places
	result["res"] = res
	return
}

/*
搜索相关位置 (填完点搜索时调用)

请求URL：user/SearchPlace

参数:

{
	"lat":1.1,//纬度
	"lng":1.1,//经度
	"keyword":"北京中关村",//输入的关键词
	"cur":1,	//页码
	"ps":10,	//每页条数
}

返回结果：

{
	"status": "ok",
	"res":{
		"places":{
			"list":[
				{
					"name":"翠微大厦",//位置名
					"uid":"11122233",//位置id
					"Address":"海淀区八达岭高速"//地址
					"location":{
						"lat":1.1,//地点纬度
						"lng":1.1//地点经度
					}
				},{...}
			],
			"pages": {
				"cur": 1,
				"total": 1,
				"ps": 2,
				"pn": 1
			}
		}
	}
}
*/
func (sm *UserModule) SearchPlace(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var lat, lng float64
	var cur, ps int
	var keyword string
	if e = req.Parse("lat", &lat, "lng", &lng, "cur", &cur, "ps", &ps, "keyword", &keyword); e != nil {
		return
	}
	list, total, e := baidu.SearchPlace(lat, lng, 50000, cur, ps, false, keyword)
	if e != nil {
		return e
	}
	pages := utils.PageInfo(total, cur, ps)
	res := make(map[string]interface{})
	places := make(map[string]interface{})
	places["list"] = list
	places["pages"] = pages
	res["places"] = places
	result["res"] = res
	return
}

/*
"uid":"111",
				"location":{
					"lat":1.1,
					"lng":1.1
				}
				"city":"北京市"
				"cityid":""

*/

/*
用户输入字符后的位置推荐

请求URL：user/SuggestionPlace

参数:

{
	"lat":1.1,//纬度
	"lng":1.1,//经度
	"province":1.1,//省
	"keyword":1.1,//输入的关键词
}

返回结果：

{
	"status": "ok",
	"res":{
		"sugestions":[
			{
				"name":"翠微大厦",//位置名
				"district":""//地点名
				"location":{
					"lat":1.1,//地点纬度
					"lng":1.1//地点经度
				}
				"city":"城市",//
				"cityid":""//
			},{...}
		]
	}
}
*/
func (sm *UserModule) SuggestionPlace(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var lat, lng float64
	var keyword, province string
	if e = req.Parse("lat", &lat, "lng", &lng, "province", &province, "keyword", &keyword); e != nil {
		return
	}
	sugestions, e := baidu.SuggestionPlace(province, keyword, lat, lng)
	res := make(map[string]interface{})
	res["sugestions"] = sugestions
	result["res"] = res
	return
}

/*
SecSetWorkArea 设置工作的写字楼

URI: s/user/SetWorkArea

如果用户直接输入的地址 则name为输入值 placeid为空字符串 其余为空字符串或者0

如果用户通过API点选的地址 则需要填写其他响应值

参数
{
	"name":"翠微大厦",//工作地点名 对应 SearchPlace返回中name
	"placeid":"ef"//工作地的百度地图id 对应 SearchPlace返回中的uid
	"Address":"清河桥西1号楼",//详细地址 对应 SearchPlace返回中Address
	"lat":1.1111,
	"lng":1.1111
}

返回值
{
		"status":"ok"
}
*/
func (sm *UserModule) SecSetWorkArea(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var lat, lng float64
	var address, placeid, name string
	if e = req.Parse("lat", &lat, "lng", &lng, "name", &name, "placeid", &placeid, "address", &address); e != nil {
		return
	}
	e = usercontrol.SetWorkArea(req.Uid, placeid, name, address, lat, lng)
	if e != nil {
		return e
	}
	user_overview.ClearUserObjects(req.Uid)
	e = discovery.UpdateDiscovery(req.Uid, "workplace", placeid)
	usercontrol.CheckVerify(req.Uid, "")
	return
}

/*
SecSetWorkunit 设置工作的公司

URI: s/user/SetWorkunit

如果用户直接输入的地址 则name为输入值 placeid为空字符串 其余为空字符串或者0

如果用户通过API点选的地址 则需要填写其他响应值

参数
{
	"name":"翠微大厦",//工作地点名 对应 SearchPlace返回中name
	"placeid":"ef"//工作地的百度地图id 对应 SearchPlace返回中的uid
	"address":"清河桥西1号楼",//详细地址 对应 SearchPlace返回中Address
	"lat":1.1111,
	"lng":1.1111
}

返回值
{
		"status":"ok"
}
*/
func (sm *UserModule) SecSetWorkunit(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var lat, lng float64
	var address, placeid, name string
	if e = req.Parse("lat", &lat, "lng", &lng, "name", &name, "placeid", &placeid, "address", &address); e != nil {
		return
	}
	e = usercontrol.SetWorkunit(req.Uid, placeid, name, address, lat, lng)
	if e != nil {
		return e
	}
	user_overview.ClearUserObjects(req.Uid)
	usercontrol.CheckVerify(req.Uid, "")
	return
}
