package main

import (
	"bytes"
	"crypto/rsa"
)

type BlockChain struct {
	Blocks []*Block
	activeKeys map[rsa.PublicKey]*Transaction
}

func NewBlockChain() *BlockChain {
	return &BlockChain{nil, make(map[rsa.PublicKey]*Transaction)}
}

func (chain *BlockChain) Last() *Block {
	if len(chain.Blocks) == 0 {
		return nil
	}

	return chain.Blocks[len(chain.Blocks)-1]
}

func (chain *BlockChain) Append(blk *Block) bool {
	for txn := range blk.Txns {
		for input := range txn.Inputs {
		}
	}

	chain.Blocks = append(chain.Blocks, blk)
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
