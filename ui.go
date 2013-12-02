package main

import (
	"bufio"
	"crypto/rsa"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
)

func inputReader(ret chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		ret <- scanner.Text()
	}
}

func mainLoop() {
	fmt.Println("Welcome to GoCoin")
	printHelp()

	input := make(chan string)

	go inputReader(input)

	fmt.Print("> ")
	for text := range input {
		switch text {
		case "": // do nothing, ignore
		case "cons":
			consWallet()
		case "pay":
			doPay(input)
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

func consWallet() {
	var total uint64
	txn := new(Transaction)

	for key, amount := range state.GetWallet() {
		if amount > 0 {
			total += amount
			txn.Inputs = append(txn.Inputs, state.GenTxnInput(key))
		}
	}

	if len(txn.Inputs) == 0 {
		fmt.Println("Wallet empty.")
		return
	}

	key := genKey()
	txn.Outputs = append(txn.Outputs, TxnOutput{key.PublicKey, total})

	state.Sign(txn)

	success := state.AddTxn(txn)
	if success {
		state.AddToWallet(key)
		network.BroadcastTxn(txn)
		fmt.Println("Wallet consolidated.")
	} else {
		fmt.Println("Failed.")
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
			len(block.Txns), block.Nonce, block.Hash()[0:12])
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

func doPay(input chan string) {
	peers := network.PeerAddrList()
	if len(peers) < 1 {
		fmt.Println("No connected peers to pay.")
		return
	}

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	defer close(interrupt)
	defer fmt.Println()

	fmt.Println("Select your payee:")
	for i, peer := range peers {
		fmt.Printf(" %2d -- %s\n", i+1, peer)
	}

	peer := ""
	for len(peer) == 0 {
		fmt.Print(">> ")
		select {
		case text := <-input:
			i, err := strconv.Atoi(text)
			if err != nil || i < 1 || i > len(peers) {
				fmt.Println("Invalid input")
			} else {
				peer = peers[i-1]
			}
		case <-interrupt:
			return
		}
	}

	var total uint64
	for _, val := range state.GetWallet() {
		total += val
	}
	var amount uint64
	fmt.Println("Pay how much? (You have", total, "in your wallet)")
	for amount == 0 {
		fmt.Print(">> ")
		select {
		case text := <-input:
			i, err := strconv.ParseInt(text, 10, 64)
			if err != nil || i < 1 || uint64(i) > total {
				fmt.Println("Invalid input")
			} else {
				amount = uint64(i)
			}
		case <-interrupt:
			return
		}
	}

	expect, err := network.RequestPayableAddress(peer)
	if err != nil {
		fmt.Print(err)
		return
	}

	var key *rsa.PublicKey
	select {
	case key = <-expect:
	case <-interrupt:
		network.CancelPayExpectation(peer)
	}

	txn := new(Transaction)

	total = 0
	for key, val := range state.GetWallet() {
		if val > 0 {
			total += val
			txn.Inputs = append(txn.Inputs, state.GenTxnInput(key))
		}
		if total >= amount {
			break
		}
	}

	txn.Outputs = append(txn.Outputs, TxnOutput{*key, amount})
	var change *rsa.PrivateKey
	if total > amount {
		// calculate change
		change = genKey()
		txn.Outputs = append(txn.Outputs, TxnOutput{change.PublicKey, total - amount})
	}

	err = state.Sign(txn)
	if err != nil {
		fmt.Print(err)
		return
	}

	success := state.AddTxn(txn)
	if success {
		if change != nil {
			state.AddToWallet(change)
		}
		network.BroadcastTxn(txn)
		fmt.Println("Payment sent.")
	} else {
		fmt.Println("Failed, please try again.")
	}
}

func printHelp() {

	fmt.Println()
	flag.Usage()

	fmt.Println()
	fmt.Println("Possible commands are:")
	fmt.Println("  help (displays this help)")
	fmt.Println("  quit (exits gocoin)")
	fmt.Println()
	fmt.Println("  state  (display blockchain and transaction state)")
	fmt.Println("  wallet (display wallet)")
	fmt.Println()
	fmt.Println("  cons (consolidate wallet into single key)")
	fmt.Println("  pay (perform a transaction to pay another client)")
	fmt.Println()
}
