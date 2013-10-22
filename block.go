package main

type Block struct {
	PrevHash []byte
	Nonce    uint64
	Txns     []Transaction
}
