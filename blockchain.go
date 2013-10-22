package main

import (
	"bytes"
)

type BlockChain struct {
	blocks []*Block
}

func (bc *BlockChain) TryAppend(blk *Block) bool {
	if !blk.Verify() {
		return false
	}

	if bc.blocks == nil && blk.PrevHash == nil {
		bc.blocks = append(bc.blocks, blk)
		return true
	}

	if bytes.Equal(bc.blocks[len(bc.blocks)-1].Hash(), blk.PrevHash) {
		bc.blocks = append(bc.blocks, blk)
		return true
	}

	return false
}
