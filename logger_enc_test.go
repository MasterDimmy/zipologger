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

	// Reset global state afterwards: SetGlobalEncryption now actually enables
	// encryption for all loggers, and SetAlsoToStdout is package-global, so
	// leaving them set would pollute later tests.
	defer func() {
		SetAlsoToStdout(false)
		mainGlobalEncryptor.m.Lock()
		mainGlobalEncryptor.key = nil
		mainGlobalEncryptor.m.Unlock()
	}()

	go func() {
		http.ListenAndServe(":9745", nil)
	}()

	SetAlsoToStdout(true)

	os.MkdirAll("./logs/", 0644)

	logger = NewLogger("./logs/test.log", 1, 1, 1, true)

	key := enc.NewKey()
	log.Printf("dec key: %s\n", key.DecryptionKey())

	if _, err := logger.SetEncryptionKey(key.EncryptionKey()); err != nil {
		t.Fatalf("SetEncryptionKey failed: %v", err)
	}

	logger.Print("test from main1")
	logger.Print("second line ")
	logger.Print("third line")
	logger.Flush()

	logger = NewLogger("./logs/test2.log", 1, 1, 1, true)

	log.Printf("dec key: %s\n", key.DecryptionKey())

	SetGlobalEncryption(key.EncryptionKey())

	logger.Print("test from main 2")
	logger.Print("second line 2")
	logger.Print("third line 2")
	logger.Flush()
}
