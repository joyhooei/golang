package main

import (
	"crypto/aes"
	"fmt"
	"yf_pkg/encrypt"
)

func main() {
	testAes()
	testAes2()
}

func testAes() {
	// AES-128。key长度：16, 24, 32 bytes 对应 AES-128, AES-192, AES-256
	key := "sfe023f_9fd&fwfl"
	result, err := encrypt.AesEncrypt16("polaris@studygolang", key)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)
	origData, err := encrypt.AesDecrypt16(result, key)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(origData))
}

func testAes2() {
	block, e := aes.NewCipher([]byte("sfe023f_9fd&fwflsfe023f_9fd&fwfl"))
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	enc := make([]byte, 100, 100)
	block.Encrypt(enc, []byte("hello world!!!!!"))
	fmt.Println(enc)
}
