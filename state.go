package main

type State struct {
	primary   *BlockChain
	alternate []*BlockChain
}
