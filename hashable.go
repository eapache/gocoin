package main

import (
	"crypto/sha256"
	"encoding/gob"
)

type Hashable struct {
}

func (h *Hashable) Hash() []byte {
	hasher := sha256.New()
	encoder := gob.NewEncoder(hasher)
	encoder.Encode(h)
	return hasher.Sum(nil)
}
