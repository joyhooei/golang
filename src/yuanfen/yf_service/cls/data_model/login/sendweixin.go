package login

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	wx_access     = "https://api.weixin.qq.com/sns/oauth2/access_token?"
	wx_userinfo   = "https://api.weixin.qq.com/sns/userinfo?"
	wx_appid      = "wx8ad559b8b71321bb"
	wx_secret     = "31ebf97a656f08c122b329cf3b6df980"
	wx_grant_type = "authorization_code"
	wx_openid     = "openid"
)

func getWeixiniID(code string) (openid string, tpken string, e error) {
	v := url.Values{}
	v.Set("appid", wx_appid)
	v.Set("secret", wx_secret)
	v.Set("code", code)
	v.Set("grant_type", wx_grant_type)

	resp, e := http.Get(wx_access + v.Encode())
	if e != nil {
		return "", "", e
	}
	body, e2 := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if e2 != nil {
		return "", "", e2
	}
	rmap := make(map[string]interface{})
	if err := json.Unmarshal(body, &rmap); err != nil {
		return "", "", err
	}
	mlog.AppendInfo(fmt.Sprintf("getWeixiniID Param appid %v,secret %v,code %v, grant_type %v,result %v", wx_appid, wx_secret, code, wx_grant_type, rmap))
	if verror, ok := rmap["errmsg"]; ok {
		return "", "", errors.New(verror.(string))
	}
	vo, ok := rmap[wx_openid]
	if !ok {
		return "", "", errors.New("缺少字段")
	} else {
		openid, ok = vo.(string)
		if !ok {
			return "", "", errors.New("缺少字段")
		}

	}
	vo, ok = rmap["access_token"]
	if !ok {
		return "", "", errors.New("缺少字段 access_token")
	} else {
		tpken, ok = vo.(string)
		if !ok {
			return "", "", errors.New("缺少字段")
		}
	}
	return

}

func getWeixiniUinfo(openid string, tpken string) (result map[string]interface{}, e error) {
	v := url.Values{}
	v.Set("access_token", tpken)
	v.Set("openid", openid)
	resp, e := http.Get(wx_userinfo + v.Encode())
	if e != nil {
		return nil, e
	}
	body, e2 := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if e2 != nil {
		return nil, e2
	}
	rmap := make(map[string]interface{})
	if err := json.Unmarshal(body, &rmap); err != nil {
		return nil, err
	}
	mlog.AppendInfo(fmt.Sprintf("getWeixiniUinfo Param access_token %v,openid %v,result %v", tpken, openid, rmap))
	if verror, ok := rmap["errmsg"]; ok {
		return nil, errors.New(verror.(string))
	}
	_, ok := rmap[wx_openid]
	if !ok {
		return nil, errors.New("缺少字段")
	}
	return rmap, nil
}

func GetWeixin(code string) (result map[string]interface{}, e error) {
	openid, tpken, e := getWeixiniID(code)
	if e != nil {
		return nil, e
	}
	return getWeixiniUinfo(openid, tpken)
}
