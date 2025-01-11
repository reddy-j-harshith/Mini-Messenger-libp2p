package main

// import (
// 	"bufio"
// 	"context"
// 	"fmt"

// 	"os"
// 	"time"

// 	"github.com/ipfs/go-log/v2"
// 	"github.com/libp2p/go-libp2p"
// 	dht "github.com/libp2p/go-libp2p-kad-dht"
// 	"github.com/libp2p/go-libp2p/core/network"
// 	"github.com/libp2p/go-libp2p/core/peer"
// 	"github.com/libp2p/go-libp2p/core/protocol"
// 	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
// 	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
// 	"github.com/multiformats/go-multiaddr"
// )

// var logger = log.Logger("rendezvous")

// func readData(rw *bufio.ReadWriter) {
// 	for {
// 		str, err := rw.ReadString('\n')
// 		if err != nil {
// 			fmt.Println("User Went Offline", err)
// 			return
// 		}

// 		if str == "" || str == "\n" {
// 			continue
// 		}

// 		fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
// 	}
// }

// func writeData(rw *bufio.ReadWriter) {
// 	stdReader := bufio.NewReader(os.Stdin)

// 	for {
// 		fmt.Print("> ")
// 		sendData, err := stdReader.ReadString('\n')
// 		if err != nil {
// 			fmt.Println("Error reading from stdin:", err)
// 			return
// 		}

// 		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
// 		if err != nil {
// 			fmt.Println("Error writing to buffer:", err)
// 			return
// 		}

// 		err = rw.Flush()
// 		if err != nil {
// 			fmt.Println("Error flushing buffer:", err)
// 			return
// 		}
// 	}
// }

// func messageProtocol(stream network.Stream) {

// 	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

// 	go readData(rw)
// 	go writeData(rw)

// }

// func main() {

// 	log.SetAllLoggers(log.LevelWarn)
// 	log.SetLogLevel("rendezvous", "info")

// 	config, err := ParseFlags()

// 	if err != nil {
// 		panic(err)
// 	}

// 	// Creating the current node
// 	host, err := libp2p.New(
// 		libp2p.ListenAddrs(
// 			[]multiaddr.Multiaddr(config.ListenAddresses)...,
// 		),
// 	)

// 	if err != nil {
// 		panic(err)
// 	}

// 	logger.Info("Node created with the ID: ", host.ID().String())

// 	// Setting a handler
// 	host.SetStreamHandler(protocol.ID(config.ProtocolID), messageProtocol)

// 	ctx := context.Background()
// 	bootstrapPeers := make([]peer.AddrInfo, len(config.BootstrapPeers))

// 	for i, addr := range config.BootstrapPeers {
// 		peerInfo, _ := peer.AddrInfoFromP2pAddr(addr)
// 		bootstrapPeers[i] = *peerInfo
// 	}

// 	// Create a local dht
// 	kademliaDHT, err := dht.New(ctx, host, dht.BootstrapPeers(bootstrapPeers...), dht.ProtocolPrefix("/custom-dht"), dht.BucketSize(5))

// 	if err != nil {
// 		panic(err)
// 	}

// 	// Clean-up scheduled
// 	defer kademliaDHT.Close()

// 	logger.Debug("Bootstrapping the node's DHT")
// 	if err = kademliaDHT.Bootstrap(ctx); err != nil {
// 		panic(err)
// 	}

// 	// Wait for the Bootstrapping to finish
// 	time.Sleep(1 * time.Second)

// 	logger.Debug("Announce your arrival")
// 	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
// 	dutil.Advertise(ctx, routingDiscovery, config.RendezvousString)
// 	logger.Debug("Successfully Announced")

// 	logger.Debug("Starting the Peer discovery")

// 	peerChan, err := routingDiscovery.FindPeers(ctx, config.RendezvousString)

// 	if err != nil {
// 		panic(err)
// 	}

// 	for peer := range peerChan {
// 		if peer.ID == host.ID() {
// 			continue
// 		}

// 		logger.Info("Found Peer: ", peer)

// 		logger.Info("Conecting to:", peer)

// 		stream, err := host.NewStream(ctx, peer.ID, protocol.ID(config.ProtocolID))

// 		if err != nil {
// 			logger.Warning("Connection Failed!!", err)
// 			continue
// 		}

// 		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

// 		go readData(rw)
// 		go writeData(rw)

// 		logger.Info("Connected to: ", peer)

// 	}

// 	select {}
// }
