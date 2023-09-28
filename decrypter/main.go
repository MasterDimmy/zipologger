package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"log"
	"os"

	"github.com/MasterDimmy/zipologger/enc"
)

func main() {
	file := flag.String("f", "", "file to decrypt")
	key := flag.String("key", "", "decryption key string")

	flag.Parse()

	if *file == "" || len(*key) < 10 {
		flag.Usage()
		return
	}

	infile, err := os.Open(*file)
	if err != nil {
		log.Fatalf("failed to open file: %s", err)
		return
	}
	defer infile.Close()

	decfile, err := os.Create(*file + ".dec")
	if err != nil {
		log.Fatalf("failed to create file: %s", err)
		return
	}
	defer decfile.Close()

	decryptor := enc.NewDecryptKey(*key)
	if decryptor == nil {
		log.Fatal("incorrect key for decryption")
		return
	}

	scanner := bufio.NewScanner(infile)
	for scanner.Scan() {
		bb, err := base64.RawStdEncoding.DecodeString(scanner.Text())
		if err != nil {
			continue
		}
		b, err := decryptor.Decrypt(bb)
		if err != nil {
			log.Fatalf("decryption error: %s\n", err.Error())
			return
		}
		decfile.Write(b)
		decfile.Write([]byte("\n"))
	}
}
