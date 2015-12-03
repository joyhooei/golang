package general

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"yf_pkg/encrypt"
	"yf_pkg/redis"
	"yf_pkg/utils"
	"yuanfen/redis_db"
)

const (
	s_SHORTMSG_URL      = "http://www.282930.cn/SMSReceiver.aspx?"
	s_SHORTMSG_USERNAME = "hnzdx"
	s_SHORTMSG_PASSWORD = "7441948"
)
const (
	s3_SHORTMSG_URL      = "http://sdk.entinfo.cn:8061/webservice.asmx/mdsmssend?"
	s3_SHORTMSG_USERNAME = "SDK-BBX-010-22729"
	s3_SHORTMSG_PASSWORD = "^-7a9c-4"
)

const (
	s4_SHORTMSG_URL      = "http://222.73.117.158/msg/HttpBatchSendSM?"
	s4_SHORTMSG_USERNAME = "VIP_jxwl"
	s4_SHORTMSG_PASSWORD = "Tch123456"
)

//第一种发送短信方式
func SendShortMsg2(phone string, smsg string) (e error) {
	v := url.Values{}
	v.Set("username", s_SHORTMSG_USERNAME)
	v.Set("password", s_SHORTMSG_PASSWORD)
	v.Set("targetdate", "")
	v.Set("mobiles", phone)
	v.Set("Content", smsg)
	resp, e := http.Get(s_SHORTMSG_URL + v.Encode())
	if e != nil {
		return e
	}
	_, e2 := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if e2 != nil {
		return e2
	}
	// fmt.Printf("%v", string(sr))
	return nil
}

//第二种发送短信方式
func SendShortMsg3(phone string, smsg string) (e error) {
	v := url.Values{}
	v.Set("sn", s3_SHORTMSG_USERNAME)
	v.Set("pwd", strings.ToUpper(encrypt.MD5Sum(s3_SHORTMSG_USERNAME+s3_SHORTMSG_PASSWORD)))

	// v.Set("pwd", s3_SHORTMSG_PASSWORD)
	v.Set("mobile", phone)
	v.Set("content", smsg)
	v.Set("ext", "")
	v.Set("stime", "")
	v.Set("rrid", "")
	v.Set("msgfmt", "")
	resp, e := http.Get(s3_SHORTMSG_URL + v.Encode())
	if e != nil {
		return e
	}
	sr, e2 := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println(fmt.Sprintf("sendshort3 %v", string(sr)))
	if e2 != nil {
		return e2
	}
	return nil
}

//第三种发送短信方式
func SendShortMsg4(phone string, smsg string) (e error) {
	v := url.Values{}
	v.Set("account", s4_SHORTMSG_USERNAME)
	v.Set("pswd", s4_SHORTMSG_PASSWORD)

	// v.Set("pwd", s3_SHORTMSG_PASSWORD)
	v.Set("mobile", phone)
	v.Set("msg", smsg)
	v.Set("needstatus", "true")
	resp, e := http.Get(s4_SHORTMSG_URL + v.Encode())
	if e != nil {
		return e
	}
	sr, e2 := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if e2 != nil {
		return e2
	}
	sa := string(sr)
	fmt.Println(fmt.Sprintf("sendshort4 %v", sa))
	var sresult string
	if ia := strings.Index(sa, ","); ia != -1 {
		s2 := sa[ia+1:]
		if ib := strings.Index(s2, "\n"); ib != -1 {
			sresult = s2[:ib]
		} else {
			sresult = s2
		}
	} else {
		return errors.New("resp has no ,")
	}
	if sresult != "0" {
		return errors.New("SendShortMsg ERROR: " + sresult)
	}
	return nil
}

const (
	sms_resend_timeout = 60
)

func SendSmsDelay(phone string, smsg string) (e error) {
	con := cache.GetWriteConnection(redis_db.CACHE_SMS_RESEND)
	defer con.Close()
	// args := make([]interface{}, 0, 0)
	// args = append(args, phone)

	i, e := redis.Int(con.Do("EXISTS", phone))
	if e != nil {
		return e
	}
	if i == 1 {
		return errors.New("发送验证码太频繁！")
	}
	e = SendShortMsg4(phone, smsg)
	if e != nil {
		return e
	}
	args := make([]interface{}, 0, 0)
	args = append(args, phone)
	args = append(args, sms_resend_timeout)
	args = append(args, utils.Now.Unix())
	_, e = con.Do("SETEX", args...)
	if e != nil {
		return e
	}
	return
}
