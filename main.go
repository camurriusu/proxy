package main

func main() {
	// Start management console
	go runConsole()
	// Run proxy
	start()
}
