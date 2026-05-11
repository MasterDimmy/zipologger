package main

import (
	"github.com/MasterDimmy/zipologger"
)

func main() {
	logger := zipologger.NewLogger("test.log", 1, 1, 1, false)
	logger.Println("Hello World")
	zipologger.Wait()
}