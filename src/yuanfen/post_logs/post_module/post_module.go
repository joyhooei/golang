package post_module

import (
	"os"
	"path/filepath"
	"yf_pkg/log"
	"yf_pkg/service"
	"yuanfen/post_logs/common"
)

type PostModule struct {
	log  *log.Logger
	path string
}

func (sm *PostModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.path = env.ModuleEnv.(*common.CustomEnv).Path

	return
}

func (sm *PostModule) Send(req *service.HttpRequest, res map[string]interface{}) (e error) {
	name := req.GetParam("name")
	file, e := os.Create(filepath.Join(sm.path, name))
	if e != nil {
		return
	}
	_, e = file.Write(req.BodyRaw)
	if e != nil {
		return
	}
	res["result"] = "ok"
	return
}
