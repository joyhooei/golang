/*
图片验证码服务

地址：

	公网：vcode.service.mumu123.cn:80
	内网：vcode.docker:80
*/
package model

import (
	"encoding/json"
	"fmt"
	"net/http"
	"yf_pkg/log"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yf_pkg/vcode"
)

const (
	TIMEOUT = 120 //超时时间（秒）
	LENGTH  = 4   //验证码长度
)

var cacheRds *redis.RedisPool
var mainLog *log.MLogger

func genKey(k string) string {
	return fmt.Sprintf("vcode_%v", k)
}

func Init(c *redis.RedisPool, lg *log.MLogger) {
	cacheRds = c
	mainLog = lg
}

/*
Verify用于验证给定key的验证码是否正确，每个验证码无论成功或失败，只能被验证一次。验证码超时时间为2分钟。

URI: /verify?key=123456&value=7890

参数:
	key: 验证码的关键字，由客户端自行生成，长度大于6个字节的字符串。
	value: 图片中验证码的值。
返回值:
	{
		"status": "ok"	//返回结果，ok-成功，fail-失败
	}

*/
func Verify(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get("key")
	value := req.URL.Query().Get("value")
	con := cacheRds.GetReadConnection(0)
	defer con.Close()
	v, e := redis.String(con.Do("GET", genKey(key)))
	if e == nil && v == value {
		j, _ := json.Marshal(map[string]interface{}{"status": "ok", "tm": utils.Now})
		w.Write(j)
	} else {
		j, _ := json.Marshal(map[string]interface{}{"status": "fail", "tm": utils.Now})
		w.Write(j)
	}
	cacheRds.Del(0, genKey(key))
	return
}

/*
Pic生成验证码图片。每个验证码的超时时间为2分钟。

URI: /pic?key=123456&width=100&height=40

参数:
	key: 验证码的关键字，由客户端自行生成，长度大于6个字节的字符串。
	width: 图片宽度
	height: 图片高度
返回值:
	一张验证码图片

*/
func Pic(w http.ResponseWriter, req *http.Request) {
	key := req.URL.Query().Get("key")
	if len(key) < 6 || len(key) > 100 {
		w.Write([]byte("key length invalid."))
		return
	}
	width, e := utils.ToInt(req.URL.Query().Get("width"))
	if e != nil {
		w.Write([]byte("param width invalid"))
		return
	}
	height, e := utils.ToInt(req.URL.Query().Get("height"))
	if e != nil {
		w.Write([]byte("param height invalid"))
		return
	}
	if width > 1000 || height > 1000 || width < 50 || height < 20 {
		w.Write([]byte("image size invalid"))
		return
	}
	w.Header().Set("Content-Type", "image/png")
	value, img := vcode.NewImage(LENGTH, width, height)
	con := cacheRds.GetWriteConnection(0)
	defer con.Close()
	_, e = con.Do("SETEX", genKey(key), TIMEOUT, value)
	if e != nil {
		w.Write([]byte("write to redis error"))
		return
	}
	//fmt.Printf("key=%v,value=%v,timeout=%v\n", key, value, codeMap[key].timeout)
	img.WriteTo(w)
}
