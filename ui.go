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
		case "show":
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
	for key, _ := range state.wallet.Keys {
		_, val := state.GetValue(key)
		fmt.Printf("%8d | %s...\n", val, key.N.String()[0:40])
		total += val
	}
	fmt.Printf("\nTotal Coins: %d\n\n", total)
}

func printHelp() {
	fmt.Println("Possible commands are:")
	fmt.Println("  help (this help)")
	fmt.Println("  quit (exits gocoin)")
	fmt.Println("  show (display wallet)")
	fmt.Println("  genk (generates a new key and adds it to the wallet)")
	fmt.Println("")
}
