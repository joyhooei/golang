package main

import yl "yf_pkg/config/yaml"

type Config struct {
	Address yl.Address "address"
	Log     struct {
		Dir   string "dir"
		Level string "level"
	} "log"
	Mysql yl.MysqlType "mysql"
	Table string       "table"
}

func (c *Config) Load(path string) error {
	return yl.Load(c, path)
}
