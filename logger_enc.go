package zipologger

import (
	"sync"
	"github.com/MasterDimmy/zipologger/enc"
)

type globalEncrypt struct {
	m   sync.Mutex
	key *enc.KeyEncrypt
}

var globalEncryptor globalEncrypt

func SetGlobalEncryption(key string) bool {
	globalEncryptor.m.Lock()
	defer globalEncryptor.m.Unlock()

	k := enc.NewEncryptKey(key)
	if k == nil {
		return false
	}
	globalEncryptor.key = k
	return true
}

func (l *Logger) SetEncryptionKey(key string) *Logger {
	l.m.Lock()
	defer l.m.Unlock()

	k := enc.NewEncryptKey(key)
	if k == nil {
		panic("cant set encryption key")
	}
	l.encryptionKey = k
	return l
}
