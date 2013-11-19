package main

import (
	"bufio"
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
		case "genk":
			_, err := state.wallet.GenKey()
			if err != nil {
				fmt.Println("Error: ", err)
			} else {
				fmt.Println("Success")
			}
		case "state":
			printBlockChain()
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

func printWallet() {
	fmt.Printf("\n  Amount | Public Key\n")
	var total uint64
	for key, val := range state.GetWallet() {
		fmt.Printf("%8d | %s...\n", val, key.N.String()[0:40])
		total += val
	}
	fmt.Printf("\nTotal Coins: %d\n\n", total)
}

func printBlockChain() {
	state.RLock()
	defer state.RUnlock()

	chain := state.primary

	fmt.Printf("\nPrimary Chain (%d Blocks)", len(chain.Blocks))
	for _, block := range chain.Blocks {
		fmt.Printf("\n\tBlock (%d Txns) - Nonce: %10d; Hash: 0x%x...",
			len(block.Txns), block.Nonce, block.Hash()[0:13])
		for _, txn := range block.Txns {
			fmt.Printf("\n\t\tTxn (%d Inputs, %d Outputs)",
				len(txn.Inputs), len(txn.Outputs))
		}
	}
	fmt.Println()
	fmt.Println()
}

func printHelp() {
	fmt.Println("Possible commands are:")
	fmt.Println("  help (displays this help)")
	fmt.Println("  quit (exits gocoin)")
	fmt.Println()
	fmt.Println("  state  (display primary blockchain state)")
	fmt.Println("  wallet (display wallet)")
	fmt.Println()
	fmt.Println("  genk (generates a new key and adds it to the wallet)")
	fmt.Println("")
}
