package main

import (
	"bytes"
	"sync"
)

type State struct {
	sync.RWMutex

	// main state
	primary    *BlockChain
	alternates []*BlockChain
	wallet     *Wallet

	pendingTxns    []*Transaction
	txnsInProgress []*Transaction
	ResetMiner     bool
}

func NewState() *State {
	s := &State{}
	s.primary = &BlockChain{}
	s.wallet = NewWallet()

	return s
}

//
// public, locked functions
//

func (s *State) ConstructBlock() *Block {
	s.Lock()
	defer s.Unlock()

	b := &Block{}
	s.txnsInProgress = s.pendingTxns
	s.pendingTxns = nil
	b.Txns = s.txnsInProgress
	b.Txns = append(b.Txns, s.wallet.NewPaymentTxn())

	if s.primary.Last() != nil {
		b.PrevHash = s.primary.Last().Hash()
	}
	s.ResetMiner = false

	return b
}

func (s *State) ChainFromHash(hash []byte) *BlockChain {
	s.RLock()
	defer s.RUnlock()
	return s.chainFromHash(hash)
}

func (s *State) AddBlockChain(chain *BlockChain) {
	s.Lock()
	defer s.Unlock()

	if len(s.primary.Blocks) < len(chain.Blocks) {
		s.alternates = append(s.alternates, s.primary)
		s.primary = chain
		s.ResetMiner = true
	}
}

// first return is if the block was accepted, second
// is if we already have the relevant chain
func (s *State) NewBlock(b *Block) (bool, bool) {
	if !b.Verify() {
		return false, false
	}

	s.Lock()
	defer s.Unlock()

	chain := s.chainFromHash(b.PrevHash)

	if chain == nil {
		return true, false
	}

	success := chain.Append(b)
	if !success {
		return false, true
	}

	s.ResetMiner = true
	return true, true
}

func (s *State) UndoBlock(b *Block) {
}

//
// private, unlocked functions *must* be called while already holding the lock
//

func (s *State) chainFromHash(hash []byte) *BlockChain {
	if hash == nil {
		return s.primary
	}

	if bytes.Equal(hash, s.primary.Last().Hash()) {
		return s.primary
	}

	for _, chain := range s.alternates {
		if bytes.Equal(hash, chain.Last().Hash()) {
			return chain
		}
	}

	return nil
}
