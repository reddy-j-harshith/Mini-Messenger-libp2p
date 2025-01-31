package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p/core/crypto"
)

func main() {
	// Generate an Ed25519 key pair
	privKey, _, err := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
	if err != nil {
		log.Fatal("Failed to generate key pair:", err)
	}

	// Extract the raw private key
	privBytes, err := privKey.Raw()
	if err != nil {
		log.Fatal("Failed to get raw private key:", err)
	}
	fmt.Println("Private Key (Hex):", hex.EncodeToString(privBytes))

	// Extract the raw public key
	pubKey := privKey.GetPublic()
	pubBytes, err := pubKey.Raw()
	if err != nil {
		log.Fatal("Failed to get raw public key:", err)
	}
	fmt.Println("Public Key (Hex):", hex.EncodeToString(pubBytes))
}
