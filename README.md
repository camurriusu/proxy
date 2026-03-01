# Web Proxy Server
## Riccardo Riggi – 23372659
### Introduction
This document outlines the design and implementation of a web proxy server written in Go. The servers sit between a client and an external web server; relaying requests sent from the client to a web server and vice versa. It supports simultaneous HTTP/HTTPS connections and improves efficiency using local HTTP caching, proven by logged timing data. This project also includes an editable host name block list through the command line interface.
The server is split into six files, each implementing a specific aspect of the server.
- `main.go`
- `server.go`
- `http.go`
- `https.go`
- `console.go`
- `safemap.go`

### Diagram
<img width="573" height="432" src="https://github.com/user-attachments/assets/5a17854d-82b8-4f4a-bd9b-065bbb9fa9c0" />

### Concurrency
The cache and blocklist data structures are defined by a struct in safemap.go that implements a thread-safe dictionary in Go.
```go
type SafeMap struct {
	mutex sync.RWMutex
	data  map[string]any
}

func (sf *SafeMap) Get(key string) (any, bool) {
    sf.mutex.RLock()
    defer sf.mutex.RUnlock()
    value, found := sf.data[key]
    return value, found
}

func (sf *SafeMap) Set(key string, value any) {
    sf.mutex.Lock()
    defer sf.mutex.Unlock()
    sf.data[key] = value
}
```
It has a getter and setter that ensure atomicity for reading or writing data, using Go’s sync.RWMutex type, which is essential since we are interacting with these data structures concurrently from potentially several goroutines. Examples of their usage include:
```go
data, found := cache.Get(key)
isBlocked, found := blocklist.Get(url)
cache.Set(key, responseDump)
blocklist.Set(targetUrl, true)
```
Throughout the project, threading is implemented by using goroutines, i.e. functions called using the `go` keyword. This is the go-to method for concurrency in Go, whether it is to perform multiple I/O operations or HTTP requests. They are very lightweight and managed by the Go runtime.
### Management console
The management console lives in the `runConsole()` method in `console.go`, running on a separate thread from the web proxy server. There are three commands: `block`, `unblock`, and `list`. Specifying a URL argument for block or unblock will appropriately update the blocklist. The list command will output all URLs marked as blocked. The blocklist is always checked before trying to begin a connection with a web server. On the event the request URL is blocked, we return a `HTTP/1.1 403 Forbidden` error code to the client. It is important to note that due to the encrypted nature of HTTPS, we check the blocklist for the requested hostname, excluding the pathname.

### On start-up
The server listens for incoming connections at an arbitrary port, in our case port number 8080, with `ln, err := net.Listen("tcp", "127.0.0.1:8080")`. On each successful connection from a client, we create a new thread to handle it in `createConn()`.
```go
for {
	conn, err := ln.Accept()
	if err != nil {
		log.Println(err)
		continue
	}
	go createConn(conn)
}
```
Then, it is important to read the method type in the request header. A `CONNECT` signals the beginning of an encrypted HTTPS (and therefore HTTP/2) connection, while any other request method, e.g. `GET`, `POST`, etc. implies a HTTP/1.1 connection. This works because a `CONNECT` request method is required to start a HTTPS connection, after which any type of request after the handshake is supported until either end closes the connection. Therefore, any other request method our proxy server may see must be `HTTP` only. 
```go
if request.Method == http.MethodConnect {
	handleHTTPS(request, conn)
} else {
	handleHTTP(request, conn)
}
```
### Handling HTTPS
Below is the flow of `handleHTTPS()`:
1.	Check if the hostname found in the request header is blocked.
2.	If port is unspecified, set it to port number 443.
3.	Establish a connection with the web server using TCP. On error, return a `502 Bad Gateway` error code and stop.
4.	Send `HTTP/1.1 200 Connection Established` to the client to begin the handshake.
5.	Blindly relay bytes of data from the client to the web server and vice versa, using anonymous threaded goroutines performing io.Copy(). We take advantage of Go’s channels to signal the end of a byte stream from either end of the connection to close the connection.

```go
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
```
Note that we do not implement any caching features for HTTPS. This is because we cannot cache specific web pages since the data passing between client and web server is encrypted. The user may rely on their browser cache.

### Handling HTTP and Caching
For HTTP, it’s a different story:
1.	Check if the request header’s URL (including pathname) is blocked.
2.	If port is unspecified, set it to port number 80.
3.	Check the cache for data using the hostname, port number (in case the same URL serves different web pages under different ports), and pathname as the dictionary key, e.g. `example.com:80/index.html`
4.	If found, we send the data back to the client and report the time elapsed to do so.
5.	Else, we begin a timer and establish a connection with the specified web server.
6.	Relay the client’s request to the web server.
7.	Read the response, and if the status code is `HTTP/1.1 200 OK`, then dump the bytes into the cache.
8.	Forward the response to the client and report the time elapsed since we started the timer.

Note that the cache is only checked and written to if the request method is `GET`. Caching `POST` and other request methods would be nonsensical since they submit data or modify the server’s data.
```go
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
// Save cache only if HTTP status is 200 OK
if response.StatusCode == http.StatusOK {
	cache.Set(key, responseDump)
}
```

### Libraries used
We make use of Go’s net and bufio libraries.
- `net.Listen()` is used to create the `net.Listener` object ln, by specifying a type of connection (e.g. `“tcp”` for both IPv4/v6, `“tcp4”` for only IPv4, etc.) and an IP and port to bind to. If you leave out the IP, it will listen on both localhost and your public IP (which I tried and got discovered by botnets). I am using `net.Listen(“tcp”, “127.0.0.1:8080”)`. Any incoming connections are buffered until accepted by calling `ln.Accept()`.
- Calling `ln.Accept()` returns a thread-safe variable implementing the net.Conn interface. A `net.Conn` variable implements the `Read()`, `Write()`, `Close()`, `LocalAddr()`, and `RemoteAddr()` methods. It is a low-level representation of any network connection, and enables us to read and write bytes, close the connection, or get the address at either end of the connection. Incoming bytes are buffered by the operating system’s TCP receive buffer until we `Read()` them.
- To make proper use of the new connection, we must be able to read incoming HTTP requests. Using `net.Conn`’s `Read()` won’t work, since it deletes the read bytes from the buffer. Since we read in chunks, this is problematic. For example, if we read 200 bytes but only need to parse the first 50, the next 150 bytes are lost. Therefore, we need our own buffer in RAM, specifically a `bufio.Reader`, from which we can safely read bytes from a `net.Conn` object using `Peek()`. This also improves performance, since we extract about 4KB from the TCP buffer at a time in one system call, and from those 4KB we can read byte by byte (which is what `http.ReadRequest()` does). Without it, to read the incoming byte stream byte by byte, we would need 4096 system calls instead of one.
- The `bufio.Reader` is passed into `http.ReadRequest()` (part of the net library) to create a pointer to a `http.Request` object. This object has wrapped our `bufio.Reader` so that it can peek bytes coming into `net.Conn`, until it spots the header delimiter `\r\n\r\n`. Only then will it `Read()` the necessary bytes, clearing our buffer. To extract the body, we read the next number of bytes specified by the header or just until we see an `EOF`.
- The request object we have created is very useful, especially for handling unencrypted HTTP requests. We can easily extract any relevant header fields. We extract the URL hostname and port and use them to create a new `net.Conn` object, creating a connection between the proxy and the automatically resolved web server. The extracted request object has a `Write()` method that can take this `net.Conn` object as an argument. This will correctly dump the bytes contained in the request (as exactly how we intercepted them from our bufio.Reader) so that they are relayed to the web server.
- To create the new `net.Conn` object, we use `net.Dial()`. This method is the opposite of `net.Listen()` and uses the same arguments. If a URL is passed, it will resolve it to an IP using our configured DNS server. Afterwards, it will complete the 3-way handshake to finally return our `net.Conn` object.
- To handle HTTPS requests, once we established our own connection with the web server, we manually write back a `HTTP/1.1 200 Connection Established` message back to client’s machine. Note that we have two `net.Conn` objects, one is between the client and us, while the second is between us and the webserver. We can set up a blind full duplex channel between the client and the webserver by manually copying bytes from one `net.Conn` object to the other using a small buffer. This is why we use `io.Copy()`. The only issue is that `io.Copy()` will block our goroutine until it receives an `EOF` signal. Therefore, both `io.Copy()` calls (one for either direction) live in separate goroutines. Our handleHTTPS goroutine waits until either `io.Copy()` is done. 
