package gw

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

type LocalCertConfig struct {
	Logger       logging.Logger
	DB           db.Database
	VaultService *vault.Service
	Hostname     string
	WorkspaceID  string
}

func generateLocalCertificate(ctx context.Context, cfg LocalCertConfig) error {
	logger := cfg.Logger
	logger.Info("Checking for existing local certificate", "hostname", cfg.Hostname)

	// Check if certificate already exists in database
	existing, err := db.Query.FindCertificateByHostname(ctx, cfg.DB.RO(), cfg.Hostname)

	if err != nil && !db.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing certificate: %w", err)
	}

	// If we found an existing certificate, use it
	if err == nil && existing.Certificate != "" && existing.EncryptedPrivateKey != "" {
		logger.Info("Using existing local certificate", "hostname", cfg.Hostname)
		return nil
	}

	logger.Info("Generating self-signed wildcard certificate", "hostname", cfg.Hostname)

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	//nolint: exhaustruct
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		//nolint: exhaustruct
		Subject: pkix.Name{
			Organization: []string{"Unkey"},
			Country:      []string{"US"},
			CommonName:   cfg.Hostname,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames: []string{
			cfg.Hostname,
			"unkey.local",
		},
	}

	// Generate certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Headers: map[string]string{},
		Type:    "CERTIFICATE",
		Bytes:   certBytes,
	})

	// Encode private key to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Headers: map[string]string{},
		Type:    "RSA PRIVATE KEY",
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Save to files for backup and user trust installation
	certDir := "./certs"
	if err = os.MkdirAll(certDir, 0o755); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	certFile := fmt.Sprintf("%s/unkey.local.crt", certDir)
	keyFile := fmt.Sprintf("%s/unkey.local.key", certDir)

	if err = os.WriteFile(certFile, certPEM, 0o600); err != nil {
		return fmt.Errorf("failed to write certificate file: %w", err)
	}

	if err = os.WriteFile(keyFile, privateKeyPEM, 0o600); err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}

	// Encrypt the private key
	encryptResp, err := cfg.VaultService.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: "unkey",
		Data:    string(privateKeyPEM),
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// Insert certificate into database
	now := time.Now().UnixMilli()
	err = db.Query.InsertCertificate(ctx, cfg.DB.RW(), db.InsertCertificateParams{
		WorkspaceID:         cfg.WorkspaceID,
		Hostname:            cfg.Hostname,
		Certificate:         string(certPEM),
		EncryptedPrivateKey: encryptResp.GetEncrypted(),
		CreatedAt:           now,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
	})
	if err != nil {
		return fmt.Errorf("failed to insert certificate: %w", err)
	}

	fmt.Println("\n================================================================================")
	fmt.Println("🔐 LOCAL CERTIFICATE GENERATED")
	fmt.Println("================================================================================")
	fmt.Printf("\nCertificate file: %s\n", certFile)
	fmt.Println("\nTo trust this certificate in your browser and system:")
	fmt.Println("\n  macOS:")
	fmt.Printf("    sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", certFile)
	fmt.Println("\n  Linux:")
	fmt.Printf("    sudo cp %s /usr/local/share/ca-certificates/unkey-local.crt\n", certFile)
	fmt.Println("    sudo update-ca-certificates")
	fmt.Println("\n  Windows:")
	fmt.Printf("    certlm.msc -> Trusted Root Certification Authorities -> Import %s\n", certFile)
	fmt.Println("\n  Chrome/Chromium (if system trust doesn't work):")
	fmt.Println("    Settings -> Privacy and Security -> Manage Certificates -> Authorities -> Import")
	fmt.Printf("    Then import: %s\n", certFile)
	fmt.Println("\n================================================================================")

	return nil
}
