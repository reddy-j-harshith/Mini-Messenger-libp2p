## This is a demonstration of a p2p messaging application with node discovery and messaging. Made using go-libp2p

Manually implemented the Gossip-protocol for propagating throughout the entire network

TODO:

-> Gracefully handle shutting down of node

    => Notify the network about shutdown so that bootstraps can update DHT
    => Remove the offline nodes from the local DHT

-> Installation

Must have Go lang installed, v1.22+

    => go mod init Messenger
    => go get github.com/libp2p/go-libp2p
    => go get github.com/libp2p/go-libp2p-kad-dht

-> Instructions

In the Bootstrap directory, run the command:

    => go run bootstrap.go

To run the individual nodes and connect to the bootstrap nodes:

    => go run . -peer {bootstrap address}