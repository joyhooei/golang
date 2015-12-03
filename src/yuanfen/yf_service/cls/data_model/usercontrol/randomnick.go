package usercontrol

import (
	"math/rand"
	"time"
	"yf_pkg/utils"
)

var nickList1, nickList2, nickList3, nickList4, nickList6 []string //nickList5,

func InitRandomNick() (e error) {

	dr, err := mdb.Query("select `name`,`type`,gender from user_nickname_random where type>0")
	if err != nil {
		return err
	}
	defer dr.Close()
	dr.Columns()
	sqlr, err := utils.ParseSqlResult(dr)
	if err != nil {
		return err
	}
	nickList1 = make([]string, 0, 0)
	nickList2 = make([]string, 0, 0)
	nickList3 = make([]string, 0, 0)
	nickList4 = make([]string, 0, 0)
	// nickList5 = make([]string, 0, 0)
	nickList6 = make([]string, 0, 0)
	for _, v := range sqlr {
		switch v["gender"] {
		case "1":
			switch v["type"] {
			case "1":
				nickList1 = append(nickList1, v["name"])
			case "2":
				nickList2 = append(nickList2, v["name"])
			case "3":
				nickList3 = append(nickList3, v["name"])
			}
		case "2":
			switch v["type"] {
			case "1":
				nickList4 = append(nickList1, v["name"])
			// case "2":
			// 	nickList5 = append(nickList2, v["name"])
			case "3":
				nickList6 = append(nickList3, v["name"])
			}
		}

	}
	return
}

func GetRandomNick() (result []string, result2 []string) {
	result = make([]string, 0, 0)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 10; i++ {
		s1 := nickList1[r.Intn(len(nickList1))]
		s2 := nickList2[r.Intn(len(nickList2))]
		s3 := nickList3[r.Intn(len(nickList3))]
		result = append(result, s1+s2+s3)
	}
	for i := 0; i < 10; i++ {
		s1 := nickList4[r.Intn(len(nickList4))]
		s2 := nickList2[r.Intn(len(nickList2))]
		s3 := nickList6[r.Intn(len(nickList6))]
		result2 = append(result2, s1+s2+s3)
	}
	return
}
