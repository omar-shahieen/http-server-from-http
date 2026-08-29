package main

import (
	"fmt"
	"github/omar-shahieen/http-server/internal/request"
	"net"
)

func main() {
	ln, err := net.Listen("tcp", ":42069")
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		fmt.Println("connection accepted")

		req, err := request.RequestFromReader(conn)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Request line: \n - Method : %v \n - Target: %v \n - Version: %v \n ",
			req.RequestLine.Method,
			req.RequestLine.RequestTarget,
			req.RequestLine.HttpVersion)
		fmt.Println("connection closed")
	}
}
