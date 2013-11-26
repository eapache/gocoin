package main

import (
	"crypto/rand"
	"crypto/rsa"
)

type Wallet struct {
	Keys map[rsa.PublicKey]*rsa.PrivateKey
}

func NewWallet() *Wallet {
	return &Wallet{Keys: make(map[rsa.PublicKey]*rsa.PrivateKey)}
}

func (w *Wallet) AddKey(key *rsa.PrivateKey) {
	w.Keys[key.PublicKey] = key
}

func (w *Wallet) GenKey() (*rsa.PublicKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	w.AddKey(priv)

	return &priv.PublicKey, nil
}
