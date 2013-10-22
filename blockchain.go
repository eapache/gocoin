package main

import (
	"bytes"
)

type BlockChain struct {
	Blocks []*Block
}

func (chain *BlockChain) Last() *Block {
	if len(chain.Blocks) == 0 {
		return nil
	}

	return chain.Blocks[len(chain.Blocks)-1]
}

func (chain *BlockChain) TryAppend(blk *Block) bool {
	if !blk.Verify() {
		return false
	}

	if chain.Blocks == nil && blk.PrevHash == nil {
		chain.Blocks = append(chain.Blocks, blk)
		return true
	}

	if bytes.Equal(chain.Last().Hash(), blk.PrevHash) {
		chain.Blocks = append(chain.Blocks, blk)
		return true
	}

	return false
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
