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
	//fmt.Println("Received request from", conn.RemoteAddr())
	defer conn.Close()

	// Read request
	reader := bufio.NewReader(conn)

	// Connection: keep-alive is default
	for {
		request, err := http.ReadRequest(reader)
		if err != nil {
			log.Println(err)
			return
		}

		// Switch to appropriate handling method
		if request.Method == http.MethodConnect {
			handleHTTPS(request, conn)
			// CONNECT already kept connection alive
			return
		} else {
			// If error occurs, close connection
			err = handleHTTP(request, conn)
			if err != nil {
				log.Println(err)
				return
			}
			// Check Connection: close
			if request.Close {
				return
			}
		}
	}
}
