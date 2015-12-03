package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"yf_pkg/lbs/baidu"
)

type Province struct {
	Name   string   `json:"name"`
	Cities []string `json:"cities"`
}

type Provinces struct {
	Data []Province `json:"data"`
}

var provinceMap map[string]string = map[string]string{"新疆": "新疆维吾尔自治区", "北京": "北京市", "天津": "天津市", "上海": "上海市", "重庆": "重庆市", "广西": "广西壮族自治区", "西藏": "西藏自治区", "宁夏": "宁夏回族自治区", "香港": "香港特别行政区", "澳门": "澳门特别行政区"}

func main() {
	file, e := os.Open(os.Args[1])
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	info, e := file.Stat()
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer file.Close()
	data := make([]byte, info.Size())
	n, e := file.Read(data)
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	if int64(n) < info.Size() {
		fmt.Sprintf("cannot read %v bytes from %v", info.Size(), os.Args[1])
		return
	}

	var ps Provinces
	e = json.Unmarshal(data, &ps)
	if e != nil {
		fmt.Println(e.Error())
		return
	}

	for _, province := range ps.Data {
		for _, city := range province.Cities {
			pname, ok := provinceMap[province.Name]
			if !ok {
				pname = province.Name + "省"
			}
			gps, _ := baidu.GetGPSByCity(city, pname)
			bCity, _, _ := baidu.GetCityByGPS(gps)
			//fmt.Printf("%s[%s]\t%s[%s]\t<%v,%v>\n", province.Name, bProvince, city, bCity, gps.Lat, gps.Lng)
			//fmt.Printf("insert into city_map(city,province,lat,lng,bCity,bProvince)values('%v','%v',%v,%v,'%v','%v');\n", city, province.Name, gps.Lat, gps.Lng, bCity, bProvince)
			fmt.Printf("update city_map set bCity='%v' where city='%v' and flag=1;\n", bCity, city)
			time.Sleep(100 * time.Millisecond)
		}
	}
}
