package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"pkg/yh_config"
	"pkg/yh_format"
	"pkg/yh_log"
	"pkg/yh_redigo/redis"
	"pkg/yh_utils"
	"runtime"
	"strconv"
	"time"
	"yuanfen/location/common"

	"container/heap"
)

const (
	REDIS_DB_ZONES    = 1                      //按区域存储的元素集合
	REDIS_DB_ELEMENTS = 2                      //元素的具体位置
	MAX_RADIUS        = 1                      //最大搜索半径
	BIG_ZONE_SIZE     = 3600                   //大区域大小(秒)
	BIG_ZONE_FACTOR   = 3600 / BIG_ZONE_SIZE   //大区域因子
	SMALL_ZONE_SIZE   = 100                    //小区域大小(秒)
	SMALL_ZONE_FACTOR = 3600 / SMALL_ZONE_SIZE //小区域因子
	ONLINE_TIMEOUT    = 3600 * 24 * 7          //在线超时时间（秒）
	LOG_INTERVAL      = 30                     //状态日志输出间隔（秒）
	MAX_ACTIVE_KEYS   = 20000                  //最多存储的待清理的key的数量
)

var keywords = map[string]bool{
	"listen":                true,
	"redis_server":          true,
	"redis_max_connections": true,
	"log":       true,
	"log_level": true,
	"procs":     true,
}

//连接池最大连接数
var maxActiveConn int

//检查配置文件是否合法
func checkConfig(conf *yh_config.Config) error {
	for key := range keywords {
		if _, found := conf.Items[key]; found == false {
			return errors.New("not found key [" + key + "]")
		}
	}
	return nil
}

func logStatus() {
	for {
		glog.Append(fmt.Sprintf("Zone redis pool >>>> active:%v, max:%v", poolZone.ActiveCount(), maxActiveConn), yh_log.NOTICE)
		glog.Append(fmt.Sprintf("Element redis pool >>>> active:%v, max:%v", poolElem.ActiveCount(), maxActiveConn), yh_log.NOTICE)
		glog.Append(fmt.Sprintf("need clear keys >>>> current:%v, max:%v", len(activeKeys), MAX_ACTIVE_KEYS), yh_log.NOTICE)
		time.Sleep(LOG_INTERVAL * time.Second)
	}
}

var activeKeys map[string]bool = make(map[string]bool, MAX_ACTIVE_KEYS*2)

func AddActiveKeys(key string) {
	if len(activeKeys) < MAX_ACTIVE_KEYS {
		activeKeys[key] = true
	}
}

func clearOld() {
	for {
		rc := poolZone.Get()
		i := 0
		for key, _ := range activeKeys {
			_, e := rc.Do("ZREMRANGEBYSCORE", key, "-inf", yh_utils.Now.Unix())
			if e != nil {
				err := common.NewError(common.ERR_REDIS, fmt.Sprintf("redis ZREMRANGEBYSCORE error : %v", e.Error()))
				glog.Append(err.Error())
				break
			}
			delete(activeKeys, key)
			if i++; i > 10 {
				break
			}
		}
		rc.Close()
		time.Sleep(1 * time.Second)
	}
}

func newPool(server string, maxActiveConn int, db int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   maxActiveConn,
		IdleTimeout: 240 * time.Second,
		Db:          db,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

var (
	poolZone *redis.Pool
	poolElem *redis.Pool
	glog     *yh_log.Logger
	levelStr map[string]int = map[string]int{"error": yh_log.ERROR, "debug": yh_log.DEBUG, "notice": yh_log.NOTICE, "warn": yh_log.WARN}
	config                  = flag.String("config", "../conf/location.conf", "config file path.")
)

func writeBack(r *http.Request, w http.ResponseWriter, result map[string]interface{}) {
	re := string(yh_format.GenerateJSON(result))
	w.Write([]byte(re))
	log := fmt.Sprintf("request : %s	response : %s", r.URL.String(), re)
	if result["status"] == "fail" {
		glog.Append(log)
	} else {
		glog.Append(log, yh_log.NOTICE)
	}
}
func writeBackErr(r *http.Request, w http.ResponseWriter, err common.Error) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"
	result["ecode"] = err.Code
	result["edesc"] = err.Desc
	writeBack(r, w, result)
}

//用户汇报自己的地理位置
func report(w http.ResponseWriter, r *http.Request) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"

	ids, found := r.URL.Query()["id"]

	if !found {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "no [id] provided."))
		return
	}
	latStr, found := r.URL.Query()["lat"]
	if !found {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "no [lat] provided."))
		return
	}
	lat, e := yh_utils.StringToFloat64(latStr[0])
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_LAT, fmt.Sprintf("Parse [lat] error : %v", e.Error())))
		return
	}
	lngStr, found := r.URL.Query()["lng"]
	if !found {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "no [lng] provided."))
		return
	}
	lng, e := yh_utils.StringToFloat64(lngStr[0])
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_LAT, fmt.Sprintf("Parse [lng] error : %v", e.Error())))
		return
	}
	sex, found := r.URL.Query()["sex"]
	if !found {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "no [sex] provided."))
		return
	}
	if sex[0] != "m" && sex[0] != "f" {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "[sex] must be 'm' or 'f'."))
		return
	}

	key := Key(sex[0], lat, lng, BIG_ZONE_FACTOR)
	AddActiveKeys(key)
	expire := yh_utils.Now.Unix() + ONLINE_TIMEOUT + 60 //比超时时间稍微长点，避免客户端汇报有延迟
	result["key_1"] = key
	result["expire"] = expire
	result["id"] = ids[0]

	rc := poolZone.Get()
	defer rc.Close()
	_, e = rc.Do("ZADD", key, expire, ids[0])
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis ZADD error : ", e.Error())))
		return
	}
	key = Key(sex[0], lat, lng, SMALL_ZONE_FACTOR)
	result["key_2"] = key
	AddActiveKeys(key)
	_, e = rc.Do("ZADD", key, expire, ids[0])
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis ZADD error : ", e.Error())))
		return
	}
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis connection close error : ", e.Error())))
		return
	}
	re := poolElem.Get()
	defer re.Close()
	pos := Pos{"", lat, lng, 0}
	_, e = re.Do("SET", ids[0], pos.Serialize())
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis error : ", e.Error())))
		return
	}
	_, e = re.Do("EXPIRE", ids[0], ONLINE_TIMEOUT+90) //比zone的超时时间稍微长一点，避免通过zone查elements时找不到
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis error : ", e.Error())))
		return
	}

	result["status"] = "ok"
	writeBack(r, w, result)
}

//获取附近的用户
func adjacent(w http.ResponseWriter, r *http.Request) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"

	num := uint(20) //默认返回20个
	numStr, found := r.URL.Query()["num"]
	if found {
		n, e := yh_utils.StringToUint(numStr[0])
		if e != nil {
			writeBackErr(r, w, common.NewError(common.ERR_INVALID_NUM, fmt.Sprintf("Parse [num] error : %v", e.Error())))
			return
		}
		num = n
	}
	latStr, found := r.URL.Query()["lat"]
	if !found {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "no [lat] provided."))
		return
	}
	lat, e := yh_utils.StringToFloat64(latStr[0])
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_LAT, fmt.Sprintf("Parse [lat] error : %v", e.Error())))
		return
	}
	lngStr, found := r.URL.Query()["lng"]
	if !found {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "no [lng] provided."))
		return
	}
	lng, e := yh_utils.StringToFloat64(lngStr[0])
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_LAT, fmt.Sprintf("Parse [lng] error : %v", e.Error())))
		return
	}
	sex, found := r.URL.Query()["sex"]
	if !found {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "no [sex] provided."))
		return
	}
	if sex[0] != "m" && sex[0] != "f" {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "[sex] must be 'm' or 'f'."))
		return
	}
	ids := make([]interface{}, 0, num*2)
	//先查看小区域是否满足需求
	rc := poolZone.Get()
	defer rc.Close()
	key := Key(sex[0], lat, lng, SMALL_ZONE_FACTOR)
	AddActiveKeys(key)
	values, e := redis.Values(rc.Do("ZRANGEBYSCORE", key, yh_utils.Now.Unix(), "+inf", "LIMIT", 0, num*2))
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis ZRANGEBYSCORE error : ", e.Error())))
		return
	}
	foundKeys := make(map[string]bool, num*3)
	//如果数量不足，则扩大范围寻找
	if uint(len(values)) < num {
		x := GetPos(lat, BIG_ZONE_FACTOR)
		y := GetPos(lng, BIG_ZONE_FACTOR)
		checked := make(map[string]bool)
		notFull := true
		for m := 0; m <= MAX_RADIUS*2 && notFull; m++ {
			for s := 0; s <= m && notFull; s++ {
				if s <= MAX_RADIUS && m-s <= MAX_RADIUS {
					for vx := -1; vx < 2 && notFull; vx += 2 {
						for vy := -1; vy < 2 && notFull; vy += 2 {
							key := KeyFromPos(sex[0], x+s*vx, y+(m-s)*vy, BIG_ZONE_FACTOR)
							AddActiveKeys(key)
							_, found := checked[key]
							if !found {
								checked[key] = true
								values, e := redis.Values(rc.Do("ZRANGEBYSCORE", key, yh_utils.Now.Unix(), "+inf", "LIMIT", 0, int(num)*2-len(values)))
								if e != nil {
									writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis ZRANGEBYSCORE error : ", e.Error())))
									return
								}
								for len(values) > 0 {
									var id string
									values, e = redis.Scan(values, &id)
									if e != nil {
										writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis Scan error : ", e.Error())))
										return
									}
									if _, found := foundKeys[id]; !found {
										ids = append(ids, id)
										foundKeys[id] = true
										if uint(len(ids)) >= num*2 {
											notFull = false
										}
									}
								}
							}
						}
					}
				}
			}
		}
	} else {
		for len(values) > 0 {
			var id string
			values, e = redis.Scan(values, &id)
			if e != nil {
				writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis Scan error : ", e.Error())))
				return
			}
			if _, found := foundKeys[id]; !found {
				ids = append(ids, id)
				foundKeys[id] = true
			}
		}
	}

	candidates := make(PosHeap, 0, len(ids))
	if len(ids) > 0 {
		re := poolElem.Get()
		defer re.Close()
		elems, e := redis.Values(re.Do("MGET", ids...))
		if e != nil {
			writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis MGET error : ", e.Error())))
			return
		}
		index := 0
		for len(elems) > 0 {
			var elem string
			elems, e = redis.Scan(elems, &elem)
			if e != nil {
				writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("redis Scan error : ", e.Error())))
				return
			}
			if elem != "" {
				pos, e := UnSerialize(elem)
				if e != nil {
					writeBackErr(r, w, common.NewError(common.ERR_REDIS, fmt.Sprintf("parse Pos error : ", e.Error())))
					return
				}
				pos.id = fmt.Sprintf("%s", ids[index])
				pos.distance = math.Abs(pos.lat-lat) + math.Abs(pos.lng-lng) //近似计算距离，不准没关系，客户端计算精准距离
				candidates = append(candidates, pos)
			}
			index++
		}
		heap.Init(&candidates)
		items := make([]map[string]interface{}, 0, candidates.Len())
		for candidates.Len() > 0 {
			pos := heap.Pop(&candidates).(Pos)
			item := make(map[string]interface{})
			item["id"] = pos.id
			item["lat"] = pos.lat
			item["lng"] = pos.lng
			items = append(items, item)
		}
		result["items"] = items
	}

	result["status"] = "ok"
	writeBack(r, w, result)
}

func main() {
	flag.Parse()
	conf, err := yh_config.NewConfig(*config)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = checkConfig(&conf)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	ps, err := strconv.Atoi(conf.Items["procs"])
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	runtime.GOMAXPROCS(ps)
	log_level, ok := levelStr[conf.Items["log_level"]]
	if !ok {
		fmt.Println("invalid log level : ", conf.Items["log_level"])
		return
	}

	glog, err = yh_log.New(conf.Items["log"], 10000, log_level)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer glog.Close()

	maxActiveConn, err = strconv.Atoi(conf.Items["redis_max_connections"])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	poolZone = newPool(conf.Items["redis_server"], maxActiveConn, REDIS_DB_ZONES)
	poolElem = newPool(conf.Items["redis_server"], maxActiveConn, REDIS_DB_ELEMENTS)

	go logStatus()
	go clearOld()

	glog.Append("service start.", yh_log.NOTICE)

	http.HandleFunc("/report", report)
	http.HandleFunc("/adjacent", adjacent)
	log.Fatal(http.ListenAndServe(conf.Items["listen"], nil))
}
