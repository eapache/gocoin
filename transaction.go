package main

import (
	"crypto/rsa"
)

type TxnInput struct {
	Key       rsa.PublicKey
	PrevHash  []byte
	Signature []byte
}

type TxnOutput struct {
	Key    rsa.PublicKey
	Amount uint64
}

type Transaction struct {
	Inputs  []TxnInput
	Outputs []TxnOutput
}
