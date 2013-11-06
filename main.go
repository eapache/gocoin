package main

import (
	"flag"
	"fmt"
)

func main() {
	initialPeer := flag.String("connect", "", "Address of peer, leave blank for new network")
	flag.Parse()

	state := NewState(*initialPeer)

	fmt.Println("Startup complete, listening on ", state.network.server.Addr())

	mainLoop(state)

	state.Close()
}
