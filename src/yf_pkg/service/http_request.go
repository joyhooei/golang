package service

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"yf_pkg/utils"
)

const MAX_PS = 1000

type HttpRequest struct {
	Body    map[string]interface{}
	BodyRaw []byte
	request *http.Request
	Uid     uint32
}

func (hr *HttpRequest) GetParam(name string) string {
	return hr.request.URL.Query().Get(name)
}

func (hr *HttpRequest) IP() string {
	ips := strings.Split(hr.request.Header.Get("X-Forwarded-For"), ",")
	if len(ips[0]) > 3 {
		return ips[0]
	} else {
		addr := strings.Split(hr.request.RemoteAddr, ":")
		return addr[0]
	}
}

func (hr *HttpRequest) Cookie(name string) (*http.Cookie, error) {
	fmt.Println(hr.request.Cookies())
	return hr.request.Cookie(name)
}

//检查Body中的字段是否齐全
func (hr *HttpRequest) EnsureBody(keys ...string) (string, bool) {
	for _, key := range keys {
		if _, ok := hr.Body[key]; !ok {
			return key, false
		}
	}
	return "", true
}

//带默认值的解析
func (hr *HttpRequest) ParseOpt(params ...interface{}) error {
	if len(params)%3 != 0 {
		return errors.New("params count invalid")
	}
	for i := 0; i < len(params); i += 3 {
		key := utils.ToString(params[i])
		v, ok := hr.Body[key]
		var e error
		switch ref := params[i+1].(type) {
		case *string:
			if ok {
				*ref = utils.ToString(v)
			} else {
				*ref = utils.ToString(params[i+2])
			}

		case *float64:
			if ok {
				*ref, e = utils.ToFloat64(v)
			} else {
				*ref, e = utils.ToFloat64(params[i+2])
			}
		case *int:
			if ok {
				*ref, e = utils.ToInt(v)
			} else {
				*ref, e = utils.ToInt(params[i+2])
			}
		case *uint32:
			if ok {
				*ref, e = utils.ToUint32(v)
			} else {
				*ref, e = utils.ToUint32(params[i+2])
			}
		case *uint64:
			if ok {
				*ref, e = utils.ToUint64(v)
			} else {
				*ref, e = utils.ToUint64(params[i+2])
			}
		case *int64:
			if ok {
				*ref, e = utils.ToInt64(v)
			} else {
				*ref, e = utils.ToInt64(params[i+2])
			}
		case *int8:
			if ok {
				*ref, e = utils.ToInt8(v)
			} else {
				*ref, e = utils.ToInt8(params[i+2])
			}
		case *uint:
			if ok {
				*ref, e = utils.ToUint(v)
			} else {
				*ref, e = utils.ToUint(params[i+2])
			}
		case *bool:
			if ok {
				*ref, e = utils.ToBool(v)
			} else {
				*ref, e = utils.ToBool(params[i+2])
			}
		case *[]string:
			if ok {
				*ref, e = utils.ToStringSlice(v)
			} else {
				*ref = params[i+2].([]string)
			}
		case *map[string]interface{}:
			if ok {
				switch m := v.(type) {
				case map[string]interface{}:
					*ref = m
				default:
					e = errors.New(fmt.Sprintf("%v is not map[string]iterface{}", key, reflect.TypeOf(v)))
				}
			} else {
				*ref = params[i+2].(map[string]interface{})
			}
		case *interface{}:
			if ok {
				*ref = v
			} else {
				*ref = params[i+2]
			}
		default:
			return errors.New(fmt.Sprintf("unknown type %v ", key, reflect.TypeOf(ref)))
		}
		if e != nil {
			return errors.New(fmt.Sprintf("parse [%v] error:%v", key, e.Error()))
		}
	}
	return nil
}

//不带默认值的解析
func (hr *HttpRequest) Parse(params ...interface{}) error {
	if len(params)%2 != 0 {
		return errors.New("params count must be odd")
	}
	for i := 0; i < len(params); i += 2 {
		key := utils.ToString(params[i])
		if v, ok := hr.Body[key]; ok {
			var e error
			switch ref := params[i+1].(type) {
			case *string:
				*ref = utils.ToString(v)
			case *float64:
				*ref, e = utils.ToFloat64(v)
			case *int:
				*ref, e = utils.ToInt(v)
			case *uint16:
				*ref, e = utils.ToUint16(v)
			case *uint32:
				*ref, e = utils.ToUint32(v)
			case *uint64:
				*ref, e = utils.ToUint64(v)
			case *int64:
				*ref, e = utils.ToInt64(v)
			case *int16:
				*ref, e = utils.ToInt16(v)
			case *int8:
				*ref, e = utils.ToInt8(v)
			case *uint:
				*ref, e = utils.ToUint(v)
			case *map[string]interface{}:
				switch m := v.(type) {
				case map[string]interface{}:
					*ref = m
				default:
					e = errors.New("value is not map[string]iterface{}")
				}
			case *[]string:
				*ref, e = utils.ToStringSlice(v)
			case *interface{}:
				*ref = v
			default:
				return errors.New(fmt.Sprintf("unknown type %v ", reflect.TypeOf(ref)))
			}
			if e != nil {
				return errors.New(fmt.Sprintf("parse [%v] error:%v", key, e.Error()))
			}
			if key == "ps" {
				ps, e := utils.ToUint64(v)
				if e == nil && ps > MAX_PS {
					return errors.New("ps too large")
				}
			}
		} else {
			return errors.New(fmt.Sprintf("%v not provided", key))
		}
	}
	return nil
}
