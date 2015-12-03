/*
aes加密和解密相关函数，经过封装，更好用一些
*/
package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// 128位aes加密, key会被md5，所以不必是16的整数倍
func AesEncrypt16(origData string, key string) (string, error) {
	raw := []byte(origData)
	h := md5.New()
	h.Write([]byte(key))
	newKey := []byte(hex.EncodeToString(h.Sum(nil)))
	result, err := AesEncrypt(raw, newKey[:16])
	//fmt.Printf("newKey=%s\n", string(newKey))
	//result, err := AesEncrypt(raw, []byte(key))
	if err != nil {
		return "", err
	}
	//fmt.Printf("result : %v\n", string(result))
	return base64.StdEncoding.EncodeToString(result), nil
}

// 128位aes解密, key会被md5，所以不必是16的整数倍
func AesDecrypt16(crypted string, key string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(crypted)
	if err != nil {
		return "", err
	}
	h := md5.New()
	h.Write([]byte(key))
	newKey := []byte(hex.EncodeToString(h.Sum(nil)))
	result, err := AesDecrypt(data, newKey[:16])
	//result, err := AesDecrypt(data, []byte(key))
	if err != nil {
		return "", err
	}
	return string(result), nil
}

//aes加密，key必须是16字节的整数倍
func AesEncrypt(origData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	fmt.Printf("blockSize=%v\n", blockSize)
	origData = pKCS5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

//aes解密，key必须是16字节的整数倍
func AesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData, err = pKCS5UnPadding(origData)
	return origData, err
}
func zeroPadding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}

func zeroUnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func pKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pKCS5UnPadding(origData []byte) ([]byte, error) {
	length := len(origData)
	if length == 0 {
		return origData, nil
	}
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length-1])
	if length < unpadding || unpadding < 0 {
		err := errors.New("decrypt failed")
		return nil, err
	}
	return origData[:(length - unpadding)], nil
}
