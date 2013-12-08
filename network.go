package main

import (
	"crypto/rsa"
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
	peers      map[string]*PeerConn
	server     net.Listener
	events     chan *NetworkMessage
	payExpects map[string]chan *rsa.PublicKey
	closing    bool
	lock       sync.RWMutex
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
		peers:      make(map[string]*PeerConn, len(peerAddrs)),
		payExpects: make(map[string]chan *rsa.PublicKey),
		events:     make(chan *NetworkMessage),
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
				network.lock.Lock()
				if !network.closing && network.peers[addr] == nil {
					network.peers[addr] = peer
					go network.ReceiveFromConn(addr)
					logger.Println("New peer:", addr)
				} else {
					conn.Close()
				}
				network.lock.Unlock()
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
			message := NetworkMessage{Type: BlockChainResponse, Value: chain}
			peer := network.Peer(msg.addr)
			peer.Send(&message)
		case BlockChainResponse:
			chain := msg.Value.(BlockChain)
			logger.Println("Received blockchain from", msg.addr)
			state.AddBlockChain(&chain)
		case BlockBroadcast:
			logger.Println("Received block from", msg.addr)
			block := msg.Value.(Block)
			valid, haveChain := state.AddBlock(&block)
			if valid && !haveChain {
				network.RequestBlockChain(block.Hash())
			}
		case TransactionRequest:
			key := genKey()
			message := NetworkMessage{Type: TransactionResponse, Value: key.PublicKey}
			peer := network.Peer(msg.addr)
			peer.Send(&message)
			state.AddToWallet(key)
		case TransactionResponse:
			network.lock.Lock()
			expect := network.payExpects[msg.addr]
			if expect != nil {
				key := msg.Value.(rsa.PublicKey)
				expect <- &key
				close(expect)
				delete(network.payExpects, msg.addr)
			}
			network.lock.Unlock()
		case TransactionBroadcast:
			logger.Println("Received txn from", msg.addr)
			txn := msg.Value.(Transaction)
			state.AddTxn(&txn)
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
				network.lock.Lock()
				delete(network.peers, msg.addr)
				network.lock.Unlock()
				logger.Println("Lost peer:", msg.addr)
				if len(network.peers) == 0 {
					if network.closing {
						close(network.events)
						return
					} else if msg.Value != io.EOF {
						panic(msg.Value)
					}
				}
			}
		default:
			panic(msg.Type)
		}
	}
}

func (network *PeerNetwork) Close() {
	network.lock.Lock()
	defer network.lock.Unlock()

	network.closing = true
	network.server.Close()
	for _, peer := range network.peers {
		peer.base.Close()
	}
}

func (network *PeerNetwork) PeerAddrList() []string {
	network.lock.RLock()
	defer network.lock.RUnlock()

	list := make([]string, 0, len(network.peers))
	for addr, _ := range network.peers {
		list = append(list, addr)
	}
	return list
}

func (network *PeerNetwork) Peer(addr string) *PeerConn {
	network.lock.RLock()
	defer network.lock.RUnlock()

	return network.peers[addr]
}

func (network *PeerNetwork) CancelPayExpectation(addr string) {
	network.lock.Lock()
	defer network.lock.Unlock()

	close(network.payExpects[addr])
	delete(network.payExpects, addr)
}

func (network *PeerNetwork) genPayExpectation(addr string) chan *rsa.PublicKey {
	network.lock.Lock()
	defer network.lock.Unlock()

	c := make(chan *rsa.PublicKey)
	network.payExpects[addr] = c
	return c
}

func (network *PeerNetwork) RequestPayableAddress(addr string) (chan *rsa.PublicKey, error) {
	peer := network.Peer(addr)

	if peer == nil {
		return nil, errors.New("Peer no longer connected")
	}

	expect := network.genPayExpectation(addr)

	message := NetworkMessage{Type: TransactionRequest}
	err := peer.Send(&message)

	if err != nil {
		network.CancelPayExpectation(addr)
		return nil, err
	}

	return expect, nil
}

func (network *PeerNetwork) RequestBlockChain(hash []byte) {
	network.lock.RLock()
	defer network.lock.RUnlock()

	// pick a random peer
	message := NetworkMessage{Type: BlockChainRequest, Value: hash}
	for _, peer := range network.peers {
		peer.Send(&message)
		return
	}
}

func (network *PeerNetwork) BroadcastBlock(b *Block) {
	network.lock.RLock()
	defer network.lock.RUnlock()

	// send to all peers
	message := NetworkMessage{Type: BlockBroadcast, Value: b}
	for _, peer := range network.peers {
		peer.Send(&message)
	}
}

func (network *PeerNetwork) BroadcastTxn(txn *Transaction) {
	network.lock.RLock()
	defer network.lock.RUnlock()

	// send to all peers
	message := NetworkMessage{Type: TransactionBroadcast, Value: txn}
	for _, peer := range network.peers {
		peer.Send(&message)
	}
}
