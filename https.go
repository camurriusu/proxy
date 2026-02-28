package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

// CONNECT
func handleHTTPS(request *http.Request, clientConn net.Conn) {
	defer clientConn.Close()

	// Check block list
	// Cannot block URL paths with HTTPS
	host := request.URL.Hostname()
	isBlocked, found := blocklist.Get(host)
	if found && isBlocked.(bool) == true {
		fmt.Println("Blocked request to", host)
		response := "HTTP/1.1 403 Forbidden\r\nContent-Type: text/html\r\n\r\n<html><body><h1>403 Forbidden: URL Blocked by Proxy</h1></body></html>\n"
		clientConn.Write([]byte(response))
		return
	}

	// Add port 443 if unspecified
	port := request.URL.Port()
	if port == "" {
		port = "443"
	}
	hostPort := net.JoinHostPort(host, port) // example.com:443

	// Establish connection with web server
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		log.Println("Error establishing connection with web server:", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer conn.Close()

	// On successful connection, send HTTP 200 OK to client to start handshake
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Println("Error sending 200 code to client:", err)
	}

	// Start full duplex communication between client and web server
	// done acts as a signal to end goroutine
	done := make(chan struct{}) // struct{} holds zero bytes

	// Copy bytes from server to client until EOF
	go func() {
		io.Copy(conn, clientConn)
		done <- struct{}{}
	}()

	// Copy bytes from client to server until EOF
	go func() {
		io.Copy(clientConn, conn)
		done <- struct{}{}
	}()

	// Block goroutine until done
	<-done
}
