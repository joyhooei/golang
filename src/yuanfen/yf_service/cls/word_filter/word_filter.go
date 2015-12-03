package word_filter

import (
	"yf_pkg/algorithm/trie"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yuanfen/yf_service/cls"
)

var db *mysql.MysqlDB
var table string
var mainLog *log.MLogger
var filter *trie.Trie

func Init(env *cls.CustomEnv) (e error) {
	db = env.SortDB
	mainLog = env.MainLog
	table = "words"

	return Reload()
}

func Reload() error {
	rows, err := db.Query("select keyword from " + table)
	if err != nil {
		return err
	}
	defer rows.Close()
	newFilter := trie.New()
	for rows.Next() {
		var keyword string
		if err := rows.Scan(&keyword); err != nil {
			return err
		}
		newFilter.AddElement(keyword)

	}
	filter = &newFilter
	return nil
}

func Search(text string) (pos int, keyword string) {
	return filter.Search(text)
}

func Replace(text string) (num int, replaced string) {
	return filter.Replace(text)
}
