package main

import (
	"bytes"
	"crypto/rsa"
	"sync"
)

type State struct {
	sync.RWMutex

	// main state
	primary    *BlockChain
	alternates []*BlockChain
	wallet     map[rsa.PublicKey]*rsa.PrivateKey
	keys       KeySet

	pendingTxns []*Transaction
	beingMined  int
	ResetMiner  bool

	pendingPayments map[string]*rsa.PrivateKey
}

func NewState() *State {
	s := &State{}
	s.primary = &BlockChain{}
	s.wallet = make(map[rsa.PublicKey]*rsa.PrivateKey)
	s.keys = make(KeySet)
	s.pendingPayments = make(map[string]*rsa.PrivateKey)

	return s
}

//
// public, locked functions
//

func (s *State) PayTo(payer string) *rsa.PublicKey {
	s.Lock()
	defer s.Unlock()

	key := s.pendingPayments[payer]

	if key == nil {
		key = genKey()
		s.pendingPayments[payer] = key
	}

	return &key.PublicKey
}

func (s *State) GenTxnInput(key rsa.PublicKey) TxnInput {
	s.RLock()
	defer s.RUnlock()

	prev := s.keys[key]
	if prev == nil {
		prev = s.primary.Keys[key]
	}
	if prev == nil {
		panic("invalid key")
	}
	input := TxnInput{key, prev.Hash(), nil}

	return input
}

func (s *State) Sign(txn *Transaction) error {
	s.RLock()
	defer s.RUnlock()

	return txn.Sign(s.wallet)
}

func (s *State) AddTxn(txn *Transaction) bool {
	s.Lock()
	defer s.Unlock()

	success := s.keys.AddTxn(txn)

	if success {
		s.pendingTxns = append(s.pendingTxns, txn)
	}

	return success
}

func (s *State) AddToWallet(key *rsa.PrivateKey) {
	s.Lock()
	defer s.Unlock()

	s.wallet[key.PublicKey] = key
}

func (s *State) GetWallet() map[rsa.PublicKey]uint64 {
	s.RLock()
	defer s.RUnlock()

	ret := make(map[rsa.PublicKey]uint64)

	for key, _ := range s.wallet {
		txn := s.keys[key]

		if txn != nil {
			_, ret[key] = txn.OutputAmount(key)
		}
	}

	return ret
}

func (s *State) ConstructBlock() (*Block, *rsa.PrivateKey) {
	s.Lock()
	defer s.Unlock()

	txn, key := NewMinersTransation()

	b := &Block{}
	b.Txns = append(b.Txns, txn)
	b.Txns = append(b.Txns, s.pendingTxns...)

	if s.primary.Last() != nil {
		b.PrevHash = s.primary.Last().Hash()
	}
	s.ResetMiner = false
	s.beingMined = len(s.pendingTxns) + 1

	return b, key
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
		s.reset()
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

	s.reset()

	return true, true
}

//
// private, unlocked functions *must* be called while already holding the lock
//

func (s *State) reset() {
	s.ResetMiner = true
	s.keys = s.primary.Keys.Copy()

	var tmp []*Transaction

	for _, txn := range s.pendingTxns {
		if s.keys.AddTxn(txn) {
			tmp = append(tmp, txn)
		} else {
			for _, input := range txn.Inputs {
				delete(s.wallet, input.Key)
			}
			for _, output := range txn.Outputs {
				delete(s.wallet, output.Key)
			}
		}
	}

	s.pendingTxns = tmp
}

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
