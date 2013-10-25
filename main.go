package main

import (
	"flag"
)

func main() {
	flag.Parse()

	network, err := NewPeerNetwork("")
	if err != nil {
		panic(err)
	}

        doUI()

	network.Close()
}
