package message

import (
	"fmt"
	"strings"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/word_filter"
)

type TagFunc func(from uint32, tag string, content interface{}, result map[string]interface{}) (msgid uint64, e error)

var taggers map[string]TagFunc = map[string]TagFunc{}

func RegisterTag(prefix string, tm TagFunc) {
	taggers[prefix] = tm
}

func SendTag(from uint32, tag string, content interface{}, res map[string]interface{}) (msgid uint64, e error) {
	for pre, tm := range taggers {
		if strings.Index(tag, pre) == 0 {
			switch value := content.(type) {
			case map[string]interface{}:
				typ := utils.ToString(value["type"])
				switch typ {
				case common.MSG_TYPE_TEXT:
					n, replaced := word_filter.Replace(utils.ToString(value["content"]))
					if n > 0 {
						origin := value["content"]
						value["content"] = replaced + "[敏感词已过滤]"
						msgid, e := tm(from, tag, content, res)
						if e != nil {
							return 0, e
						}
						sql := "insert into bad_message(id,`from`,origin,replaced,num)values(?,?,?,?,?)"
						if _, e = msgdb.Exec(sql, msgid, from, origin, replaced, n); e != nil {
							mainLog.Append(fmt.Sprintf("add to bad_message table error:%v", e.Error()))
						}
						return msgid, nil
					}
				}
			}
			return tm(from, tag, content, res)
		}
	}
	return 0, service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("unknown tag prefix : %v", tag))
}
