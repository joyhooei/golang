package event

import (
	"encoding/json"
	"time"
	"yf_pkg/format"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/unread"
)

var mdb *mysql.MysqlDB
var mainLog *log.MLogger

func Init(env *cls.CustomEnv) (e error) {
	mdb = env.MainDB
	mainLog = env.MainLog

	unread.Register(common.UNREAD_EVENT, UnreadNum)
	return e
}

var sql_detail string = "select title,style,pic,content,tm,timeout from events where id=?"

func Detail(id uint32) (event map[string]interface{}, e error) {
	var content []byte
	var style, title, pic, tmStr, toStr string
	e = mdb.QueryRow(sql_detail, id).Scan(&title, &style, &pic, &content, &tmStr, &toStr)
	if e != nil {
		return nil, e
	}
	tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
	if e != nil {
		return nil, e
	}
	to, e := utils.ToTime(toStr, format.TIME_LAYOUT_1)
	if e != nil {
		return nil, e
	}
	var j map[string]interface{}
	if e := json.Unmarshal(content, &j); e != nil {
		return nil, e
	}
	return map[string]interface{}{"id": id, "title": title, "style": style, "pic": pic, "tm": tm, "content": j, "timeout": to}, nil
}

func List(cur, ps int) (events []map[string]interface{}, total int, e error) {
	sql := "select id,title,style,pic,content,tm,timeout from events order by timeout desc" + utils.BuildLimit(cur, ps)
	rows, e := mdb.Query(sql)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	sql = "select count(*) from events"
	e = mdb.QueryRow(sql).Scan(&total)
	if e != nil {
		return nil, 0, e
	}
	events = make([]map[string]interface{}, 0, ps)
	for rows.Next() {
		var id uint32
		var content []byte
		var style, title, pic, tmStr, timeoutStr string
		if e = rows.Scan(&id, &title, &style, &pic, &content, &tmStr, &timeoutStr); e != nil {
			return nil, 0, e
		}
		tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
		if e != nil {
			return nil, 0, e
		}
		timeout, e := utils.ToTime(timeoutStr, format.TIME_LAYOUT_1)
		if e != nil {
			return nil, 0, e
		}
		var j map[string]interface{}
		if e := json.Unmarshal(content, &j); e != nil {
			return nil, 0, e
		}
		events = append(events, map[string]interface{}{"id": id, "title": title, "style": style, "pic": pic, "tm": tm, "timeout": timeout, "content": j})
	}
	return
}

var sql_focus string = "select pic,text,action from focus where `type`=? and status=1 order by position"

func FocusList(tp string) (focus []map[string]interface{}, e error) {
	rows, e := mdb.Query(sql_focus, tp)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	focus = make([]map[string]interface{}, 0, 10)
	for rows.Next() {
		var action []byte
		var pic, text string
		if e = rows.Scan(&pic, &text, &action); e != nil {
			return nil, e
		}
		var j map[string]interface{}
		if e := json.Unmarshal(action, &j); e != nil {
			return nil, e
		}
		focus = append(focus, map[string]interface{}{"text": text, "pic": pic, "action": j})
	}
	return
}

var sql_unread string = "select count(*) from events where tm > ? and timeout > ?"

// 加幸运未读消息
func UnreadNum(uid uint32, k string, from time.Time) (total uint32, show string) {
	switch k {
	case common.UNREAD_EVENT:
		mdb.QueryRow(sql_unread, from, utils.Now).Scan(&total)
	}
	if total > 0 {
		return 1, "1"
	} else {
		return 0, ""
	}
}
