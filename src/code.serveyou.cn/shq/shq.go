/*
	账户注册、登录、验证等功能
*/
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/location"
	"code.serveyou.cn/model"
	"code.serveyou.cn/pkg/config"
	"code.serveyou.cn/pkg/encrypt"
	"code.serveyou.cn/pkg/format"
	"code.serveyou.cn/pkg/log"
	"code.serveyou.cn/search"
)

//日志
var glog *log.Logger

//数据库连接池
var db *sql.DB
var domainDb *sql.DB

//位置模块
var loc *location.Location

var accountHandler *AccountHandler
var transactionHandler *TransactionHandler
var searchHandler *SearchHandler

//城市、小区字典
var Cities = model.CityMap{}
var Communities = model.CommunityMap{}
var CSearcher *search.Searcher

//用户基本信息
var Users = model.UserMap{}
var Phones = model.PhoneUserMap{}

//HKP按小区、Job分类的数据
var CommunityHKPs = model.CommunityHKPJobMap{}

//UID-->HKP信息的映射
var HKPs = model.UidHKPJobMap{}

//焦点图数据
var Recommend format.JSON

func writeBackErr(r *http.Request, w http.ResponseWriter, err common.Error) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"
	result["ecode"] = err.Code
	result["edesc"] = err.Desc
	writeBack(r, w, string(format.GenerateJSON(result)))
}

func writeBack(r *http.Request, w http.ResponseWriter, result string) {
	w.Write([]byte(result))
	log := fmt.Sprintf("request : %s	response : %s", r.URL.String(), result)
	glog.Append(log)
}

//安全请求，Post内容是经过加密的
func secureHandler(w http.ResponseWriter, r *http.Request) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"
	tmpuid, e := format.ParseUint64(r.URL.Query().Get("uid"))
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "parse uid error : "+e.Error()))
		return
	}
	uid := common.UIDType(tmpuid)
	devid := r.URL.Query().Get("devid")
	password, err := accountHandler.GetPassword(uid, devid)
	if err.Code != common.ERR_NOERR {
		writeBackErr(r, w, err)
		return
	}
	body, e := ioutil.ReadAll(r.Body)
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_INTERNAL, "read http data error : "+e.Error()))
		return
	}
	if len(body) == 0 || string(body) == "" {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_FORMAT, "format error : no post data"))
		return
	}
	decrypted, e := encrypt.AesDecrypt16(string(body), password)
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_ENCRYPT_ERROR, "decrypt error : "+e.Error()))
		return
	}
	if strings.Index(decrypted, "!!encrypt_head=shq365.cn") != 0 {
		writeBackErr(r, w, common.NewError(common.ERR_ENCRYPT_ERROR, "decrypt error : invalid header"))
		return
	}

	cmd := strings.Split(r.URL.Path[3:], "/")
	if len(cmd) < 2 {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "unknown secure command : "+r.URL.Path[1:]))
		return
	}
	switch cmd[0] {
	case "account":
		err = accountHandler.ProcessSec(cmd[1], r, uid, devid, decrypted, result)
	case "trans":
		err = transactionHandler.ProcessSec(cmd[1], r, uid, devid, decrypted, result)
	case "search":
		err = searchHandler.ProcessSec(cmd[1], r, uid, devid, decrypted, result)
	default:
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "unknown secure command : "+r.URL.Path[1:]))
		return
	}

	if err.Code == common.ERR_NOERR {
		result["status"] = "ok"
		writeBack(r, w, string(format.GenerateJSON(result)))
	} else {
		writeBackErr(r, w, err)
	}
}

//普通请求，全部明文
func nonSecureHandler(w http.ResponseWriter, r *http.Request) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"
	var err common.Error
	fmt.Println("cmd = " + r.URL.Path[1:])
	cmd := strings.Split(r.URL.Path[1:], "/")
	if len(cmd) < 2 {
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "invalid command format : "+r.URL.Path[1:]))
		return
	}

	body_bytes, e := ioutil.ReadAll(r.Body)
	if e != nil {
		writeBackErr(r, w, common.NewError(common.ERR_INTERNAL, "read http data error : "+e.Error()))
		return
	}
	body := string(body_bytes)
	switch cmd[0] {
	case "account":
		err = accountHandler.Process(cmd[1], r, body, result)
	case "trans":
		err = transactionHandler.Process(cmd[1], r, body, result)
	case "search":
		err = searchHandler.Process(cmd[1], r, body, result)
	default:
		writeBackErr(r, w, common.NewError(common.ERR_INVALID_PARAM, "unknown command : "+r.URL.Path[1:]))
		return
	}
	if err.Code == common.ERR_NOERR {
		result["status"] = "ok"
		writeBack(r, w, string(format.GenerateJSON(result)))
	} else {
		writeBackErr(r, w, err)
	}
}

//配置文件中的必要项
var keywords = map[string]bool{
	"ip":             true,
	"port":           true,
	"log":            true,
	"my_user":        true,
	"my_pwd":         true,
	"my_db":          true,
	"my_ip":          true,
	"my_port":        true,
	"db_domain_user": true,
	"db_domain_pwd":  true,
	"db_domain_db":   true,
	"db_domain_ip":   true,
	"db_domain_port": true,
}

//检查配置文件是否合法
func checkConfig(conf *config.Config) error {
	for key := range keywords {
		if _, found := conf.Items[key]; found == false {
			return errors.New("not found key [" + key + "]")
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s [config]\n", os.Args[0])
		return
	}

	rand.Seed(time.Now().UnixNano())
	conf, err := config.NewConfig(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = checkConfig(&conf)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	glog, err = log.New(conf.Items["log"], 10000)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer glog.Close()
	glog.Append("start service " + os.Args[0])

	//初始化数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", conf.Items["my_user"], conf.Items["my_pwd"],
		conf.Items["my_ip"], conf.Items["my_port"], conf.Items["my_db"])
	if db, err = common.NewDB(dsn); err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}
	defer db.Close()
	dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", conf.Items["db_domain_user"], conf.Items["db_domain_pwd"],
		conf.Items["db_domain_ip"], conf.Items["db_domain_port"], conf.Items["db_domain_db"])
	if domainDb, err = common.NewDB(dsn); err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}
	defer domainDb.Close()

	//初始化业务模块
	accountHandler = NewAccountHandler(db)
	var e common.Error
	transactionHandler, e = NewTransactionHandler(db)
	if e.Code != common.ERR_NOERR {
		fmt.Println(e.Error())
		glog.Append(e.Error())
		return
	}
	searchHandler = NewSearchHandler(db)

	//初始化全局变量
	if err = common.Variables.InitVariables(db); err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}

	//初始化角色工作映射表
	common.RoleJob.Init()

	//读取城市、小区数据
	if err = Cities.Init(domainDb); err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}
	if err = Communities.Init(db); err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}
	CSearcher = search.NewSearcher(Communities)

	//初始化位置服务
	el, err := GetCommunityLocations()
	if err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}
	loc = location.NewLocation(el)

	//读取服务提供者信息
	if err = InitHKPs(db); err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}

	//读取焦点图数据
	Recommend, err = GetRecommend(db)
	if err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}

	//监听端口，提供http服务
	http.HandleFunc("/s/", secureHandler)
	http.HandleFunc("/", nonSecureHandler)
	err = http.ListenAndServe(conf.Items["ip"]+":"+conf.Items["port"], nil)
	if err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
	}
	time.Sleep(2 * time.Second)
}
