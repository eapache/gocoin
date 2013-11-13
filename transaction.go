package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/gob"
	"errors"
)

type TxnInput struct {
	Key       rsa.PublicKey
	PrevHash  []byte
	Signature []byte
}

type TxnOutput struct {
	Key    rsa.PublicKey
	Amount uint64
}

type Transaction struct {
	Inputs  []TxnInput
	Outputs []TxnOutput
}

func (txn *Transaction) Hash() []byte {
	hasher := sha256.New()
	encoder := gob.NewEncoder(hasher)

	// transaction hash value does not include signatures (or signing would change the hash,
	// which would make this impossible) so we save and restore the signature fields
	tmpSigs := make([][]byte, len(txn.Inputs))
	for i := range txn.Inputs {
		tmpSigs[i] = txn.Inputs[i].Signature
		txn.Inputs[i].Signature = nil
	}

	err := encoder.Encode(txn)
	if err != nil {
		panic(err)
	}

	for i := range txn.Inputs {
		txn.Inputs[i].Signature = tmpSigs[i]
	}

	return hasher.Sum(nil)
}

func (txn *Transaction) Sign(w *Wallet) (err error) {
	hash := txn.Hash()

	for i := range txn.Inputs {
		privKey := w.Keys[txn.Inputs[i].Key]
		if privKey == nil {
			return errors.New("could not sign transaction, missing private key")
		}
		txn.Inputs[i].Signature, err = rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, hash)
		if err != nil {
			return err
		}
	}

	return nil
}

func (txn *Transaction) VerifySignatures() bool {
	hash := txn.Hash()

	for i := range txn.Inputs {
		err := rsa.VerifyPKCS1v15(&txn.Inputs[i].Key, crypto.SHA256, hash, txn.Inputs[i].Signature)
		if err != nil {
			return false
		}
	}
	return true
}

func (txn *Transaction) OutputAmount(key rsa.PublicKey) (uint64, error) {
	for i := range txn.Outputs {
		if key == txn.Outputs[i].Key {
			return txn.Outputs[i].Amount, nil
		}
	}

	return 0, errors.New("could not fetch output amount, key not found")
}
