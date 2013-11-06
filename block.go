package main

import (
	"crypto/sha256"
	"encoding/gob"
)

type Block struct {
	PrevHash []byte
	Nonce    uint32
	Txns     []*Transaction
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
	// first 17 bits zero? timed for 5-20 seconds per solve on modern CPU
	return hash[0] == 0 && hash[1] == 0 && hash[2] & 0x80 == 0
}
