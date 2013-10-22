package main

type Transaction struct {
	Inputs, Outputs        []Address
	PrevHashes, Signatures [][]byte
}
