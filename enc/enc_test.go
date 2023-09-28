package enc

import (
	"bytes"
	"testing"

	"log"
)

func Test_encrypt_decrypt(t *testing.T) {

	basicKey := NewKey()

	encryptionKeyString := basicKey.EncryptionKey()
	log.Printf("encryption string: %s\n", encryptionKeyString)

	decryptionKeyString := basicKey.DecryptionKey()
	log.Printf("decryption string: %s\n", decryptionKeyString)

	text := "Hello world"

	keyForEncrypt := NewEncryptKey(encryptionKeyString)

	encoded, err := keyForEncrypt.EncryptString(text)
	if err != nil {
		log.Fatal()
	}

	log.Printf("encoded: %x", encoded)

	keyForDecrypt := NewDecryptKey(decryptionKeyString)

	decoded, err := keyForDecrypt.Decrypt(encoded)
	if err != nil {
		log.Fatal()
	}

	if bytes.Compare([]byte(text), decoded) != 0 {
		log.Fatal()
	}

	log.Printf("decoded: %s", decoded)
}
