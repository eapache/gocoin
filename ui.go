package main

import (
	"bufio"
	"fmt"
	"os"
)

func doUI() {
	printHelp()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("> ")
	for scanner.Scan() {
		switch scanner.Text() {
		case "": // do nothing, ignore
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
	// TODO
}
