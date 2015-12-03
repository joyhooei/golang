package utils

import "math"

const (
	LAT_1_SEC    = 30.87  //纬度1秒的长度（单位米），精确值
	LNG_1_SEC    = 25.0   //经度1秒的长度（单位米），仅为中国地区的平均值
	LNG_1_DEGREE = 90000  //经度1度的距离（米）仅限中国地区
	LAT_1_DEGREE = 111000 //纬度1度的距离(米)
)

const EARTH_RADIUS = 6378.137 //地球半径

type Coordinate struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

//把GPS给出的经纬度转换成秒
func GPSToSecond(latOrLng float64) int {
	return int(latOrLng * 3600)
}

//把公里数转换成纬度度数
func KmToLat(km float64) float64 {
	return km * 1000 / LAT_1_SEC / 3600
}

//把公里数转换成经度度数
func KmToLng(km float64) float64 {
	return km * 1000 / LNG_1_SEC / 3600
}

//两点之间大约的距离(米)
func Distence(p1 Coordinate, p2 Coordinate) float64 {
	radLat1 := rad(p1.Lat)
	radLat2 := rad(p2.Lat)
	a := radLat1 - radLat2
	b := rad(p1.Lng) - rad(p2.Lng)

	return 2 * math.Asin(math.Sqrt(math.Pow(math.Sin(a/2), 2)+math.Cos(radLat1)*math.Cos(radLat2)*math.Pow(math.Sin(b/2), 2))) * EARTH_RADIUS
}

func rad(d float64) float64 {
	return d * math.Pi / 180.0
}
