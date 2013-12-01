package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
)

func genKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}

	return key
}

func keysEql(a, b *rsa.PublicKey) bool {
	return a.N.Cmp(b.N) == 0 && a.E == b.E
}

func keyHash(in *rsa.PublicKey) []byte {
	hasher := sha256.New()
	encoder := gob.NewEncoder(hasher)
	err := encoder.Encode(in)
	if err != nil {
		panic(err)
	}
	return hasher.Sum(nil)
}
