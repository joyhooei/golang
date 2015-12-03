package general

import (
	"fmt"
	"strings"
	"time"
	// "yf_pkg/utils"
)

type province_item struct {
	Name  string   `json:"name"`
	Citys []string `json:"citys"`
}

type trade_item struct {
	Name string   `json:"name"`
	Jobs []string `json:"jobs"`
}

var newver uint32

//省市列表
var province_List []*province_item

//行业职业列表
var trade_List []*trade_item

func loadProvinceList() (e error) {
	sql := "select city,province from city_map where flag=0"
	rows, e := mdb.Query(sql)
	if e != nil {
		fmt.Println("Load Province_List error:", e.Error())
		return
	}
	defer rows.Close()
	tmplist := make([]*province_item, 0, 0)
	for rows.Next() {
		var city, province string
		if err := rows.Scan(&city, &province); err != nil {
			fmt.Println("mysql Scan error:", err.Error())
			return
		}
		var item *province_item
		for _, v := range tmplist {
			if v.Name == province {
				item = v
				break
			}
		}
		if item == nil {
			item = new(province_item)
			item.Name = province
			item.Citys = make([]string, 0, 0)
			tmplist = append(tmplist, item)
		}
		item.Citys = append(item.Citys, city)
	}
	province_List = tmplist
	return
}

func loadTradeList() (e error) {
	sql := "select trade,jobs from trades"
	rows, e := mdb.Query(sql)
	if e != nil {
		fmt.Println("Load Province_List error:", e.Error())
		return
	}
	defer rows.Close()
	tmplist := make([]*trade_item, 0, 0)
	for rows.Next() {
		var trade, jobs string
		if err := rows.Scan(&trade, &jobs); err != nil {
			fmt.Println("mysql Scan error:", err.Error())
			return
		}
		item := new(trade_item)
		item.Name = trade
		item.Jobs = strings.Split(jobs, ",")
		tmplist = append(tmplist, item)

	}
	trade_List = tmplist
	return
}

func loadallset() {
	var tmpver uint32
	e := mdb.QueryRow("select ver from face_ver where `key`='set_ver'").Scan(&tmpver)
	if e != nil {
		return
	}
	// fmt.Println(fmt.Sprintf("loadallset %v", tmpver))
	if tmpver != newver {
		if e := loadProvinceList(); e != nil {
			fmt.Println(fmt.Sprintf("loadProvinceList error %v", e))
			return
		}
		if e := loadTradeList(); e != nil {
			fmt.Println(fmt.Sprintf("loadTradeList error %v", e))
			return
		}
		newver = tmpver

	}
	return
}

func updateset() (e error) {
	for {
		time.Sleep(60 * time.Second)
		loadallset()
	}
}

func initListSet() {
	loadallset()
	go updateset()
	return
}

func GetListSet(ver uint32) (result map[string]interface{}, e error) {
	result = make(map[string]interface{})
	if newver > ver {
		result["newver"] = newver
		result["trades"] = trade_List
		result["provinces"] = province_List
		result["mentags"] = []string{"事业型", "稳重型", "浪漫型", "肌肉男"}
		result["womentags"] = []string{"端庄大气", "高冷", "娇小可爱", "温柔"}
	} else {
		result["newver"] = 0 //没有版本信息
	}

	return
}
