package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage : %s post_data\n", os.Args[0])
		os.Exit(1)
	}
	data := []byte("73eb095de9d3c1b04248217f:6cf3076ab3b9433f9058579e")
	key := "Basic " + base64.StdEncoding.EncodeToString(data)
	req, err := http.NewRequest("POST", "https://api.jpush.cn/v3/push", strings.NewReader(os.Args[1]))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	req.Header.Set("Authorization", key)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(string(body))
		}
	}
}
