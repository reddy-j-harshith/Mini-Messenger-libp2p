package main

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"os"
	"time"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("rendezvous")

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("User Went Offline", err)
			return
		}

		if str == "" || str == "\n" {
			continue
		}

		fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
	}
}

func messageProtocol(stream network.Stream) {

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw)
	// go writeData(rw)

}

func main() {

	log.SetAllLoggers(log.LevelWarn)
	log.SetLogLevel("rendezvous", "info")

	// Parsing the flags
	config, err := ParseFlags()
	if err != nil {
		panic(err)
	}

	// Creating the current node
	host, err := libp2p.New(
		libp2p.ListenAddrs(
			[]multiaddr.Multiaddr(config.ListenAddresses)...,
		),
	)
	if err != nil {
		panic(err)
	}

	logger.Info("Node created with the ID: ", host.ID().String())

	// Setting a handler
	host.SetStreamHandler(protocol.ID(config.ProtocolID), messageProtocol)

	// Extract Bootstrap peers
	ctx := context.Background()
	bootstrapPeers := make([]peer.AddrInfo, len(config.BootstrapPeers))

	// Add the Bootstrap peers to the peer slice
	for i, addr := range config.BootstrapPeers {
		peerInfo, _ := peer.AddrInfoFromP2pAddr(addr)
		bootstrapPeers[i] = *peerInfo
	}

	// Create a local dht with custom buket size
	kademliaDHT, err := dht.New(ctx, host, dht.BootstrapPeers(bootstrapPeers...), dht.ProtocolPrefix("/custom-dht"), dht.BucketSize(5))
	if err != nil {
		panic(err)
	}

	// Clean-up scheduled
	defer kademliaDHT.Close()

	logger.Debug("Bootstrapping the node's DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	// Wait for the Bootstrapping to finish
	time.Sleep(2 * time.Second)

	// Announce your arrival
	logger.Debug("Announcing your arrival")
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, config.RendezvousString)
	logger.Debug("Successfully Announced")

	logger.Debug("Starting the Peer discovery")

	peerArray := make([]peer.AddrInfo, 0)
	peerSet := make(map[peer.ID]bool)

	go func() {

		for {
			// Create a peer channel for all the available users
			peerChan, err := routingDiscovery.FindPeers(ctx, config.RendezvousString)
			if err != nil {
				logger.Error("Error finding peers:", err)
				time.Sleep(2 * time.Second)
				continue
			}

			for peer := range peerChan {

				if peer.ID == host.ID() {
					continue
				}

				if _, exists := peerSet[peer.ID]; exists {
					continue
				}

				if err := host.Connect(ctx, peer); err != nil {
					logger.Warn("Failed to connect to peer:", err)
				} else {
					logger.Info("Connected to peer:", peer.ID)
					peerArray = append(peerArray, peer)
					peerSet[peer.ID] = true
				}
			}

			// The local DHT updates for every two seconds
			time.Sleep(2 * time.Second)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	var userStream network.Stream = nil

	for {

		if userStream == nil {

			// Taking the input
			print("> Enter user index\n>")

			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading the input")
				continue
			}

			index, _ := strconv.ParseInt(input, 10, 32)

			if index > int64(len(peerArray)) {
				println("Please Enter a valid index!!")
				continue
			}

			logger.Info("Connecting to user")

			fmt.Println(peerArray[int(index)].ID)

			// Establish the Stream
			newStream, err := host.NewStream(ctx, peerArray[int(index)].ID, protocol.ID(config.ProtocolID))
			if err != nil {
				println("Error occured creating a stream!\n")
				continue
			}

			// Set the current stream
			userStream = newStream

			logger.Info("Connected to: ", peerArray[int(index)])
		}

		println("> Enter Message for the user")

		rw := bufio.NewReadWriter(bufio.NewReader(userStream), bufio.NewWriter(userStream))

		fmt.Print("> ")
		sendData, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin:", err)
			return
		}

		sendData = strings.TrimSpace(sendData)

		if sendData == "Cancel" {
			userStream.Close()
			userStream = nil
			continue
		}

		// Send the message
		_, err = rw.WriteString(fmt.Sprintf("> Message from: %s => %s\n", host.ID().String(), sendData))
		if err != nil {
			fmt.Println("Error writing to buffer:", err)
			continue
		}

		// Flush the errors
		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer:", err)
			continue
		}
	}

}

/*

Gracefully handle the exit of the users

*/
