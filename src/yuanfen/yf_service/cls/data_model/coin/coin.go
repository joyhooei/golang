package coin

import (
	//"fmt"
	// "database/sql"
	// "strconv"
	"yf_pkg/mysql"
	// "yf_pkg/service"

	"yf_pkg/service"
	"yf_pkg/utils"

	"yuanfen/yf_service/cls"
)

//能执行Exec的对象 tx 和MysqlDB
const (
	COIN_DAY_MAX       = 1000
	COIN_DAY_OTHER_MAX = 10000
)

var mdb *mysql.MysqlDB

func Init(env *service.Env) {
	mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
}

// 用户消费或赢得金币

// func UserCoinChange( uid uint32, forid uint32, tp int, exttp int, coin int, info string) (e error) {

// 	txx, err := mdb.Begin()
// 	if err != nil {
// 		return err
// 	}
// 	txx.Exec(query, ...)
// 	row, err := txx.Query("select goldcoin from user_main where uid=? FOR UPDATE", uid)
// 	if err != nil {
// 		txx.Rollback()
// 		return err
// 	}
// 	defer row.Close()
// 	sqlr, e2 := utils.ParseSqlResult(row)
// 	if e2 != nil {
// 		return err
// 	}
// 	if len(sqlr) <= 0 {
// 		txx.Rollback()
// 		return err
// 	}
// 	var rmap = sqlr[0]
// 	goldcoin, err := strconv.Atoi(rmap["goldcoin"])
// 	if err != nil {
// 		txx.Rollback()
// 		return err
// 	}
// 	if goldcoin < -coin {
// 		txx.Rollback()
// 		return err
// 	}
// 	_, err = txx.Exec("update user_main set goldcoin=goldcoin+? where uid=?", coin, uid)
// 	if err != nil {
// 		txx.Rollback()
// 		return err
// 	}
// 	_, err = txx.Exec("insert into user_coin_log (uid,forid,type,exttype,coin,info,status,create_time)values(?,?,?,?,?,?,1,?)", uid, forid, tp, exttp, coin, info, utils.Now)
// 	if err != nil {
// 		txx.Rollback()
// 		return err
// 	}
// 	err = txx.Commit()
// 	if err != nil {
// 		return err
// 	}
// 	return
// }

func GetUserCoinInfo(uid uint32) (coin int, isvip int, e error) {
	sql := "select goldcoin from user_main where uid=?"
	e = mdb.QueryRowFromMain(sql, uid).Scan(&coin)
	return
}

func GetUserCoin(tx utils.SqlObj, uid uint32) (coin int, e error) {
	sql := "select goldcoin from user_main where uid=?"
	e = tx.QueryRow(sql, uid).Scan(&coin)
	return
}

//男性用户当日可获得的金币总和
func CheckTodayWork(uid uint32) (result bool, e error) {
	var total int
	err := mdb.QueryRow("select ifnull(sum(coin),0) from user_coin_log where type=? and create_time>? and ((uid=? and forid=0)or forid=?)", EARN_WORK, utils.Now.AddDate(0, 0, -1), uid, uid).Scan(&total)
	if err != nil {
		return false, err
	}
	return total < COIN_DAY_MAX, nil
}

//女性用户当日可接受的金币总和
func CheckTodayFend(uid uint32) (result bool, e error) {
	var total int
	err := mdb.QueryRow("select ifnull(sum(coin),0) from user_coin_log where type=?  and uid=? and forid>0 and create_time>?", EARN_WORK, uid, utils.Now.AddDate(0, 0, -1)).Scan(&total)
	if err != nil {
		return false, err
	}
	return total < COIN_DAY_OTHER_MAX, nil
}

//uid是金币变更主体 forid是关联变更用户
//玩游戏 uid 为付款的男生 消费为负 扣金币 forid为女生
//供养 uid为得款的女生 消费为正 加金币 forid为供养者 男生
func UserCoinChange(tx utils.SqlObj, uid uint32, forid uint32, tp int, exttp int, coin int, info string) service.Error {
	r, err := tx.Exec("update user_main set goldcoin=goldcoin+? where uid=? and goldcoin>=?", coin, uid, -coin)
	if err != nil {
		return service.NewError(service.ERR_MYSQL, err.Error())
	}

	i, err := r.RowsAffected()
	if err != nil {
		return service.NewError(service.ERR_MYSQL, err.Error())
	}

	if i <= 0 {
		return service.NewError(service.ERR_NOT_ENOUGH_MONEY, "余额不足")
	}
	_, err = tx.Exec("insert into user_coin_log (uid,forid,type,exttype,coin,info,status,create_time)values(?,?,?,?,?,?,1,?)", uid, forid, tp, exttp, coin, info, utils.Now)
	if err != nil {
		return service.NewError(service.ERR_MYSQL, err.Error())
	}

	return service.NewError(service.ERR_NOERR, "")
}

//查询A给B的供养
func FendCoinTotal(fromid uint32, toid uint32) (total int, e error) {
	err := mdb.QueryRow("select ifnull(sum(coin),0) from user_coin_log where type=? and uid=? and forid=?", EARN_WORK, toid, fromid).Scan(&total)
	if err != nil {
		return 0, err
	}
	return
}

//查询给我的供养总数
func MyFendCoin(uid uint32) (total int, count int, e error) {
	err := mdb.QueryRow("select ifnull(sum(coin),0),count(*) from user_coin_log where type=? and forid>0 and uid=?", EARN_WORK, uid).Scan(&total, &count)
	if err != nil {
		return 0, 0, err
	}
	return
}

//消费类型，对应类型 1为充值，2为赠送,3为打工获取,4为消费
//供养列表
func FeedLog(uid uint32, cur, ps int) (result []map[string]string, e error) {
	sql := "select info,coin,create_time as time,forid as uid from user_coin_log where type=? and forid>0 and uid=? order by create_time desc " + utils.BuildLimit(cur, ps)
	rows, err := mdb.Query(sql, EARN_WORK, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list, err := utils.ParseSqlResult(rows)
	if err != nil {
		return nil, err
	}
	return list, nil
}
