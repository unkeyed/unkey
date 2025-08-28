package acme

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"
)

func privateKeyToString(privateKey *ecdsa.PrivateKey) (string, error) {
	// Marshal the private key to DER format
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Encode to PEM format
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privKeyBytes,
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

func getCertificateExpiry(certPEM string) (time.Time, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return time.Time{}, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert.NotAfter, nil
}
