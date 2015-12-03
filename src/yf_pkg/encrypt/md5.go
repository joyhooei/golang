/*
md5加密，封装一下，更简单易用
*/
package encrypt

import (
	"crypto/md5"
	"encoding/hex"
)

//MD5加密
func MD5Sum(key string) string {
	h := md5.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}
