package common

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"code.serveyou.cn/pkg/format"
	"github.com/djimenez/iconv-go"
)

var key string = "Basic " + base64.StdEncoding.EncodeToString([]byte("a64ab9cd1ef54050ba7a9d61:14cda3cb0268bce5c58843ce"))

//var key string = "Basic " + base64.StdEncoding.EncodeToString([]byte("73eb095de9d3c1b04248217f:6cf3076ab3b9433f9058579e"))
var hclient *http.Client = &http.Client{}

func SendSMS(phone string, content string) (err error) {
	return
	content += "【生活圈】"
	output, err := iconv.ConvertString(content, "utf-8", "gbk")
	if err != nil {
		return
	}
	url := fmt.Sprintf("http://sdk.entinfo.cn:8060/z_mdsmssend.aspx?sn=SDK-BBX-010-20121&pwd=AD2694543FA3A0514AB7032E10688EEA&mobile=%v&content=%v", phone, output)
	fmt.Printf("URL: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("Response error : status code %v", resp.StatusCode))
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}
	bStr := string(body)
	fmt.Printf("Response: %s\n", bStr)
	if bStr == "1" || strings.Index(bStr, "-") == 0 {
		err = errors.New(fmt.Sprintf("response error : %s", bStr))
		return
	}
	return
}
func PushNotification(alias []UIDType, tags []string, title string, content string, extras map[string]interface{}) (err error) {
	data := make(map[string]interface{})
	data["platform"] = "all"
	audience := make(map[string]interface{})
	if tags != nil {
		audience["tag"] = tags
	}
	if alias != nil {
		audience["alias"] = alias
	}
	data["audience"] = audience

	ar := make(map[string]interface{})
	ar["title"] = title
	ar["extras"] = extras

	ios := make(map[string]interface{})
	ios["extras"] = extras
	ios["bagde"] = 1
	ios["alert"] = content
	ios["sound"] = "default"
	ios["content-available"] = 1

	noti := make(map[string]interface{})
	noti["alert"] = content
	noti["android"] = ar
	noti["ios"] = ios

	data["notification"] = noti

	msg := make(map[string]interface{})
	msg["title"] = title
	msg["msg_content"] = content
	msg["content_type"] = 1
	msg["extras"] = extras

	data["message"] = msg

	opt := make(map[string]interface{})
	opt["apns_production"] = "False"
	data["options"] = opt

	json := format.GenerateJSON(data)
	fmt.Println(json)
	req, err := http.NewRequest("POST", "https://api.jpush.cn/v3/push", strings.NewReader(string(json)))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	req.Header.Set("Authorization", key)
	resp, err := hclient.Do(req)
	if err != nil {
		return
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		} else {
			r := string(body)
			if strings.Index(r, "\"error\"") >= 0 {
				err := errors.New(r)
				return err
			} else {
				fmt.Println(r)
			}
		}
	}
	return
}
func PushMessage(alias []UIDType, tags []string, title string, content string, extras map[string]interface{}) (err error) {
	data := make(map[string]interface{})
	data["platform"] = "all"
	audience := make(map[string]interface{})
	if tags != nil {
		audience["tag"] = tags
	}
	if alias != nil {
		audience["alias"] = alias
	}
	data["audience"] = audience

	msg := make(map[string]interface{})
	msg["title"] = title
	msg["msg_content"] = content
	msg["content_type"] = 1
	msg["extras"] = extras

	data["message"] = msg

	json := format.GenerateJSON(data)
	fmt.Println(json)
	req, err := http.NewRequest("POST", "https://api.jpush.cn/v3/push", strings.NewReader(string(json)))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	req.Header.Set("Authorization", key)
	resp, err := hclient.Do(req)
	if err != nil {
		return
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		} else {
			r := string(body)
			if strings.Index(r, "{\"error\"") >= 0 {
				err := errors.New(r)
				return err
			} else {
				fmt.Println(r)
			}
		}
	}
	return
}
