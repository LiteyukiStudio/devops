package id

import (
	"crypto/rand"
	"encoding/hex"
)

func New(prefix string) string {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(bytes[:])
}
