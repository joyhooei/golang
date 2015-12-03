package main

import (
	"code.serveyou.cn/model"
	"fmt"
)

func main(){
	a := model.NewUser()
	fmt.Println(a.Birthday())

	m := make(map[string]string)
	b := m["a"]
	if b == "" {
		fmt.Println("b = \"\"")
	}
}
