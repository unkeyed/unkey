package acme

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func privateKeyToString(privateKey *ecdsa.PrivateKey) (string, error) {
	// Marshal the private key to DER format
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Encode to PEM format
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Headers: map[string]string{},
		Type:    "EC PRIVATE KEY",
		Bytes:   privKeyBytes,
	})

	return string(privKeyPEM), nil
}

func stringToPrivateKey(pemString string) (*ecdsa.PrivateKey, error) {
	// Decode PEM format
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Parse the EC private key
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EC private key: %w", err)
	}

	return privateKey, nil
}
