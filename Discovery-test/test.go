package main

import (
	"context"
	"fmt"
	"time"

	log "github.com/ipfs/go-log/v2"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

func main() {
	log.SetAllLoggers(log.LevelWarn)
	log.SetLogLevel("rendezvous", "info")
	logger := log.Logger("rendezvous")

	// Parse flags
	config, err := ParseFlags()
	if err != nil {
		panic(err)
	}

	// Create a new libp2p host
	host, err := libp2p.New(
		libp2p.ListenAddrs(
			[]multiaddr.Multiaddr(config.ListenAddresses)...,
		),
	)
	if err != nil {
		panic(err)
	}

	logger.Info("Node created with ID: ", host.ID().String())

	ctx := context.Background()

	// Extract bootstrap peers
	bootstrapPeers := make([]peer.AddrInfo, len(config.BootstrapPeers))
	for i, addr := range config.BootstrapPeers {
		peerInfo, _ := peer.AddrInfoFromP2pAddr(addr)
		bootstrapPeers[i] = *peerInfo
	}

	// Create a local DHT
	kademliaDHT, err := dht.New(ctx, host, dht.BootstrapPeers(bootstrapPeers...), dht.ProtocolPrefix("/custom-dht"), dht.BucketSize(5))
	if err != nil {
		panic(err)
	}
	defer kademliaDHT.Close()

	logger.Debug("Bootstrapping the DHT")
	if err := kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	// Wait for bootstrapping to complete
	time.Sleep(1 * time.Second)

	// Announce presence
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, config.RendezvousString)

	// Detect new peers joining the network
	go func() {

		for {
			// Start peer discovery
			peerChan, err := routingDiscovery.FindPeers(ctx, config.RendezvousString)
			if err != nil {
				panic(err)
			}
			for peer := range peerChan {
				if peer.ID == host.ID() {
					continue
				}
				fmt.Printf("New peer joined: %s\n", peer.ID)
				logger.Info("New peer joined: ", peer)

				if err := host.Connect(ctx, peer); err != nil {
					logger.Warn("Failed to connect to new peer: ", err)
				} else {
					logger.Info("Connected to new peer: ", peer.ID)
				}
			}
		}
	}()

	// Keep the program running
	select {}
}
