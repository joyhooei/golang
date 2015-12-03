package general

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"math/rand"
	"strings"
	"yf_pkg/cachedb"
	"yf_pkg/lbs/baidu"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/common"

	_ "github.com/go-sql-driver/mysql"
)

var mdb *mysql.MysqlDB
var msgdb *mysql.MysqlDB
var rdb *redis.RedisPool
var cache *redis.RedisPool
var cachedb2 *cachedb.CacheDB
var alertLog, mlog *log.Logger
var upload_service_url string // 上传图片和图片检测服务地址

//百度省市名称和我们自己的映射表
//百度的市->我们的市
var city_map map[string]string = map[string]string{}

//我们的市->百度的市
var bcity_map map[string]string = map[string]string{}

//百度的省->我们的省
var province_map map[string]string = map[string]string{}

//我们的省->百度的省
var bprovince_map map[string]string = map[string]string{}

/*
 公用模块，未读消息，推送等相关模块
*/
func Init(env *cls.CustomEnv, conf service.Config) {
	//	fmt.Println("init service_game")
	mdb = env.MainDB
	rdb = env.MainRds
	msgdb = env.MsgDB
	cache = env.CacheRds
	cachedb2 = env.CacheDB
	l, err := log.New2(conf.LogDir+"/sgeneral.log", 10000, conf.LogLevel)
	if err != nil {
		fmt.Println("初始化日志error:", err.Error())
	}
	mlog = l
	l, err = log.New2(conf.LogDir+"/alert.log", 1000, log.NOTICE_STR)
	if err != nil {
		fmt.Println("初始化日志error:", err.Error())
	}
	alertLog = l
	if env.Mode == cls.MODE_PRODUCTION {
		Alert("init", "service restart")
	}
	if err = UpdateAdmin(); err != nil {
		fmt.Println("init admin map error :", err.Error())
	}
	upload_service_url = env.UploadServiceUrl

	initProvinceCityMap()
	initListSet()
}

// 获取指定尺寸的图片url
func GetImgSizeUrl(url string, size int) (u string) {
	suffix := "jpg"
	arr := strings.Split(url, ".")
	if len(arr) > 0 {
		suffix = arr[len(arr)-1]
	}
	sz := utils.IntToString(size)
	return url + "@1e_" + sz + "w_" + sz + "h_1c_0i_1o_95Q_1x." + suffix
}

var admin map[uint32]bool

func UpdateAdmin() (e error) {
	sql := "select uid from admin"
	rows, e := mdb.Query(sql)
	if e != nil {
		return e
	}
	newAdmin := map[uint32]bool{}
	defer rows.Close()
	var uid uint32
	for rows.Next() {
		if err := rows.Scan(&uid); err != nil {
			return e
		}
		newAdmin[uid] = true
	}
	admin = newAdmin
	return
}
func IsSystemUser(uid uint32) bool {
	return uid <= common.UID_MAX_SYSTEM
}
func IsAdmin(uid uint32) bool {
	_, ok := admin[uid]
	return ok
}

//添加一条记录到报警日志，title用来区分是不是同一类报警，相同title的报警5分钟内只会发一条报警短信
func Alert(title string, detail string) {
	alertLog.Append(fmt.Sprintf("%v:%v:%v", utils.Now.Unix(), title, detail), log.NOTICE)
}

//随机获取[min,max]一个值
func RandNum(min, max int) (r int) {
	arr := make([]int, 0, 10)
	for i := min; i <= max; i++ {
		arr = append(arr, i)
	}
	index := rand.Intn(len(arr))
	return arr[index]
}

//随机获取[min,max]的n个值(不重复)
func RandNumMap(min, max int, n int) (r map[int]int) {
	r = make(map[int]int)
	arr := make([]int, 0, 10)
	var flag bool
	if n >= (max - min) {
		flag = true
	}
	for i := min; i <= max; i++ {
		arr = append(arr, i)
		if flag {
			r[i] = i
		}
	}
	if flag {
		return
	}
	for i := 0; i <= n+100; i++ {
		if len(r) > n {
			return
		}
		index := rand.Intn(len(arr))
		if _, ok := r[index]; ok {
			continue
		}
		r[index] = index
	}
	return
}

func MakeKey(keys ...interface{}) string {
	if len(keys) == 0 {
		return ""
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("%v", keys[0]))
	for i := 1; i < len(keys); i++ {
		buf.WriteString(fmt.Sprintf("_%v", keys[i]))
	}
	return buf.String()
}
func SplitKey(key string) []string {
	return strings.Split(key, "_")
}

// 根据字符串求取md5值
func Md5(s string) (md5_str string) {
	md5_str = fmt.Sprintf("%x", md5.Sum([]byte(s)))
	return
}

func initProvinceCityMap() {
	sql := "select city,province,bCity,bProvince from city_map where flag=0"
	rows, e := mdb.Query(sql)
	if e != nil {
		fmt.Println("init city_map and province_map error:", e.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var city, bCity, province, bProvince string
		if err := rows.Scan(&city, &province, &bCity, &bProvince); err != nil {
			fmt.Println("mysql Scan error:", err.Error())
			return
		}
		city_map[bCity] = city
		bcity_map[city] = bCity
		province_map[bProvince] = province
		bprovince_map[province] = bProvince
	}
}

func BaiduToOurProvinceCity(bProvince, bCity string) (province, city string) {
	var ok bool
	if province, ok = province_map[bProvince]; !ok {
		province = bProvince
	}
	if city, ok = city_map[bCity]; !ok {
		if baidu.IsZXS(bCity) {
			city = province
		} else {
			city = bCity
		}
	}
	return
}
func BaiduToOurProvince(bProvince string) (province string) {
	var ok bool
	if province, ok = province_map[bProvince]; !ok {
		province = bProvince
	}
	return
}

func OurToBaiduProvince(province string) (bProvince string) {
	var ok bool
	if bProvince, ok = bprovince_map[province]; !ok {
		bProvince = province
	}
	return
}
