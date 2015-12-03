package main

import (
	"fmt"
	"code.serveyou.cn/pkg/encrypt"
)

func main(){
	data, key := "hello", "world"
	fmt.Println("data=",data," key=",key)
	cr, _ := encrypt.AesEncrypt16(data, key)
	fmt.Println("encrypted=",cr)
	de, _ := encrypt.AesDecrypt16(cr, key)
	fmt.Println("decrypted=", de)
	md5 := "hello"
	fmt.Println(md5, " md5 ",encrypt.Md5Sum(md5))
}
