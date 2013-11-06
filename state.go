package main

import (
	"bytes"
	"crypto/rsa"
	"math/rand"
	"sync"
	"time"
)

type signal int

const (
	sigNewBlock signal = iota
	sigQuit     signal = iota
)

type State struct {
	sync.RWMutex

	// main state
	primary    *BlockChain
	alternates []*BlockChain
	wallet     *Wallet
	activeKeys map[rsa.PublicKey]*Transaction

	network *PeerNetwork

	pendingTxns    []*Transaction
	txnsInProgress []*Transaction
	signals        chan signal
}

func NewState(initialPeer string) *State {
	s := &State{}
	s.primary = &BlockChain{}
	s.activeKeys = make(map[rsa.PublicKey]*Transaction)
	s.wallet = NewWallet()
	s.signals = make(chan signal, 1)

	var err error
	s.network, err = NewPeerNetwork(initialPeer)
	if err != nil {
		panic(err)
	}
	s.network.state = s // woohoo, circular references

	s.network.RequestBlockChain(nil) // nil hash for the primary chain

	go s.MineForGold()

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
		s.signals <- sigNewBlock
	}
}

func (s *State) newBlock(b *Block) {
	if !b.Verify() {
		return
	}

	s.Lock()
	defer s.Unlock()

	chain := s.chainFromHash(b.PrevHash)

	if chain == nil {
		s.network.RequestBlockChain(b.Hash())
	} else {
		chain.Append(b)
		s.signals <- sigNewBlock
	}
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
		s.Unlock()

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for {
			select {
			case sig := <-s.signals:
				switch sig {
				case sigNewBlock:
					continue mineNewBlock
				case sigQuit:
					return
				}
			default:
				b.Nonce = r.Uint32()
				if b.Verify() {
					success := false
					s.Lock()
					if bytes.Equal(s.primary.Last().Hash(), b.PrevHash) {
						s.primary.Append(b)
						success = true
					}
					s.Unlock()
					if success {
						s.network.BroadcastBlock(b)
					}
					continue mineNewBlock
				}
			}
		}
	}
}

func (s *State) Close() {
	s.signals <- sigQuit
}
