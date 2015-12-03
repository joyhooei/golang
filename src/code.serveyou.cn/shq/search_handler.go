package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/model"
	"code.serveyou.cn/pkg/format"
)

type SearchHandler struct {
	sdb *SearchDBAdapter
}

func NewSearchHandler(db *sql.DB) (s *SearchHandler) {
	s = new(SearchHandler)
	s.sdb = NewSearchDBAdapter(db)
	return
}

func (s *SearchHandler) ProcessSec(cmd string, r *http.Request, uid common.UIDType, devid string, body string, result map[string]interface{}) (err common.Error) {
	switch cmd {
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, "unknown sec command : "+cmd)
		return
	}
	return
}

func (s *SearchHandler) Process(cmd string, r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	switch cmd {
	case "find_users":
		err = s.findUsers(r, body, result)
	case "find_communities":
		err = s.findCommunities(r, body, result)
	case "adjacent_communities":
		err = s.adjacentCommunities(r, body, result)
	case "recommend":
		err = s.recommend(r, body, result)
	case "notifications":
		err = s.listNotifications(r, body, result)
	case "notification":
		err = s.GetNotification(r, body, result)
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, "unknown command : "+cmd)
		return
	}
	return
}

func (s *SearchHandler) findUsers(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"role", "community", "job", "start"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	search := model.NewSearchCondition()
	if e = search.SetCommunityStr(m["community"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set community error : %v", e.Error()))
		return
	}
	if e = search.SetRoleStr(m["role"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set role error : %v", e.Error()))
		return
	}
	if e = search.SetJobStr(m["job"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set job error : %v", e.Error()))
		return
	}
	if e = search.SetStartStr(m["start"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set start error : %v", e.Error()))
		return
	}
	switch search.Role() {
	case common.ROLE_HKP:
		hkps, err := s.findHKPs(search)
		if err.Code != 0 {
			return err
		}
		re := make([]map[string]interface{}, 0, search.Rn())
		for _, hkp := range hkps {
			uinfo, found := Users[hkp.Job.Uid]
			if !found {
				err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("user %v not found", hkp.Job.Uid))
				return err
			}
			m := make(map[string]interface{})
			m["uid"] = uinfo.Id()
			m["idcardtype"] = uinfo.IdCardType()
			masked, e := common.MaskIDCardNum(uinfo.IdCardNum())
			if e != nil {
				err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("invalid id card %v", uinfo.IdCardNum()))
				return err
			}
			m["idcardnum"] = masked
			m["nickname"] = uinfo.NickName()
			m["firstname"] = uinfo.FirstName()
			m["lastname"] = uinfo.LastName()
			m["sex"] = uinfo.Sex()
			//		m["phone"] = uinfo.Cellphone()
			m["birthday"] = uinfo.Birthday().Format(format.TIME_LAYOUT_2)
			m["birthplace"] = uinfo.Birthplace()
			m["rank"] = hkp.Rank.RankAll()
			m["rank_desc"] = hkp.Rank.RankAllDesc()
			m["speed"] = hkp.Rank.Speed()
			m["quality"] = hkp.Rank.Quality()
			m["attitude"] = hkp.Rank.Attitude()
			m["times"] = hkp.Rank.SuccessTimes()
			m["price"] = hkp.Job.Price
			m["jobversion"] = hkp.Job.Version
			if uinfo.HealthCardTimeout().After(time.Now()) {
				m["health"] = true
			} else {
				m["health"] = false
			}
			re = append(re, m)
		}
		result["hkps"] = re
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("unknown role %v", search.Role()))
	}
	return
}
func (s *SearchHandler) findHKPs(search *model.SearchCondition) (hkps []model.HKPDetail, err common.Error) {
	//先要找到附近两倍距离的小区
	communities, e := loc.Adjacent(search.Community(), common.Variables.AdjacentRange()*2)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Adjacent error : %v", e.Error()))
		return
	}
	noTimeHKPs, e := GetHKPsNoTime(s.sdb.db, communities, search.Role(), search.Start())
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetHKPsNoTime error : %v", e.Error()))
		return
	}
	hkps = selectHKPs(search, noTimeHKPs)
	return
}

func selectHKPs(search *model.SearchCondition, noTimeHKPs map[common.UIDType]bool) (hkps []model.HKPDetail) {
	//TODO:每个阿姨随机分配一个初始权重(0-100)，然后再根据阿姨的评价、服务成功率等调整权重，最后再按权重从高到低排序，返回给客户前100个
	communities, err := loc.Adjacent(search.Community(), common.Variables.AdjacentRange())
	if err != nil {
		return
	}
	score := make(common.RandomScoreElems, 0, 30)
	for _, elem := range communities {
		j, found := CommunityHKPs[elem.Id]
		fmt.Printf("j=%v found=%v\n", j, found)
		if found {
			u, found := j[search.Job()]
			if found {
				for uid, d := range u {
					if _, found := noTimeHKPs[uid]; !found {
						log := fmt.Sprintf("uid=%v ", uid)
						var priority int32 = rand.Int31n(int32(common.Variables.HKPListLimit()))
						log += fmt.Sprintf("priority=%v ", priority)
						if d.Rank.Times >= 5 {
							priority += int32(d.Rank.RankAll() * 2)
							log += fmt.Sprintf("rank+%v ", int(d.Rank.RankAll()*2))
							priority -= int32(d.Rank.RejectTimes * 10 / d.Rank.Times)
							log += fmt.Sprintf("RejectTimes+%v ", (-int(d.Rank.RejectTimes * 10 / d.Rank.Times)))
							priority -= int32(d.Rank.FailTimes * 5 / d.Rank.Times)
							log += fmt.Sprintf("FailTime+%v ", (-int(d.Rank.FailTimes * 5 / d.Rank.Times)))
						}
						score = append(score, common.ScoreElem{uid, elem.Id, 0, priority})
						log += fmt.Sprintf("total=%v", priority)
						fmt.Println(log)
					}
				}
			}
		}
	}
	sort.Sort(score)
	num := uint(len(score))
	if num > common.Variables.HKPListLimit() {
		num = common.Variables.HKPListLimit()
	}

	fmt.Printf("num=%v\n", num)
	hkps = make([]model.HKPDetail, 0, num)
	for i := uint(0); i < num; i++ {
		hkps = append(hkps, CommunityHKPs[score[i].Comm][search.Job()][score[i].Uid])
	}

	return
}
func (s *SearchHandler) findCommunities(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"city", "keyword", "pn", "rn"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	city, e := format.ParseUint(m["city"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse city error : %v", e.Error()))
		return
	}

	pn, e := format.ParseUint(m["pn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse pn error : %v", e.Error()))
		return
	}
	rn, e := format.ParseUint(m["rn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse rn error : %v", e.Error()))
		return
	}

	res := CSearcher.Search(city, m["keyword"], pn, rn)
	list := make([]map[string]interface{}, 0, rn)
	for _, doc := range res {
		comm, found := Communities[doc.Id]
		if !found {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("search result error : no community of id [%v]", doc.Id))
			return
		}
		kv := make(map[string]interface{})
		kv["id"] = doc.Id
		kv["rank"] = doc.Rank
		kv["name"] = comm.Name()
		kv["address"] = comm.Address()
		kv["lat"] = comm.Latitude()
		kv["lng"] = comm.Longitude()
		kv["city_id"] = Cities[comm.City].Id
		kv["city_name"] = Cities[comm.City].Name
		list = append(list, kv)
	}
	result["communities"] = list
	return
}
func (s *SearchHandler) adjacentCommunities(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"lat", "lng"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	lat, e := format.ParseFloat(m["lat"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse lat error : %v", e.Error()))
		return
	}
	lng, e := format.ParseFloat(m["lng"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse lng error : %v", e.Error()))
		return
	}
	communities, e := loc.Adjacent2(lat, lng, common.Variables.AdjacentRange())
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Adjacent2 error : %v", e.Error()))
		return
	}
	list := make([]map[string]interface{}, 0, 10)
	for _, c := range communities {
		comm, found := Communities[c.Id]
		if !found {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("search result error : no community of id [%v]", c.Id))
			return
		}
		kv := make(map[string]interface{})
		kv["id"] = c.Id
		kv["distance"] = c.Distance
		kv["name"] = comm.Name()
		kv["address"] = comm.Address()
		kv["lat"] = comm.Latitude()
		kv["lng"] = comm.Longitude()
		kv["city_id"] = Cities[comm.City].Id
		kv["city_name"] = Cities[comm.City].Name
		list = append(list, kv)
	}
	result["communities"] = list
	return
}

func (s *SearchHandler) recommend(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	result["items"] = Recommend
	return
}
func (s *SearchHandler) listNotifications(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"pn", "rn"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	var ps, cis, cos []string
	provinces := make([]uint8, 0, 2)
	if _, found := m["provinces"]; found {
		ps = format.ParseVector(m["provinces"], ",")
		for _, value := range ps {
			if p, e := format.ParseUint8(value); e != nil {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse provinces error : %v", e.Error()))
				return
			} else {
				provinces = append(provinces, p)
			}
		}
	}
	cities := make([]uint, 0, 2)
	if _, found := m["cities"]; found {
		cis = format.ParseVector(m["cities"], ",")
		for _, value := range cis {
			if c, e := format.ParseUint(value); e != nil {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse cities error : %v", e.Error()))
				return
			} else {
				cities = append(cities, c)
			}
		}
	}
	communities := make([]uint, 0, 2)
	if _, found := m["communities"]; found {
		cos = format.ParseVector(m["communities"], ",")
		for _, value := range cos {
			if c, e := format.ParseUint(value); e != nil {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse communities error : %v", e.Error()))
				return
			} else {
				communities = append(communities, c)
			}
		}
	}
	pn, e := format.ParseUint(m["pn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse pn error : %v", e.Error()))
		return
	}
	rn, e := format.ParseUint(m["rn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse rn error : %v", e.Error()))
		return
	}
	notis, e := s.sdb.ListNotifications(provinces, cities, communities, pn, rn)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("ListNotifications error : %v", e.Error()))
		return
	}

	ns := make([]interface{}, 0, rn)
	for _, no := range notis {
		n := make(map[string]interface{})
		n["id"] = no.Id
		n["time"] = no.Time.Format(format.TIME_LAYOUT_1)
		n["title"] = no.Title
		n["pic"] = no.Pic
		n["content"] = no.Content
		n["url"] = no.Url
		ns = append(ns, n)
	}
	result["notifications"] = ns
	return
}
func (s *SearchHandler) GetNotification(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"id"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	id, e := format.ParseUint(m["id"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse id error : %v", e.Error()))
		return
	}
	no, e := s.sdb.GetNotification(id)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetNotification error : %v", e.Error()))
		return
	}

	result["id"] = no.Id
	result["time"] = no.Time.Format(format.TIME_LAYOUT_1)
	result["title"] = no.Title
	result["pic"] = no.Pic
	result["content"] = no.Content
	result["url"] = no.Url
	return
}
