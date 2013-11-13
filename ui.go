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
			fmt.Println("Wallet:")
			for key, _ := range state.wallet.Keys {
				fmt.Println(key)
				fmt.Println()
			}
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

func printHelp() {
	fmt.Println("Possible commands are:")
	fmt.Println("  help (this help)")
	fmt.Println("  quit (exits gocoin)")
	fmt.Println("  show (display wallet)")
	fmt.Println("  genk (generates a new key and adds it to the wallet)")
	fmt.Println("")
}
