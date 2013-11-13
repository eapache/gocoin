package main

import (
	"bytes"
	"crypto/rsa"
	"math/rand"
	"sync"
	"time"
)

type State struct {
	sync.RWMutex

	// main state
	primary    *BlockChain
	alternates []*BlockChain
	wallet     *Wallet
	activeKeys map[rsa.PublicKey]*Transaction

	pendingTxns    []*Transaction
	txnsInProgress []*Transaction
	resetMiner     bool
	stopper        chan bool
}

func NewState(initialPeer string) *State {
	s := &State{}
	s.primary = &BlockChain{}
	s.activeKeys = make(map[rsa.PublicKey]*Transaction)
	s.wallet = NewWallet()
	s.stopper = make(chan bool, 1)

	return s
}

func (s *State) ChainFromHash(hash []byte) *BlockChain {
	s.RLock()
	defer s.RUnlock()
	return s.chainFromHash(hash)
}

func (s *State) chainFromHash(hash []byte) *BlockChain {
	// unlocked version

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

func (s *State) addBlockChain(chain *BlockChain) {
	s.Lock()
	defer s.Unlock()

	if len(s.primary.Blocks) < len(chain.Blocks) {
		s.alternates = append(s.alternates, s.primary)
		s.primary = chain
		s.resetMiner = true
	}
}

// returns true if we need to request the whole chain
func (s *State) newBlock(b *Block) bool {
	if !b.Verify() {
		return false
	}

	s.Lock()
	defer s.Unlock()

	chain := s.chainFromHash(b.PrevHash)

	if chain == nil {
		return true
	}

	chain.Append(b)
	s.resetMiner = true
	return false
}

func (s *State) MineForGold() {
mineNewBlock:
	for {
		s.Lock()
		b := &Block{}
		s.txnsInProgress = s.pendingTxns
		s.pendingTxns = nil
		b.Txns = s.txnsInProgress
		b.Txns = append(b.Txns, s.wallet.NewPaymentTxn())

		if s.primary.Last() != nil {
			b.PrevHash = s.primary.Last().Hash()
		}
		s.resetMiner = false
		s.Unlock()

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for {
			if s.resetMiner {
				continue mineNewBlock
			}
			select {
			case <-s.stopper:
				return
			default:
				b.Nonce = r.Uint32()
				if b.Verify() {
					success := false
					s.Lock()
					prev := s.primary.Last()
					if prev == nil || bytes.Equal(prev.Hash(), b.PrevHash) {
						s.primary.Append(b)
						success = true
					}
					s.Unlock()
					if success {
						network.BroadcastBlock(b)
					}
					continue mineNewBlock
				}
			}
		}
	}
}

func (s *State) Close() {
	s.stopper <- true
}
