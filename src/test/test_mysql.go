package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

func main() {
	// 打开数据库，sns是我的数据库名字，需要替换你自己的名字，（官网给的没有加tcp，跑不起来，具体有时&nbsp;间看看源码分析下为何）
	db, err := sql.Open("mysql", "root:root@tcp(localhost:3306)/test?charset=utf8&parseTime=true&loc=Asia%2FShanghai")
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	stmtin, err := db.Prepare("insert into TestTime(tm,dt)values(?,?)")
	if err != nil {
		return
	}
	defer stmtin.Close()
	_, err = stmtin.Exec(time.Now(), time.Now())
	if err != nil {
		fmt.Println(err)
	}
	stmt, err := db.Prepare("SELECT id,tm,dt FROM `TestTime`")
	if err != nil {
		return
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var tm, dt mysql.NullTime
		var id uint
		e := rows.Scan(&id, &tm, &dt)
		if e != nil {
			fmt.Println(e)
		}
		if dt.Valid {
			fmt.Printf("id=%v\ttm=%v\ttm.local=%v\tdt=%v\n", id, tm.Time, tm.Time.Local(), dt.Time)
		} else {
			fmt.Println("null")
		}
	}

}
