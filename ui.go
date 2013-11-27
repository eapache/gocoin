package main

import (
	"bufio"
	"crypto/rsa"
	"fmt"
	"os"
)

func mainLoop() {
	fmt.Println("Welcome to GoCoin")
	printHelp()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("> ")
	for scanner.Scan() {
		switch scanner.Text() {
		case "": // do nothing, ignore
		case "cons":
			txn, key := consWallet()
			success := state.AddTxn(txn)
			if success {
				state.AddToWallet(key)
				fmt.Println("Wallet consolidated.")
			} else {
				fmt.Println("Failed.")
			}
		case "state":
			printState()
		case "wallet":
			printWallet()
		case "help":
			printHelp()
		case "quit":
			return
		default:
			fmt.Println("Unknown input, try 'help' or 'quit'")
		}

		fmt.Print("> ")
	}
}

func consWallet() (*Transaction, *rsa.PrivateKey) {
	var total uint64
	txn := new(Transaction)

	for key, amount := range state.GetWallet() {
		if amount > 0 {
			total += amount
			txn.Inputs = append(txn.Inputs, state.GenTxnInput(key))
		}
	}

	key := genKey()
	txn.Outputs = append(txn.Outputs, TxnOutput{key.PublicKey, total})

	state.Sign(txn)

	return txn, key
}

func printWallet() {
	fmt.Printf("\n  Amount | Public Key\n")
	var total uint64
	for key, val := range state.GetWallet() {
		fmt.Printf("%8d | %s...\n", val, key.N.String()[0:40])
		total += val
	}
	fmt.Printf("\nTotal Coins: %d\n\n", total)
}

func printState() {
	state.RLock()
	defer state.RUnlock()

	fmt.Printf("\nPrimary Chain (%d Blocks)", len(state.primary.Blocks))
	printBlockChain(state.primary)

	fmt.Printf("\n%d Pending Transactions (%d being mined)\n", len(state.pendingTxns), state.beingMined)
	for _, txn := range state.pendingTxns {
		printTxn(txn)
	}
	fmt.Println()
}

func printBlockChain(chain *BlockChain) {
	if len(chain.Blocks) > 0 {
		fmt.Println()
	}
	for _, block := range chain.Blocks {
		fmt.Printf("\tBlock (%d Txns) - Nonce: %10d; Hash: 0x%x...",
			len(block.Txns), block.Nonce, block.Hash()[0:13])
		if len(block.Txns) > 0 {
			fmt.Println()
		}
		for _, txn := range block.Txns {
			fmt.Printf("\t\t")
			printTxn(txn)
		}
	}
}

func printTxn(txn *Transaction) {
	fmt.Printf("Txn (%d Inputs, %d Outputs)\n",
		len(txn.Inputs), len(txn.Outputs))
}

func printHelp() {
	fmt.Println("Possible commands are:")
	fmt.Println("  help (displays this help)")
	fmt.Println("  quit (exits gocoin)")
	fmt.Println()
	fmt.Println("  state  (display blockchain and transaction state)")
	fmt.Println("  wallet (display wallet)")
	fmt.Println()
	fmt.Println("  cons (consolidate wallet into single key)")
	fmt.Println("")
}
