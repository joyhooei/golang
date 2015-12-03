//带redis缓存的数据库对象操作模块
//使用者不需要对缓存进行管理，只需要实现从mysql读写数据的接口即可
package cachedb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"yf_pkg/mysql"
	"yf_pkg/redis"
)

type DBObject interface {
	//返回对象的ID，如果没有设置ID，则第二个返回值为false
	ID() (interface{}, bool)
	//缓存超时时间(秒)，-1表示不超时，0表示默认值10分钟
	Expire() int

	//新增或更新数据
	Save(mysqldb *mysql.MysqlDB) (id interface{}, e error)
	//获取数据内容
	Get(id interface{}, mysqldb *mysql.MysqlDB) (e error)
	//批量从数据库取数据
	GetMap(ids []interface{}, mysqldb *mysql.MysqlDB) (objs map[interface{}]DBObject, e error)
}

type CacheDB struct {
	mysqldb *mysql.MysqlDB
	cache   *redis.RedisPool
	rdbid   int
}

func fullID(id interface{}, typ reflect.Type) string {
	return fmt.Sprintf("%v_%v", typ, id)
}

//新建CacheDB对象
func New(mysqldb *mysql.MysqlDB, cache *redis.RedisPool, rdbid int) *CacheDB {
	c := &CacheDB{mysqldb, cache, rdbid}
	return c
}

//新增或更新数据，如果redis里有缓存，也一并更新
func (db *CacheDB) Save(data DBObject) (id interface{}, e error) {
	id, e = data.Save(db.mysqldb)
	if e != nil {
		return nil, e
	}
	e = db.delRedis(id, reflect.TypeOf(data))
	return id, e
}

//清除缓存
func (db *CacheDB) ClearCache(obj DBObject) (e error) {
	id, ok := obj.ID()
	if ok {
		e = db.delRedis(id, reflect.TypeOf(obj))
	}
	return e
}

//读取数据，redis里有则从redis读取，否则从mysql读取，同时放入redis缓存
func (db *CacheDB) Get(id interface{}, data DBObject) (e error) {
	e = db.getRedis(id, data)
	if e != nil {
		e = data.Get(id, db.mysqldb)
		if e != nil {
			return e
		}
		e = db.saveRedis(id, data)
		return e
	}
	return nil
}

//批量读取数据，策略与Get一样
//newDBObject函数用来生成用户自定义的DBObject
func (db *CacheDB) GetMap(data map[interface{}]DBObject, newDBObject func(id interface{}) DBObject) (e error) {
	noValues, e := db.getRedisMap(data, newDBObject)
	if e != nil {
		return e
	}
	if len(noValues) > 0 {
		objects, e := newDBObject(nil).GetMap(noValues, db.mysqldb)
		if e != nil {
			return e
		}
		e = db.saveRedisMap(objects)
		if e == nil {
			for key, value := range objects {
				data[key] = value
			}
		}
		return e
	}
	return nil
}

func (db *CacheDB) getRedis(id interface{}, data DBObject) (e error) {
	fID := fullID(id, reflect.TypeOf(data))
	conn := db.cache.GetReadConnection(db.rdbid)
	defer conn.Close()
	b, e := redis.Bytes(conn.Do("GET", fID))
	if e != nil {
		return e
	}
	e = json.Unmarshal(b, data)
	return e
}

func (db *CacheDB) getRedisMap(data map[interface{}]DBObject, newDBObject func(id interface{}) DBObject) (noValues []interface{}, e error) {
	if len(data) == 0 {
		return
	}
	ids := make([]interface{}, 0, 10)
	fids := make([]interface{}, 0, 10)
	typ := reflect.TypeOf(newDBObject(nil))
	for key, _ := range data {
		fids = append(fids, fullID(key, typ))
		ids = append(ids, key)
	}
	conn := db.cache.GetReadConnection(db.rdbid)
	defer conn.Close()
	values, e := redis.Values(conn.Do("MGET", fids...))
	if e != nil {
		return nil, e
	}
	noValues = make([]interface{}, 0, 10)
	for index, id := range ids {
		if values[index] == nil {
			noValues = append(noValues, id)
		} else {
			obj := newDBObject(id)
			b, e := redis.Bytes(values[index], nil)
			if e != nil {
				return nil, e
			}
			e = json.Unmarshal(b, &obj)
			if e != nil {
				return nil, e
			}
			data[id] = obj
		}
	}
	return noValues, e
}
func (db *CacheDB) saveRedisMap(data map[interface{}]DBObject) (e error) {
	for id, value := range data {
		e = db.saveRedis(id, value)
		if e != nil {
			return e
		}
	}
	return e
}

func (db *CacheDB) saveRedis(id interface{}, data DBObject) (e error) {
	fID := fullID(id, reflect.TypeOf(data))
	j, e := json.Marshal(data)
	conn := db.cache.GetWriteConnection(db.rdbid)
	defer conn.Close()
	expire := data.Expire()
	if expire == 0 {
		expire = 600
	}
	_, e = conn.Do("SETEX", fID, expire, j)
	return
}

func (db *CacheDB) delRedis(id interface{}, typ reflect.Type) (e error) {
	fID := fullID(id, typ)
	conn := db.cache.GetWriteConnection(db.rdbid)
	defer conn.Close()
	_, e = conn.Do("DEL", fID)
	return
}
