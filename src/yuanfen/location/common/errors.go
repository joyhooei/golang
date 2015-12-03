package common

import "fmt"

const (
	ERR_NOERR    = 0    //没有错误
	ERR_UNKNOWN  = 1001 //未知错误
	ERR_INTERNAL = 1002 //内部错误

	ERR_INVALID_PARAM   = 2001 //请求参数错误
	ERR_INVALID_FORMAT  = 2002 //格式错误
	ERR_ENCRYPT_ERROR   = 2003 //加密错误
	ERR_INVALID_REQUEST = 2004 //不合法的请求

	ERR_INVALID_LAT    = 9001 //不合法的纬度
	ERR_INVALID_LNG    = 9002 //验证不合法的经度
	ERR_REDIS          = 9003 //redis出错
	ERR_INVALID_RADIUS = 9004 //不合法的半径
	ERR_INVALID_NUM    = 9005 //不合法的数量
)

type Error struct {
	Code uint
	Desc string
}

func NewError(ecode uint, desc string) (err Error) {
	err = Error{ecode, desc}
	return
}
func (e *Error) Error() (re string) {
	return fmt.Sprintf("ecode=%v, desc=%v", e.Code, e.Desc)
}
