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

// generates a new payment of 10 coins from mining, returning the transaction
// and the private key that will be payed if the mining is successful
func NewMinersTransation() (*Transaction, *rsa.PrivateKey) {
	priv := genKey()
	txn := &Transaction{}
	txn.Outputs = append(txn.Outputs, TxnOutput{priv.PublicKey, 10})
	return txn, priv
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

func (txn *Transaction) Sign(wallet map[rsa.PublicKey]*rsa.PrivateKey) (err error) {
	hash := txn.Hash()

	for i := range txn.Inputs {
		privKey := wallet[txn.Inputs[i].Key]
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

func (txn *Transaction) OutputAmount(key rsa.PublicKey) (bool, uint64) {
	for i := range txn.Outputs {
		if keysEql(&key, &txn.Outputs[i].Key) {
			return true, txn.Outputs[i].Amount
		}
	}

	return false, 0
}
