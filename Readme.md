## This is a mini demonstration of a p2p messaging application with node discovery and messaging. Made using go-libp2p

TODO:

-> Gracefully handle shutting down of node

    => Notify the network about shutdown so that bootstraps can update DHT
    => clean up of any resources (Trivial as Go has an in built garbage collector)