package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
)

func start() {
	// Start proxy at port 8080
	ln, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}

	// Create goroutine for each new client connection
	fmt.Println("Listening on", ln.Addr())
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go createConn(conn)
	}
}

func createConn(conn net.Conn) {
	fmt.Println("Received request from", conn.RemoteAddr())

	// Read request
	reader := bufio.NewReader(conn)
	request, err := http.ReadRequest(reader)
	if err != nil {
		log.Println(err)
		return
	}

	method := request.Method

	// Switch to appropriate handling method
	switch method {
	case http.MethodGet:
		handleHTTP(request, conn)
	case http.MethodConnect:
		handleHTTPS(request, conn)
	default:
		log.Println(method, "method is not supported. Must be GET or CONNECT")
		return
	}
}
