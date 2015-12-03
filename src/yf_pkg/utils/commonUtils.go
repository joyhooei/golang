package utils

import (
	"database/sql"
	"strconv"
	"strings"
)

const (
	DER_ERR_MSG  = "获取数据失败，请重试"
	DER_ERR_PARM = "参数错误"
)

// 判断字符参数，并转换int如果没有设置则返回默认值
func GetCommonIntParam(p string, def int) int {
	tmp, err := strconv.Atoi(p)
	if err != nil {
		return def
	}
	return tmp
}

// 根据字符串获取uint32
func GetCommonUInt32Param(p string, def uint32) uint32 {
	tmp, err := StringToUint32(p)
	if err != nil {
		return def
	}
	return tmp
}

// 判断传入值是否为nil且为float64，是则转化成int,否则返回默认值
func GetComonFloat64ToUint32(p interface{}, def uint32) uint32 {
	if p != nil {
		if v, ok := p.(float64); ok {
			return uint32(v)
		}
	}
	return def
}

// 判断传入值是否为nil且为float64，是则转化成int,否则返回默认值
func GetComonFloat64ToInt(p interface{}, def int) int {
	if p != nil {
		if v, ok := p.(float64); ok {
			return int(v)
		}
	}
	return def
}

// 判断传入值是否为nil，是则转化成uint32,否则返回默认值
func InterfaceToUint32(p interface{}, def uint32) uint32 {
	if p != nil {
		if v, ok := p.(uint32); ok {
			return v
		}
	}
	return def
}

// 解析数据库结果集，封装成数组(该解析,只能只针对于转化为string对象)
func ParseSqlResult(rows *sql.Rows) ([]map[string]string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]sql.RawBytes, len(cols))
	scans := make([]interface{}, len(cols))
	res_arr := make([]map[string]string, 0, 0)
	for i := range values {
		scans[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			return res_arr, err
		}
		row := make(map[string]string)
		for j, v := range values {
			key := cols[j]
			row[key] = string(v)
		}
		res_arr = append(res_arr, row)
	}
	return res_arr, nil
}

// 将uint32的数组转为字符串，以逗号连接
func Uint32ArrTostring(ids []uint32) string {
	if ids == nil || len(ids) <= 0 {
		return "0"
	}
	sids := make([]string, 0, 0)
	for _, v := range ids {
		sids = append(sids, ToString(v))
	}
	return strings.Join(sids, ",")
}

// 将整数数组转为字符串，以逗号连接
func ArrTostring(v interface{}, sep string) string {
	r := make([]string, 0, 10)
	switch slice := v.(type) {
	case []int:
		for _, item := range slice {
			r = append(r, ToString(item))
		}
	case []uint32:
		for _, item := range slice {
			r = append(r, ToString(item))
		}
	case []int32:
		for _, item := range slice {
			r = append(r, ToString(item))
		}
	case []interface{}:
		for _, item := range slice {
			r = append(r, ToString(item))
		}
	default:
		return "0"
	}
	return strings.Join(r, sep)
}

// 求两个数组的差集，返回存在于a1却不存在a2中的值
func Uint32ArrDiff(a1, a2 []uint32) (diff []uint32) {
	diff = make([]uint32, 0, 10)
	if len(a1) <= 0 {
		return
	}
	if len(a2) <= 0 {
		return a1
	}
	m := make(map[uint32]uint32)
	for _, v := range a2 {
		m[v] = v
	}
	for _, v := range a1 {
		if _, ok := m[v]; !ok {
			diff = append(diff, v)
		}
	}
	return
}

// 求两个数组的交集，返回存在于a1并且存在a2中的值
func Uint32ArrIntersection(a1, a2 []uint32) (both []uint32) {
	both = make([]uint32, 0, 10)
	if len(a1) <= 0 {
		return
	}
	if len(a2) <= 0 {
		return
	}
	m := make(map[uint32]uint32)
	for _, v := range a2 {
		m[v] = v
	}
	for _, v := range a1 {
		if _, ok := m[v]; ok {
			both = append(both, v)
		}
	}
	return
}

// 求两个数组的差集，返回存在于a1却不存在a2中的值
func StringArrDiff(a1, a2 []string) (diff []string) {
	diff = make([]string, 0, len(a1))
	if len(a1) <= 0 {
		return
	}
	if len(a2) <= 0 {
		return a1
	}
	m := make(map[string]string)
	for _, v := range a2 {
		m[v] = v
	}
	for _, v := range a1 {
		if _, ok := m[v]; !ok {
			diff = append(diff, v)
		}
	}
	return
}

// 字符串分割并转化成数字数组
func StringToUint32Arr(str string, sep string) (r []uint32) {
	r = make([]uint32, 0, 10)
	if str == "" {
		return
	}
	for _, s := range strings.Split(str, sep) {
		v, e := ToUint32(s)
		if e != nil {
			continue
		}
		r = append(r, v)
	}
	return
}
