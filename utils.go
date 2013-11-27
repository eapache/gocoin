package main

import (
	"crypto/rand"
	"crypto/rsa"
)

func genKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}

	return key
}
