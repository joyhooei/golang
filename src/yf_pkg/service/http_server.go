/*
简易的Http框架，以json为传输格式。

错误返回值：
	{
		"code":2001,	//错误码
		"detail":"uid not provided",	//内部使用的错误详情
		"msg":"参数错误",	//客户端显示的错误原因
		"status":"fail"
	}
*/
package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"
	"yf_pkg/log"
	"yf_pkg/utils"

	_ "github.com/go-sql-driver/mysql"
)

type Server struct {
	modules      map[string]Module
	sysLog       *log.MLogger
	conf         *Config
	parseBody    bool //是否把POST的内容解析为json对象
	customResult bool //返回结果中是否包含result和tm项
}

func New(conf *Config, args ...bool) (server *Server, err error) {
	sysLog, err := log.NewMLogger(conf.LogDir+"/system", 10000, conf.LogLevel)
	if err != nil {
		return nil, err
	}
	server = &Server{make(map[string]Module), sysLog, conf, true, false}
	server.AddModule("default", &DefaultModule{})
	if len(args) >= 1 {
		server.parseBody = args[0]
	}
	if len(args) >= 2 {
		server.customResult = args[1]
	}
	return server, nil
}

func (server *Server) AddModule(name string, module Module) (err error) {
	fmt.Printf("add module %s... ", name)
	mlog, err := log.NewMLogger(server.conf.LogDir+"/"+name, 10000, server.conf.LogLevel)
	if err != nil {
		fmt.Println("failed")
		return err
	}
	env := server.conf.GetEnv(name)
	env.Log = mlog
	err = module.Init(env)
	if err != nil {
		return
	}
	fmt.Println("ok")
	mlog.Append("add module success", log.NOTICE)
	server.modules[name] = module
	return
}

func (server *Server) StartService() error {
	handler := http.NewServeMux()
	handler.HandleFunc("/s/", server.secureHandler)
	handler.HandleFunc("/", server.nonSecureHandler)
	s := &http.Server{
		Addr:           server.conf.IpPort,
		Handler:        handler,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
		MaxHeaderBytes: 0,
	}
	l := fmt.Sprint("service start at ", server.conf.IpPort, " ...")
	server.sysLog.Append(l, log.NOTICE)
	fmt.Println(l)
	return s.ListenAndServe()
}

func (server *Server) writeBackErr(r *http.Request, w http.ResponseWriter, reqBody []byte, err Error, duration int64) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"
	result["code"] = err.Code
	result["msg"] = err.Show
	result["detail"] = err.Desc
	res, _ := json.Marshal(result)
	server.writeBack(r, w, reqBody, res, false, duration)
}

func (server *Server) writeBack(r *http.Request, w http.ResponseWriter, reqBody []byte, result []byte, success bool, duration int64) {
	w.Write(result)
	var l string
	var uid, response string
	uidCookie, e := r.Cookie("uid")
	if e != nil {
		uid = ""
	} else {
		uid = uidCookie.Value
	}
	if reqBody != nil {
		response = string(reqBody)
	}
	format := "\nduration: %.2fms\n"
	format += "uid: %s\n"
	format += "uri: %s\n"
	format += "param: %s\n"
	format += "response:\n%s\n"
	format += "------------------------------------------------------------------"

	l = fmt.Sprintf(format, float64(duration)/1000000, uid, r.URL.String(), response, string(result))
	server.sysLog.Append(l, log.DEBUG)
	if !success {
		server.sysLog.Append(l, log.ERROR)
	}
}

func (server *Server) secureHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UnixNano()
	var result map[string]interface{} = make(map[string]interface{})
	var err error

	uid := server.conf.IsValidUser(r)
	var body []byte
	if uid > 0 {
		fields := strings.Split(r.URL.Path[1:], "/")
		if len(fields) >= 3 {
			body, err = server.handleRequest(fields[1], "Sec"+fields[2], uid, r, result)
		} else {
			err = NewError(ERR_INVALID_PARAM, "invalid url format : "+r.URL.Path)
		}
	} else {
		err = NewError(ERR_INVALID_USER, "invalid user")
	}

	end := time.Now().UnixNano()
	server.processError(w, r, err, body, result, end-start)
}

func (server *Server) nonSecureHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now().UnixNano()
	var result map[string]interface{} = make(map[string]interface{})
	var err error

	fields := strings.Split(r.URL.Path[1:], "/")
	var body []byte
	if len(fields) >= 2 {
		body, err = server.handleRequest(fields[0], fields[1], 0, r, result)
	} else {
		err = NewError(ERR_INVALID_PARAM, "invalid url format : "+r.URL.Path)
	}
	end := time.Now().UnixNano()
	server.processError(w, r, err, body, result, end-start)
}

func (server *Server) processError(w http.ResponseWriter, r *http.Request, err error, reqBody []byte, result map[string]interface{}, duration int64) {
	var re Error
	switch e := err.(type) {
	case nil:
	case Error:
		re = e
	default:
		re = NewError(ERR_INTERNAL, e.Error(), "未知错误")
	}
	// 302跳转
	if re.Code == SERVER_REDIRECT {
		url := utils.ToString(result[SERVER_REDIRECT_KEY])
		http.Redirect(w, r, url, http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	//w.Header().Set("Content-Encoding", "utf-8")
	//w.Header().Set("Charset", "utf-8")
	if re.Code == ERR_NOERR {
		if !server.customResult {
			result["status"] = "ok"
			result["tm"] = utils.Now.Unix()
		}
		res, e := json.Marshal(result)
		//res, e := json.MarshalIndent(result, "", " ")
		if e == nil {
			server.writeBack(r, w, reqBody, res, true, duration)
		} else {
			server.writeBackErr(r, w, reqBody, NewError(ERR_INTERNAL, fmt.Sprintf("Marshal result error : %v", e.Error())), duration)
		}
	} else {
		server.writeBackErr(r, w, reqBody, re, duration)
	}
}

func (server *Server) handleRequest(moduleName string, methodName string, uid uint32, r *http.Request, result map[string]interface{}) ([]byte, error) {
	bodyBytes, e := ioutil.ReadAll(r.Body)
	if e != nil {
		return nil, NewError(ERR_INTERNAL, "read http data error : "+e.Error())
	}
	var body map[string]interface{}
	if len(bodyBytes) == 0 {
		//可能是Get请求
		body = make(map[string]interface{})
	} else if server.parseBody {
		e = json.Unmarshal(bodyBytes, &body)
		if e != nil {
			return bodyBytes, NewError(ERR_INVALID_PARAM, "read body error : "+e.Error())
		}
	}
	var values []reflect.Value
	module, ok := server.modules[moduleName]
	if ok {
		method := reflect.ValueOf(module).MethodByName(methodName)
		if method.IsValid() {
			values = method.Call([]reflect.Value{reflect.ValueOf(&HttpRequest{body, bodyBytes, r, uid}), reflect.ValueOf(result)})
		} else {
			method = reflect.ValueOf(server.modules["default"]).MethodByName("ErrorMethod")
			values = method.Call([]reflect.Value{reflect.ValueOf(&HttpRequest{body, bodyBytes, r, uid}), reflect.ValueOf(result)})
		}
	} else {
		method := reflect.ValueOf(server.modules["default"]).MethodByName("ErrorModule")
		values = method.Call([]reflect.Value{reflect.ValueOf(&HttpRequest{body, bodyBytes, r, uid}), reflect.ValueOf(result)})
	}
	if len(values) != 1 {
		return bodyBytes, NewError(ERR_INTERNAL, fmt.Sprintf("method %s.%s return value is not 2.", moduleName, methodName))
	}
	switch x := values[0].Interface().(type) {
	case nil:
		return bodyBytes, nil
	default:
		return bodyBytes, x.(error)
	}
}
