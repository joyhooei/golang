package mall

import (
	"fmt"
	"yf_pkg/format"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
)

var mdb *mysql.MysqlDB
var mainLog *log.MLogger

func Init(env *cls.CustomEnv) (e error) {
	mdb = env.MainDB
	mainLog = env.MainLog

	return e
}

var sql_detail string = "select title,style,pic,content,tm,timeout from malls where id=?"

func Buy(uid, id uint32) (e error) {
	tx, e := mdb.Begin()
	if e != nil {
		return e
	}
	var itemId, gold, ugold int
	if e = tx.QueryRow("select i.id,m.gold from mall m,mall_inventory i where m.id=i.goods_id and i.goods_id=? and i.status=0", id).Scan(&itemId, &gold); e != nil {
		tx.Rollback()
		if e == mysql.ErrNoRows {
			return service.NewError(service.ERR_INVENTORY_EMPTY, "inventory empty", "库存不足")
		}
		return e
	}
	if e = tx.QueryRow("select goldcoin from user_main where uid=?", uid).Scan(&ugold); e != nil {
		tx.Rollback()
		return e
	}
	if ugold < gold {
		tx.Rollback()
		return service.NewError(service.ERR_NOT_ENOUGH_MONEY, "not enough diamond", "钻石不足")
	}
	//扣钱
	if _, e = tx.Exec("update user_main set goldcoin=? where uid=?", ugold-gold, uid); e != nil {
		tx.Rollback()
		return e
	}
	//实例物品更新为已售出
	if _, e = tx.Exec("update mall_inventory set status=1 where id=?", itemId); e != nil {
		tx.Rollback()
		return e
	}
	//增加一条用户的购买记录
	if _, e = tx.Exec("insert into mall_buy_history(uid,item_id)values(?,?)", uid, itemId); e != nil {
		tx.Rollback()
		return e
	}
	if e = tx.Commit(); e != nil {
		tx.Rollback()
		return e
	}
	return nil
}

func List(uid uint32, cur, ps int) (malls []map[string]interface{}, total int, e error) {
	sql := "select count(*) from mall m,mall_inventory i,mall_buy_history b where m.id=i.goods_id and i.id=b.item_id and b.uid=?"
	e = mdb.QueryRow(sql, uid).Scan(&total)
	if e != nil {
		return nil, 0, e
	}
	sql = "select b.id,m.title,m.pic,b.tm from mall m,mall_inventory i,mall_buy_history b where m.id=i.goods_id and i.id=b.item_id and b.uid=? order by b.id desc" + utils.BuildLimit(cur, ps)
	rows, e := mdb.Query(sql, uid)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	malls = make([]map[string]interface{}, 0, ps)
	for rows.Next() {
		var id uint32
		var title, pic, tmStr string
		if e = rows.Scan(&id, &title, &pic, &tmStr); e != nil {
			return nil, 0, e
		}
		tm, e := utils.ToTime(tmStr, format.TIME_LAYOUT_1)
		if e != nil {
			return nil, 0, e
		}
		malls = append(malls, map[string]interface{}{"id": id, "title": title, "pic": pic, "url": fmt.Sprintf("http://test.a.imswing.cn:10080/mall/detail?id=%v", id), "tm": tm})
	}
	return
}
