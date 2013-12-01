package main

import (
	"bytes"
	"log"
)

type KeySet map[string]*Transaction

func (set KeySet) Copy() KeySet {
	tmp := make(KeySet, len(set))
	for k, v := range set {
		tmp[k] = v
	}
	return tmp
}

func (set KeySet) AddTxn(txn *Transaction) bool {
	valid := txn.VerifySignatures()
	if !valid {
		log.Println("Failed to verify txn signatures")
		return false
	}

	var inTotal, outTotal uint64

	for _, input := range txn.Inputs {
		prev := set[input.Key.N.String()]
		if prev == nil || !bytes.Equal(prev.Hash(), input.PrevHash) {
			log.Println("Transaction missing input")
			return false
		}
		exists, amount := prev.OutputAmount(input.Key)
		if !exists {
			log.Println("Keyset corrupt!")
			return false
		}
		inTotal += amount
		delete(set, input.Key.N.String())
	}

	for _, output := range txn.Outputs {
		outTotal += output.Amount
		set[output.Key.N.String()] = txn
	}

	if txn.Inputs == nil && len(txn.Outputs) == 1 {
		// a miner's bonus txn
		inTotal = 10
	}

	if inTotal != outTotal {
		log.Println("Txn corrupt!")
		return false
	}

	return true
}
