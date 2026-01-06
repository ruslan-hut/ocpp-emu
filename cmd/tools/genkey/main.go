// Package main provides a utility to generate API keys and their hashes for config files.
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	// Generate random 32-byte key
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating random key: %v\n", err)
		os.Exit(1)
	}

	// Hex encode the key for use in requests
	keyStr := hex.EncodeToString(key)

	// Generate SHA-256 hash for config file
	hash := sha256.Sum256([]byte(keyStr))
	hashStr := "sha256:" + hex.EncodeToString(hash[:])

	fmt.Println("API Key (use this in X-API-Key header):")
	fmt.Println(keyStr)
	fmt.Println()
	fmt.Println("Key Hash (add this to config.yaml under api_keys.key_hash):")
	fmt.Println(hashStr)
}
