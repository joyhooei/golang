package general

import (
	"encoding/json"
	"strings"
	"yf_pkg/net/http"
	"yf_pkg/utils"
)

type ImgResult struct {
	Url    string `json:"url"`
	Status int    `json:"status"` //图片检查状态 -1 待处理,0 正常 1 不正常 2 待扩展
}

/*
图片md5
*/
type ImgMd5 struct {
	Url         string  // 图片地址
	Md5         string  // 图片md5
	SexyRate    float64 // 色情评级可靠度
	SexyFlag    int     // 色情等级， 0：色情； 1：性感； 2：正常；
	SexyReview  int     // 是否需要人工复审 0 无 1 需要
	AdRate      float64 // 广告识别概率
	AdFlag      int     // 广告等级 0：正常； 1：二维码； 2：带文字图片；
	AdReview    int     // 是否需要人工复审 0 无 1 需要
	HumanRate   float64 // 人物识别概率
	HumanFlag   int     // 是否人物， 0：男人； 1：女人； 2：其他； 3：多人；
	HumanReview int     // 是否需要人工复审 0 无 1 需要
	Type        string  // 图片类型
	Status      int     // 图片检查状态 -1 待处理,0 正常 1 不正常 2 待扩展
	Item        int     // 自动检测组合类型，0 未检测任何类型，1 色情+广告 2 色情+是否人物 待扩展
}

type ImgRes struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Res  []ImgResult `json:"res"`
}

const (
	IMGCHECK_SEXY_AND_AD    = 1 // 色情+广告
	IMGCHECK_SEXY_AND_HUMAN = 2 //色情+是否人物
)

// 单张图片做审核
func CheckImgByUrl(item int, url string) (r ImgResult, e error) {
	r = ImgResult{Url: url, Status: 1}
	rm, e := CheckImg(item, url)
	mlog.AppendObj(e, "---CheckImg ---", rm)
	if e != nil {
		return
	}
	r, _ = rm[url]
	return
}

/*
添加图片md5

参数：
	item:自动检测组合类型，0 未检测任何类型，1 色情+广告 2 色情+是否人物 待扩展
	imgs:检测图片,可多张，但不超过10张(避免请求时间过长)
返回值：
	res: map[string]ImgResult  key 为图片地址,value 为ImgResult 对象
	具体结构见 http://120.131.64.91:8182/pkg/yuanfen/yf_service/cls/data_model/general/#ImgResult
*/
func CheckImg(item int, urls ...string) (res map[string]ImgResult, e error) {
	if len(urls) <= 0 {
		return
	}
	if item <= 0 {
		return
	}
	images := make([]string, 0, len(urls))
	urlMap := make(map[string]string) // 保存url和原始url的对应关系
	// 去掉url中压缩的后缀代码
	for _, url := range urls {
		origin_url := getOriginImg(url)
		images = append(images, origin_url)
		urlMap[origin_url] = url
	}
	// 检测自身库，是否已经有该图片检测结果
	db_res, e := checkInDb(item, images)
	mlog.AppendObj(e, "chekc in db :", images, db_res)
	if e != nil {
		return
	}
	db_images := make([]string, 0, len(db_res))
	if len(db_res) > 0 {
		for _, r := range db_res {
			db_images = append(db_images, r.Url)
		}
	}
	// 去除已经检测过的图片,如果还有图片未检测，需要向第三方发送请求
	check_images := utils.StringArrDiff(images, db_images)
	//	mlog.AppendObj(nil, "all:  ", images, "db:  ", db_images, " check: ", check_images)
	// 第三方请求
	check_res, e := doCheck(item, check_images)
	mlog.AppendObj(e, "doCheck rs :", check_images, check_res)
	if len(check_res) > 0 {
		db_res = append(db_res, check_res...)
	}
	res = make(map[string]ImgResult)
	if len(db_res) > 0 {
		for _, v := range db_res {
			n_url := v.Url
			if u, ok := urlMap[n_url]; ok {
				v.Url = u
			}
			res[v.Url] = v
		}
	}
	mlog.AppendObj(nil, " final res :  ", images, res)
	return
}

//先从自己数据库中查看数据
func checkInDb(item int, images []string) (res []ImgResult, e error) {
	if len(images) <= 0 {
		return
	}
	res = make([]ImgResult, 0, len(images))
	var q string
	for _, url := range images {
		q += "\"" + url + "\","
	}
	q = utils.SubString(q, 0, len(q)-1)
	// 查询这些照片中是否已经存在相同的md5，如果有相同的存在，则判断是否合法
	s := "select url,md5 from image_md5 where url in (" + q + ") "
	rows, e := mdb.Query(s)
	if e != nil {
		return
	}
	defer rows.Close()
	var md5_str string
	md5Map := make(map[string]string)
	for rows.Next() {
		var img ImgMd5
		if e = rows.Scan(&img.Url, &img.Md5); e != nil {
			return
		}
		md5Map[img.Md5] = img.Url
		md5_str += "\"" + img.Md5 + "\","
	}
	if md5_str == "" || len(md5_str) <= 0 {
		return
	}
	md5_str = utils.SubString(md5_str, 0, len(md5_str)-1)

	s2 := "select url,md5,sexy_rate,sexy_flag,sexy_review,ad_rate,ad_flag,ad_review,type,status,human_rate,human_flag,human_review,item from image_md5 where md5 in (" + md5_str + " )  and status >=0 and item =? group by md5"
	rows2, e := mdb.Query(s2, item)
	if e != nil {
		return
	}
	defer rows2.Close()
	// 获取已经有的结论的图片
	ok_arr := make([]ImgMd5, 0, len(md5_str))
	for rows2.Next() {
		var i ImgMd5
		if e = rows2.Scan(&i.Url, &i.Md5, &i.SexyRate, &i.SexyFlag, &i.SexyReview, &i.AdRate, &i.AdFlag, &i.AdReview, &i.Type, &i.Status, &i.HumanRate, &i.HumanFlag, &i.HumanReview, &i.Item); e != nil {
			return
		}
		ok_arr = append(ok_arr, i)
	}
	if len(ok_arr) > 0 {
		// 将数据保存在新的url中
		s3 := "update image_md5 set sexy_rate=?,sexy_flag=?,sexy_review=?,ad_rate=?,ad_flag=?,ad_review=?,status=?,human_rate=?,human_flag=?,human_review=?,item=? where url = ?"
		st, e2 := mdb.PrepareExec(s3)
		if e2 != nil {
			return res, e2
		}
		defer st.Close()
		for _, v := range ok_arr {
			var q_url string
			if v, ok := md5Map[v.Md5]; ok {
				q_url = v
			}
			var r ImgResult
			r.Url = q_url
			r.Status = v.Status
			res = append(res, r)
			if _, e = st.Exec(v.SexyRate, v.SexyFlag, v.SexyReview, v.AdRate, v.AdFlag, v.AdReview, v.Status, v.HumanRate, v.HumanFlag, v.HumanReview, v.Item, q_url); e != nil {
				return
			}
		}
	}
	//mlog.AppendObj(nil, "-checkInDb-", res)
	return
}

// 发送图片检测请求
func doCheck(item int, images []string) (res []ImgResult, e error) {
	if len(images) <= 0 {
		return
	}
	//	mlog.AppendObj(nil, "-----doCheck-1-------", images)
	data := make(map[string]string)
	data["type"] = utils.ToString(item)
	data["imgs"] = strings.Join(images, ",")
	r, e := http.HttpGet(upload_service_url, "/Index/checkImg", data, 15)
	if e != nil {
		Alert("img", "checkimg is error")
		mlog.AppendObj(e, "--------doCheck is error--", string(r))
		return
	}
	mlog.AppendObj(nil, "-----doCheck-2-------", upload_service_url, string(r))
	var ir ImgRes
	ir.Code = 1
	if e = json.Unmarshal(r, &ir); e != nil {
		mlog.AppendObj(e, "-----doCheck-2-1-error-json-----", (r))
		return
	}
	//	mlog.AppendObj(nil, "-----doCheck-3-------", ir)
	if ir.Code == 0 {
		res = ir.Res
		//	mlog.AppendObj(nil, "-----doCheck----ok-------", ir)
	}
	//	mlog.AppendObj(nil, "-----doCheck-4-------", res)
	return
}

/*
根据url，换取原始图片地址
*/
func getOriginImg(url string) (origin string) {
	index := strings.LastIndex(url, "@")
	origin = url
	if index > 0 {
		origin = utils.SubString(url, 0, index)
	}
	return
}

/*
删除因图片审核失败的图片消息
msgid:消息id
from:发送方uid
url:图片地址
t: 群聊1 私聊 0
*/
func DeleBadPicMessage(msgid uint64, from uint32, url string, t int) (e error) {
	var s string
	if t == 0 {
		s = "delete from message where id = ?"
	} else if t == 1 {
		s = "delete from tag_message where id = ?"
	}
	mlog.AppendObj(nil, "---", s, msgid, t)
	if s == "" {
		return
	}
	_, e = msgdb.Exec(s, msgid)
	if e != nil {
		mlog.AppendObj(e, "DeleBadPicMessage--1 is error", msgid)
		return e
	}
	s2 := "insert into bad_message(id,`from`,origin,replaced,num)values(?,?,?,?,?)"
	if _, e = msgdb.Exec(s2, msgid, from, url, "", 0); e != nil {
		mlog.AppendObj(e, "add to bad_message table error")
	}
	return
}
