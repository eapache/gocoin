package main

import (
	"encoding/gob"
	"errors"
	"net"
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
)

type PeerConn struct {
	base net.Conn
	encoder *gob.Encoder
	decoder *gob.Decoder
}

type PeerNetwork struct {
	peers map[string]*PeerConn
	server net.Listener
}

func NewPeerNetwork(startPeer string) (*PeerNetwork, error) {
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

	tmpAddrs := make([]string, 0)
	err = decoder.Decode(&tmpAddrs)
	if err != nil {
		return nil, err
	}

	if len(tmpAddrs) == 0 {
		return nil, errors.New("Initial peer returned empty peer list")
	}

	network := &PeerNetwork{peers: make(map[string]*PeerConn, len(tmpAddrs))}
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

		network.peers[addr] = &PeerConn{conn, encoder, decoder}
	}

	return network, nil
}
