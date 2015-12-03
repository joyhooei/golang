package common

/*
Web页面相关接口文档(php)

WEB交互HOST：  http://test.a.imswing.cn:10080 (测试)   http://www.imswing.cn  (线上)

web与app交互cmd定义：

商品列表URL：/mall/list

商品购买参数格式：
	{
		'cmd':'mall_buy',    // 执行命令
		'tip':'',
		'def':false,
		'data':{
			"id":12
			"content":"'你将消耗1200钻石购买 \“星巴克88元会员卡一张\”', // 提示框文字"
			"gold":1200   // 商品需要钻石数量
		}
 	}

进入详情页参数：
	{
		'cmd':'mall_info',    //打开商品详情
		'tip':'',
		'def':false,
		'data':{
			'url':"http://test.a.imswing.cn:10080/mall/info?id=1"   // 详情页加载url
		}
 	}

打开充值页面：
	{
		'cmd':'cmd_mycharge',    //开发充值页面
		'tip':'',
		'def':false,
		'data':{	}
 	}

约会相关接口：
	咖啡交友页面：http://test.a.imswing.cn:10080/client/date
	详细页面：http://test.a.imswing.cn:10080/client/dateInfo

打开交友页面
	{
		'cmd':'client_date',    //打开商品详情
		'tip':'',
		'def':false,
		'data':{
			'url':"http://test.a.imswing.cn:10080/mall/info?id=1"   // 详情页加载url
		}
 	}
打开详情页面：
	{
		'cmd':'client_nav',    //导航
		'tip':'',
		'def':false,
		'data':{
		}
 	}





*/
const API_DOC = "其他接口说明文档"
