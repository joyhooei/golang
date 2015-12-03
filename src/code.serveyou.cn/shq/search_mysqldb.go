package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/model"
	"code.serveyou.cn/pkg/format"
	_ "github.com/go-sql-driver/mysql"
)

type SearchDBAdapter struct {
	db *sql.DB
}

func NewSearchDBAdapter(db *sql.DB) (dba *SearchDBAdapter) {
	dba = new(SearchDBAdapter)
	dba.db = db
	return
}

func (s *SearchDBAdapter) ListNotifications(provinces []uint8, cities []uint, communities []uint, pn uint, rn uint) (notis []model.Notification, err error) {
	var ps, cis, cos bytes.Buffer
	ps.WriteString("0")
	for _, item := range provinces {
		ps.WriteString(fmt.Sprintf(",%v", item))
	}
	cis.WriteString("0")
	for _, item := range cities {
		cis.WriteString(fmt.Sprintf(",%v", item))
	}
	cos.WriteString("0")
	for _, item := range communities {
		cos.WriteString(fmt.Sprintf(",%v", item))
	}
	sql := fmt.Sprintf(common.SQL_ListNotifications, ps.String(), cis.String(), cos.String())
	rows, err := s.db.Query(sql, pn, rn)
	if err != nil {
		return
	}
	defer rows.Close()
	notis = make([]model.Notification, 0, rn)
	for rows.Next() {
		var n model.Notification
		var timeStr string
		rows.Scan(&n.Id, &n.Title, &n.Pic, &n.Content, &n.Url, &timeStr)
		n.Time, err = time.Parse(format.TIME_LAYOUT_1, timeStr)
		if err != nil {
			return
		}
		notis = append(notis, n)
	}
	return
}
func (s *SearchDBAdapter) GetNotification(id uint) (n model.Notification, err error) {
	var t string
	err = s.db.QueryRow(common.SQL_GetNotification, id).Scan(&n.Title, &n.Pic, &n.Content, &n.Url, &t)
	if err != nil {
		return
	}
	n.Time, err = time.Parse(format.TIME_LAYOUT_1, t)
	if err != nil {
		return
	}
	return
}
