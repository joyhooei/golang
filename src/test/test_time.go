package main

import (
	"fmt"
	"time"
)

func main() {
	t := time.Now()
	fmt.Println(t.Unix())
	fmt.Println(t.Local())
	fmt.Println(t.Local().Format("2006-01-02 15:04:05"))
	fmt.Println(t.Format("2006-01-02 15:04:05"))
	fmt.Println(t.Format("2006-01-02"))
	loc, e := time.LoadLocation("Local")
	fmt.Println(loc, e)
	fmt.Println(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local))
}
