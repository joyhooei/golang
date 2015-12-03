package service

import (
	"net/http"
	"yf_pkg/log"
)

type Env struct {
	Log       *log.MLogger
	ModuleEnv interface{}
}

func NewEnv(data interface{}) *Env {
	return &Env{nil, data}
}

type Config struct {
	IpPort      string
	LogDir      string
	LogLevel    string
	GetEnv      func(module string) *Env
	IsValidUser func(r *http.Request) (uid uint32) //如果不是合法用户，需要返回0
}
