package building

import (
	"fmt"
	"yf_pkg/cachedb"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
)

var mdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *cachedb.CacheDB

func Init(env *cls.CustomEnv) (e error) {
	mdb = env.MainDB
	rdb = env.MainRds
	cache = env.CacheDB
	return
}

//获取建筑信息，如果某个ID的建筑信息找不到，则对应的值为nil。
func GetBuildings(ids ...string) (buildings map[string]*Building, e error) {
	if len(ids) == 0 {
		return
	}
	objs := make(map[interface{}]cachedb.DBObject)
	for _, v := range ids {
		objs[v] = nil
	}
	e = cache.GetMap(objs, NewBuilding)
	if e != nil {
		return nil, e
	} else {
		buildings := make(map[string]*Building)
		for id, building := range objs {
			if building != nil {
				buildings[utils.ToString(id)] = building.(*Building)
			}
		}
	}
	return
}

//获取建筑信息，如果某个ID的建筑信息找不到，则对应的值不变。
func GetBuildingMap(buildings map[string]*Building) (e error) {
	if len(buildings) == 0 {
		return
	}
	objs := make(map[interface{}]cachedb.DBObject)
	for v, _ := range buildings {
		objs[v] = nil
	}
	e = cache.GetMap(objs, NewBuilding)
	if e != nil {
		return e
	} else {
		for id, building := range objs {
			fmt.Println("id:", id, "building:", building)
			if building != nil {
				buildings[utils.ToString(id)] = building.(*Building)
			}
		}
	}
	return
}
