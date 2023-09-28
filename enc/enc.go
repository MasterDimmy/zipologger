package enc

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

type KeyEncrypt struct {
	pub *ecies.PublicKey
}

type KeyDecrypt struct {
	key *ecdsa.PrivateKey
}

type Key struct {
	KeyEncrypt
	KeyDecrypt
}

func NewKey() *Key {
	key, err := ecies.GenerateKey(rand.Reader, crypto.S256(), ecies.ECIES_AES256_SHA256)
	if err != nil {
		return nil
	}

	return &Key{
		KeyDecrypt: KeyDecrypt{key.ExportECDSA()},
		KeyEncrypt: KeyEncrypt{&key.PublicKey},
	}
}

func (k *KeyEncrypt) EncryptionKey() string {
	pb := k.pub.ExportECDSA()
	pk := crypto.FromECDSAPub(pb)
	return hex.EncodeToString(pk)
}

func (k *KeyDecrypt) DecryptionKey() string {
	b := crypto.FromECDSA(k.key)
	return hex.EncodeToString(b)
}

func NewEncryptKey(encKey string) *KeyEncrypt {
	k, err := hex.DecodeString(strings.TrimSpace(encKey))
	if err != nil {
		return nil
	}

	pub, err := crypto.UnmarshalPubkey(k)
	if err != nil {
		return nil
	}

	pb := ecies.ImportECDSAPublic(pub)

	return &KeyEncrypt{
		pub: pb,
	}
}

func (k *KeyEncrypt) EncryptString(str string) ([]byte, error) {
	return k.Encrypt([]byte(str))
}

func (k *KeyEncrypt) Encrypt(msg []byte) ([]byte, error) {
	kk := &ecies.PublicKey{
		X:     k.pub.X,
		Y:     k.pub.Y,
		Curve: crypto.S256(),
	}
	return ecies.Encrypt(rand.Reader, kk, msg, nil, nil)
}

func NewDecryptKey(encKey string) *KeyDecrypt {
	k, err := hex.DecodeString(strings.TrimSpace(encKey))
	if err != nil {
		return nil
	}
	key, err := crypto.ToECDSA(k)
	if err != nil {
		return nil
	}
	return &KeyDecrypt{key: key}
}

func (k *KeyDecrypt) Decrypt(ciphertext []byte) ([]byte, error) {
	kk := &ecies.PrivateKey{
		PublicKey: ecies.PublicKey{
			X:     k.key.X,
			Y:     k.key.Y,
			Curve: crypto.S256(),
		},
		D: k.key.D,
	}
	return kk.Decrypt(ciphertext, nil, nil)
}
