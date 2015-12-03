package main

import (
	"pkg/yh_utils"
	"strconv"
	"strings"
)

type Pos struct {
	id       string
	lat      float64
	lng      float64
	distance float64
}

type PosHeap []Pos

func (h PosHeap) Len() int           { return len(h) }
func (h PosHeap) Less(i, j int) bool { return h[i].distance < h[j].distance }
func (h PosHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *PosHeap) Push(x interface{}) {
	*h = append(*h, x.(Pos))
}

func (h *PosHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func (p *Pos) Serialize() string {
	return yh_utils.Float64ToString(p.lat) + "," + yh_utils.Float64ToString(p.lng)
}
func UnSerialize(v string) (pos Pos, err error) {
	items := strings.Split(v, ",")
	if len(items) >= 2 {
		tmp, err := yh_utils.StringToFloat64(items[0])
		if err != nil {
			return pos, err
		}
		pos.lat = tmp
		tmp, err = yh_utils.StringToFloat64(items[1])
		if err != nil {
			return pos, err
		}
		pos.lng = tmp
	}
	return pos, nil
}

func Key(sex string, lat float64, lng float64, factor int) string {
	return sex + "_" + strconv.Itoa(factor) + "_" + yh_utils.IntToString(int(lat*float64(factor))) + "-" + yh_utils.IntToString(int(lng*float64(factor)))
}
func KeyFromPos(sex string, x int, y int, factor int) string {
	return sex + "_" + strconv.Itoa(factor) + "_" + yh_utils.IntToString(x) + "-" + yh_utils.IntToString(y)
}

func GetPos(latOrLng float64, factor int) int {
	return int(latOrLng * float64(factor))
}
