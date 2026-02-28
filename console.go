package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Thread safe block list
var blocklistMutex sync.RWMutex
var blocklist = make(map[string]bool) // key: url, value: boolean

func runConsole() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// Extract user input into a whitespace separated slice
		args := strings.Fields(strings.TrimSpace(scanner.Text()))

		// Ignore if user input is empty
		if len(args) == 0 {
			continue
		}

		switch args[0] {
		// block <url>
		case "block":
			if len(args) < 2 {
				fmt.Println("Usage: block <url>")
				continue
			}
			// Mark URL as blocked
			targetUrl := args[1]
			blocklistMutex.Lock()
			blocklist[targetUrl] = true
			blocklistMutex.Unlock()
			fmt.Println(targetUrl, "blocked")
		// unblock <url>
		case "unblock":
			if len(args) < 2 {
				fmt.Println("Usage: unblock <url>")
				continue
			}
			// Mark URL as unblocked
			targetUrl := args[1]
			blocklistMutex.Lock()
			blocklist[targetUrl] = false
			blocklistMutex.Unlock()
			fmt.Println(targetUrl, "unblocked")
		// list
		case "list":
			// List all blocked URLs
			blocklistMutex.RLock()
			for url, val := range blocklist {
				if val == true {
					fmt.Println(url)
				}
			}
			blocklistMutex.RUnlock()
		default:
			fmt.Println("Unknown command")
		}
	}
}
