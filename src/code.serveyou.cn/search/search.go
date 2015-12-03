package search

import (
	"sort"

	"code.serveyou.cn/common"
	"code.serveyou.cn/model"
)

type Document struct {
	Id   uint
	Rank uint
}
type DocumentsByID []Document
type DocumentsByRank []Document

func (a DocumentsByID) Len() int           { return len(a) }
func (a DocumentsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DocumentsByID) Less(i, j int) bool { return a[i].Id < a[j].Id }

func (a DocumentsByRank) Len() int           { return len(a) }
func (a DocumentsByRank) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DocumentsByRank) Less(i, j int) bool { return a[i].Rank > a[j].Rank }

type Searcher struct {
	revertTable map[uint]map[string]DocumentsByID
	cache       map[uint]map[string]DocumentsByRank
}

func wordLength(pre uint8) (sp int) {
	switch {
	case pre >= 0xC0 && pre < 0xE0:
		sp = 2
	case pre >= 0xE0 && pre < 0xF0:
		sp = 3
	case pre >= 0xF0 && pre < 0xF8:
		sp = 4
	case pre >= 0xF8 && pre < 0xFC:
		sp = 5
	case pre >= 0xFC:
		sp = 6
	default:
		sp = 1
	}
	return
}

func NewSearcher(communities map[uint]model.Community) (s *Searcher) {
	s = &Searcher{}
	tmp1 := make(map[uint]map[string]map[uint]uint, 1000)
	for id, c := range communities {
		tmp3, found := tmp1[c.City]
		if !found {
			tmp3 = make(map[string]map[uint]uint, 200)
			tmp1[c.City] = tmp3
		}
		a := c.Name()
		for i := 0; i < len(a); {
			sp := wordLength(a[i])
			tmp2, found := tmp3[a[i:i+sp]]
			if !found {
				tmp2 = make(map[uint]uint)
				tmp3[a[i:i+sp]] = tmp2
			}
			tmp2[id] += 1
			i += sp
		}
	}
	s.revertTable = make(map[uint]map[string]DocumentsByID)
	for city, se := range tmp1 {
		tmp := make(map[string]DocumentsByID)
		s.revertTable[city] = tmp
		for key, value := range se {
			docs := make(DocumentsByID, 0, 100)
			for id, rank := range value {
				docs = append(docs, Document{id, rank})
			}
			sort.Sort(docs)
			tmp[key] = docs
		}
	}
	s.cache = make(map[uint]map[string]DocumentsByRank)
	return
}
func (s *Searcher) Search(city uint, keyword string, pn uint, rn uint) (docs []Document) {
	if rn == 0 {
		return
	}
	searcher, found := s.revertTable[city]
	if !found {
		return
	}
	var dr DocumentsByRank
	ci, found := s.cache[city]
	if !found {
		ci = make(map[string]DocumentsByRank)
		s.cache[city] = ci
	}
	dr, found = ci[keyword]
	if !found {
		cas := make([]DocumentsByID, 0, common.MAX_KEYWORD_LEN)
		wordNum := 0
		for i := 0; i < len(keyword); {
			sp := wordLength(keyword[i])
			v, found := searcher[keyword[i:i+sp]]
			if found {
				cas = append(cas, v)
			}
			i += sp
			wordNum++
			if wordNum >= common.MAX_KEYWORD_LEN {
				break
			}
		}
		index := make([]int, len(cas))
		dr = make(DocumentsByRank, 0, 100)
		finished := false
		min := 0
		for !finished {
			finished = true
			for i := 0; i < len(cas); i++ {
				if index[i] < len(cas[i]) {
					finished = false
					if len(cas[min]) <= index[min] || cas[i][index[i]].Id < cas[min][index[min]].Id {
						min = i
					}
				}
			}
			if finished {
				break
			}
			if len(dr) > 0 && dr[len(dr)-1].Id == cas[min][index[min]].Id {
				dr[len(dr)-1].Rank += cas[min][index[min]].Rank
			} else {
				dr = append(dr, cas[min][index[min]])
			}
			index[min]++
		}
		sort.Sort(dr)
		ci[keyword] = dr
	}
	if int(pn*rn) >= len(dr) {
		return
	}
	max := (pn + 1) * rn
	if int(max) >= len(dr) {
		max = uint(len(dr))
	}

	return dr[pn*rn : max]
}
