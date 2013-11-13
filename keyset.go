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

	if txn.Inputs == nil && len(txn.Outputs) == 1 {
		// a miner's bonus txn
		inTotal = 10
	}

	if inTotal != outTotal {
		return false
	}

	return true
}

