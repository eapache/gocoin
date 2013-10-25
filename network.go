package main

import (
	"encoding/gob"
	"errors"
	"net"
	"sync"
)

type MsgType int32

const (
	PeerListRequest MsgType = iota
	PeerBroadcast   MsgType = iota

	BlockChainRequest MsgType = iota
	BlockBroadcast    MsgType = iota

	TransactionRequest   MsgType = iota
	TransactionBroadcast MsgType = iota
)

type PeerConn struct {
	base    net.Conn
	encoder *gob.Encoder
	decoder *gob.Decoder
	txLock  sync.Mutex
}

type PeerEvent struct {
	addr  string
	value interface{}
}

type PeerNetwork struct {
	peers    map[string]*PeerConn
	server   net.Listener
	events   chan PeerEvent
	closing  bool
	peerLock sync.RWMutex
}

func NewPeerNetwork(startPeer string) (network *PeerNetwork, err error) {
	tmpAddrs := make([]string, 0)

	if startPeer != "" {
		conn, err := net.Dial("udp", startPeer)
		if err != nil {
			return nil, err
		}
		defer conn.Close()

		encoder := gob.NewEncoder(conn)
		decoder := gob.NewDecoder(conn)

		err = encoder.Encode(PeerListRequest)
		if err != nil {
			return nil, err
		}

		err = decoder.Decode(&tmpAddrs)
		if err != nil {
			return nil, err
		}

		if len(tmpAddrs) == 0 {
			return nil, errors.New("Initial peer returned empty peer list")
		}
	}

	network = &PeerNetwork{
		peers:  make(map[string]*PeerConn, len(tmpAddrs)),
		events: make(chan PeerEvent),
	}
	network.server, err = net.Listen("udp", ":0")
	if err != nil {
		return nil, err
	}

	for _, addr := range tmpAddrs {
		conn, err := net.Dial("udp", addr)
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
			network.events <- PeerEvent{"", err}
			return
		}

		encoder := gob.NewEncoder(conn)
		decoder := gob.NewDecoder(conn)

		var nextType MsgType
		err = decoder.Decode(&nextType)
		if err != nil {
			conn.Close()
			continue
		}

		switch nextType {
		case PeerListRequest:
			peerList := network.PeerAddrList()
			err = encoder.Encode(peerList)
			conn.Close()
		case PeerBroadcast:
			var addr string
			err = decoder.Decode(&addr)
			network.peerLock.Lock()
			if err != nil || network.peers[addr] != nil {
				network.peerLock.Unlock()
				conn.Close()
				continue
			}
			network.peers[addr] = &PeerConn{base: conn, encoder: encoder, decoder: decoder}
			go network.ReceiveFromConn(addr)
			network.peerLock.Unlock()
		default:
			conn.Close()
		}
	}
}

func (network *PeerNetwork) ReceiveFromConn(addr string) {
	peer := network.peers[addr]

	var err error
	var nextType MsgType

	for {
		err = peer.decoder.Decode(&nextType)
		if err != nil {
			network.events <- PeerEvent{addr, err}
			return
		}

		switch nextType {
		default:
			network.events <- PeerEvent{addr, errors.New("Unknown message type received")}
			return
		}
	}
}

func (network *PeerNetwork) HandleEvents() {
	for event := range network.events {
		switch val := event.value.(type) {
		case error:
			if event.addr == "" {
				if network.closing {
					if len(network.peers) == 0 {
						close(network.events)
						return
					}
				} else {
					panic(val)
				}
			} else {
				network.peerLock.Lock()
				delete(network.peers, event.addr)
				network.peerLock.Unlock()
				if len(network.peers) == 0 {
					if network.closing {
						close(network.events)
						return
					} else {
						panic(val)
					}
				}
			}
		}
	}
}

func (network *PeerNetwork) Close() {
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
