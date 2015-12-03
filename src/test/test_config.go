package main

import (
	"fmt"
	"os"

	"code.serveyou.cn/pkg/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("invalid args")
		return
	}
	ic, err := config.NewConfig(os.Args[1])
	if err == nil {
		for k, v := range ic.Items {
			fmt.Printf("key=%s, value=%s\n", k, v)
		}
	} else {
		fmt.Println(err.Error())
	}
}
