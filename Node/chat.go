package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"os"
	"time"

	"github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
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

var (
	// Set the user
	User host.Host

	// Local DHT
	kademliaDHT *dht.IpfsDHT

	// Make the message database
	m_id     int32                       = 1
	least    map[string]int32            = map[string]int32{}
	database map[string]map[int32]string = map[string]map[int32]string{}

	// Maintain a set of neighbors
	peerArray []peer.AddrInfo  = []peer.AddrInfo{}
	peerSet   map[peer.ID]bool = map[peer.ID]bool{}
)

func gossipProtocol(stream network.Stream) {

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	go gossipExecute(rw, stream)

}

var peerMutex sync.RWMutex

func gossipExecute(rw *bufio.ReadWriter, strm network.Stream) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic in gossipExecute:", r)
		}
	}()

	for {
		line, err := rw.ReadString('\n')
		if err != nil {
			return
		}

		if line == "" || line == "\n" {
			continue
		}

		var message Message
		err = json.Unmarshal([]byte(line), &message)
		if err != nil {
			fmt.Println("Failed to parse JSON:", err)
			continue
		}

		peerMutex.Lock()
		if small, exist := least[message.Sender]; !exist || message.Message_Id > small {
			least[message.Sender] = message.Message_Id
		} else {
			peerMutex.Unlock()
			continue
		}
		peerMutex.Unlock()

		// Database Access with Mutex
		peerMutex.Lock()
		if _, exists := database[message.Sender]; !exists {
			database[message.Sender] = make(map[int32]string)
			if _, exists := database[message.Sender][message.Message_Id]; !exists {
				database[message.Sender][message.Message_Id] = message.Content
			} else {
				peerMutex.Unlock()
				continue
			}
		}
		peerMutex.Unlock()

		fmt.Printf("\x1b[32m> Message Sent by: %s\n> Message: %s\n> Sent from %s\x1b[0m\n", message.Sender, message.Content, strm.Conn().RemotePeer())

		peerMutex.RLock()
		peers := peerArray
		peerMutex.RUnlock()

		for _, peer := range peers {
			if peer.ID == strm.Conn().RemotePeer() || peer.ID.String() == message.Sender {
				continue
			}

			stream, err := User.NewStream(context.Background(), peer.ID, protocol.ID("/chat/1.0.0/gossip"))
			if err != nil {
				continue
			}

			rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

			data, err := json.Marshal(message)
			if err != nil {
				fmt.Println("Failed to serialize message:", err)
				stream.Close()
				continue
			}

			_, err = rw.WriteString(string(data) + "\n")
			if err != nil {
				fmt.Println("Failed to send message to peer:", peer.ID, err)
				stream.Close()
				continue
			}

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
	fmt.Println("Enter the private key:")

	reader := bufio.NewReader(os.Stdin)
	privKeyString, _ := reader.ReadString('\n')

	privKeyString = strings.TrimSpace(privKeyString)

	privKeyBytes, _ := hex.DecodeString(privKeyString)

	privKey, _ := crypto.UnmarshalSecp256k1PrivateKey(privKeyBytes)

	// Creating the current node
	host, _ := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrs(
			[]multiaddr.Multiaddr(config.ListenAddresses)...,
		),
	)

	// Extract the raw public key
	pubKey := privKey.GetPublic()
	pubBytes, err := pubKey.Raw()
	if err != nil {
		logger.Fatal("Failed to get raw public key:", err)
	}
	fmt.Println("Public Key (Hex):", hex.EncodeToString(pubBytes)) // Extract the raw public key

	nodeID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		panic(err)
	}
	fmt.Println("Computed Node ID:", nodeID.String())

	// Assigning this user globally for the node
	User = host

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
	kademliaDHT, err = dht.New(ctx, host, dht.BootstrapPeers(bootstrapPeers...), dht.ProtocolPrefix("/custom-dht"), dht.BucketSize(5))
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
					logger.Info("Connected to peer:", peer.ID.String())
					peerArray = append(peerArray, peer)
					peerSet[peer.ID] = true
				}
			}

			// The local DHT updates for every two seconds
			time.Sleep(2 * time.Second)
		}
	}()

	var userStream network.Stream = nil

	for {
		// Mode Selection: Direct Message or Gossip Mode
		print("> Select Mode (1: Direct Message, 2: Gossip Mode, 3: Exit)\n> ")
		mode, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading the input")
			continue
		}

		mode = strings.TrimSpace(mode)

		// Exit the program if selected
		if mode == "3" {
			fmt.Println("Exiting...")
			break
		}

		// Handle Direct Message Mode
		if mode == "1" {

			fmt.Print(kademliaDHT)

			// Direct Message logic (same as before)
			for {
				if userStream == nil {

					// Taking the input to select user index
					print("> Enter user index\n>")
					input, err := reader.ReadString('\n')
					if err != nil {
						fmt.Println("Error reading the input")
						continue
					}

					input = strings.TrimSpace(input)

					index, _ := strconv.ParseInt(input, 10, 64)

					fmt.Println(index)

					if index >= int64(len(peerArray)) {
						println("Please Enter a valid index!!")
						continue
					}

					logger.Info("Connecting to user")

					// Establish the Stream
					newStream, err := host.NewStream(ctx, peerArray[index].ID, protocol.ID(config.ProtocolID+"/message"))
					if err != nil {
						println("Error occurred creating a stream!\n")
						continue
					}

					// Set the current stream
					userStream = newStream

					logger.Info("Connected to: ", peerArray[index].ID.String())
				}

				println("> Enter Message for the user (type 'Cancel' to go back to mode selection)")

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
					break
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

		// Handle Gossip Mode
		if mode == "2" {
			// Taking the input to send a message in gossip mode
			println("> Enter Message for Gossip (type 'Cancel' to go back to mode selection)")

			sendData, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading from stdin:", err)
				continue
			}

			sendData = strings.TrimSpace(sendData)

			if sendData == "Cancel" {
				break
			}

			// Increment global m_id for Gossip messages
			m_id++

			// Create a new Message
			message := Message{
				Sender:     host.ID().String(),
				Message_Id: m_id,
				Content:    sendData,
			}

			// Send the message to all connected peers (Gossip)
			for _, peer := range peerArray {
				// Create a new stream for the peer
				stream, err := host.NewStream(ctx, peer.ID, protocol.ID(config.ProtocolID+"/gossip"))
				if err != nil {
					fmt.Println("Failed to create stream with peer:", peer.ID)
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
}
