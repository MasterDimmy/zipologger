package zipologger

import (
	"fmt"

	"github.com/MasterDimmy/zipologger/enc"
)

func SetGlobalEncryption(key string) bool {
	mainGlobalEncryptor.m.Lock()
	defer mainGlobalEncryptor.m.Unlock()

	k := enc.NewEncryptKey(key)
	if k == nil {
		return false
	}
	mainGlobalEncryptor.key = k
	return true
}

func (l *Logger) SetEncryptionKey(key string) (*Logger, error) {
	l.m.Lock()
	defer l.m.Unlock()

	k := enc.NewEncryptKey(key)
	if k == nil {
		return l, fmt.Errorf("invalid encryption key")
	}
	l.encryptionKey = k
	return l, nil
}
