package main

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

	_ "github.com/go-sql-driver/mysql"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	pdb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

var Cmd = &cli.Command{
	Name:  "local-tls",
	Usage: "Manage self-signed TLS certificates for local development",
	Description: `Generate and install self-signed wildcard certificates for *.unkey.local
This command creates certificates and stores them in the database with encrypted private keys,
allowing the gateway to serve HTTPS traffic for local development.`,
	Commands: []*cli.Command{
		{
			Name:  "generate",
			Usage: "Generate and install self-signed certificate for *.unkey.local",
			Flags: []cli.Flag{
				cli.String("mysql-dsn", "MySQL connection string for partition database",
					cli.Default("unkey:password@tcp(localhost:3306)/partition_001?parseTime=true&interpolateParams=true"),
					cli.EnvVar("UNKEY_DATABASE_PARTITION")),

				cli.String("vault-s3-url", "S3 URL for vault service",
					cli.Default("http://localhost:3902"),
					cli.EnvVar("UNKEY_VAULT_S3_URL")),

				cli.String("vault-s3-bucket", "S3 bucket for vault service",
					cli.Default("acme-vault"),
					cli.EnvVar("UNKEY_VAULT_S3_BUCKET")),

				cli.String("vault-s3-access-key", "S3 access key ID",
					cli.Default("minio_root_user"),
					cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_ID")),

				cli.String("vault-s3-secret", "S3 access key secret",
					cli.Default("minio_root_password"),
					cli.EnvVar("UNKEY_VAULT_S3_ACCESS_KEY_SECRET")),

				cli.String("vault-master-keys", "Vault master keys",
					cli.Default("Ch9rZWtfMmdqMFBJdVhac1NSa0ZhNE5mOWlLSnBHenFPENTt7an5MRogENt9Si6wms4pQ2XIvqNSIgNpaBenJmXgcInhu6Nfv2U="),
					cli.EnvVar("UNKEY_VAULT_MASTER_KEYS")),

				cli.String("workspace-id", "Workspace ID for the certificate",
					cli.Default("unkey")),

				cli.String("hostname", "Hostname for the certificate",
					cli.Default("*.unkey.local")),

				cli.Int("days", "Certificate validity in days",
					cli.Default(365)),

				cli.String("cert-dir", "Directory to save certificate files",
					cli.Default("./certs")),
			},
			Action: generateCertificate,
		},
	},
}

func generateCertificate(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("Generating self-signed wildcard certificate...")

	hostname := cmd.String("hostname")
	days := cmd.Int("days")
	certDir := cmd.String("cert-dir")

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Unkey Local Development"},
			Country:      []string{"US"},
			CommonName:   "unkey.local",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Duration(days) * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames: []string{
			hostname,
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
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// Encode private key to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	fmt.Println("Certificate generated successfully!")

	// Save to files for backup
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	certFile := fmt.Sprintf("%s/unkey.local.crt", certDir)
	keyFile := fmt.Sprintf("%s/unkey.local.key", certDir)

	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write certificate file: %w", err)
	}

	if err := os.WriteFile(keyFile, privateKeyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}

	fmt.Printf("Certificate files saved to %s/\n", certDir)

	// Store in database
	return storeCertificateInDB(cmd, string(certPEM), string(privateKeyPEM), certFile)
}

func storeCertificateInDB(cmd *cli.Command, certPEM, privateKeyPEM, certFile string) error {
	fmt.Println("\nStoring certificate in database...")

	logger := logging.New()

	// Connect to MySQL using db.New
	partitionDB, err := db.New(db.Config{
		PrimaryDSN: cmd.String("mysql-dsn"),
		Logger:     logging.New(),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize storage for vault
	s3Storage, err := storage.NewS3(storage.S3Config{
		S3URL:             cmd.String("vault-s3-url"),
		S3Bucket:          cmd.String("vault-s3-bucket"),
		S3AccessKeyID:     cmd.String("vault-s3-access-key"),
		S3AccessKeySecret: cmd.String("vault-s3-secret"),
		Logger:            logger,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize S3 storage: %w", err)
	}

	// Initialize vault service
	vaultService, err := vault.New(vault.Config{
		Logger:     logger,
		Storage:    s3Storage,
		MasterKeys: []string{cmd.String("vault-master-keys")},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize vault service: %w", err)
	}

	bgCtx := context.Background()

	// Encrypt the private key
	encryptResp, err := vaultService.Encrypt(bgCtx, &vaultv1.EncryptRequest{
		Keyring: "unkey",
		Data:    privateKeyPEM,
	})
	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// Insert certificate into database
	workspaceID := cmd.String("workspace-id")
	hostname := cmd.String("hostname")
	now := time.Now().UnixMilli()

	err = pdb.Query.InsertCertificate(bgCtx, partitionDB.RW(), pdb.InsertCertificateParams{
		WorkspaceID:         workspaceID,
		Hostname:            hostname,
		Certificate:         certPEM,
		EncryptedPrivateKey: encryptResp.Encrypted,
		CreatedAt:           now,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
	})

	if err != nil {
		return fmt.Errorf("failed to insert certificate: %w", err)
	}

	fmt.Println("Certificate successfully stored in database!")
	fmt.Printf("\nSetup complete! The gateway can now use the certificate for %s\n", hostname)

	// Print instructions for trusting the certificate
	fmt.Println("\n🔐 To trust this certificate in your browser and system:")
	fmt.Printf("\n  Certificate file: %s\n", certFile)
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

	return nil
}

func main() {
	app := &cli.Command{
		Name:        "localtls",
		Usage:       "Run localtls",
		Description: `LocalTLS CLI – run and administer LocalTLS services.`,
		Commands: []*cli.Command{
			Cmd,
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
