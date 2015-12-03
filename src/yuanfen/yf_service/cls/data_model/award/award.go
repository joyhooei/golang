package award

import (
	"errors"
	"fmt"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/coin"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/notify"
)

const (
	AWARD_STATUS_WAIT     = 0 //待领取
	AWARD_STATUS_RECVED   = 1 //已领取
	AWARD_STATUS_WAITSEND = 2 //等待充值（发货）
	AWARD_STATUS_COMPLETE = 3 //已完成
)

const (
	AWARD_FROM_GAME  = 1 //游戏
	AWARD_FROM_WABAO = 2 //挖宝
	AWARD_FROM_LUKKY = 3 //抽奖
)

//能执行Exec的对象 tx 和MysqlDB

var mdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool

func Init(env *service.Env) {
	mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds
}

func PushAwardSysMessage(to uint32, msg map[string]interface{}) (msgid uint64, e error) {
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_TEXT
	content["FOLDER_KEY"] = common.FOLDER_OTHER
	for k, v := range msg {
		content[k] = v
	}
	return general.SendMsg(common.UID_AWARD, to, content, "")
}

//新增奖品方法
func AwardAdd(tx utils.SqlObj, uid uint32, award_id int, from int, from_ext string, f_uid int, flag int, cnum int, frominfo string) (aid int, e service.Error) {
	var itype, price, virtualtype, virtualcount int
	var name string
	// fmt.Println(fmt.Sprintf("award_id %v", award_id))
	err := mdb.QueryRow("select `type`,name,price,virtualtype,virtualcount from award_config where id=?", award_id).Scan(&itype, &name, &price, &virtualtype, &virtualcount)
	if err != nil {
		return 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	// fmt.Println(fmt.Sprintf("AwardAdd %v", itype))
	r, err := tx.Exec("insert into award_record (uid,award_id,`from`,from_ext,f_uid,flag,cnum,frominfo)values(?,?,?,?,?,?,?,?)", uid, award_id, from, from_ext, f_uid, flag, cnum, frominfo)
	if err != nil {
		return 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	id, err := r.LastInsertId()
	if err != nil {
		return 0, service.NewError(service.ERR_MYSQL, err.Error())
	}
	if v, err := utils.ToInt(id); err != nil {
		return 0, service.NewError(service.ERR_MYSQL, err.Error())
	} else {
		aid = v
	}
	switch itype {
	case 1: //金币
		_, err := tx.Exec("update award_record set oper_tm=?,status=? where id=? ", utils.Now, 3, aid)
		if err != nil {
			return 0, service.NewError(service.ERR_MYSQL, err.Error())
		}
		if err := coin.UserCoinChange(tx, uid, 0, coin.EARN_AWARD, 0, price, "获得奖品 "+name); err.Code != service.ERR_NOERR {
			return 0, err
		}

		msg := make(map[string]interface{})
		if c, _, err := coin.GetUserCoinInfo(uid); err == nil {
			msg[common.USER_BALANCE] = c
		}
		not, e := notify.GetNotify(uid, notify.NOTIFY_COIN, nil, "系统消息", "获得奖品 "+name, uid)
		if e == nil {
			msg[notify.NOTIFY_KEY] = not
		}
		msg["content"] = "获得奖品 " + name
		PushAwardSysMessage(uid, msg)
		// fmt.Println(fmt.Sprintf("SysMessage %v", msg))
	case 4: //虚拟奖品
		_, err := tx.Exec("update award_record set oper_tm=?,status=? where id=? ", utils.Now, 3, aid)
		if err != nil {
			return 0, service.NewError(service.ERR_MYSQL, err.Error())
		}
		switch virtualtype {
		case 1: //领取飞行场次
			//		if err := service_game.UpdateGameNum(tx, uid, virtualcount, true); err != nil { //赠送获得 最后参数填写true
			//		return 0, service.NewError(service.ERR_MYSQL, err.Error())
			//	}
			un := make(map[string]interface{})
			//	un[common.UNREAD_PLANE_FREE] = 0
			//	unread.GetUnreadNum(uid, un)
			msg := make(map[string]interface{})
			msg[common.UNREAD_KEY] = un
			not, e := notify.GetNotify(uid, notify.NOTIFY_PLANE_NUM, nil, "系统消息", "获得奖品 "+name, uid)
			if e == nil {
				msg[notify.NOTIFY_KEY] = not
			}
			msg["content"] = "获得奖品 " + name
			PushAwardSysMessage(uid, msg)
			// fmt.Println(fmt.Sprintf("SysMessage %v", msg))
		}
	case 5: //金币
		_, err := tx.Exec("update award_record set oper_tm=?,status=? where id=? ", utils.Now, 3, aid)
		if err != nil {
			return 0, service.NewError(service.ERR_MYSQL, err.Error())
		}
		if err := coin.UserCoinChange(tx, uid, 0, coin.EARN_AWARD, 0, cnum, fmt.Sprintf("获得奖品 %v %v", cnum, name)); err.Code != service.ERR_NOERR {
			return 0, err
		}

		msg := make(map[string]interface{})
		if c, _, err := coin.GetUserCoinInfo(uid); err == nil {
			msg[common.USER_BALANCE] = c
		}
		not, e := notify.GetNotify(uid, notify.NOTIFY_COIN, nil, "系统消息", fmt.Sprintf("获得奖品 %v %v", cnum, name), uid)
		if e == nil {
			msg[notify.NOTIFY_KEY] = not
		}
		msg["content"] = "获得奖品 " + name
		PushAwardSysMessage(uid, msg)
	}

	return
}

//收取奖品
func AwardReceive(uid uint32, id int) (e error) {
	sql := "update award_record set oper_tm=?,status=? where id=? and uid=? and status<=1"
	r, err := mdb.Exec(sql, utils.Now, 1, id, uid)
	if err != nil {
		return err
	}
	if r, e := r.RowsAffected(); e != nil {
		return err
	} else {
		if r == 0 {
			return errors.New("无可领取的奖品")
		}
	}
	return
}

func getAddr(transid uint32) (result map[string]interface{}, e error) {
	var name, num, address string
	sql := "select address_name,address_phone,address from award_trans where id=?"
	err := mdb.QueryRow(sql, transid).Scan(&name, &num, &address)
	if err != nil {
		return nil, err
	}
	result = make(map[string]interface{})
	result["address_name"] = name
	result["address_phone"] = num
	result["address"] = address
	return result, nil
}

//获取奖品详情
func AwardDetail(uid uint32, id int) (result map[string]interface{}, e error) {
	result = make(map[string]interface{})
	var status, from, log_id, tp, cnum uint32
	var name, img, info, charge_phone string
	sql := "select a.`status`,`from`,charge_phone,log_id,name,b.`type`,img,info,cnum from award_record a LEFT JOIN award_config b on a.award_id=b.id where a.id=? and uid=?"
	err := mdb.QueryRow(sql, id, uid).Scan(&status, &from, &charge_phone, &log_id, &name, &tp, &img, &info, &cnum)
	if err != nil {
		return nil, err
	}
	result["status"] = status
	result["from"] = from
	result["type"] = tp

	result["img"] = img
	if tp == 5 {
		result["info"] = fmt.Sprintf("%v %v", cnum, info)
		result["name"] = fmt.Sprintf("%v %v", cnum, name)
	} else {
		result["info"] = info
		result["name"] = name
	}
	result["charge_phone"] = charge_phone
	switch status {
	case 0, 4, 5:
	case 2, 3: //等待充值（发货）
		switch tp {
		case 2: //实物
			if r, e := getAddr(log_id); e != nil {
				return nil, e
			} else {
				result["trans"] = r
			}
		case 3: //手机充值卡
			result["charge_phone"] = charge_phone
		}
	}
	return
}
