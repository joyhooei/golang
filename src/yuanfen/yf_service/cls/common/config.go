package common

import yl "yf_pkg/config/yaml"

type Config struct {
	PrivateAddr yl.Address "private_addr"
	PublicAddr  yl.Address "public_addr"
	Log         struct {
		Dir   string "dir"
		Level string "level"
	} "log"
	PushAddr yl.Address "push"
	Mysql    struct {
		Main    yl.MysqlType "main"
		Sort    yl.MysqlType "sort"
		Message yl.MysqlType "message"
		Stat    yl.MysqlType "stat"
		DStat   yl.MysqlType "dstat"
	} "mysql"
	Redis struct {
		Main  yl.RedisType "main"
		Cache yl.RedisType "cache"
	} "redis"
	WebServiceUrl    string "web_service_url"
	Mode             string "mode" //develop/test/production
	UploadServiceUrl string "upload_service_url"
}

func (c *Config) Load(path string) error {
	return yl.Load(c, path)
}
