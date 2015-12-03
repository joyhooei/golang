package common

import yl "yf_pkg/config/yaml"

type Config struct {
	HttpAddr yl.Address   "http_addr"
	TcpAddr  yl.Addresses "tcp_addr"
	Log      struct {
		Dir   string "dir"
		Level string "level"
	} "log"
	Mysql struct {
		Main   yl.MysqlType "main"
		Online yl.MysqlType "online"
	} "mysql"
	Redis struct {
		Main yl.RedisType "main"
	} "redis"
}

func (c *Config) Load(path string) error {
	return yl.Load(c, path)
}
