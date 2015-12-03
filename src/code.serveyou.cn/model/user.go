package model

import (
	"database/sql"
	"errors"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/pkg/format"
)

type User struct {
	id         common.UIDType
	nickName   string
	lastName   string
	firstName  string
	cellphone  string
	idCardType uint8
	idCardNum  string
	sex        uint8
	birthday   time.Time
	birthplace string
	regTime    time.Time
	mail       string
	inviter    common.UIDType

	healthCardTimeout time.Time //健康证到期时间
	//defaultAddr uint8
}

type Balance struct {
	UserID  common.UIDType
	Balance float32
	Score   uint
	Coupons uint
}

type UserMap map[common.UIDType]User
type PhoneUserMap map[string]common.UIDType

func NewBalance(uid common.UIDType) (b *Balance) {
	return &Balance{uid, 0, 0, 0}
}
func NewUser() (u *User) {
	//u = &User{0, "", "", "", cellphone, 0, "", common.SEX_UNKNOWN, common.InitDate, "", common.InitDate, "", 0, 0}
	u = &User{0, "", "", "", "", 0, "", common.SEX_UNKNOWN, common.InitDate, "", common.InitDate, "", 0, common.InitDate}
	return u
}

func (u *User) InitByDB(rows *sql.Rows) (err error) {
	var birthday, regTime, hCard string
	//err = rows.Scan(&u.id, &u.nickName, &u.lastName, &u.firstName, &u.cellphone, &u.idCardType, &u.idCardNum, &u.sex, &birthday, &u.birthplace, &regTime, &u.mail, &u.inviter, &u.defaultAddr)
	err = rows.Scan(&u.id, &u.nickName, &u.lastName, &u.firstName, &u.cellphone, &u.idCardType, &u.idCardNum, &u.sex, &birthday, &u.birthplace, &regTime, &u.mail, &u.inviter, &hCard)
	if err != nil {
		return
	}
	u.birthday, err = time.ParseInLocation(format.TIME_LAYOUT_2, birthday, time.Local)
	if err != nil {
		return
	}
	u.healthCardTimeout, err = time.ParseInLocation(format.TIME_LAYOUT_2, hCard, time.Local)
	if err != nil {
		return
	}
	u.regTime, err = time.ParseInLocation(format.TIME_LAYOUT_1, regTime, time.Local)
	return
}

func (u *User) Id() common.UIDType {
	return u.id
}
func (u *User) SetId(id common.UIDType) {
	u.id = id
}

func (u *User) NickName() string {
	return u.nickName
}
func (u *User) SetNickName(n string) (err error) {
	if len(n) > 50 {
		err = errors.New("nick name too long : [" + n + "]")
		return
	} else if len(n) < 4 {
		err = errors.New("nick name too short : [" + n + "]")
		return
	}
	u.nickName = n
	return nil
}

func (u *User) LastName() string {
	return u.lastName
}
func (u *User) SetLastName(n string) (err error) {
	if len(n) > 10 {
		err = errors.New("last name too long : [" + n + "]")
		return
	}
	u.lastName = n
	return nil
}

func (u *User) FirstName() string {
	return u.firstName
}
func (u *User) SetFirstName(n string) (err error) {
	if len(n) > 20 {
		err = errors.New("first name too long : [" + n + "]")
		return
	}
	u.firstName = n
	return nil
}

func (u *User) Cellphone() string {
	return u.cellphone
}
func (v *User) SetCellphone(c string) (err error) {
	if !format.CheckCellphone(c) {
		err = errors.New("cellphone number [" + c + "] format error.")
		return
	}
	v.cellphone = c
	return
}

func (u *User) IdCardType() uint8 {
	return u.idCardType
}
func (u *User) SetIdCardType(t uint8) {
	u.idCardType = t
}

func (u *User) IdCardNum() string {
	return u.idCardNum
}
func (u *User) SetIdCardNum(n string) (err error) {
	if len(n) > 20 {
		err = errors.New("id card number too long : [" + n + "]")
		return
	}
	u.idCardNum = n
	return nil
}

func (u *User) Sex() uint8 {
	return u.sex
}
func (u *User) SetSex(s uint8) (err error) {
	if s > 3 {
		err = errors.New("invalid sex. sex only can be 0(secret), 1(male), 2(female), 3(unset).")
		return
	}
	u.sex = s
	return
}

func (u *User) Birthday() time.Time {
	return u.birthday
}
func (u *User) SetBirthday(b time.Time) {
	u.birthday = b
}

func (u *User) Birthplace() string {
	return u.birthplace
}
func (u *User) SetBirthplace(b string) (err error) {
	if len(b) > 20 {
		err = errors.New("birthplace too long : [" + b + "]")
		return
	}
	u.birthplace = b
	return nil
}

func (u *User) RegTime() time.Time {
	return u.regTime
}
func (u *User) SetRegTime(r time.Time) {
	u.regTime = r
}

func (u *User) Mail() string {
	return u.mail
}
func (u *User) SetMail(m string) (err error) {
	if len(m) > 50 {
		err = errors.New("mail string [" + m + "] too long.")
		return
	}
	if !format.CheckEmail(m) {
		err = errors.New("mail format invalid : " + m)
		return
	}
	u.mail = m
	return nil
}

func (u *User) Inviter() common.UIDType {
	return u.inviter
}
func (u *User) SetInviter(i common.UIDType) (err error) {
	u.inviter = i
	return
}

func (u *User) HealthCardTimeout() time.Time {
	return u.healthCardTimeout
}
func (u *User) SetHealthCardTimeout(t time.Time) {
	u.healthCardTimeout = t
}

/*
func (u *User) DefaultAddr() uint8 {
	return u.defaultAddr
}
func (u *User) SetDefaultAddr(d uint8) {
	u.defaultAddr = d
}
*/
