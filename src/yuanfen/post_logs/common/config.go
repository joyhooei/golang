package common

import yl "yf_pkg/config/yaml"

type Config struct {
	PublicAddr yl.Address "public_addr"
	FilePath   string     "path"
	Log        struct {
		Dir   string "dir"
		Level string "level"
	} "log"
}

func (c *Config) Load(path string) error {
	return yl.Load(c, path)
}
