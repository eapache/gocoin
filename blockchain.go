package main

import (
	"bytes"
)

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
		if !tmpKeys.AddTxn(txn) {
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
