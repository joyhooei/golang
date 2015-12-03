package location

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"code.serveyou.cn/common"
)

type Element struct {
	Id  uint
	Lat float32
	Lng float32
}

type AdjElement struct {
	Id       uint
	Distance uint
}

type Position struct {
	LatPos int
	LngPos int
}

type Location struct {
	elements map[uint]Element  //元素列表
	position map[uint]Position //元素在下面列表中的位置
	sortLat  []uint            //根据纬度大小排列的元素列表
	sortLng  []uint            //根据经度大小排列的元素列表
}

type ByLat []Element

func (a ByLat) Len() int           { return len(a) }
func (a ByLat) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLat) Less(i, j int) bool { return a[i].Lat < a[j].Lat }

type ByLng []Element

func (a ByLng) Len() int           { return len(a) }
func (a ByLng) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLng) Less(i, j int) bool { return a[i].Lng < a[j].Lng }

type ByDistance []AdjElement

func (a ByDistance) Len() int           { return len(a) }
func (a ByDistance) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDistance) Less(i, j int) bool { return a[i].Distance < a[j].Distance }

func NewLocation(elems []Element) (location *Location) {
	tmpElems := make([]Element, len(elems))
	copy(tmpElems, elems)

	position := make(map[uint]Position)
	elements := make(map[uint]Element)

	sort.Sort(ByLat(tmpElems))
	sortLat := []uint{}
	for i, e := range tmpElems {
		sortLat = append(sortLat, e.Id)
		position[e.Id] = Position{i, 0}
		elements[e.Id] = e
	}
	sort.Sort(ByLng(tmpElems))
	sortLng := []uint{}
	for i, e := range tmpElems {
		sortLng = append(sortLng, e.Id)
		p := position[e.Id]
		p.LngPos = i
		position[e.Id] = p
	}

	location = &Location{elements, position, sortLat, sortLng}
	return
}

//返回以id所在位置为中心，边长为2*distance（米）的正方形范围内的所有元素
func (l *Location) Adjacent(id uint, di uint) (e []Element, err error) {
	var lat_degree float32 = float32(di) / float32(common.DISTANCE_LATITUDE)
	var lng_degree float32 = float32(di) / float32(common.DISTANCE_LONGITUDE)
	fmt.Printf("di=%v,LAT=%v,lat_degree=%v, lng_degree=%v\n", di, common.DISTANCE_LATITUDE, lat_degree, lng_degree)
	candidates := make(map[uint]bool)
	p, found := l.position[id]
	if !found {
		err = errors.New(fmt.Sprintf("find adjacent element error : not found element [id=%v].", id))
		return
	}
	center := l.elements[id]
	for i := p.LatPos; i >= 0; i-- {
		if center.Lat-l.elements[l.sortLat[i]].Lat <= lat_degree {
			candidates[l.sortLat[i]] = true
		} else {
			break
		}
	}
	for i := p.LatPos + 1; i < len(l.sortLat); i++ {
		if l.elements[l.sortLat[i]].Lat-center.Lat <= lat_degree {
			candidates[l.sortLat[i]] = true
		} else {
			break
		}
	}
	fmt.Printf("candidates : %v\n", candidates)
	e = make([]Element, 0, 10)
	for i := p.LngPos; i >= 0; i-- {
		if center.Lng-l.elements[l.sortLng[i]].Lng <= lng_degree {
			if candidates[l.sortLng[i]] && distance(center, l.elements[l.sortLng[i]]) <= di {
				e = append(e, l.elements[l.sortLng[i]])
			}
		} else {
			break
		}
	}
	for i := p.LngPos + 1; i < len(l.sortLng); i++ {
		if l.elements[l.sortLng[i]].Lng-center.Lng <= lng_degree {
			if candidates[l.sortLng[i]] && distance(center, l.elements[l.sortLng[i]]) <= di {
				e = append(e, l.elements[l.sortLng[i]])
			}
		} else {
			break
		}
	}
	fmt.Printf("Adjacent : %v\n", e)
	return
}
func rad(d float32) float64 {
	return float64(d) * math.Pi / 180.0
}

func (l *Location) Adjacent2(lat float32, lng float32, di uint) (e ByDistance, err error) {
	var lat_degree float32 = float32(di) / float32(common.DISTANCE_LATITUDE)
	var lng_degree float32 = float32(di) / float32(common.DISTANCE_LONGITUDE)
	fmt.Printf("lat_degree=%v, lng_degree=%v\n", lat_degree, lng_degree)
	candidates := make(map[uint]bool)
	left := 0
	right := len(l.sortLat) - 1
	if l.elements[l.sortLat[left]].Lat >= lat {
		right = left
	} else if l.elements[l.sortLat[right]].Lat <= lat {
		left = right
	} else {
		for left < right-1 {
			tmp := (left + right) / 2
			if l.elements[l.sortLat[tmp]].Lat < lat {
				left = tmp
			} else {
				right = tmp
			}
		}
	}
	fmt.Printf("lat left=%v right=%v\n", left, right)
	for i := left; i >= 0; i-- {
		if lat-l.elements[l.sortLat[i]].Lat <= lat_degree {
			candidates[l.sortLat[i]] = true
		} else {
			break
		}
	}
	for i := right; i < len(l.sortLat); i++ {
		if l.elements[l.sortLat[i]].Lat-lat <= lat_degree {
			candidates[l.sortLat[i]] = true
		} else {
			break
		}
	}
	fmt.Printf("lat candidates %v\n", candidates)

	left = 0
	right = len(l.sortLng) - 1
	if l.elements[l.sortLng[left]].Lng >= lng {
		right = left
	} else if l.elements[l.sortLng[right]].Lng <= lng {
		left = right
	} else {
		for left < right-1 {
			tmp := (left + right) / 2
			if l.elements[l.sortLng[tmp]].Lat < lng {
				left = tmp
			} else {
				right = tmp
			}
		}
	}
	fmt.Printf("lng left=%v right=%v\n", left, right)
	e = make(ByDistance, 0, 10)
	center := Element{0, lat, lng}
	for i := left; i >= 0; i-- {
		if lng-l.elements[l.sortLng[i]].Lng <= lng_degree {
			d := distance(center, l.elements[l.sortLng[i]])
			if candidates[l.sortLng[i]] && d <= di {
				e = append(e, AdjElement{l.sortLng[i], d})
			}
		} else {
			break
		}
	}
	if right == left {
		right++
	}
	for i := right; i < len(l.sortLng); i++ {
		if l.elements[l.sortLng[i]].Lng-lng <= lng_degree {
			d := distance(center, l.elements[l.sortLng[i]])
			if candidates[l.sortLng[i]] && d <= di {
				e = append(e, AdjElement{l.sortLng[i], d})
			}
		} else {
			break
		}
	}
	sort.Sort(e)
	return
}

func distance(e1 Element, e2 Element) uint {
	radLat1 := rad(e1.Lat)
	radLat2 := rad(e2.Lat)
	a := radLat1 - radLat2
	b := rad(e1.Lng) - rad(e2.Lng)

	s := 2 * math.Asin(math.Sqrt(math.Pow(math.Sin(a/2), 2)+math.Cos(radLat1)*math.Cos(radLat2)*math.Pow(math.Sin(b/2), 2)))
	v := uint(s * common.EARTH_RADIUS)
	return v
}
