package main

import (
	"github.com/MasterDimmy/zipologger"
)

func main() {
	defer zipologger.Wait()

	sessionLog := zipologger.NewLogger("./logs/conns/3.log", 2, 2, 2, false)
	sessionLog.Println("123 with wait")
}