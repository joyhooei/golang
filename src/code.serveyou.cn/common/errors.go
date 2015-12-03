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
	ERR_VERIFY_FAIL     = 2005 //验证失败
	ERR_VCODE_TIMEOUT   = 2006 //验证码超时

	ERR_CELLPHONE_EXISTS       = 9001 //手机号已存在
	ERR_USER_NOT_FOUND         = 9002 //未找到用户
	ERR_ROLE_NOT_MATCH         = 9003 //角色不匹配
	ERR_MAX_ADDRESS_LIMIT      = 9004 //超过地址数量上限
	ERR_ADDRESS_EMPTY          = 9005 //地址为空
	ERR_DELETE_DEFAULT_ADDRESS = 9006 //删除默认地址
	ERR_CELLPHONE_NOT_EXISTS   = 9007 //手机不存在
	ERR_INVALID_PHASE          = 9008 //处于不合法的阶段
	ERR_COLLECT_EXISTS         = 9009 //收藏的小时工已存在
	ERR_NOT_ENOUGH_MONEY       = 9010 //余额不足
	ERR_VOTED                  = 9011 //已投过票
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
