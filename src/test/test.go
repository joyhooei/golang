package main

import (
	"fmt"
	"yf_pkg/utils"
)

func main() {
	fmt.Println(utils.DurationTo(1, 6, 0, 0).Seconds())
}
