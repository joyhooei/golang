package login

import (
	"encoding/json"
	"errors"
	// "fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	wb_user_url   = "https://api.weibo.com/2/users/show.json?"
	wb_appkey     = "2156353279"
	wb_App_Secret = "5314de949b3b7cb4a2fdc9faa115c9d3"
	wb_tmp_token  = "2.00TI1JGGbzpv2C177486e25etzNuwB"
)

func GetWeiboUinfo(token string, openid string) (result map[string]interface{}, e error) {
	v := url.Values{}
	v.Set("source", wb_appkey)
	v.Set("access_token", token)
	v.Set("uid", openid)
	resp, e := http.Get(wb_user_url + v.Encode())
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
	if verror, ok := rmap["error"]; ok {
		return nil, errors.New(verror.(string))
	}
	return rmap, nil
}
