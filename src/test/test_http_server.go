package main

import (
	"io"
	"log"
	"net/http"
)

type Server struct {
	a int
}

// hello world, the web server
func (s *Server) HelloServer(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello, world!\n")
}

func main() {
	var s Server
	http.HandleFunc("/hello", s.HelloServer)
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
