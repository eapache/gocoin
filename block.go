package main

import (
	"crypto/sha256"
	"encoding/gob"
	"math/rand"
	"time"
)

type Block struct {
	PrevHash []byte
	Nonce    uint32
	Txns     []Transaction
}

func (b *Block) Hash() []byte {
	hasher := sha256.New()
	encoder := gob.NewEncoder(hasher)
	err := encoder.Encode(b)
	if err != nil {
		panic(err)
	}
	return hasher.Sum(nil)
}

func (b *Block) Verify() bool {
	hash := b.Hash()
	return hash[0] == 0 && hash[1] == 0
}

func (b *Block) Solve() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b.Nonce = r.Uint32()
	for !b.Verify() {
		b.Nonce = r.Uint32()
	}
}
