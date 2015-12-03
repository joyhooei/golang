package login

import (
	"math/rand"
	"strings"
	"time"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
)

const (
	check_TIMEOUT = 3600
)

func GetCodeRandom() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := r.Intn(1000000)
	scode := utils.ToString(code)
	scode = strings.Repeat("0", 6-len(scode)) + scode
	return scode
}

func GetCodeRandom2() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := r.Intn(10000)
	scode := utils.ToString(code)
	scode = strings.Repeat("0", 4-len(scode)) + scode
	return scode
}

func AddPhoneCode(phone string, code string) (e error) {
	err := rdb.SetEx(redis_db.REDIS_PHONE_CODE, phone, check_TIMEOUT, code)
	if err != nil {
		return err
	}

	return
}

func CheckPhoneCode(phone string, code string) (result int, e error) {
	v, err := redis.String(rdb.Get(redis_db.REDIS_PHONE_CODE, phone))
	if err != nil {
		return 0, err
	}
	if code != v {
		return 2, nil
	} else {
		rdb.Expire(redis_db.REDIS_PHONE_CODE, check_TIMEOUT, phone)
		return 1, nil
	}
	return
}

func GetPhoneCode(phone string) (code string, e error) {
	code, err := redis.String(rdb.Get(redis_db.REDIS_PHONE_CODE, phone))
	if err != nil {
		return "", err
	}
	return
}

func DelPhone(phone string) (e error) {
	err := rdb.Del(redis_db.REDIS_PHONE_CODE, phone)
	if err != nil {
		return err
	}
	return
}
