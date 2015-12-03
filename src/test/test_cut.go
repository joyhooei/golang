package main

import "fmt"

func split(pre uint8) (sp int) {
	sp = 1
	switch {
	case pre >= 0xC0 && pre < 0xE0:
		sp = 2
	case pre >= 0xE0 && pre < 0xF0:
		sp = 3
	case pre >= 0xF0 && pre < 0xF8:
		sp = 4
	case pre >= 0xF8 && pre < 0xFC:
		sp = 5
	case pre >= 0xFC:
		sp = 6
	}
	return
}

func main() {
	a := "您23好a啥地方开始的"
	sp := 1
	for i := 0; i < len(a); {
		sp = split(a[i])
		fmt.Println(string(a[i : i+sp]))
		i += sp
	}

}
