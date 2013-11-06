package main

import (
	"crypto/rsa"
	"math/rand"
	"sync"
	"time"
)

type State struct {
	primary    *BlockChain
	alternates []*BlockChain
	network *PeerNetwork
	wallet *Wallet

	txnLock sync.Mutex
	pendingTxns []*Transaction
	txnsInProgress []*Transaction
	activeKeys map[rsa.PublicKey]*Transaction

	restart chan bool
}

func NewState(initialPeer string) *State {
	s := &State{}
	s.primary = &BlockChain{}
	s.activeKeys = make(map[rsa.PublicKey]*Transaction)
	s.wallet = NewWallet()
	s.restart = make(chan bool)

	var err error
	s.network, err = NewPeerNetwork(initialPeer)
	if err != nil {
		panic(err)
	}

	go s.MineForGold()

	return s
}

func (s *State) MineForGold() {
	mineNewBlock:
	for {
		s.txnLock.Lock()
		b := &Block{}
		s.txnsInProgress = s.pendingTxns
		s.pendingTxns = nil
		b.Txns = s.txnsInProgress
		s.txnLock.Unlock()

		if s.primary.Last() != nil {
			b.PrevHash = s.primary.Last().Hash()
		}

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for {
			select {
			case <-s.restart:
				continue mineNewBlock
			default:
				b.Nonce = r.Uint32()
				if b.Verify() {
					return // TODO
				}
			}
		}
	}
}

func (s *State) Close() {
}
