package general

import "yf_pkg/utils"

// 通用配置信息
type Config struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

type AppImg struct {
	Url  string `json:"url"`
	Tm   string `json:"tm"`
	Type int    `json:"type"`
}

// 获取通用配置数据
func GetComConfig() (c_arr []*Config, e error) {
	if ex, rs, e := readConfigCache(); ex && rs != nil && e == nil {
		return rs, nil
	}
	sql := "select `key` , value , type  from sys_config "
	rows, e := mdb.Query(sql)
	if e != nil {
		return
	}
	defer rows.Close()
	c_arr = make([]*Config, 0, 20)
	for rows.Next() {
		c := new(Config)
		if e = rows.Scan(&c.Key, &c.Value, &c.Type); e != nil {
			return
		}
		c_arr = append(c_arr, c)
	}
	if len(c_arr) > 0 {
		if e := writeConfigCache(c_arr); e != nil {
			return nil, e
		}
	}
	return
}

// 分类别处理配置数据
func FormatConfig(c_arr []*Config) map[string]map[string]interface{} {
	m := make(map[string]map[string]interface{})
	if c_arr == nil || len(c_arr) <= 0 {
		return m
	}
	for _, c := range c_arr {
		item := make(map[string]interface{})
		//	c_item := make(map[string]interface{})
		if arr, ok := m[c.Type]; ok {
			item = arr
		}
		item[c.Key] = c.Value
		m[c.Type] = item
	}
	return m
}

// 获取所有的app背景图（注册图）
func GetAppImgs() (r []AppImg, e error) {
	if ex, rs, e := readAppImgCache(); ex && rs != nil && e == nil {
		return rs, nil
	}
	r = make([]AppImg, 0, 10)
	s := "select url,tm,type from app_imgs where status = 0 order by id desc"
	rows, e := mdb.Query(s)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var ai AppImg
		if e = rows.Scan(&ai.Url, &ai.Tm, &ai.Type); e != nil {
			return
		}
		r = append(r, ai)
	}
	if len(r) > 0 {
		if e := writeAppImgCache(r); e != nil {
			return nil, e
		}
	}
	return
}

// 获取所有的app背景图（注册图）
func FormatAppImgs(v []AppImg) (r []map[string]interface{}, e error) {
	if len(v) <= 0 {
		return
	}
	r = make([]map[string]interface{}, 0, len(v))
	for _, ai := range v {
		item := make(map[string]interface{})
		item["url"] = ai.Url
		tm, e := utils.ToTime(ai.Tm)
		if e != nil {
			return r, e
		}
		item["tm"] = tm
		item["type"] = 1
		r = append(r, item)
	}
	return
}
