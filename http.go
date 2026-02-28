package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

// GET
func handleHTTP(request *http.Request, clientConn net.Conn) {
	defer clientConn.Close()

	// Check block list for full URL
	url := request.URL.String()
	isBlocked, found := blocklist.Get(url)
	if found && isBlocked.(bool) == true {
		fmt.Println("Blocked request to", url)
		response := "HTTP/1.1 403 Forbidden\r\nContent-Type: text/html\r\n\r\n<html><body><h1>403 Forbidden: URL Blocked by Proxy</h1></body></html>\n"
		clientConn.Write([]byte(response))
		return
	}

	host := request.URL.Hostname()
	port := request.URL.Port()
	// Add port 80 if unspecified
	if port == "" {
		port = "80"
	}
	// e.g. key = example.com:80/index.html
	hostPort := net.JoinHostPort(host, port) // example.com:80
	key := hostPort + request.URL.RequestURI()

	// Check cache
	data, found := cache.Get(key)
	if found {
		startHit := time.Now()
		clientConn.Write(data.([]byte))
		fmt.Printf("HIT %s in %v\n", key, time.Since(startHit))
		return
	}

	startMiss := time.Now()

	// Establish connection with web server
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		log.Println("Error establishing connection with web server:", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer conn.Close()

	// Send request to web server
	err = request.Write(conn)
	if err != nil {
		log.Println("Error forwarding request to web server:", err)
		return
	}

	// Read response from web server
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		log.Println("Error reading response from web server:", err)
		return
	}

	// Dump response bytes
	responseDump, err := httputil.DumpResponse(response, true)
	if err != nil {
		log.Println("Error dumping response:", err)
		// If error, forward response back to client
		err = response.Write(clientConn)
		if err != nil {
			log.Println("Error relaying response to client:", err)
			return
		}
	}

	// (Needs Ctrl+Shift+R in browser to bypass local cache)
	// Save cache only if HTTP status is 200 OK
	if response.StatusCode == http.StatusOK {
		cache.Set(key, responseDump)
	}

	// Send cached web page to client
	clientConn.Write(responseDump)
	fmt.Printf("MISS %s in %v\n", key, time.Since(startMiss))
}
