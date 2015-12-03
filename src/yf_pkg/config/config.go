package config

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
)

type Config struct {
	Items map[string]string
}

func NewConfig(filePath string) (config Config, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return config, err
	}
	defer f.Close()

	config.Items = make(map[string]string)
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return config, err
		} else {
			blocks := strings.Split(line, "#")
			if strings.Trim(blocks[0], " \t") == "" {
				continue
			}
			kv := strings.Split(blocks[0], "=")
			if len(kv) != 2 {
				err = errors.New("invalid format: " + line)
				return config, err
			} else {
				key := strings.Trim(kv[0], " \t\n")
				_, found := config.Items[key]
				if found {
					err = errors.New("duplicate key [" + key + "]")
					return config, err
				}
				config.Items[key] = strings.Trim(kv[1], " \t\n")
			}
		}
	}
	return config, nil
}

//检查配置文件是否合法
func (c *Config) IsValid(keywords map[string]bool) error {
	for key := range keywords {
		if _, found := c.Items[key]; found == false {
			return errors.New("not found key [" + key + "]")
		}
	}
	return nil
}
