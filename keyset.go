package main

import (
	"bytes"
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
		logger.Println("Failed to verify txn signatures")
		return false
	}

	var inTotal, outTotal uint64

	for _, input := range txn.Inputs {
		prev := set[input.Key.N.String()]
		if prev == nil || !bytes.Equal(prev.Hash(), input.PrevHash) {
			logger.Println("Transaction missing input")
			return false
		}
		exists, amount := prev.OutputAmount(input.Key)
		if !exists {
			logger.Println("Keyset corrupt!")
			return false
		}
		inTotal += amount
		delete(set, input.Key.N.String())
	}

	for _, output := range txn.Outputs {
		outTotal += output.Amount
		set[output.Key.N.String()] = txn
	}

	if inTotal != outTotal && !txn.IsMiner() {
		logger.Println("Txn corrupt!", inTotal, outTotal)
		return false
	}

	return true
}
