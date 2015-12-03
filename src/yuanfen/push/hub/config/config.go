package config

import yl "yf_pkg/config/yaml"

type Config struct {
	Address yl.Address "address"
	Log     struct {
		Dir   string "dir"
		Level string "level"
	} "log"
	Mysql struct {
		Main yl.MysqlType "main"
	} "mysql"
	Redis struct {
		Main yl.RedisType "main"
	} "redis"
}

func (c *Config) Load(path string) error {
	return yl.Load(c, path)
}
