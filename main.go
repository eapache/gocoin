package main

import (
	"flag"
	"fmt"
)

var network *PeerNetwork
var state *State

func main() {
	initialPeer := flag.String("connect", "", "Address of peer, leave blank for new network")
	flag.Parse()

	var err error
	network, err = NewPeerNetwork(*initialPeer)
	if err != nil {
		panic(err)
	}

	state = NewState(*initialPeer)

	network.RequestBlockChain(nil) // nil hash for the primary chain

	go state.MineForGold()

	fmt.Println("Startup complete, listening on ", network.server.Addr())

	mainLoop(state)

	state.Close()
	network.Close()
}
