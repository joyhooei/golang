package general

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"yf_pkg/format"
	"yf_pkg/net/http"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
)

type Result struct {
	IsOk int `json:"isok"`
	Code int `json:"code"`
	Data struct {
		Err      int    `json:"err"`
		Address  string `json:"address"`
		Sex      string `json:"sex"`
		Birthday string `json:"birthday"`
	} `json:"data"`
}

func IsMatch(id string, name string) (match bool, info map[string]interface{}, e error) {
	rows, err := mdb.Query("select birthday,sex from idcard where id= ? and name=?", id, name)
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var bir string
		var sex int
		if e := rows.Scan(&bir, &sex); e != nil {
			return false, nil, err
		}
		return true, map[string]interface{}{"birthday": bir, "gender": sex}, nil
	} else {
		b, err := http.HttpGet("api.id98.cn", "api/idcard", map[string]string{"appkey": "bf3dc06b9c1f1fce3c7fddf88be8ebee", "name": name, "cardno": id, "output": "json"}, 6)
		if err != nil {
			fmt.Println(err)
			return false, nil, err
		}
		fmt.Println("raw : ", string(b))
		var ret Result
		if err := json.Unmarshal(b, &ret); err != nil {
			return false, nil, err
		}
		fmt.Println("unmarshaled : ", ret)
		if ret.IsOk == 1 {
			if ret.Code == 1 {
				var sex int
				var birthday time.Time
				birthday, err = utils.ToTime(ret.Data.Birthday, format.TIME_LAYOUT_2)
				if err != nil {
					birthday, _ = utils.ToTime("1900-01-01", format.TIME_LAYOUT_2)
				}
				if ret.Data.Sex == "M" {
					sex = common.GENDER_MAN
				} else {
					sex = common.GENDER_WOMAN
				}
				_, err := mdb.Exec("insert into idcard(id,name,sex,birthday,address)values(?,?,?,?,?)", id, name, sex, birthday, ret.Data.Address)
				if err != nil {
					fmt.Println(err)
					return false, nil, err
				}
				return true, map[string]interface{}{"birthday": ret.Data.Birthday + " 00:00:00", "gender": sex}, nil
			} else {
				return false, nil, nil
			}
		} else {
			if ret.Code == 12 {
				Alert("idcard", "余额不足")
			} else {
				Alert("idcard", fmt.Sprintf("errcode=%v", ret.Code))
			}
			return false, nil, errors.New(fmt.Sprintf("query error. code=%v", ret.Code))
		}
	}
}
