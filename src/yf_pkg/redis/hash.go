package redis

import (
	"errors"
	"fmt"
)

/*
HSet批量设置HashSet中的值

	db: 数据库表ID
	args: 必须是<key,id,value>的列表
*/
func (rp *RedisPool) HMultiSet(db int, args ...interface{}) (e error) {
	if len(args)%3 != 0 {
		return errors.New("invalid arguments number")
	}
	fcon := rp.GetWriteConnection(db)
	defer fcon.Close()
	if e := fcon.Send("MULTI"); e != nil {
		return e
	}
	for i := 0; i < len(args); i += 3 {
		if e := fcon.Send("HSET", args[i], args[i+1], args[i+2]); e != nil {
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

func (rp *RedisPool) HSet(db int, key interface{}, id interface{}, value interface{}) (e error) {
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	_, e = scon.Do("HSET", key, id, value)
	return
}

func (rp *RedisPool) HGet(db int, key interface{}, name interface{}) (value interface{}, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	fmt.Println("HGET", key, name)
	return scon.Do("HGET", key, name)
}

/*
HMGet针对同一个key获取hashset中的部分元素的值

参数：
	args: 第一个值必须是key，后续的值都是id
	values: 必须是数组的引用，如果某个id不存在，会把对应数据类型的零值放在数组对应位置上
*/
func (rp *RedisPool) HMGet(db int, values interface{}, key interface{}, ids ...interface{}) (e error) {
	if len(ids) == 0 {
		return
	}
	args := make([]interface{}, len(ids)+1)
	args[0] = key
	copy(args[1:], ids)
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	fmt.Println("HMGET", args)
	vs, e := Values(scon.Do("HMGET", args...))
	if e != nil {
		return e
	}
	return ScanSlice(vs, values)
}

/*
HMultiGet批量获取HashSet中多个key中ID的值

参数：
	db: 数据库表ID
	args: 必须是<key,id>的列表
返回值：
	values: 一个两层的map，第一层的key是参数中的key，第二层的key是参数中的id
*/
func (rp *RedisPool) HMultiGet(db int, args ...interface{}) (values map[interface{}]map[interface{}]interface{}, e error) {
	if len(args)%2 != 0 {
		return nil, errors.New("invalid arguments number")
	}
	conn := rp.GetReadConnection(db)
	defer conn.Close()
	for i := 0; i < len(args); i += 2 {
		if e := conn.Send("HGET", args[i], args[i+1]); e != nil {
			return nil, e
		}
	}
	conn.Flush()
	values = make(map[interface{}]map[interface{}]interface{}, len(args))
	for i := 0; i < len(args); i += 2 {
		v, e := conn.Receive()
		switch e {
		case nil:
			idm, ok := values[args[i]]
			if !ok {
				idm = make(map[interface{}]interface{})
				values[args[i]] = idm
			}
			idm[args[i+1]] = v
		case ErrNil:
		default:
			return nil, e
		}
	}
	return values, nil
}

/*
HDel批量删除某个Key中的元素
	args: 第一个必须是key，后面的都是id
*/
func (rp *RedisPool) HDel(db int, args ...interface{}) (e error) {
	if len(args) <= 1 {
		return nil
	}
	scon := rp.GetWriteConnection(db)
	defer scon.Close()
	_, e = scon.Do("HDEL", args...)
	return
}

/*
HGetAll针对同一个key获取hashset中的所有元素的值
*/
func (rp *RedisPool) HGetAll(db int, key interface{}) (reply interface{}, e error) {
	scon := rp.GetReadConnection(db)
	defer scon.Close()
	return scon.Do("HGETALL", key)
}

/*
MHGetAll批量获取多个key所有的字段
*/
func (rp *RedisPool) MHGetAll(db int, args ...interface{}) (reply map[interface{}][]interface{}, e error) {
	fcon := rp.GetReadConnection(db)
	defer fcon.Close()
	for _, key := range args {
		if e = fcon.Send("HGETALL", key); e != nil {
			return
		}
	}
	fcon.Flush()
	reply = make(map[interface{}][]interface{})
	for _, key := range args {
		r, e := Values(fcon.Receive())
		if e != nil {
			return reply, e
		}
		reply[key] = r
	}
	return
}
