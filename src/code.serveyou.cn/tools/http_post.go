package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"code.serveyou.cn/pkg/encrypt"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage : %s url post_data [key]\n", os.Args[0])
		os.Exit(1)
	}
	s := strings.Replace(os.Args[2], "\\r\\n", "\r\n", -1)
	u, err := url.Parse(os.Args[1])
	if err != nil {
		panic(err.Error())
		os.Exit(1)
	}
	var es string
	if u.Path[:3] == "/s/" {
		if len(os.Args) < 4 {
			panic("no key provided")
			return
		}
		s = "!!encrypt_head=shq365.cn\r\n" + s
		es, err = encrypt.AesEncrypt16(s, os.Args[3])
		if err != nil {
			panic(err.Error())
			os.Exit(1)
		}
	} else {
		fmt.Printf("s=%v\n", s)
		es = s
	}
	resp, err := http.Post(os.Args[1], "html/text", strings.NewReader(es))
	if err != nil {
		fmt.Println(err.Error())
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(string(body))
		}
	}
}
