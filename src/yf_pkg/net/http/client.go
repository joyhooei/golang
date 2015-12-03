package http

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

var client http.Client

const (
	DEFAULT_TIMEOUT = 2
)

func init() {
	client.Timeout = DEFAULT_TIMEOUT * time.Second
}

//发送http的GET或POST请求
//data如果为nil，则发GET请求，否则发POST请求
func HttpSend(host string, path string, params map[string]string, cookies map[string]string, data []byte) (body []byte, e error) {
	return Send("http", host, path, params, nil, cookies, data)
}

func HttpGet(host string, path string, params map[string]string, timeout int) (body []byte, e error) {
	return send("http", host, path, params, nil, nil, nil, timeout)
}

//发送http(s)的GET或POST请求
//data如果为nil，则发GET请求，否则发POST请求
func Send(protocal string, host string, path string, params map[string]string, header map[string]string, cookies map[string]string, data []byte, timeout ...int) (body []byte, e error) {
	to := DEFAULT_TIMEOUT
	if len(timeout) > 0 {
		to = timeout[0]
	}
	return send(protocal, host, path, params, header, cookies, data, to)
}

func send(protocal string, host string, path string, params map[string]string, header map[string]string, cookies map[string]string, data []byte, timeout int) (body []byte, e error) {
	m := "GET"
	if data != nil {
		m = "POST"
	}
	v := url.Values{}
	for key, value := range params {
		v.Set(key, value)
	}
	req_url := &url.URL{
		Host:     host,
		Scheme:   protocal,
		Path:     path,
		RawQuery: v.Encode(),
	}
	req, e := http.NewRequest(m, req_url.String(), bytes.NewBuffer(data))
	if e != nil {
		return nil, e
	}
	for k, v := range header {
		req.Header.Add(k, v)
	}
	for k, v := range cookies {
		var cookie http.Cookie
		cookie.Name = k
		cookie.Value = v
		req.AddCookie(&cookie)
	}
	c := &client
	if timeout != DEFAULT_TIMEOUT {
		c = &http.Client{}
		c.Timeout = time.Duration(timeout) * time.Second
	}
	//fmt.Println(req.URL)
	resp, e := c.Do(req)
	if e != nil {
		return nil, e
	}
	body, e = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	return
}
