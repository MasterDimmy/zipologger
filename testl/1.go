package main

import (
	"fmt"
	"github.com/MasterDimmy/zipologger"
)

func main() {
	log := zipologger.NewLogger("./logs/test.log",1,1,1,false)
	fmt.Println(log.Printf("aaa : %s", "123"))
}
