package main

import (
	"encoding/gob"
	"errors"
	"io"
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

func NewPeerConn(conn net.Conn) *PeerConn {
	return &PeerConn{conn, gob.NewEncoder(conn), gob.NewDecoder(conn)}
}

func (peer *PeerConn) Send(msg *NetworkMessage) error {
	err := peer.encoder.Encode(msg)

	if err == nil {
		return nil
	}

	switch err.(type) {
	case net.Error:
		// caller may choose to ignore this
		return err
	default:
		// probably a gob error which we want to know about
		panic(err)
	}
}

func (peer *PeerConn) Receive() (*NetworkMessage, error) {
	msg := new(NetworkMessage)
	err := peer.decoder.Decode(msg)

	if err == nil {
		return msg, nil
	}

	switch err.(type) {
	case net.Error:
		// caller may choose to ignore this
		return nil, err
	default:
		// probably a gob error which we want to know about
		panic(err)
	}
}

type PeerNetwork struct {
	peers    map[string]*PeerConn
	server   net.Listener
	events   chan *NetworkMessage
	closing  bool
	peerLock sync.RWMutex
}

func NewPeerNetwork(address, startPeer string) (network *PeerNetwork, err error) {
	var peerAddrs []string

	if startPeer != "" {
		conn, err := net.Dial("tcp", startPeer)
		if err != nil {
			return nil, err
		}
		defer conn.Close()

		peer := NewPeerConn(conn)

		err = peer.Send(&NetworkMessage{Type: PeerListRequest})
		if err != nil {
			return nil, err
		}

		msg, err := peer.Receive()
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
	network.server, err = net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	go network.AcceptNewConns()
	go network.HandleEvents()

	msg := NetworkMessage{Type: PeerBroadcast, Value: network.server.Addr().String()}
	for _, addr := range peerAddrs {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}

		peer := NewPeerConn(conn)

		err = peer.Send(&msg)
		if err != nil {
			conn.Close()
			continue
		}

		network.peers[addr] = peer
		go network.ReceiveFromConn(addr)
	}

	return network, nil
}

func (network *PeerNetwork) AcceptNewConns() {
	for {
		conn, err := network.server.Accept()

		if err != nil {
			network.events <- &NetworkMessage{Error, err, ""}
			return
		}

		peer := NewPeerConn(conn)

		msg, err := peer.Receive()
		if err != nil {
			conn.Close()
			continue
		}

		switch msg.Type {
		case PeerListRequest:
			response := NetworkMessage{Type: PeerListResponse, Value: append(network.PeerAddrList(), network.server.Addr().String())}
			peer.Send(&response)
			conn.Close()
		case PeerBroadcast:
			switch addr := msg.Value.(type) {
			case string:
				network.peerLock.Lock()
				if !network.closing && network.peers[addr] == nil {
					network.peers[addr] = peer
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
			chain := state.ChainFromHash(hash)
			message := &NetworkMessage{Type: BlockChainResponse, Value: chain}
			peer := network.Peer(msg.addr)
			peer.Send(message)
		case BlockChainResponse:
			chain := msg.Value.(BlockChain)
			state.AddBlockChain(&chain)
		case BlockBroadcast:
			block := msg.Value.(Block)
			valid, haveChain := state.NewBlock(&block)
			if valid && !haveChain {
				network.RequestBlockChain(block.Hash())
			}
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
					} else if msg.Value != io.EOF {
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

func (network *PeerNetwork) Peer(addr string) *PeerConn {
	network.peerLock.RLock()
	defer network.peerLock.RUnlock()

	return network.peers[addr]
}

func (network *PeerNetwork) RequestBlockChain(hash []byte) {
	network.peerLock.RLock()
	defer network.peerLock.RUnlock()

	// pick a random peer
	message := NetworkMessage{Type: BlockChainRequest, Value: hash}
	for _, peer := range network.peers {
		peer.Send(&message)
		return
	}
}

func (network *PeerNetwork) BroadcastBlock(b *Block) {
	network.peerLock.RLock()
	defer network.peerLock.RUnlock()

	// send to all peers
	message := NetworkMessage{Type: BlockBroadcast, Value: b}
	for _, peer := range network.peers {
		peer.Send(&message)
	}
}
