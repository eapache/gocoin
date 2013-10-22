package main

import (
	"crypto/rsa"
)

type Wallet struct {
	keys map[Address]rsa.PrivateKey
}
