package dynamics

import (
	"encoding/json"
	"fmt"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
)

const (
	CACAHE_KEY_DYNAMIC_ARTICLE = "cache_dynamic_article" // 文章动态缓存key 60 秒
)

func makeDynamicCachaeKey(id uint32) string {
	return "cache_dynamic_" + utils.ToString(id)
}

// 清除动态缓存
func ClearDynamicCache(id uint32) (e error) {
	e = cache.Del(redis_db.CACHE_DYNAMIC, makeDynamicCachaeKey(id))
	return
}

// 获取动态缓存
func readDynamicCache(ids []uint32) (dy map[uint32]Dynamic, un []uint32, e error) {
	un = make([]uint32, 0, 10)
	if len(ids) <= 0 {
		un = ids
		return
	}
	keys := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, makeDynamicCachaeKey(id))
	}
	arr, e := redis.Strings(cache.MGet(redis_db.CACHE_DYNAMIC, keys...))
	if e != nil {
		return
	}
	dy = make(map[uint32]Dynamic)
	for k, s := range arr {
		if s != "" {
			var d Dynamic
			if e = json.Unmarshal([]byte(s), &d); e == nil && d.Id > 0 {
				dy[d.Id] = d
				continue
			} else {
				mlog.AppendObj(e, "ReadDynamicCache json is error", s)
			}
		}
		un = append(un, ids[k])
	}
	return
}

// 写入动态缓存
func writeDymamicCache(arr []Dynamic) (e error) {
	if len(arr) <= 0 {
		return
	}
	keys := make([]interface{}, 0, len(arr))
	kv := make([]interface{}, 0, 2*len(arr))
	for _, dy := range arr {
		key := makeDynamicCachaeKey(dy.Id)
		b, e := json.Marshal(dy)
		if e != nil {
			mlog.AppendObj(e, "writeDymamicCache is error ", dy, dy.Id)
			continue
		}
		v := string(b)
		keys = append(keys, key)
		kv = append(kv, key, v)
	}
	if len(kv) <= 0 {
		return
	}
	if _, e = cache.MSet(redis_db.CACHE_DYNAMIC, kv...); e != nil {
		return
	}
	e = cache.MultiExpire(redis_db.CACHE_DYNAMIC, 3600, keys...)
	fmt.Println("set cache data ", keys, kv)
	return
}

// 获取动态缓存
func readDynamicArticleCache() (d Dynamic, e error) {
	if ok, e := cache.Exists(redis_db.CACHE_DYNAMIC, CACAHE_KEY_DYNAMIC_ARTICLE); !ok || e != nil {
		return d, e
	}
	js, e := redis.String(cache.Get(redis_db.CACHE_DYNAMIC, CACAHE_KEY_DYNAMIC_ARTICLE))
	if e != nil || js == "" {
		return
	}
	e = json.Unmarshal([]byte(js), &d)
	return
}

// 写入文章缓存
func writeDymamicArticleCache(d Dynamic) (e error) {
	b, e := json.Marshal(d)
	if e != nil {
		return
	}
	if e = cache.Set(redis_db.CACHE_DYNAMIC, CACAHE_KEY_DYNAMIC_ARTICLE, string(b)); e != nil {
		return
	}
	e = cache.Expire(redis_db.CACHE_DYNAMIC, 60, CACAHE_KEY_DYNAMIC_ARTICLE)
	return
}
