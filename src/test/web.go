package main

import (
	"net/http"
	"fmt"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("杨希、贾涛、唐子君！"))
	fmt.Println("Request from ", r.RemoteAddr)
}
func main() {
	http.HandleFunc("/hello",HelloHandler)
	http.ListenAndServe(":8888", nil)

}
