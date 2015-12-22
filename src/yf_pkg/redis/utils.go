package redis

import redigo "github.com/garyburd/redigo/redis"

func Values(reply interface{}, err error) ([]interface{}, error) {
	return redigo.Values(reply, err)
}
func Scan(src []interface{}, dest ...interface{}) ([]interface{}, error) {
	return redigo.Scan(src, dest...)
}

func ScanSlice(src []interface{}, dest interface{}, fieldNames ...string) error {
	return redigo.ScanSlice(src, dest, fieldNames...)
}
func ScanStruct(src []interface{}, dest interface{}) error {
	return redigo.ScanStruct(src, dest)
}
func Bytes(reply interface{}, err error) ([]byte, error) {
	return redigo.Bytes(reply, err)
}
func String(reply interface{}, err error) (string, error) {
	return redigo.String(reply, err)
}
func Strings(reply interface{}, err error) ([]string, error) {
	return redigo.Strings(reply, err)
}
func Bool(reply interface{}, err error) (bool, error) {
	return redigo.Bool(reply, err)
}
func Int(reply interface{}, err error) (int, error) {
	return redigo.Int(reply, err)
}
func Int64(reply interface{}, err error) (int64, error) {
	return redigo.Int64(reply, err)
}
func Uint64(reply interface{}, err error) (uint64, error) {
	return redigo.Uint64(reply, err)
}
func Uint32(reply interface{}, err error) (uint32, error) {
	v, e := redigo.Uint64(reply, err)
	if e != nil {
		return 0, e
	}
	return uint32(v), nil
}

func Float64(reply interface{}, err error) (float64, error) {
	return redigo.Float64(reply, err)
}

func Int64Map(reply interface{}, err error) (map[string]int64, error) {
	return redigo.Int64Map(reply, err)
}

func (rp *RedisPool) Exists(db int, key interface{}) (bool, error) {
	conn := rp.GetReadConnection(db)
	defer conn.Close()
	return redigo.Bool(conn.Do("EXISTS", key))
}

func (rp *RedisPool) Del(db int, key interface{}) error {
	conn := rp.GetWriteConnection(db)
	defer conn.Close()
	_, e := conn.Do("DEL", key)
	return e
}

/*
设置key的有效时间
*/
func (rp *RedisPool) Expire(db, expire int, key interface{}) (e error) {
	conn := rp.GetWriteConnection(db)
	defer conn.Close()
	_, e = conn.Do("EXPIRE", key, expire)
	return

}

/*
MultiExpire 批量设置key的有效时间

	db: 数据库表ID
	expire:缓存失效时间(秒值)
	args:key的列表
*/
func (rp *RedisPool) MultiExpire(db, expire int, args ...interface{}) (e error) {
	if len(args) <= 0 {
		return
	}
	fcon := rp.GetWriteConnection(db)
	defer fcon.Close()
	if e := fcon.Send("MULTI"); e != nil {
		return e
	}
	for _, key := range args {
		if e := fcon.Send("EXPIRE", key, expire); e != nil {
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

func (rp *RedisPool) Multi(db int, cmd func(con redigo.Conn) error) error {
	fcon := rp.GetWriteConnection(db)
	defer fcon.Close()
	if e := fcon.Send("MULTI"); e != nil {
		return e
	}
	if e := cmd(fcon); e != nil {
		fcon.Send("DISCARD")
		return e
	}
	if _, e := fcon.Do("EXEC"); e != nil {
		return e
	}
	return nil
}
