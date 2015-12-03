package main

import (
	"yf_pkg/algorithm/trie"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/service"
)

type FilterModule struct {
	db     *mysql.MysqlDB
	table  string
	log    *log.MLogger
	filter *trie.Trie
}

func (fm *FilterModule) Init(env *service.Env) (err error) {
	fm.log = env.Log
	data := env.ModuleEnv.(map[string]interface{})
	fm.db = data["db"].(*mysql.MysqlDB)
	fm.table = data["table"].(string)
	return fm.reload()
}

func (fm *FilterModule) reload() error {
	rows, err := fm.db.Query("select keyword from " + fm.table)
	if err != nil {
		return err
	}
	defer rows.Close()
	filter := trie.New()
	for rows.Next() {
		var keyword string
		if err := rows.Scan(&keyword); err != nil {
			return err
		}
		filter.AddElement(keyword)

	}
	fm.filter = &filter
	return nil
}
func (fm *FilterModule) Reload(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	err := fm.reload()
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	return
}

func (fm *FilterModule) Search(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	content, ok := req.Body["content"].(string)
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "no [content] provided")
	}
	pos, v := fm.filter.Search(content)
	res["pos"] = pos
	res["keyword"] = v
	return
}

func (fm *FilterModule) Replace(req *service.HttpRequest, res map[string]interface{}) (e service.Error) {
	content, ok := req.Body["content"].(string)
	if !ok {
		return service.NewError(service.ERR_INVALID_PARAM, "no [content] provided")
	}
	num, replaced := fm.filter.Replace(content)
	res["num"] = num
	res["replaced"] = replaced
	return
}
