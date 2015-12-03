package common

import (
	"database/sql"
	"time"
	"code.serveyou.cn/pkg/format"

	_ "github.com/go-sql-driver/mysql"
)

const (
	DISTANCE_LATITUDE  = 111319
	DISTANCE_LONGITUDE = 70000
	EARTH_RADIUS       = 6378137
)

const (
	DB_VTYPE_REGLOGIN = 1 //验证注册或登陆
	DB_VTYPE_MAX      = 2 //无意义，仅用来表示上限
)

const (
	PLATFORM_ANDROID  = 1
	PLATFORM_IPHONE   = 2
	PLATFORM_IPAD     = 3
	PLATFORM_WINPHONE = 4
	PLATFORM_ANPAD    = 5
	PLATFORM_MAX      = 6
)

const (
	ROLE_ALL      = 0 //所有角色，用于检索
	ROLE_CUSTOMER = 1 //顾客
	ROLE_HKP      = 2 //家政服务
)

const (
	SEX_SECRET  = 0 //保密
	SEX_MALE    = 1 //男
	SEX_FEMALE  = 2 //女
	SEX_UNKNOWN = 3 //未设置
)

const (
	HKP_JOB_ALL  = 0 //所有Job，用于检索
	HKP_JOB_RCBJ = 1 //日常保洁
	HKP_JOB_KH   = 2 //开荒
	HKP_JOB_DSC  = 3 //大扫除
	HKP_JOB_MAX  = 4 //无意义，仅用来表示最大值
)

const (
	HKP_PHASE_AUTO_DISPATCH    = 1  //自动分发阶段
	HKP_PHASE_CSR              = 2  //人工联系阶段
	HKP_PHASE_RECOMMEND        = 3  //推荐阶段
	HKP_PHASE_ORDER_SUCCESS    = 4  //下单成功
	HKP_PHASE_SERVICE_COMPLETE = 5  //服务完成
	HKP_PHASE_RANK_COMPLETE    = 6  //评价完成
	HKP_PHASE_ORDER_FAIL       = 7  //下单失败
	HKP_PHASE_SERVICE_FAIL     = 8  //服务失败，不区分客户还是服务提供者
	HKP_PHASE_CUSTOMER_CANCEL  = 9  //客户取消
	HKP_PHASE_PROVIDER_CANCEL  = 10 //服务提供者取消
)

//小时工是否确认服务
const (
	HKP_PROVIDER_CONFIRM_UNKNOWN = 0
	HKP_PROVIDER_CONFIRM_YES     = 1
	HKP_PROVIDER_CONFIRM_NO      = 2
)

//交易类型
const (
	TRANS_REG         = 1 //注册奖励
	TRANS_INVITE      = 2 //邀请奖励
	TRANS_FIRST_ORDER = 3 //首次下单奖励
	TRANS_EACH_ORDER  = 4 //每次下单奖励
	TRANS_BUY_GOODS   = 5 //购买物品
)

//通知类型
const (
	NOTI_ADV    = 1 //推广通知
	NOTI_REWARD = 2 //奖励通知
	NOTI_ORDER  = 3 //订单通知
)

const MAX_RANK_VALUE = 5
const MAX_KEYWORD_LEN = 10
const MAX_ADDRESS_NUM = 10
const MAX_COMMENT_LEN = 255

type GlobalVariables struct {
	cRegBonus             uint    //客户的注册奖励（元）
	cInvBonus             uint    //邀请客户注册成功的奖励（元）
	cFirstOrderBonus      uint    //第一次成功下单且评价的现金奖励（元）
	cOrderBonus           uint    //正常下单且评价的积分奖励
	adjacentRange         uint    //寻找附近阿姨的范围（米）
	hKPsServiceStart      uint    //HKP服务起始时间（几点）
	hKPsServiceStop       uint    //HKP服务结束时间（几点）
	hKPRankAllBase        float32 //HKP综合服务质量评价的基础值
	hKPRankSpeedBase      float32 //HKP服务速度评价的基础值
	hKPRankQualityBase    float32 //HKP服务质量评价的基础值
	hKPRankAttitudeBase   float32 //HKP服务态度评价的基础值
	hKPMaxCount           uint8   //订单中HKP的最大数量
	hKPListLimit          uint    //搜索HKP时最多返回的个数
	vCodeTimeout          uint    //验证码过期时间（分钟）
	hKPServiceConfirmTime uint    //服务开始多久后可以确认是否完成（分钟）
	androidVersion        string  //安卓版本号
	iPhoneVersion         string  //iPhone版本号
	androidUrl            string  //安卓新版下载地址
	iPhoneUrl             string  //iPhone新版下载地址
	androidForce          uint    //安卓更新是否是强制更新
	iPhoneForce           uint    //iPhone更新是否是强制更新
}
type RoleJobMap map[uint]uint
type UIDType uint

var Variables GlobalVariables
var InitDate time.Time = time.Date(1901, time.January, 1, 0, 0, 0, 0, time.Local)
var RoleJob RoleJobMap

func (r *RoleJobMap) Init() {
	RoleJob = make(RoleJobMap)
	RoleJob[ROLE_HKP] = HKP_JOB_MAX
}
func (r *RoleJobMap) IsValidJob(rid uint, jid uint) bool {
	max, found := RoleJob[rid]
	if found && max > jid && jid > 0 {
		return true
	} else {
		return false
	}
}

func (v *GlobalVariables) InitVariables(db *sql.DB) (err error) {
	stmt, err := db.Prepare(SQL_InitVariables)
	if err != nil {
		return
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		return
	}
	defer rows.Close()

	var name, value string
	for rows.Next() {
		if err = rows.Scan(&name, &value); err != nil {
			return
		}
		switch name {
		case "CRegBonus":
			Variables.cRegBonus, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "CInvBonus":
			Variables.cInvBonus, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "AdjacentRange":
			Variables.adjacentRange, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "HKPServiceStart":
			Variables.hKPsServiceStart, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "HKPServiceStop":
			Variables.hKPsServiceStop, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "HKPRankAllBase":
			Variables.hKPRankAllBase, err = format.ParseFloat(value)
			if err != nil {
				return err
			}
		case "HKPRankSpeedBase":
			Variables.hKPRankSpeedBase, err = format.ParseFloat(value)
			if err != nil {
				return err
			}
		case "HKPRankQualityBase":
			Variables.hKPRankQualityBase, err = format.ParseFloat(value)
			if err != nil {
				return err
			}
		case "HKPRankAttitudeBase":
			Variables.hKPRankAttitudeBase, err = format.ParseFloat(value)
			if err != nil {
				return err
			}
		case "HKPMaxCount":
			Variables.hKPMaxCount, err = format.ParseUint8(value)
			if err != nil {
				return err
			}
		case "HKPListLimit":
			Variables.hKPListLimit, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "VCodeTimeout":
			Variables.vCodeTimeout, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "HKPServiceConfirmTime":
			Variables.hKPServiceConfirmTime, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "CFirstOrderBonus":
			Variables.cFirstOrderBonus, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "COrderBonus":
			Variables.cOrderBonus, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "AndroidVersion":
			Variables.androidVersion = value
		case "IPhoneVersion":
			Variables.iPhoneVersion = value
		case "AndroidUrl":
			Variables.androidUrl = value
		case "IPhoneUrl":
			Variables.iPhoneUrl = value
		case "AndroidForce":
			Variables.androidForce, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		case "IPhoneForce":
			Variables.iPhoneForce, err = format.ParseUint(value)
			if err != nil {
				return err
			}
		default:
		}
	}
	if err = rows.Err(); err != nil {
		return
	}
	return
}

func (v *GlobalVariables) CRegBonus() uint {
	return v.cRegBonus
}
func (v *GlobalVariables) CInvBonus() uint {
	return v.cInvBonus
}
func (v *GlobalVariables) AdjacentRange() uint {
	return v.adjacentRange
}
func (v *GlobalVariables) HKPServiceStart() uint {
	return v.hKPsServiceStart
}
func (v *GlobalVariables) HKPServiceStop() uint {
	return v.hKPsServiceStop
}
func (v *GlobalVariables) HKPRankAllBase() float32 {
	return v.hKPRankAllBase
}
func (v *GlobalVariables) HKPRankSpeedBase() float32 {
	return v.hKPRankSpeedBase
}
func (v *GlobalVariables) HKPRankQualityBase() float32 {
	return v.hKPRankQualityBase
}
func (v *GlobalVariables) HKPRankAttitudeBase() float32 {
	return v.hKPRankAttitudeBase
}
func (v *GlobalVariables) HKPMaxCount() uint8 {
	return v.hKPMaxCount
}
func (v *GlobalVariables) HKPListLimit() uint {
	return v.hKPListLimit
}
func (v *GlobalVariables) VCodeTimeout() uint {
	return v.vCodeTimeout
}
func (v *GlobalVariables) HKPServiceConfirmTime() uint {
	return v.hKPServiceConfirmTime
}
func (v *GlobalVariables) CFirstOrderBonus() uint {
	return v.cFirstOrderBonus
}
func (v *GlobalVariables) COrderBonus() uint {
	return v.cOrderBonus
}
func (v *GlobalVariables) AndroidVersion() string {
	return v.androidVersion
}
func (v *GlobalVariables) IPhoneVersion() string {
	return v.iPhoneVersion
}
func (v *GlobalVariables) AndroidUrl() string {
	return v.androidUrl
}
func (v *GlobalVariables) IPhoneUrl() string {
	return v.iPhoneUrl
}
func (v *GlobalVariables) AndroidForce() uint {
	return v.androidForce
}
func (v *GlobalVariables) IPhoneForce() uint {
	return v.iPhoneForce
}
