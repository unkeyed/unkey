package tls

import (
	"crypto/tls"
	"os"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Config is a type alias for the standard library's tls.Config.
//
// It is reexported here to provide a cleaner import path when used within the application.
// This allows consumers to use this package's functions without also importing crypto/tls.
type Config = tls.Config

// New creates a TLS configuration with the provided certificate and key data.
// It validates the input data, parses the certificate and key, and returns
// a TLS configuration ready to use with an HTTP server.
//
// The certPEMBlock parameter should contain PEM-encoded certificate data.
// The keyPEMBlock parameter should contain the PEM-encoded private key data.
//
// New returns a properly configured *tls.Config with secure defaults:
//   - TLS 1.3 or higher is enforced
//   - The provided certificate is configured for use
//
// If validation fails or the certificate cannot be parsed, New returns a descriptive error.
// Common error cases include empty certificate or key data, or malformed PEM content.
//
// Edge Cases:
//   - Passing nil for either parameter will result in a validation error
//   - Certificate chains (with intermediate certs) are supported, but must be properly ordered
//   - RSA, ECDSA, and ED25519 keys are supported, following Go's standard library support
//   - The function does not verify certificate expiration or validity
//   - If multiple certificates are needed, you'll need to modify the returned config
//
// Example:
//
//	// Load certificate and key data from memory
//	certPEM := []byte("-----BEGIN CERTIFICATE-----\n...")
//	keyPEM := []byte("-----BEGIN PRIVATE KEY-----\n...")
//
//	tlsConfig, err := tls.New(certPEM, keyPEM)
//	if err != nil {
//	    log.Fatalf("Failed to create TLS config: %v", err)
//	}
//
//	// Use with http.Server
//	server := &http.Server{
//	    Addr:      ":443",
//	    TLSConfig: tlsConfig,
//	}
//
// This function is commonly used when certificates are stored in memory or retrieved
// from a secrets management system rather than from files.
//
// See [NewFromFiles] for loading certificates directly from the filesystem.
func New(certPEMBlock, keyPEMBlock []byte) (*tls.Config, error) {
	err := assert.All(
		assert.NotEmpty(certPEMBlock, "TLS certificate must not be empty"),
		assert.NotEmpty(keyPEMBlock, "TLS key must not be empty"),
	)
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to parse TLS certificate"), fault.Public("Invalid certificate or key format"))
	}

	// nolint:exhaustruct
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// NewFromFiles loads TLS certificate and key from the specified files and creates
// a TLS configuration ready to use with an HTTP server.
//
// The certFile parameter should be the path to a PEM-encoded certificate file.
// The keyFile parameter should be the path to a PEM-encoded private key file.
//
// The function reads both files, verifies their content, and creates a properly
// configured TLS configuration with secure defaults (TLS 1.3+).
//
// If the files cannot be read or contain invalid certificate/key data, the function
// returns a descriptive error. Common error cases include non-existent files, permission
// issues, or malformed certificate/key content.
//
// Example:
//
//	// Load certificate and key from files
//	tlsConfig, err := tls.NewFromFiles("/path/to/server.crt", "/path/to/server.key")
//	if err != nil {
//	    log.Fatalf("Failed to load TLS certificates: %v", err)
//	}
//
//	// Use with http.Server
//	server := &http.Server{
//	    Addr:      ":443",
//	    TLSConfig: tlsConfig,
//	}
//
//	// Start HTTPS server
//	if err := server.ListenAndServeTLS("", ""); err != nil {
//	    log.Fatalf("Failed to start HTTPS server: %v", err)
//	}
//
// This function is commonly used in applications that store certificates and keys
// as files, such as command-line tools or services with configuration files.
//
// The file permissions should be properly set to restrict access to the private key
// (typically 0600).
//
// Edge Cases:
//   - If files don't exist or aren't readable, a descriptive error is returned
//   - Very large files (>1GB) may cause memory issues as they're fully read into memory
//   - If certificate and key don't match, an error will be returned during parsing
//   - Absolute and relative paths are both supported
//   - Symlinks are followed
//
// See [New] for creating a TLS configuration from in-memory certificate data.
func NewFromFiles(certFile, keyFile string) (*tls.Config, error) {
	certPEMBlock, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to read certificate file"))
	}

	keyPEMBlock, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to read key file"))
	}

	return New(certPEMBlock, keyPEMBlock)
}
