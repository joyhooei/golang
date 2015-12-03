package push

import "database/sql"

var db *sql.DB = nil

func Init(d *sql.DB) {
	db = d
}

func Send(uid uint64, devid string, content string) (err error) {
	return
}
