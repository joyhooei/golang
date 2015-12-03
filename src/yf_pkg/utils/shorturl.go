package utils

import (
	"encoding/json"
	"errors"
	"yf_pkg/net/http"
)

const (
	cWeiboShortKey  = "2156353279"
	cWeiboShortHost = "http://api.weibo.com/2/short_url/shorten.json?"
)

func UrlToShort(url_long string) (url_short string, e error) {
	body, e := http.Send("http", "api.weibo.com", "2/short_url/shorten.json", map[string]string{"source": cWeiboShortKey, "url_long": url_long}, nil, nil, nil)
	if e != nil {
		return "", e
	}
	res := map[string]interface{}{}
	if e := json.Unmarshal(body, &res); e != nil {
		return "", e
	}
	switch v := res["urls"].(type) {
	case []interface{}:
		if len(v) >= 1 {
			switch v1 := v[0].(type) {
			case map[string]interface{}:
				switch url_short := v1["url_short"].(type) {
				case string:
					return url_short, nil
				}
			}
		}
	}
	return "", errors.New("parse return value failed")
}
