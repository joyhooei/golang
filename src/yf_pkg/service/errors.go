package service

import "fmt"

const (
	SERVER_REDIRECT = 302 // 302跳转特殊错误码

	ERR_NOERR     = 0    //没有错误
	ERR_UNKNOWN   = 1001 //未知错误
	ERR_INTERNAL  = 1002 //内部错误
	ERR_MYSQL     = 1003 //mysql错误
	ERR_REDIS     = 1004 //redis错误
	ERR_NOT_FOUND = 1005 //未找到

	ERR_INVALID_PARAM         = 2001 //请求参数错误
	ERR_INVALID_FORMAT        = 2002 //格式错误
	ERR_ENCRYPT_ERROR         = 2003 //加密错误
	ERR_INVALID_REQUEST       = 2004 //不合法的请求
	ERR_VERIFY_FAIL           = 2005 //验证失败
	ERR_VCODE_TIMEOUT         = 2006 //验证码超时
	ERR_INVALID_USER          = 2007 //用户验证不通过
	ERR_PERMISSION_DENIED     = 2008 //权限不足
	ERR_VCODE_ERROR           = 2009 //验证码错误
	ERR_TOO_MANY              = 2010 //次数过多
	ERR_IN_BLACKLIST          = 2011 //在黑名单中
	ERR_MUSTINFO_NOT_COMPLETE = 2012 //必填项没有填写完整
	ERR_INVALID_IMG           = 2013 // 图片检查未通过

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
	ERR_CANTBY_ITEM            = 9012 //不能购买道具，当前推荐展示没用完
	ERR_PAY_INVALID            = 9013 //充值数据不合法
	ERR_TOPIC_EXIST            = 9014 //话题已存在
	ERR_NOT_SAME_PROVINCE      = 9015 //不在同一省（直辖市）
	ERR_INVENTORY_EMPTY        = 9016 //库存不足

	ERR_POP_NOTIFY = 10001 //特殊错误码，当初该错误码时，需要解析desc字段，并做弹窗处理
)

//302跳转key
const (
	SERVER_REDIRECT_KEY = "redirect_url"
)

type Error struct {
	Code uint
	Desc string
	Show string //客户端显示的内容
}

func NewError(ecode uint, desc string, show ...string) (err Error) {

	if len(show) > 0 {
		err = Error{ecode, desc, show[0]}
	} else {
		switch ecode {
		case ERR_INVALID_PARAM:
			err = Error{ecode, desc, "参数错误"}
		case ERR_INVALID_REQUEST:
			err = Error{ecode, desc, "不合法的请求"}
		case ERR_MYSQL, ERR_REDIS:
			err = Error{ecode, desc, "数据库错误"}
		default:
			err = Error{ecode, desc, "内部错误"}
		}
	}
	return
}
func (e Error) Error() (re string) {
	return fmt.Sprintf("ecode=%v, desc=%v", e.Code, e.Desc)
}
