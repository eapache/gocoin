package main

import (
	"flag"
	"fmt"
)

func main() {
	initialPeer := flag.String("connect", "", "Address of peer, leave blank for new network")
	flag.Parse()

	network, err := NewPeerNetwork(*initialPeer)
	if err != nil {
		panic(err)
	}

	fmt.Println("Startup complete, listening on ", network.server.Addr())

	doUI()

	network.Close()
}
