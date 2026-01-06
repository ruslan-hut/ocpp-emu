// Package main provides a utility to generate bcrypt password hashes for config files.
package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: hashpw <password>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Generates a bcrypt hash for the given password.")
		fmt.Fprintln(os.Stderr, "Use the output in config.yaml under users.password_hash")
		os.Exit(1)
	}

	password := os.Args[1]

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(hash))
}
