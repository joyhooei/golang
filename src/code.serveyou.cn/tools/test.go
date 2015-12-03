package main

import (
	"fmt"
	"os"
	"strings"

	"crypto/md5"
	"encoding/hex"
	"code.serveyou.cn/pkg/encrypt"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage : %s url post_data [key]\n", os.Args[0])
		os.Exit(1)
	}
	m := md5.New()
	m.Write([]byte(os.Args[3]))
	a := m.Sum(nil)
	println(len(a))
	println(hex.EncodeToString(m.Sum(nil)))
	s := strings.Replace(os.Args[2], "\\r\\n", "\r\n", -1)
	es, err := encrypt.AesEncrypt16(s, os.Args[3])
	fmt.Println(es)
	if err != nil {
		panic(err.Error())
		os.Exit(1)
	}
}
