package main

import (
	"bytes"
	"crypto/rsa"
)

type KeySet map[rsa.PublicKey]*Transaction

func (set KeySet) Copy() KeySet {
	tmp := make(KeySet, len(set))
	for k, v := range set {
		tmp[k] = v
	}
	return tmp
}

func (set KeySet) AddTxn(txn *Transaction) bool {
	var inTotal, outTotal uint64

	for _, input := range txn.Inputs {
		prev := set[input.Key]
		if prev == nil || !bytes.Equal(prev.Hash(), input.PrevHash) {
			return false
		}
		amount, err := prev.OutputAmount(input.Key)
		if err != nil {
			return false
		}
		inTotal += amount
		delete(set, input.Key)
	}

	for _, output := range txn.Outputs {
		outTotal += output.Amount
		set[output.Key] = txn
	}

	if inTotal != outTotal {
		return false
	}

	return true
}

type BlockChain struct {
	Blocks     []*Block
	ActiveKeys KeySet
}

func NewBlockChain() *BlockChain {
	return &BlockChain{nil, make(KeySet)}
}

func (chain *BlockChain) Last() *Block {
	if len(chain.Blocks) == 0 {
		return nil
	}

	return chain.Blocks[len(chain.Blocks)-1]
}

func (chain *BlockChain) Append(blk *Block) bool {
	tmpKeys := chain.ActiveKeys.Copy()
	for _, txn := range blk.Txns {
		valid := txn.VerifySignatures()
		if !valid {
			return false
		}

		valid = tmpKeys.AddTxn(txn)
		if !valid {
			return false
		}
	}

	chain.Blocks = append(chain.Blocks, blk)
	chain.ActiveKeys = tmpKeys
	return true
}

func (chain *BlockChain) Verify() bool {
	if len(chain.Blocks) == 0 {
		return true // an empty chain is always valid
	}

	if chain.Blocks[0].PrevHash != nil || !chain.Blocks[0].Verify() {
		return false //first block has no PrevHash, but it must still have a valid nonce
	}

	actualPrevHash := chain.Blocks[0].Hash()

	for _, block := range chain.Blocks[1:] {
		if !bytes.Equal(block.PrevHash, actualPrevHash) || !block.Verify() {
			return false
		}

		actualPrevHash = block.Hash()
	}

	return true
}
