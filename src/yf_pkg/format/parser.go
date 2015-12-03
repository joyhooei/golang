package format

import (
	"errors"
	"strings"
)

//把字符串解析成key-value对，br代表所使用的换行符
//解析的过程中不会去除空格和tab字符
func ParseKV(data string, br string) (map[string]string, error) {
	values := make(map[string]string)
	for _, line := range strings.Split(data, br) {
		if line == "" {
			continue
		}
		items := strings.SplitN(line, "=", 2)
		if len(items) < 2 {
			err := errors.New("Parse key-value pairs error : " + line)
			return nil, err
		} else {
			values[items[0]] = items[1]
		}
	}
	return values, nil
}

func ParseKVGroup(data string, br string, groupBr string) ([]map[string]string, error) {
	values := make([]map[string]string, 0, 3)
	for _, group := range strings.Split(data, groupBr) {
		kvg := make(map[string]string)
		for _, line := range strings.Split(group, br) {
			items := strings.SplitN(line, "=", 2)
			if len(items) < 2 {
				err := errors.New("Parse key-value pairs error : " + line)
				return nil, err
			} else {
				kvg[items[0]] = items[1]
			}
		}
		values = append(values, kvg)
	}
	return values, nil
}

//检查Map中是否包含所需要的kv
//如果缺少某些kv，则返回第一个找不到的key
func Contains(kv map[string]string, keys []string) (bool, string) {
	for _, key := range keys {
		if _, found := kv[key]; !found {
			return false, key
		}
	}
	return true, ""
}

//把字符串解析成数组对，br代表所使用的分隔符
//解析的过程中不会去除空格和tab字符
func ParseVector(data string, br string) []string {
	values := make([]string, 0, 5)
	for _, item := range strings.Split(data, br) {
		values = append(values, item)
	}
	return values
}
