package main

import (
	"crypto/rsa"
)

type State struct {
	Primary    *BlockChain
	Alternate  []*BlockChain
	ActiveKeys map[rsa.PublicKey]*Transaction
}
