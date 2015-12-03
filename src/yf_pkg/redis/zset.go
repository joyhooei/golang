package redis

import (
	"errors"
	"fmt"
	"yf_pkg/utils"
)

type ItemScore struct {
	Key   string
	Score float64
}

func (rp *RedisPool) ZCount(db int, key interface{}, min, max float64) (count uint32, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	fmt.Println("ZCOUNT", key, min, max)
	n, e := Uint64(scon.Do("ZCOUNT", key, min, max))
	if e != nil {
		return 0, errors.New(fmt.Sprintf("ZCOUNT error: %v", e.Error()))
	}
	return uint32(n), nil
}

func (rp *RedisPool) ZCard(db int, key interface{}) (num uint64, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	return Uint64(scon.Do("ZCARD", key))
}

func (rp *RedisPool) ZRangeByScore(db int, key interface{}, min, max interface{}) (items []ItemScore, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	values, e := Values(scon.Do("ZRANGEBYSCORE", key, min, max, "WITHSCORES"))
	if e != nil {
		return nil, errors.New(fmt.Sprintf("ZRANGEBYSCORE error: %v", e.Error()))
	}
	if e = ScanSlice(values, &items); e != nil {
		return nil, errors.New(fmt.Sprintf("ScanSlice error: %v", e.Error()))
	}
	return
}

func (rp *RedisPool) ZREVRangeWithScoresPS(db int, key interface{}, cur int, ps int) (items []ItemScore, total int, e error) {
	return rp.zRangeWithScoresPS(db, key, cur, ps, false)
}
func (rp *RedisPool) ZRangeWithScoresPS(db int, key interface{}, cur int, ps int) (items []ItemScore, total int, e error) {
	return rp.zRangeWithScoresPS(db, key, cur, ps, true)
}
func (rp *RedisPool) ZREVRangeWithScores(db int, key interface{}, start, end int) (items []ItemScore, total int, e error) {
	return rp.zRangeWithScores(db, key, start, end, false)
}
func (rp *RedisPool) ZRangeWithScores(db int, key interface{}, start, end int) (items []ItemScore, total int, e error) {
	return rp.zRangeWithScores(db, key, start, end, true)
}

//分页获取SortedSet的ID集合
func (rp *RedisPool) ZRangePS(db int, key interface{}, cur int, ps int, asc bool, results interface{}) (total int, e error) {
	start, end := utils.BuildRange(cur, ps, total)
	return rp.ZRange(db, key, start, end, asc, results)
}

//获取SortedSet的ID集合
func (rp *RedisPool) ZRange(db int, key interface{}, start, end int, asc bool, results interface{}) (total int, e error) {
	cmd := "ZRANGE"
	if !asc {
		cmd = "ZREVRANGE"
	}
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	total, e = Int(scon.Do("ZCARD", key))
	if e != nil {
		return 0, errors.New(fmt.Sprintf("ZCARD error: %v", e.Error()))
	}
	values, e := Values(scon.Do(cmd, key, start, end))
	if e != nil {
		return 0, e
	}
	if e = ScanSlice(values, results); e != nil {
		return 0, errors.New(fmt.Sprintf("ScanSlice error: %v", e.Error()))
	}
	return
}

//分页获取带积分的SortedSet值
func (rp *RedisPool) zRangeWithScoresPS(db int, key interface{}, cur int, ps int, asc bool) (items []ItemScore, total int, e error) {
	if ps > 100 {
		ps = 100
	}
	start, end := utils.BuildRange(cur, ps, total)
	return rp.zRangeWithScores(db, key, start, end, asc)
}

//获取带积分的SortedSet值
func (rp *RedisPool) zRangeWithScores(db int, key interface{}, start, end int, asc bool) (items []ItemScore, total int, e error) {
	cmd := "ZRANGE"
	if !asc {
		cmd = "ZREVRANGE"
	}
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	total, e = Int(scon.Do("ZCARD", key))
	if e != nil {
		return nil, 0, errors.New(fmt.Sprintf("ZCARD error: %v", e.Error()))
	}
	items = make([]ItemScore, 0, 100)
	values, e := Values(scon.Do(cmd, key, start, end, "WITHSCORES"))
	if e != nil {
		return nil, 0, errors.New(fmt.Sprintf("%v error: %v", cmd, e.Error()))
	}
	if e = ScanSlice(values, &items); e != nil {
		return nil, 0, errors.New(fmt.Sprintf("ScanSlice error: %v", e.Error()))
	}
	return
}

/*
批量添加到sorted set类型的表中

	db: 数据库表ID
	args: 必须是<key,score,id>的列表
*/
func (rp *RedisPool) ZAdd(db int, args ...interface{}) error {
	if len(args)%3 != 0 {
		return errors.New("invalid arguments number")
	}
	fcon := rp.GetWriteConnection(db)
	defer fcon.Close()
	if e := fcon.Send("MULTI"); e != nil {
		return e
	}
	for i := 0; i < len(args); i += 3 {
		if e := fcon.Send("ZADD", args[i], args[i+1], args[i+2]); e != nil {
			fcon.Send("DISCARD")
			return e
		}
	}
	if _, e := fcon.Do("EXEC"); e != nil {
		fcon.Send("DISCARD")
		return e
	}
	return nil
}

/*
批量添加到sorted set类型的表中

	db: 数据库表ID
	opt: 可选参数，必须是NX|XX|CH|INCR|""中的一个
	args: 必须是<key,score,id>的列表
*/
func (rp *RedisPool) ZAddOpt(db int, opt string, args ...interface{}) error {
	if len(args)%3 != 0 {
		return errors.New("invalid arguments number")
	}
	fmt.Println("ZADD", opt, args)
	fcon := rp.GetWriteConnection(db)
	defer fcon.Close()
	if e := fcon.Send("MULTI"); e != nil {
		return e
	}
	for i := 0; i < len(args); i += 3 {
		if e := fcon.Send("ZADD", args[i], opt, args[i+1], args[i+2]); e != nil {
			fcon.Send("DISCARD")
			return e
		}
	}
	if _, e := fcon.Do("EXEC"); e != nil {
		return e
	}
	fmt.Println("success")
	return nil
}

/*
ZRem批量删除sorted set表中的元素

参数：
	db: 数据库表ID
	args: 必须是<key,id>的列表
返回值：
	affected: 每条命令影响的行数
*/
func (rp *RedisPool) ZRem(db int, args ...interface{}) (affected []int, e error) {
	if len(args)%2 != 0 {
		return nil, errors.New("invalid arguments number")
	}
	fcon := rp.GetWriteConnection(db)
	defer fcon.Close()
	if e := fcon.Send("MULTI"); e != nil {
		return nil, e
	}
	for i := 0; i < len(args); i += 2 {
		if e := fcon.Send("ZREM", args[i], args[i+1]); e != nil {
			fcon.Send("DISCARD")
			return nil, e
		}
	}
	if replies, e := Values(fcon.Do("EXEC")); e != nil {
		return nil, e
	} else {
		affected = []int{}
		if e := ScanSlice(replies, &affected); e != nil {
			return nil, e
		}
	}
	return
}

func (rp *RedisPool) ZExists(db int, key interface{}, id interface{}) (bool, error) {
	conn := rp.GetReadConnection(db)
	defer conn.Close()
	_, e := Int64(conn.Do("ZSCORE", key, id))
	switch e {
	case nil:
		return true, nil
	case ErrNil:
		return false, nil
	default:
		return false, e
	}
}

func (rp *RedisPool) ZScore(db int, key interface{}, item interface{}) (score int64, e error) {
	conn := rp.GetReadConnection(db)
	defer conn.Close()
	return Int64(conn.Do("ZSCORE", key, item))
}

//批量获取有序集合的元素的得分
func (rp *RedisPool) ZMultiScore(db int, key interface{}, items ...interface{}) (scores map[interface{}]int64, e error) {
	conn := rp.GetReadConnection(db)
	defer conn.Close()
	for _, id := range items {
		if e := conn.Send("ZSCORE", key, id); e != nil {
			return nil, e
		}
	}
	conn.Flush()
	scores = make(map[interface{}]int64, len(items))
	for _, id := range items {
		score, e := Int64(conn.Receive())
		switch e {
		case nil:
			scores[id] = score
		case ErrNil:
		default:
			return nil, e
		}
	}
	return scores, nil
}

//批量判断是否是有序集合中的元素
func (rp *RedisPool) ZMultiIsMember(db int, key interface{}, items map[interface{}]bool) error {
	conn := rp.GetReadConnection(db)
	defer conn.Close()
	ids := make([]interface{}, 0, len(items))
	for id, _ := range items {
		if e := conn.Send("ZSCORE", key, id); e != nil {
			return e
		}
		ids = append(ids, id)
	}
	conn.Flush()
	for _, id := range ids {
		_, e := Int64(conn.Receive())
		switch e {
		case nil:
			items[id] = true
		case ErrNil:
			items[id] = false
		default:
			return e
		}
	}
	return nil
}

/*
根据score 获取有序集 ZREVRANGEBYSCORE min <=score < max  按照score 从大到小排序, ps 获取条数
*/
func (rp *RedisPool) ZREVRangeByScoreWithScores(db int, key interface{}, min, max int64, ps int) (items []ItemScore, e error) {
	return rp.zRangeByScoreWithScores(db, key, min, max, ps, false)
}

/*
根据score 获取有序集 ZRANGEBYSCORE min <score <= max  按照score 从小到大排序, ps 获取条数
*/
func (rp *RedisPool) ZRangeByScoreWithScores(db int, key interface{}, min, max int64, ps int) (items []ItemScore, e error) {
	return rp.zRangeByScoreWithScores(db, key, min, max, ps, true)
}

//根据积分的SortedSet值
func (rp *RedisPool) zRangeByScoreWithScores(db int, key interface{}, min, max int64, ps int, asc bool) (items []ItemScore, e error) {
	var s1, s2, cmd string
	if asc {
		s1 = "(" + utils.ToString(min)
		s2 = utils.ToString(max)
		cmd = "ZRANGEBYSCORE"
	} else {
		s1 = "(" + utils.ToString(max)
		s2 = utils.ToString(min)
		cmd = "ZREVRANGEBYSCORE"
	}
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	items = make([]ItemScore, 0, 100)
	values, e := Values(scon.Do(cmd, key, s1, s2, "WITHSCORES", "LIMIT", 0, ps))
	fmt.Println(s1, s2, key, ps)
	if e != nil {
		return nil, errors.New(fmt.Sprintf("%v error: %v", cmd, e.Error()))
	}
	if e = ScanSlice(values, &items); e != nil {
		return nil, errors.New(fmt.Sprintf("ScanSlice error: %v", e.Error()))
	}
	return
}

/*
移除有序集 key 中，所有 score 值介于 min 和 max 之间(包括等于 min 或 max )的成员
*/
func (rp *RedisPool) ZRemRangeByScore(db int, key interface{}, min, max int64) error {
	conn := rp.GetWriteConnection(db)
	defer conn.Close()
	_, e := conn.Do("ZREMRANGEBYSCORE", key, min, max)
	return e
}

/*
合并多个有序集合，其中权重weights 默认为1 ，AGGREGATE 默认使用sum
ZUNIONSTORE destination numkeys key [key ...] [WEIGHTS weight [weight ...]] [AGGREGATE SUM|MIN|MAX]
dest_key：合并目标key
keys: 带合并的keys集合 <key> 的列表
expire: 有效时间 （秒值）
aggregate: 聚合方式： SUM | MIN | MAX
*/
func (rp *RedisPool) ZUnionSrore(db int, dest_key interface{}, expire int, keys []interface{}, weights []interface{}, aggregate string) error {
	if len(keys) != len(weights) || len(keys) <= 0 {
		return errors.New("invalid numbers of keys and weights")
	}
	args := make([]interface{}, 0, 2*len(keys)+10)
	args = append(args, dest_key, len(keys))
	args = append(args, keys...)
	args = append(args, "WEIGHTS")
	args = append(args, weights...)
	args = append(args, "AGGREGATE", aggregate)
	conn := rp.GetWriteConnection(db)
	fmt.Println("ZUnionSrore : ", args)
	defer conn.Close()
	if _, e := conn.Do("ZUNIONSTORE", args...); e != nil {
		return e
	}
	_, e := conn.Do("EXPIRE", dest_key, expire)
	return e
}

//ZIsMember判断是否是有序集合的成员
func (rp *RedisPool) ZIsMember(db int, key interface{}, item interface{}) (isMember bool, e error) {
	conn := rp.GetReadConnection(db)
	defer conn.Close()
	_, e = Int64(conn.Do("ZSCORE", key, item))
	switch e {
	case nil:
		return true, nil
	case ErrNil:
		return false, nil
	default:
		return false, e
	}
}
