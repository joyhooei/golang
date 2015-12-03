package tag

import (
	"errors"
	"fmt"
	"yf_pkg/format"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

const (
	TAG_LIST_TIMEOUT = 3600 //标签列表的超时时间
	TAG_LIST_MAXLEN  = 500  //标签列表的最大长度
)

var mdb *mysql.MysqlDB
var cache *redis.RedisPool
var mainLog *log.MLogger

func Init(env *cls.CustomEnv) (e error) {
	mdb = env.MainDB
	cache = env.CacheRds
	mainLog = env.MainLog

	return e
}
func key(tType string, gender int) string {
	return fmt.Sprintf("%v_%v", tType, gender)
}

func getGender(uid uint32) (int, error) {
	uinfos, e := user_overview.GetUserObjects(uid)
	if e != nil {
		return 0, errors.New(fmt.Sprintf("Get uid %v info error :%v", uid, e.Error()))
	}
	gender := common.GENDER_BOTH
	uinfo := uinfos[uid]
	if uinfo != nil {
		gender = uinfo.Gender
	}
	return gender, nil
}

func UseTag(uid uint32, tType string, tag string) error {
	return nil
	/*
		gender, e := getGender(uid)
		if e != nil {
			return e
		}
		if e := useTag(gender, tType, tag); e != nil {
			return e
		}
		if e := useTag(common.GENDER_BOTH, tType, tag); e != nil {
			return e
		}
		return nil
	*/
}
func useTag(gender int, tType string, tag string) error {
	weight := 2
	if gender == common.GENDER_BOTH {
		weight = 1
	}
	tx, e := mdb.Begin()
	if e != nil {
		return e
	}
	sql := "select cnt,begin_tm from tag where type=? and name=? and gender=?"
	rows, e := tx.Query(sql, tType, tag, gender)
	if e != nil {
		tx.Rollback()
		return e
	}
	var cnt int
	var beginTmStr string
	updateTmStr := utils.Now.Format(format.TIME_LAYOUT_1)
	if rows.Next() {
		e = rows.Scan(&cnt, &beginTmStr)
		if e != nil {
			tx.Rollback()
			return e
		}
		rows.Close()
		value := cnt % 100
		beginTm, _ := utils.ToTime(beginTmStr, format.TIME_LAYOUT_1)
		hours := (utils.Now.Sub(beginTm).Minutes() + 1.0) / 60.0
		if hours <= 0 {
			tx.Rollback()
			return errors.New("now - begin_tm < 0, check whether begin_tm is valid?")
		}
		score := float64(value) / hours //score的分数实际上就是该标签一周被使用的次数
		var sql string
		switch {
		case value == 99:
			sql = fmt.Sprintf("update tag set cnt=%v,begin_tm='%v',update_tm='%v',score=%v where type=? and name=? and gender=?", cnt+weight, updateTmStr, updateTmStr, score)
		case value >= 5:
			sql = fmt.Sprintf("update tag set cnt=%v,update_tm='%v',score=%v where type=? and name=? and gender=?", cnt+weight, updateTmStr, score)
		default:
			sql = fmt.Sprintf("update tag set cnt=%v,update_tm='%v' where type=? and name=? and gender=?", cnt+weight, updateTmStr)
		}
		if _, e = tx.Exec(sql, tType, tag, gender); e != nil {
			tx.Rollback()
			return e
		}
	} else {
		if tag == "" {
			tx.Rollback()
			return nil
		}
		sql := fmt.Sprintf("insert into tag(type,name,gender,cnt,begin_tm,update_tm,score)values(?,?,?,1,'%v','%v',0)", updateTmStr, updateTmStr)
		if _, e = tx.Exec(sql, tType, tag, gender); e != nil {
			tx.Rollback()
			return e
		}
	}
	return tx.Commit()
}

func GetTags(uid uint32, tType string, cur int, ps int) (tags []string, total int, e error) {
	gender, e := getGender(uid)
	if e != nil {
		return nil, 0, e
	}

	tag_key := key(tType, gender)
	exists, e := cache.Exists(redis_db.CACHE_TAG, tag_key)
	if e != nil {
		return nil, 0, e
	}
	if exists {
		conn := cache.GetReadConnection(redis_db.CACHE_TAG)
		defer conn.Close()
		total, e = redis.Int(conn.Do("LLEN", tag_key))
		if e != nil {
			return nil, 0, e
		}
		begin, end := utils.BuildRange(cur, ps, total)
		values, e := redis.Values(conn.Do("LRANGE", tag_key, begin, end))
		if e != nil {
			return nil, 0, e
		}
		if e = redis.ScanSlice(values, &tags); e != nil {
			return nil, 0, e
		}
	} else {
		sql := "select distinct name from tag where type=? and gender" + mysql.In([]int{common.GENDER_BOTH, gender}) + " order by score desc limit ?"
		rows, e := mdb.Query(sql, tType, TAG_LIST_MAXLEN)
		if e != nil {
			return nil, 0, e
		}
		defer rows.Close()
		tagsWithKey := make([]interface{}, 0, TAG_LIST_MAXLEN+1)
		name := ""
		tagsWithKey = append(tagsWithKey, tag_key)
		for rows.Next() {
			e = rows.Scan(&name)
			if e != nil {
				return nil, 0, e
			}
			tagsWithKey = append(tagsWithKey, name)
		}
		total = len(tagsWithKey) - 1
		conn := cache.GetWriteConnection(redis_db.CACHE_TAG)
		defer conn.Close()
		if _, e = conn.Do("DEL", tag_key); e != nil {
			return nil, 0, e
		}
		if _, e = conn.Do("RPUSH", tagsWithKey...); e != nil {
			return nil, 0, e
		}
		if _, e = conn.Do("EXPIRE", tag_key, TAG_LIST_TIMEOUT); e != nil {
			return nil, 0, e
		}
		begin, end := utils.BuildRange(cur, ps, total)
		tags = make([]string, 0, ps)
		for i := begin; i <= end; i++ {
			tags = append(tags, tagsWithKey[i+1].(string))
		}
	}
	return
}
