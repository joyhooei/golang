//用时间和一个16位的标签共同生成sorted set的分值，时间放在高位，这样既不影响按时间排序，还能记录一些额外的信息。
package zscore

import (
	"fmt"
	"time"
)

//生成score，时间会以秒数的形式存储在64位整型的前(64-bits)位，tag会存储在后bits位。
func MakeZScore(tm time.Time, tag uint32, bits uint) int64 {
	score := (tm.Unix() << bits) + int64(tag)
	fmt.Printf("tm=%b,tag=%b,score=%b(%u)\n", tm.Unix(), tag, score, score)
	return score
}

//从score中提取标签的值
func GetTagFromScore(score int64, bits uint) uint32 {
	fmt.Printf("score=%b,tag=%b\n", score, score&((int64(1)<<bits)-1))
	return uint32(score & ((int64(1) << bits) - 1))
}

//从score中提取时间
func GetTimeFromScore(score int64, bits uint) time.Time {
	fmt.Printf("tm=%b\n", score>>bits)
	return time.Unix(score>>bits, 0)
}
