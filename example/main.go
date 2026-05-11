package main

import (
	"github.com/MasterDimmy/zipologger"
)

func main() {
	sessionLog := zipologger.NewLogger("./logs/conns/2.log", 2, 2, 2, false)
	sessionLog.Println("123")
	sessionLog.Flush()
}
