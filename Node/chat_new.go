package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"os"
	"time"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("rendezvous")

type Message struct {
	Sender     string `json:"sender"`
	Message_Id int32  `json:"m_id"`
	Content    string `json:"content"`
}

func (m *Message) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Sender     string `json:"sender"`
		Message_Id int32  `json:"m_id"`
		Content    string `json:"content"`
	}{
		Sender:     m.Sender,
		Message_Id: m.Message_Id,
		Content:    m.Content,
	})
}

func (m *Message) UnmarshalJSON(data []byte) error {
	aux := &struct {
		Sender     string `json:"sender"`
		Message_Id int32  `json:"m_id"`
		Content    string `json:"content"`
	}{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	m.Sender = aux.Sender
	m.Message_Id = aux.Message_Id
	m.Content = aux.Content
	return nil
}

// Set the user
var User host.Host

// Make the message database
var m_id int32 = 1
var database map[string]map[int32]string = map[string]map[int32]string{}

// Maintain a set of neighbors
var peerArray []peer.AddrInfo = []peer.AddrInfo{}
var peerSet map[peer.ID]bool = map[peer.ID]bool{}

func gossipProtocol(stream network.Stream) {

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	go gossipExecute(rw, stream)

}

func gossipExecute(rw *bufio.ReadWriter, strm network.Stream) {
	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("User Went Offline:", err)
			return
		}

		if line == "" || line == "\n" {
			continue
		}

		// Deserialize the JSON object
		var message Message
		err = json.Unmarshal([]byte(line), &message)
		if err != nil {
			fmt.Println("Failed to parse JSON:", err)
			continue
		}

		// Check Data base
		if _, exists := database[message.Sender][message.Message_Id]; exists {
			continue
		}

		// Add the message to the database
		database[message.Sender][message.Message_Id] = message.Content

		// For each neighbor
		for _, peer := range peerArray {

			// Skip the sender since he already knows
			if peer.ID == strm.Conn().RemotePeer() {
				continue
			}

			// Create a new stream for the peer
			stream, err := User.NewStream(context.Background(), peer.ID, "/gossip/1.0.0")
			if err != nil {
				continue
			}

			// Create a buffered writer for the stream
			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			// Serialize the message to JSON
			data, err := json.Marshal(message)
			if err != nil {
				fmt.Println("Failed to serialize message:", err)
				stream.Close()
				continue
			}

			// Send the serialized message
			_, err = rw.WriteString(string(data) + "\n")
			if err != nil {
				fmt.Println("Failed to send message to peer:", peer.ID, err)
				stream.Close()
				continue
			}

			// Flush the buffer to ensure data is sent
			err = rw.Flush()
			if err != nil {
				fmt.Println("Failed to flush data to peer:", peer.ID, err)
				stream.Close()
				continue
			}

			stream.Close()
		}
	}
}

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

	// Setting a handlers
	host.SetStreamHandler(protocol.ID(config.ProtocolID+"/message"), messageProtocol)
	host.SetStreamHandler(protocol.ID(config.ProtocolID+"/gossip"), gossipProtocol)

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
			newStream, err := host.NewStream(ctx, peerArray[int(index)].ID, protocol.ID(config.ProtocolID+"/message"))
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
