package zipologger

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"testing"

	"github.com/MasterDimmy/zipologger/enc"
)

var logger *Logger

func Test_EncryptedLog(t *testing.T) {
	defer Wait()

	SetAlsoToStdout(true)

	os.MkdirAll("./logs/", 0644)

	logger = NewLogger("./logs/test.log", 1, 1, 1, true)

	key := enc.NewKey()
	log.Printf("enc key: %s\n", key.EncryptionKey())
	log.Printf("dec key: %s\n", key.DecryptionKey())

	logger.SetEncryptionKey(key.EncryptionKey())

	go func() {
		http.ListenAndServe(":9745", nil)
	}()

	logger.Print("test from main1\n")
	logger.Print("second line \n")
	logger.Print("third line \n")
	logger.Flush()
}
