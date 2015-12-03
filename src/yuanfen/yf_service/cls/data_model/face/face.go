package face

import (
	// "fmt"
	"errors"
	"strings"
	"sync"
	"time"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
)

type FaceItem struct {
	Id     uint32   `json:"id"`
	Name   string   `json:"name"`
	Ico    string   `json:"ico"`
	Pic    string   `json:"pic"`
	Gender int      `json:"gender"`
	Res    []string `json:"res"`
	// Group string `json:"group"`
}

type FaceList struct {
	Id     uint32      `json:"id"`
	Type   string      `json:type`
	Name   string      `json:"name"`
	Ico    string      `json:"ico"`
	Pic    string      `json:"pic"`
	Gender int         `json:"gender"`
	List   []*FaceItem `json:"list"`
}

type InputItem struct {
	Key  string   `json:"key"`
	List []uint32 `json:"list"`
}

var menlist, womenlist, bothlist []*FaceList
var gamelist *FaceList
var mendefaultpic, womendefaultpic string
var facever uint32
var meninput, womeninput []*InputItem
var facemap map[uint32]*FaceItem
var lock sync.RWMutex

var mdb *mysql.MysqlDB
var mlog *log.MLogger

func Init(env *cls.CustomEnv) {
	mdb = env.MainDB
	mlog = env.MainLog
	loadFaceList()
	go updateface()
	return
}

func addtogroup(name string, list []*FaceList, item *FaceItem) (rlist []*FaceList) {
	rlist = list
	for _, v := range rlist {
		if v.Name == name {
			v.List = append(v.List, item)
			return
		}
	}
	p := new(FaceList)
	p.Name = name
	p.List = make([]*FaceItem, 0, 0)
	p.List = append(p.List, item)
	rlist = append(rlist, p)
	return
}

func addtoinput(key string, faceid uint32, list []*InputItem) (rlist []*InputItem) {
	rlist = list
	for _, v := range rlist {
		if v.Key == key {
			v.List = append(v.List, faceid)
			return
		}
	}
	p := new(InputItem)
	p.Key = key
	p.List = make([]uint32, 0, 0)
	p.List = append(p.List, faceid)
	rlist = append(rlist, p)
	return rlist
}

func loadFaceList() (e error) {
	var tmpver uint32
	e = mdb.QueryRow("select ver from face_ver where `key`='face_ver'").Scan(&tmpver)
	if e != nil {
		return
	}

	var tmpdefaultpic, tmpdefaultwomen string
	// tmpmale := make([]*FaceList, 0, 0)
	// tmpfemale := make([]*FaceList, 0, 0)
	tmpboth := make([]*FaceList, 0, 0)
	// tmpgame := new(FaceList)
	menmap := make(map[uint32]int)
	womenmap := make(map[uint32]int)
	tmpmap := make(map[uint32]*FaceItem)
	tmpgame := make(map[uint32]*FaceItem)
	rows, e := mdb.Query("select id,name,icon,`type`,gender,pic from face where parentid=0 order by rank")
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var id uint32
		var gender int
		var name, icon, tp, pic string
		if e = rows.Scan(&id, &name, &icon, &tp, &gender, &pic); e != nil {
			return e
		}
		pboth := &FaceList{id, tp, name, icon, pic, gender, make([]*FaceItem, 0, 0)}
		rows2, e := mdb.Query("select id,name,icon,pic,gender,`default`,res from face where parentid=? order by rank", id)
		if e != nil {
			return e
		}
		defer rows2.Close()
		for rows2.Next() {
			var id uint32
			var name, pic, icon, res string
			var gender, ifdefault int
			if e = rows2.Scan(&id, &name, &icon, &pic, &gender, &ifdefault, &res); e != nil {
				return e
			}

			if ifdefault == 1 {
				switch gender {
				case common.GENDER_MAN:
					tmpdefaultpic = pic
				case common.GENDER_WOMAN:
					tmpdefaultwomen = pic
				case common.GENDER_BOTH:
					tmpdefaultpic = pic
					tmpdefaultwomen = pic
				}
			}
			item := &FaceItem{id, name, icon, pic, gender, strings.Split(res, ",")}
			if tp == "minigame" {
				tmpgame[id] = item
			}
			pboth.List = append(pboth.List, item)
			tmpmap[id] = item
			switch gender {
			case common.GENDER_MAN:
				menmap[id] = 1
			case common.GENDER_WOMAN:
				womenmap[id] = 1
			case common.GENDER_BOTH:
				menmap[id] = 1
				womenmap[id] = 1
			}
		}

		tmpboth = append(tmpboth, pboth)
	}

	rows, e = mdb.Query("select `key`,faceid from face_input")
	if e != nil {
		return e
	}
	tmpmeninput := make([]*InputItem, 0, 0)
	tmpwomeninput := make([]*InputItem, 0, 0)
	for rows.Next() {
		var key string
		var faceid uint32
		if e = rows.Scan(&key, &faceid); e != nil {
			return e
		}

		if _, ok := menmap[faceid]; ok {
			tmpmeninput = addtoinput(key, faceid, tmpmeninput)
		}
		if _, ok := womenmap[faceid]; ok {
			tmpwomeninput = addtoinput(key, faceid, tmpwomeninput)
		}
	}
	// fmt.Println(fmt.Sprintf("Load face %v", tmpver))
	// fmt.Println(fmt.Sprintf("Load tmpmale %v", tmpmale))
	// fmt.Println(fmt.Sprintf("Load tmpfemale %v", tmpfemale))
	// lock.Lock()
	// menlist = tmpmale
	// womenlist = tmpfemale
	gamemap = tmpgame
	bothlist = tmpboth
	// gamelist = tmpgame
	facever = tmpver
	meninput = tmpmeninput
	womeninput = tmpwomeninput
	mendefaultpic = tmpdefaultpic
	womendefaultpic = tmpdefaultwomen
	facemap = tmpmap
	// lock.Unlock()
	return
}

func updateface() {
	for {
		time.Sleep(60 * time.Second)
		loadFaceList()
	}
}

func GetFace(ver uint32) (result map[string]interface{}, e error) {
	result = make(map[string]interface{})
	if facever > ver {
		result["newver"] = facever
		result["mendefault"] = mendefaultpic
		result["meninput"] = meninput
		result["womendefault"] = womendefaultpic
		result["womeninput"] = womeninput
		result["facelist"] = bothlist
		// result["minigame"] = gamelist

	} else {
		result["newver"] = 0 //没有版本信息
	}
	return
}

func SendBigFace(f_uid, t_uid uint32, faceid uint32) (msgid uint64, e error) {
	item, ok := facemap[faceid]
	if !ok {
		return 0, errors.New("invalid faceid")
	}
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_BIGFACE
	content["id"] = faceid
	content["name"] = item.Name
	content["ico"] = item.Ico
	content["pic"] = item.Pic
	return general.SendMsg(f_uid, t_uid, content, "")
}
