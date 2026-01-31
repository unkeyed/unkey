package main

import (
	"encoding/hex"
	"fmt"

	"github.com/unkeyed/unkey/pkg/encryption"
)

func main() {
	masterKey := "02a7a5b6c686c6c6f776f726c64"
	workspaceId := "ws_5ZUapZkYqhPpbkfL"
	testToken := "test_access_token_12345"

	fmt.Println("Testing encryption...")
	fmt.Println("MASTER_KEY length:", len(masterKey), "bytes")
	fmt.Println("MASTER_KEY bytes:", hex.EncodeToString([]byte(masterKey)))

	// Create encryption service
	enc, err := encryption.NewWorkspaceEncryption([]byte(masterKey))
	if err != nil {
		fmt.Printf("Error creating encryption: %v\n", err)
		return
	}

	// Encrypt
	encrypted, err := enc.EncryptToken(workspaceId, testToken)
	if err != nil {
		fmt.Printf("Error encrypting: %v\n", err)
		return
	}
	fmt.Println("Encrypted:", encrypted)

	// Decrypt
	decrypted, err := enc.DecryptToken(workspaceId, encrypted)
	if err != nil {
		fmt.Printf("Error decrypting: %v\n", err)
		return
	}
	fmt.Println("Decrypted:", decrypted)

	// Verify
	if decrypted == testToken {
		fmt.Println("✓ SUCCESS: Encryption/decryption works!")
	} else {
		fmt.Println("✗ FAILURE: Decrypted token doesn't match")
		fmt.Println("Expected:", testToken)
		fmt.Println("Got:", decrypted)
	}
}