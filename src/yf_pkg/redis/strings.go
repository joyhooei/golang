package redis

import "errors"

func (rp *RedisPool) Get(db int, key interface{}) (value interface{}, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	return scon.Do("GET", key)
}

func (rp *RedisPool) Set(db int, key interface{}, value interface{}) (e error) {
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	_, e = scon.Do("SET", key, value)
	return
}

func (rp *RedisPool) SetEx(db int, key interface{}, seconds int, value interface{}) (e error) {
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	_, e = scon.Do("SETEX", key, seconds, value)
	return
}

func (rp *RedisPool) Incr(db int, key interface{}) (int64, error) {
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	return Int64(scon.Do("INCR", key))
}

func (rp *RedisPool) IncrBy(db int, key interface{}, value interface{}) (int64, error) {
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	return Int64(scon.Do("INCRBY", key, value))
}

// 批量获取
func (rp *RedisPool) MGet(db int, keys ...interface{}) (value interface{}, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	return scon.Do("MGET", keys...)
}

/*
批量设置
< key value > 序列
*/
func (rp *RedisPool) MSet(db int, kvs ...interface{}) (value interface{}, e error) {
	if len(kvs)%2 != 0 {
		return nil, errors.New("invalid arguments number")
	}
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	return scon.Do("MSET", kvs...)
}
