package config

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
)

type JSONConfig struct {
	Items map[string]interface{}
}

func New(filePath string) (config JSONConfig, err error) {
	f, err := os.Open(filePath)
	if err != nil {
		return config, err
	}
	defer f.Close()

	config.Items = make(map[string]interface{})
	reader := bufio.NewReader(f)
	d, err := ioutil.ReadAll(reader)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(d, &config.Items)
	return config, err
}
