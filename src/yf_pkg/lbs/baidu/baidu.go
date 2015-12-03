package baidu

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	yf_http "yf_pkg/net/http"
	"yf_pkg/utils"
)

const (
	AK     = "FGV8C3GjFeAGstAa1INmabge"
	APIKEY = "98d278cdc3fdb4e71b1469bdfd144624"
)

var ZXS []string = []string{"北京市", "天津市", "重庆市", "上海市", "澳门特别行政区", "香港特别行政区"}

//是否是直辖市
func IsZXS(province string) bool {
	for _, p := range ZXS {
		if p == province {
			return true
		}
	}
	return false
}

/*
SearchPlace根据关键字搜索附近的实体，关键词之间是或的关系。

参数：
	lat,lng: 中心点，经纬度
	radius: 半径，米
	cur: 请求的页码，从1开始
	ps: 每页的数量，不能超过20
	sortByDistence: 是否按距离排序，如果为false，则根据百度地图的默认排序规则（貌似是根据热度排序）
	keywords: 搜索关键词
*/
func SearchPlace(lat, lng float64, radius, cur, ps int, sortByDistence bool, keywords ...string) (places []Place, total int, e error) {
	if ps > 20 {
		return nil, 0, errors.New("invalid ps")
	}
	q := strings.Join(keywords, "$")
	var body []byte
	if sortByDistence {
		body, e = yf_http.Send("http", "api.map.baidu.com", "place/v2/search", map[string]string{"output": "json", "ak": AK, "scope": "1", "page_num": utils.ToString(cur - 1), "page_size": utils.ToString(ps), "location": fmt.Sprintf("%v,%v", lat, lng), "radius": utils.ToString(radius), "query": q, "filter": "sort_name:distance|sort_rule:1"}, nil, nil, nil, 4)
	} else {
		body, e = yf_http.Send("http", "api.map.baidu.com", "place/v2/search", map[string]string{"output": "json", "ak": AK, "scope": "1", "page_num": utils.ToString(cur - 1), "page_size": utils.ToString(ps), "location": fmt.Sprintf("%v,%v", lat, lng), "radius": utils.ToString(radius), "query": q}, nil, nil, nil, 4)
	}
	if e != nil {
		return nil, 0, e
	}
	var result PlaceResult
	if e := json.Unmarshal(body, &result); e != nil {
		return nil, 0, e
	}
	if result.Status != 0 {
		return nil, 0, errors.New(result.Message)
	}
	return result.Results, result.Total, nil
}

/*
SearchRegion在城市范围内根据关键字搜索实体，关键词之间是或的关系。

参数：
	cur: 请求的页码，从1开始
	ps: 每页的数量，不能超过20
	keywords: 搜索关键词
*/
func SearchCity(city string, cur, ps int, keywords ...string) (places []Place, total int, e error) {
	if ps > 20 {
		return nil, 0, errors.New("invalid ps")
	}
	q := strings.Join(keywords, "$")
	body, e := yf_http.Send("http", "api.map.baidu.com", "place/v2/search", map[string]string{"output": "json", "ak": AK, "scope": "1", "page_num": utils.ToString(cur - 1), "page_size": utils.ToString(ps), "region": city, "query": q}, nil, nil, nil, 4)
	if e != nil {
		return nil, 0, e
	}
	var result PlaceResult
	if e := json.Unmarshal(body, &result); e != nil {
		return nil, 0, e
	}
	if result.Status != 0 {
		return nil, 0, errors.New(result.Message)
	}
	return result.Results, result.Total, nil
}

/*
SearchProvince在省范围内根据关键字搜索实体，关键词之间是或的关系，搜索结果返回省内城市存在匹配结果的数量。

参数：
	province: 省或者“全国”，如果是直辖市则返回空数组
	cur: 请求的页码，从1开始
	ps: 每页的数量，不能超过20
	keywords: 搜索关键词
*/
func SearchProvince(province string, cur, ps int, keywords ...string) (cities []CityNum, total int, e error) {
	if IsZXS(province) {
		return
	}
	if ps > 20 {
		return nil, 0, errors.New("invalid ps")
	}
	q := strings.Join(keywords, "$")
	body, e := yf_http.Send("http", "api.map.baidu.com", "place/v2/search", map[string]string{"output": "json", "ak": AK, "scope": "1", "page_num": utils.ToString(cur - 1), "page_size": utils.ToString(ps), "region": province, "query": q}, nil, nil, nil, 4)
	if e != nil {
		return nil, 0, e
	}
	var result CityNumResult
	if e := json.Unmarshal(body, &result); e != nil {
		return nil, 0, e
	}
	if result.Status != 0 {
		return nil, 0, errors.New(result.Message)
	}
	return result.Results, result.Total, nil
}

/*
SuggestionPlace根据用户输入返回建议的结果。

参数：
	region: 区域，例如"北京市"
	coordinate: 中心点，经纬度(先纬度后经度)，如果传这两个值，则以此坐标为中心返回建议结果。
	keyword: 搜索关键词
*/
func SuggestionPlace(region string, keyword string, coordinate ...float64) (sugestions []Suggestion, e error) {
	body := []byte{}
	if len(coordinate) >= 2 {
		body, e = yf_http.Send("http", "api.map.baidu.com", "place/v2/suggestion", map[string]string{"output": "json", "ak": AK, "region": region, "location": fmt.Sprintf("%v,%v", coordinate[0], coordinate[1]), "query": keyword}, nil, nil, nil, 4)
	} else {
		body, e = yf_http.Send("http", "api.map.baidu.com", "place/v2/suggestion", map[string]string{"output": "json", "ak": AK, "region": region, "query": keyword}, nil, nil, nil, 4)
	}
	if e != nil {
		return nil, e
	}
	var result SuggestionResult
	if e := json.Unmarshal(body, &result); e != nil {
		return nil, e
	}
	if result.Status != 0 {
		return nil, errors.New(result.Message)
	}
	return result.Results, nil
}

//根据IP获取城市
func GetCityByIP(ip string) (province, city string, e error) {
	body, e := yf_http.Send("http", "api.map.baidu.com", "location/ip", map[string]string{"ak": AK, "ip": ip}, nil, nil, nil)
	if e != nil {
		return "", "", e
	}
	var result IPAddressResult
	if e := json.Unmarshal(body, &result); e != nil {
		return "", "", e
	}
	if result.Status != 0 {
		return "", "", errors.New(result.Message)
	}
	return result.Content.AddressDetail.Province, result.Content.AddressDetail.City, nil
}

//根据城市获取GPS
func GetGPSByCity(city string, province string) (pos utils.Coordinate, e error) {
	body, e := yf_http.Send("http", "api.map.baidu.com", "geocoder/v2/", map[string]string{"ak": AK, "output": "json", "address": province + city, "city": city}, nil, nil, nil)
	if e != nil {
		return pos, e
	}
	var result CityGPSResult
	if e := json.Unmarshal(body, &result); e != nil {
		return pos, e
	}
	if result.Status != 0 {
		return pos, errors.New("baidu error :" + result.Message)
	}
	pos.Lat, pos.Lng = result.Result.Location.Lat, result.Result.Location.Lng
	return pos, nil
}

//根据GPS获取城市
func GetCityByGPS(pos utils.Coordinate) (city, province string, e error) {
	body, e := yf_http.Send("http", "api.map.baidu.com", "geocoder/v2/", map[string]string{"ak": AK, "output": "json", "location": fmt.Sprintf("%v,%v", pos.Lat, pos.Lng)}, nil, nil, nil)
	if e != nil {
		return "", "", e
	}
	var result GPSCityResult
	if e := json.Unmarshal(body, &result); e != nil {
		return "", "", e
	}
	if result.Status != 0 {
		return "", "", errors.New(result.Message)
	}
	for _, name := range ZXS {
		if name == result.Result.Address.City {
			return result.Result.Address.Distict, result.Result.Address.City, nil
		}
	}
	if strings.Index(result.Result.Address.City, "直辖县级行政单位") >= 0 {
		return result.Result.Address.Distict, result.Result.Address.Province, nil
	}
	return result.Result.Address.City, result.Result.Address.Province, nil
}

//根据手机号码获取城市
func GetCityByPhone(phone string) (city, province, supplier string, e error) {
	body, e := yf_http.Send("http", "apis.baidu.com", "apistore/mobilephoneservice/mobilephone", map[string]string{"tel": phone}, map[string]string{"apikey": APIKEY}, nil, nil)
	if e != nil {
		return "", "", "", e
	}
	var result CityByPhoneResult
	if e := json.Unmarshal(body, &result); e != nil {
		return "", "", "", e
	}
	if result.ErrNum != 0 {
		return "", "", "", errors.New(result.ErrMsg)
	}

	return result.RetData.City, result.RetData.Province, result.RetData.Supplier, nil
}
