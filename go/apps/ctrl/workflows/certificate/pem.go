package certificate

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"
)

// privateKeyToString converts an ECDSA private key to PEM-encoded string format.
//
// The key is marshaled to DER format and then encoded as PEM with the "EC PRIVATE KEY"
// block type. This format is compatible with standard TLS libraries and certificate
// authorities.
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

// stringToPrivateKey parses a PEM-encoded ECDSA private key from a string.
//
// Returns an error if the PEM block cannot be decoded or if the key format is invalid.
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

// getCertificateExpiry extracts the expiration time from a PEM-encoded certificate.
//
// Returns the NotAfter timestamp from the certificate, which indicates when the
// certificate expires and needs renewal.
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
