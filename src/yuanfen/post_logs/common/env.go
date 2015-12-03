package common

import (
	"yf_pkg/log"
	"yf_pkg/service"
)

type CustomEnv struct {
	Path    string
	MainLog *log.Logger
}

func (c *CustomEnv) Init(conf *Config) (err error) {
	c.Path = conf.FilePath
	c.MainLog, err = log.New2(conf.Log.Dir+"/main.log", 10000, conf.Log.Level)
	return err
}
func (c *CustomEnv) GetEnv(module string) *service.Env {
	return service.NewEnv(c)
}
