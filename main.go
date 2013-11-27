package main

import (
	"crypto/rsa"
	"encoding/gob"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"runtime"
	"time"
)

var network *PeerNetwork
var state *State

func main() {
	// these are used as interface values so must be registered first
	gob.Register(Block{})
	gob.Register(BlockChain{})
	gob.Register(Transaction{})

	// XXX so it appears that "gob" assigns type IDs consecutively as they are used, which
	// means that if two processes encode different types first, the same type will get different IDs,
	// meaning that the same object in the two processes will hash to different values! This is obviously
	// a problem for us, since we have to verify hashes across processes, so encode all of our types right
	// away in a specific order so that all processes assign them the same type IDs
	encoder := gob.NewEncoder(ioutil.Discard)
	encoder.Encode(rsa.PublicKey{})
	encoder.Encode(TxnInput{})
	encoder.Encode(TxnOutput{})
	encoder.Encode(Transaction{})
	encoder.Encode(Block{})
	encoder.Encode(BlockChain{})

	initialPeer := flag.String("connect", "", "Address of peer to connect to, leave blank for new network")
	address := flag.String("listen", ":0", "Listening address of peer, default is random localhost port")
	flag.Parse()

	// XXX so mining doesn't block everything, since the goroutine scheduler only kicks in on
	// system calls which mining doesn't make in the CPU-intensive path
	runtime.GOMAXPROCS(2)

	var err error
	network, err = NewPeerNetwork(*address, *initialPeer)
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
		b, key := state.ConstructBlock()
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
					state.AddToWallet(key)
					network.BroadcastBlock(b)
					continue mineNewBlock
				}
			}
		}
	}
}
