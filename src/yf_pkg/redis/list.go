package redis

//批量插入队尾
func (rp *RedisPool) RPush(db int, key interface{}, values ...interface{}) (value interface{}, e error) {
	if len(values) == 0 {
		return
	}
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	vs := []interface{}{key}
	vs = append(vs, values...)
	return scon.Do("RPUSH", vs...)
}

/*
获取队列数据
*/
func (rp *RedisPool) LRange(db int, key interface{}, start, stop interface{}) (value interface{}, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	return scon.Do("LRANGE", key, start, stop)
}

/*
获取队列长度，如果key不存在，length=0，不会报错。
*/
func (rp *RedisPool) LLen(db int, key interface{}) (length int, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	return Int(scon.Do("LLEN", key))
}
