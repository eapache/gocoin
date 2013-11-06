package main

import (
	"encoding/gob"
	"errors"
	"net"
	"sync"
)

type MsgType int32

const (
	PeerListRequest  MsgType = iota
	PeerListResponse MsgType = iota
	PeerBroadcast    MsgType = iota

	BlockChainRequest  MsgType = iota
	BlockChainResponse MsgType = iota
	BlockBroadcast     MsgType = iota

	TransactionRequest   MsgType = iota
	TransactionResponse  MsgType = iota
	TransactionBroadcast MsgType = iota

	Error MsgType = iota
)

type NetworkMessage struct {
	Type  MsgType
	Value interface{}

	addr string // filled on the receiving side
}

type PeerConn struct {
	base    net.Conn
	encoder *gob.Encoder
	decoder *gob.Decoder
}

type PeerNetwork struct {
	peers    map[string]*PeerConn
	server   net.Listener
	events   chan *NetworkMessage
	closing  bool
	peerLock sync.RWMutex
	state    *State
}

func NewPeerNetwork(startPeer string) (network *PeerNetwork, err error) {
	var msg NetworkMessage
	var peerAddrs []string

	if startPeer != "" {
		conn, err := net.Dial("tcp", startPeer)
		if err != nil {
			return nil, err
		}
		defer conn.Close()

		encoder := gob.NewEncoder(conn)
		decoder := gob.NewDecoder(conn)

		err = encoder.Encode(&NetworkMessage{Type: PeerListRequest})
		if err != nil {
			return nil, err
		}

		err = decoder.Decode(&msg)
		if err != nil {
			return nil, err
		}

		if msg.Type != PeerListResponse {
			return nil, errors.New("Received message not a PeerListResponse")
		}

		switch v := msg.Value.(type) {
		case []string:
			peerAddrs = v
		default:
			return nil, errors.New("Unknown value in PeerListResponse")
		}
	}

	network = &PeerNetwork{
		peers:  make(map[string]*PeerConn, len(peerAddrs)),
		events: make(chan *NetworkMessage),
	}
	network.server, err = net.Listen("tcp", ":0")
	if err != nil {
		return nil, err
	}

	for _, addr := range peerAddrs {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}

		encoder := gob.NewEncoder(conn)

		err = encoder.Encode(PeerBroadcast)
		if err != nil {
			conn.Close()
			continue
		}

		err = encoder.Encode(network.server.Addr().String())
		if err != nil {
			conn.Close()
			continue
		}

		decoder := gob.NewDecoder(conn)

		network.peers[addr] = &PeerConn{base: conn, encoder: encoder, decoder: decoder}
		go network.ReceiveFromConn(addr)
	}

	go network.AcceptNewConns()
	go network.HandleEvents()

	return network, nil
}

func (network *PeerNetwork) AcceptNewConns() {
	for {
		conn, err := network.server.Accept()

		if err != nil {
			network.events <- &NetworkMessage{Error, err, ""}
			return
		}

		encoder := gob.NewEncoder(conn)
		decoder := gob.NewDecoder(conn)

		var msg NetworkMessage
		err = decoder.Decode(&msg)
		if err != nil {
			conn.Close()
			continue
		}

		switch msg.Type {
		case PeerListRequest:
			response := NetworkMessage{Type: PeerListResponse, Value: network.PeerAddrList()}
			encoder.Encode(&response) // XXX anything to handle error?
			conn.Close()
		case PeerBroadcast:
			switch addr := msg.Value.(type) {
			case string:
				network.peerLock.Lock()
				if !network.closing && network.peers[addr] == nil {
					network.peers[addr] = &PeerConn{base: conn, encoder: encoder, decoder: decoder}
					go network.ReceiveFromConn(addr)
				} else {
					conn.Close()
				}
				network.peerLock.Unlock()
			default:
				conn.Close()
			}
		default:
			conn.Close()
		}
	}
}

func (network *PeerNetwork) ReceiveFromConn(addr string) {
	peer := network.peers[addr]

	var err error
	var msg NetworkMessage

	for {
		err = peer.decoder.Decode(&msg)
		if err != nil {
			network.events <- &NetworkMessage{Error, err, addr}
			return
		}

		msg.addr = addr

		network.events <- &msg
	}
}

func (network *PeerNetwork) HandleEvents() {
	for msg := range network.events {
		switch msg.Type {
		case BlockChainRequest:
			hash := msg.Value.([]byte)
			chain := network.state.ChainFromHash(hash)
			message := &NetworkMessage{Type: BlockChainResponse, Value: chain}
			peer := network.peers[msg.addr] // XXX lock map?
			peer.encoder.Encode(&message)   // XXX anything to handle error?
		case BlockChainResponse:
			chain := msg.Value.(*BlockChain)
			network.state.addBlockChain(chain)
		case BlockBroadcast:
			block := msg.Value.(*Block)
			network.state.newBlock(block)
		case Error:
			if msg.addr == "" {
				if network.closing {
					if len(network.peers) == 0 {
						close(network.events)
						return
					}
				} else {
					panic(msg.Value)
				}
			} else {
				network.peerLock.Lock()
				delete(network.peers, msg.addr)
				network.peerLock.Unlock()
				if len(network.peers) == 0 {
					if network.closing {
						close(network.events)
						return
					} else {
						panic(msg.Value)
					}
				}
			}
		}
	}
}

func (network *PeerNetwork) Close() {
	network.peerLock.Lock()
	defer network.peerLock.Unlock()

	network.closing = true
	network.server.Close()
	for _, peer := range network.peers {
		peer.base.Close()
	}
}

func (network *PeerNetwork) PeerAddrList() []string {
	network.peerLock.RLock()
	defer network.peerLock.RUnlock()

	list := make([]string, 0, len(network.peers))
	for addr, _ := range network.peers {
		list = append(list, addr)
	}
	return list
}

func (network *PeerNetwork) RequestBlockChain(hash []byte) {
	network.peerLock.RLock()
	defer network.peerLock.RUnlock()

	// pick a random peer
	for _, peer := range network.peers {
		message := &NetworkMessage{Type: BlockChainRequest, Value: hash}
		peer.encoder.Encode(&message) // XXX anything to handle error?
		return
	}
}

func (network *PeerNetwork) BroadcastBlock(b *Block) {
	network.peerLock.RLock()
	defer network.peerLock.RUnlock()

	// pick a random peer
	message := &NetworkMessage{Type: BlockBroadcast, Value: b}
	for _, peer := range network.peers {
		peer.encoder.Encode(&message) // XXX anything to handle error?
	}
}
