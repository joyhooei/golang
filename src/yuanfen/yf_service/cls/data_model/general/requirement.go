package general

import "fmt"

const (
	RANGE_MAX  = 30000
	AGE_MAX    = 999
	HEIGHT_MAX = 999
)

//目标用户要符合的条件
type Requirement struct {
	Range         uint   `json:"range"`          //覆盖范围，RANGE_MAX表示不限
	Gender        int    `json:"gender"`         //性别，默认值为common.GENDER_BOTH（不限）
	Province      string `json:"province"`       //省，""表示不限
	City          string `json:"city"`           //市，""表示不限
	MinAge        int    `json:"min_age"`        //年龄下限，0表示不限
	MaxAge        int    `json:"max_age"`        //年龄上限，AGE_MAX表示不限
	MinHeight     int    `json:"min_height"`     //身高下限（厘米），0表示不限
	MaxHeight     int    `json:"max_height"`     //身高上限（厘米），HEIGHT_MAX表示不限
	Star          int    `json:"star"`           //星座，0-不限
	Online        int    `json:"online"`         //是否在线，0-不限
	Income        int    `json:"income"`         //收入下线，0-不限
	CertifyPhone  int    `json:"certify_phone"`  //手机认证用户，0-不限
	CertifyVideo  int    `json:"certify_video"`  //视频认证用户，0-不限
	CertifyIDcard int    `json:"certify_idcard"` //身份证认证用户，0-不限
}

//NewRequirement初始化对象，默认条件是都不限
//
//务必通过此方法获得Requirement的实例，否则身高上限、年龄上限等默认条件将会是0
func NewRequirement() (req Requirement) {
	req.Range = RANGE_MAX
	req.MaxAge = AGE_MAX
	req.MaxHeight = HEIGHT_MAX
	req.Gender = -1
	return
}

func (r *Requirement) NoRequirement() bool {
	return r.Range == RANGE_MAX && r.Gender == -1 && r.Province == "" && r.City == "" && r.MinAge == 0 && r.MaxAge == AGE_MAX && r.MinHeight == 0 && r.MaxHeight == HEIGHT_MAX && r.Star == 0 && r.Online == 0 && r.Income == 0 && r.CertifyPhone == 0 && r.CertifyVideo == 0 && r.CertifyIDcard == 0
}

//MatchMyRequirement检验是否满足自己的条件
func (r *Requirement) MatchMyRequirement(gender int, province, city string, distence float64, age, height, star, income, online, certifyPhone, certifyVideo, certifyIDcard int) bool {
	if r.Gender > 0 && r.Gender != gender {
		//fmt.Println("性别不符")
		return false
	}
	if r.CertifyPhone > certifyPhone || r.CertifyVideo > certifyVideo || r.CertifyIDcard > certifyIDcard {
		//fmt.Println("认证不符")
		return false
	}
	if r.MinAge > age || r.MaxAge < age {
		//fmt.Println("年龄不符")
		return false
	}
	if r.MinHeight > height || r.MaxHeight < height {
		//fmt.Println("身高不符")
		return false
	}
	if r.Star > 0 && r.Star != star {
		//fmt.Println("星座不符")
		return false
	}
	if r.Income > 0 && r.Income > income {
		//fmt.Println("收入不符")
		return false
	}
	if float64(r.Range) < distence {
		//fmt.Println("距离不符: 要求：", r.Range, "实际：", distence)
		return false
	}
	if r.Province != "" && province != r.Province {
		//fmt.Println("省份不符:target:", province, "my:", r.Province)
		return false
	}
	if r.City != "" && city != r.City {
		//fmt.Println("城市不符:target:", city, "my:", r.City)
		return false
	}
	if r.Online > online {
		//fmt.Println("在线不符")
		return false
	}
	//fmt.Println("符合条件")
	return true
}

func (r *Requirement) Key() string {
	return fmt.Sprintf("%v_%v_%v_%v_%v_%v_%v_%v_%v_%v", r.Range, r.MinAge, r.MaxAge, r.MinHeight, r.MaxHeight, r.Star, r.Online, r.CertifyPhone, r.CertifyVideo, r.CertifyIDcard)
}

func (r *Requirement) ToString() string {
	v := "择友条件："
	if r.Range == RANGE_MAX {
		v += fmt.Sprintf("距离：不限,")
	} else {
		v += fmt.Sprintf("距离<%v公里,", r.Range)
	}
	if r.Province == "" {
		v += fmt.Sprintf("省：不限,")
	} else {
		v += fmt.Sprintf("省：%v,", r.Province)
	}
	if r.City == "" {
		v += fmt.Sprintf("市：不限,")
	} else {
		v += fmt.Sprintf("市：%v,", r.City)
	}
	if r.Income == 0 {
		v += fmt.Sprintf("收入：不限,")
	} else {
		v += fmt.Sprintf("收入>=%v,", r.Income)
	}
	v += fmt.Sprintf("年龄：%v-%v, ", r.MinAge, r.MaxAge)
	v += fmt.Sprintf("身高：%v-%v, ", r.MinHeight, r.MaxHeight)
	if r.Star == 0 {
		v += fmt.Sprintf("星座：不限,")
	} else {
		v += fmt.Sprintf("星座：%v,", r.Star)
	}
	v += fmt.Sprintf("在线：%v, ", r.Online)
	v += fmt.Sprintf("电话认证：%v, ", r.CertifyPhone)
	v += fmt.Sprintf("视频认证：%v, ", r.CertifyVideo)
	v += fmt.Sprintf("身份证认证：%v", r.CertifyIDcard)
	return v
}
