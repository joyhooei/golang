package main

import (
	"net/http"
	"os"
	"strings"
	"fmt"
	"io/ioutil"
)

func main(){
	s := strings.Replace(os.Args[2],"\\r\\n","\r\n", -1)
	resp, err := http.Post(os.Args[1],"html/text",strings.NewReader(s))
	if err != nil {
		fmt.Println(err.Error())
	}else{
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err.Error())
		}else{
			fmt.Println(string(body))
		}
	}
}
