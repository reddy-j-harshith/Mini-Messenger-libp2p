package main

import (
	"context"
	"fmt"
	"log"

	"crypto/rand"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
)

func generateRSAKey() crypto.PrivKey {
	privateKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		panic(err)
	}
	return privateKey
}

func main() {
	// Define the multiaddress for the bootstrap node
	port := "4001" // Hardcoded port
	listenAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%s", port)

	// Create a libp2p host listening on the specified address
	host, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.Identity(generateRSAKey()),
	)
	if err != nil {
		log.Fatalf("Failed to create libp2p host: %v", err)
	}

	// Create a new DHT for the host
	_, err = dht.New(context.Background(), host, dht.Mode(dht.ModeServer), dht.ProtocolPrefix("/custom-dht"), dht.BucketSize(5))
	if err != nil {
		log.Fatalf("Failed to create DHT: %v", err)
	}

	// Set up a notification system to listen for new peers
	host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			fmt.Printf("New node joined! Peer ID: %s\n", c.RemotePeer().String())
		},
	})

	// Print the multiaddress of the bootstrap node
	for _, addr := range host.Addrs() {
		fmt.Printf("Bootstrap node address: %s/p2p/%s\n", addr, host.ID().String())
	}

	// Keep the bootstrap node running
	select {}
}
