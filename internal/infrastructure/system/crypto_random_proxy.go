package system

import "crypto/rand"

type CryptoRandomProxy struct{}

func NewCryptoRandomProxy() *CryptoRandomProxy {
	return &CryptoRandomProxy{}
}

func (cryptoRandomProxy *CryptoRandomProxy) GenerateBytes(length int) []byte {
	randomBytes := make([]byte, length)
	// crypto/rand.Read never returns an error since Go 1.24; it crashes the process instead
	_, _ = rand.Read(randomBytes)
	return randomBytes
}
