package service

type DefaultModule struct {
	env *Env
}

func (module *DefaultModule) Init(env *Env) error {
	module.env = env
	return nil
}

func (module *DefaultModule) Hello(req *HttpRequest, res map[string]interface{}) (e Error) {
	res["result"] = "World!"
	return
}
func (module *DefaultModule) SecHello(req *HttpRequest, res map[string]interface{}) (e Error) {
	res["result"] = "Secure World!"
	return
}
func (module *DefaultModule) ErrorModule(req *HttpRequest, res map[string]interface{}) (e Error) {
	e.Desc = "Invalid Module Name"
	e.Code = ERR_INVALID_PARAM
	return
}
func (module *DefaultModule) ErrorMethod(req *HttpRequest, res map[string]interface{}) (e Error) {
	e.Desc = "Invalid Method Name"
	e.Code = ERR_INVALID_PARAM
	return
}
