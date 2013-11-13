package main

import (
	"flag"
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

var network *PeerNetwork
var state *State

func main() {
	initialPeer := flag.String("connect", "", "Address of peer, leave blank for new network")
	flag.Parse()

	// XXX so mining doesn't block everything, since the goroutine scheduler only kicks in on
	// system calls which mining doesn't make in the CPU-intensive path
	runtime.GOMAXPROCS(2)

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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

mineNewBlock:
	for {
		b := state.ConstructBlock()
		for {
			if state.ResetMiner {
				state.UndoBlock(b)
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
