package main

import (
	"flag"
	"strings"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	maddr "github.com/multiformats/go-multiaddr"
)

type AddrList []maddr.Multiaddr

// Join all the adddress in the multi address list into a single string separated by commas
func (addrList *AddrList) String() string {
	strs := make([]string, len(*addrList))
	for i, addr := range *addrList {
		strs[i] = addr.String()
	}

	return strings.Join(strs, ",")
}

// Obtain a MultiAddr Object and append to the addrList
func (addrList *AddrList) Set(value string) error {
	addr, err := maddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*addrList = append(*addrList, addr)
	return nil
}

// Provide an array of multi address Strings
func StringsToAddrs(addrString []string) (maddrs []maddr.Multiaddr, err error) {
	for _, addrString := range addrString {
		addr, err := maddr.NewMultiaddr(addrString)
		if err != nil {
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}

type Config struct {
	RendezvousString string
	BootstrapPeers   AddrList
	ListenAddresses  AddrList
	ProtocolID       string
}

func ParseFlags() (Config, error) {
	config := Config{}

	// flag.StringVar(pointer, name, defaultValue, usage)
	flag.StringVar(&config.RendezvousString, "rendezvous", "meet me here",
		"Unique string to identify group of nodes. Share this with your friends to let them connect with you")
	flag.Var(&config.BootstrapPeers, "peer", "Adds a peer multiaddress to the bootstrap list")
	flag.Var(&config.ListenAddresses, "listen", "Adds a multiaddress to the listen list")
	flag.StringVar(&config.ProtocolID, "pid", "/chat/1.0.0", "Sets a protocol id for stream headers")
	flag.Parse()

	if len(config.BootstrapPeers) == 0 {
		config.BootstrapPeers = dht.DefaultBootstrapPeers
	}

	return config, nil
}
