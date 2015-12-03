package redis

import "errors"

func (rp *RedisPool) SAdd(db int, key interface{}, values ...interface{}) (e error) {
	args := make([]interface{}, len(values)+1)
	args[0] = key
	copy(args[1:], values)
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	_, e = scon.Do("SADD", args...)
	return
}

/*
批量添加到set类型的表中

	db: 数据库表ID
	args: 必须是<key,id>的列表
*/
func (rp *RedisPool) SMultiAdd(db int, args ...interface{}) error {
	if len(args)%2 != 0 {
		return errors.New("invalid arguments number")
	}
	fcon := rp.GetWriteConnection(db)
	defer fcon.Close()
	if e := fcon.Send("MULTI"); e != nil {
		return e
	}
	for i := 0; i < len(args); i += 2 {
		if e := fcon.Send("SADD", args[i], args[i+1]); e != nil {
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
批量删除set类型表中的元素

	db: 数据库表ID
	args: 必须是<key,id>的列表
*/
func (rp *RedisPool) SMultiRem(db int, args ...interface{}) error {
	if len(args)%2 != 0 {
		return errors.New("invalid arguments number")
	}
	fcon := rp.GetWriteConnection(db)
	defer fcon.Close()
	if e := fcon.Send("MULTI"); e != nil {
		return e
	}
	for i := 0; i < len(args); i += 2 {
		if e := fcon.Send("SREM", args[i], args[i+1]); e != nil {
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

func (rp *RedisPool) SRem(db int, key interface{}, values ...interface{}) (e error) {
	args := make([]interface{}, len(values)+1)
	args[0] = key
	copy(args[1:], values)
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	_, e = scon.Do("SREM", args...)
	return
}

func (rp *RedisPool) SIsMember(db int, key interface{}, value interface{}) (isMember bool, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	return Bool(scon.Do("SISMEMBER", key, value))
}

/*
SMembers获取某个key下的所有元素

参数：
	values: 必须是数组的引用
*/
func (rp *RedisPool) SMembers(db int, values interface{}, key interface{}) (e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	vs, e := Values(scon.Do("SMEMBERS", key))
	if e != nil {
		return e
	}
	return ScanSlice(vs, values)
}
