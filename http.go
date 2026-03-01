package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

// GET
func handleHTTP(request *http.Request, clientConn net.Conn) error {
	start := time.Now()

	// Check block list for full URL
	url := request.URL.String()
	isBlocked, found := blocklist.Get(url)
	if found && isBlocked.(bool) == true {
		fmt.Println("Blocked request to", url)
		response := "HTTP/1.1 403 Forbidden\r\nConnection: close\r\nContent-Type: text/html\r\n\r\n<html><body><h1>403 Forbidden: URL Blocked by Proxy</h1></body></html>\n"
		clientConn.Write([]byte(response))
		return errors.New("Request to " + url + " blocked by proxy")
	}

	host := request.URL.Hostname()
	port := request.URL.Port()
	// Add port 80 if unspecified
	if port == "" {
		port = "80"
	}
	// e.g. key = example.com:80/index.html
	key := request.URL.String()
	// Check cache only if request is GET
	if request.Method == http.MethodGet {
		data, found := cache.Get(key)
		if found {
			clientConn.Write(data.([]byte))
			fmt.Printf("HIT %s in %v\n", key, time.Since(start))
			return nil
		}
	}

	// Establish connection with web server
	hostPort := net.JoinHostPort(host, port) // example.com:80
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		log.Println("Error establishing connection with web server:", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return err
	}
	// While this breaks keep-alive between proxy and server,
	// it greatly simplifies code unless http.Transport is used
	defer conn.Close()

	// Send request to web server
	err = request.Write(conn)
	if err != nil {
		log.Println("Error forwarding request to web server:", err)
		return err
	}

	// Read response from web server
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		log.Println("Error reading response from web server:", err)
		return err
	}
	// Required by http.ReadResponse
	defer response.Body.Close()
	// Dump response bytes
	responseDump, err := httputil.DumpResponse(response, true)
	if err != nil {
		log.Println("Error dumping response:", err)
		// If error, forward response back to client without saving byte dump
		err = response.Write(clientConn)
		if err != nil {
			log.Println("Error relaying response to client:", err)
			return err
		}
		// Avoid caching broken dump
		return nil
	}

	// Save cache only if HTTP status is 200 OK
	if response.StatusCode == http.StatusOK && request.Method == http.MethodGet {
		cache.Set(key, responseDump)
	}

	// Send cached web page to client
	clientConn.Write(responseDump)
	fmt.Printf("MISS %s in %v\n", key, time.Since(start))

	// Easy way of closing keep-alive loop in server.go
	if response.Close {
		request.Close = true
	}

	return nil
}
