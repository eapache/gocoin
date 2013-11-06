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

func (w *Wallet) GenKey() (*rsa.PublicKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	w.Keys[priv.PublicKey] = priv

	return &priv.PublicKey, nil
}

// generates a new payment of 10 coins into this wallet from mining
func (w *Wallet) NewPaymentTxn() *Transaction {
	return nil // TODO
}
