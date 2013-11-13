package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"
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

	state = NewState()

	network.RequestBlockChain(nil) // nil hash for the primary chain

	stopper := make(chan bool)
	go MineForGold(stopper)

	fmt.Println("Startup complete, listening on ", network.server.Addr())

	mainLoop()

	stopper <- true
	network.Close()
}

func MineForGold(stopper chan bool) {
mineNewBlock:
	for {
		b := state.ConstructBlock()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for {
			if state.ResetMiner {
				continue mineNewBlock
			}
			select {
			case <-stopper:
				return
			default:
				b.Nonce = r.Uint32()
				success, _ := state.NewBlock(b)
				if success {
					network.BroadcastBlock(b)
					continue mineNewBlock
				}
			}
		}
	}
}
