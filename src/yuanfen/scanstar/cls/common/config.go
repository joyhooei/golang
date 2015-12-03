package common

import yl "yf_pkg/config/yaml"

type Config struct {
	Log struct {
		Dir   string "dir"
		Level string "level"
	} "log"
	Mysql struct {
		Main yl.MysqlType "main"
		Stat yl.MysqlType "stat"
	} "mysql"
}

func (c *Config) Load(path string) error {
	return yl.Load(c, path)
}
