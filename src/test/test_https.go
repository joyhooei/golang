package main

import (
	"fmt"
	"yf_pkg/push"
)

func main() {
	e := push.XiaoMi("Xk4V+iDrgHmZVQbjaH1t0w==", push.XIAOMI_MODE_NOTIFICATION, "alias", []string{"jiatao", "hello"}, "test", "你好")
	if e != nil {
		fmt.Println(e.Error())
	} else {
		fmt.Println("success")
	}
	e = push.XinGe("2100096315", "7e71c8d87cd8195d0696b83dacb4dc90", push.XINGE_MODE_MESSAGE, "alias", []string{"jiatao", "hello"}, "你好", "{\"content\":\"{\\\"hello\\\":1}\"}")
	if e != nil {
		fmt.Println(e.Error())
	} else {
		fmt.Println("success")
	}
}
