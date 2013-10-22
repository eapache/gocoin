package main

import (
	"crypto/rsa"
)

type Address struct {
	rsa.PublicKey
	Amount uint64
}
