## This is a demonstration of a p2p messaging application with node discovery and messaging. Made using go-libp2p

TODO:

-> Gracefully handle shutting down of node

    => Notify the network about shutdown so that bootstraps can update DHT
    => clean up of any resources (Trivial as Go has an in built garbage collector)

-> Instructions

    => go mod init Messenger
    => go get github.com/libp2p/go-libp2p
    => go get github.com/libp2p/go-libp2p-kad-dht