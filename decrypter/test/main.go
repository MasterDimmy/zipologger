package main

import (
	"flag"
	"fmt"
)

func main() {
	key := flag.String("key", "", "key")
	f := flag.String("f", "", "f")

	flag.Parse()

	fmt.Printf("key=%s f=%s\n", *key, *f)
}
