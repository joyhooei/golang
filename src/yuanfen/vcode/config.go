package main

import yl "yf_pkg/config/yaml"

type Config struct {
	Address yl.Address "address"
	Redis   struct {
		Cache yl.RedisType "cache"
	} "redis"
	Log struct {
		Dir   string "dir"
		Level string "level"
	} "log"
}

func (c *Config) Load(path string) error {
	return yl.Load(c, path)
}
