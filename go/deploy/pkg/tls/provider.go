// Package tls provides TLS configuration abstraction with support for multiple backends.
//
// The package supports three TLS modes:
//   - disabled: No TLS (deprecated, for testing only)
//   - file: Traditional certificate files
//   - spiffe: SPIFFE/SPIRE identity (recommended default)
//
// Example usage:
//
//	provider, err := tls.NewProvider(ctx, tls.Config{
//		Mode:     tls.ModeFile,
//		CertFile: "/path/to/cert.pem",
//		KeyFile:  "/path/to/key.pem",
//	})
//	if err != nil {
//		return err
//	}
//	defer provider.Close()
//
//	client := provider.HTTPClient()
//	grpcConn, err := grpc.Dial("server:443", provider.GRPCDialOption())
package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/deploy/pkg/spiffe"
)

// Provider abstracts TLS configuration source across different backends.
type Provider interface {
	// ServerTLSConfig returns server TLS configuration.
	// Returns nil config for disabled mode.
	ServerTLSConfig() (*tls.Config, error)

	// ClientTLSConfig returns client TLS configuration.
	// Returns nil config for disabled mode.
	ClientTLSConfig() (*tls.Config, error)

	// HTTPClient returns an HTTP client configured with appropriate TLS settings.
	HTTPClient() *http.Client

	// Close releases any resources held by the provider.
	Close() error
}

// Mode defines the TLS configuration mode.
type Mode string

const (
	// ModeDisabled provides no TLS encryption (development only).
	ModeDisabled Mode = "disabled"
	// ModeFile uses traditional X.509 certificate files.
	ModeFile Mode = "file"
	// ModeSPIFFE uses SPIFFE/SPIRE for identity-based mTLS.
	ModeSPIFFE Mode = "spiffe"
)

// Config configures TLS provider creation.
type Config struct {
	// Mode specifies the TLS configuration mode.
	Mode Mode `json:"mode" default:"spiffe"`

	// CertFile is the path to the certificate file (file mode).
	CertFile string `json:"cert_file,omitempty"`
	// KeyFile is the path to the private key file (file mode).
	KeyFile string `json:"-"`
	// CAFile is the path to the CA certificate file for mutual TLS (file mode).
	CAFile string `json:"ca_file,omitempty"`

	// SPIFFESocketPath is the SPIRE agent socket path (spiffe mode).
	SPIFFESocketPath string `json:"spiffe_socket_path,omitempty"`
	// SPIFFETimeout configures SPIFFE connection timeout (spiffe mode).
	SPIFFETimeout string `json:"spiffe_timeout,omitempty"`

	// EnableCertCaching enables certificate caching for performance (file mode).
	EnableCertCaching bool `json:"enable_cert_caching,omitempty"`
	// CertCacheTTL sets certificate cache duration (defaults to 5s).
	CertCacheTTL time.Duration `json:"cert_cache_ttl,omitempty"`
}

// NewProvider creates a TLS provider based on the configuration mode.
// For SPIFFE mode, it falls back to disabled mode if the agent socket is unavailable.
func NewProvider(ctx context.Context, cfg Config) (Provider, error) {
	switch cfg.Mode {
	case ModeDisabled:
		return &disabledProvider{}, nil

	case ModeFile:
		if cfg.EnableCertCaching {
			cacheTTL := cfg.CertCacheTTL
			if cacheTTL == 0 {
				cacheTTL = 5 * time.Second
			}

			return newCachedFileProvider(cfg, cacheTTL)
		}

		return newFileProvider(cfg)

	case ModeSPIFFE:
		if _, err := os.Stat(cfg.SPIFFESocketPath); os.IsNotExist(err) {
			log.Printf("SPIRE agent socket not found at %s", cfg.SPIFFESocketPath)
			return &disabledProvider{}, nil
		}

		return newSPIFFEProvider(ctx, cfg)

	default:
		return nil, fmt.Errorf("unknown TLS mode: %s", cfg.Mode)
	}
}

// disabledProvider provides no TLS (plain HTTP)
type disabledProvider struct{}

func (p *disabledProvider) ServerTLSConfig() (*tls.Config, error) {
	return nil, nil //nolint: all
}

func (p *disabledProvider) ClientTLSConfig() (*tls.Config, error) {
	return nil, nil //nolint: all
}

func (p *disabledProvider) HTTPClient() *http.Client {
	return &http.Client{}
}

func (p *disabledProvider) Close() error {
	return nil
}

// fileProvider uses traditional certificate files
type fileProvider struct {
	certFile string
	keyFile  string
	caFile   string
	// AIDEV-NOTE: Don't store tlsConfig - create it dynamically to support rotation
}

func newFileProvider(cfg Config) (Provider, error) {
	// AIDEV-NOTE: Validate file paths to prevent directory traversal attacks
	if err := validateFilePath(cfg.CertFile); err != nil {
		return nil, fmt.Errorf("invalid cert file path: %w", err)
	}
	if err := validateFilePath(cfg.KeyFile); err != nil {
		return nil, fmt.Errorf("invalid key file path: %w", err)
	}
	if err := validateFilePath(cfg.CAFile); err != nil {
		return nil, fmt.Errorf("invalid CA file path: %w", err)
	}

	p := &fileProvider{
		certFile: cfg.CertFile,
		keyFile:  cfg.KeyFile,
		caFile:   cfg.CAFile,
	}

	// Validate that we can load certificates at startup
	if p.certFile != "" && p.keyFile != "" {
		_, err := p.loadTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("validate certificates: %w", err)
		}
	}

	return p, nil
}

// validateFilePath validates file paths to prevent directory traversal attacks.
func validateFilePath(path string) error {
	if path == "" {
		return nil
	}

	cleanPath := filepath.Clean(path)

	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal: %s", path)
	}

	if strings.HasPrefix(cleanPath, "/etc/") ||
		strings.HasPrefix(cleanPath, "/usr/") ||
		strings.HasPrefix(cleanPath, "/var/") {
		if !strings.HasPrefix(cleanPath, "/etc/ssl/") &&
			!strings.HasPrefix(cleanPath, "/etc/pki/") &&
			!strings.HasPrefix(cleanPath, "/etc/unkey/") &&
			!strings.HasPrefix(cleanPath, "/var/lib/unkey/") {
			return fmt.Errorf("path points to system directory: %s", path)
		}
	}

	return nil
}

// loadTLSConfig loads certificates from disk to support rotation.
func (p *fileProvider) loadTLSConfig() (*tls.Config, error) {
	if p.certFile == "" || p.keyFile == "" {
		return nil, nil //nolint:all
	}

	//nolint:exhaustruct
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
			if err != nil {
				return nil, fmt.Errorf("load cert/key: %w", err)
			}
			return &cert, nil
		},
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	if p.caFile != "" {
		caCert, err := os.ReadFile(p.caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("invalid CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig.Clone(), nil
}

func (p *fileProvider) ServerTLSConfig() (*tls.Config, error) {
	return p.loadTLSConfig()
}

func (p *fileProvider) ClientTLSConfig() (*tls.Config, error) {
	return p.loadTLSConfig()
}

func (p *fileProvider) HTTPClient() *http.Client {
	tlsConfig, _ := p.loadTLSConfig()
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		//nolint:exhaustruct // net.Dialer's zero values are intentional and recommended
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func (p *fileProvider) Close() error {
	return nil
}

// spiffeProvider uses SPIFFE/SPIRE
type spiffeProvider struct {
	client *spiffe.Client
}

func newSPIFFEProvider(ctx context.Context, cfg Config) (Provider, error) {
	socketPath := cfg.SPIFFESocketPath
	if socketPath == "" {
		socketPath = "/var/lib/spire/agent/agent.sock"
	}

	if !strings.HasPrefix(socketPath, "unix://") && !strings.HasPrefix(socketPath, "tcp://") {
		socketPath = "unix://" + socketPath
	}

	client, err := spiffe.NewWithOptions(ctx, spiffe.Options{
		SocketPath: socketPath,
	})
	if err != nil {
		return nil, fmt.Errorf("init SPIFFE: %w", err)
	}

	return &spiffeProvider{client: client}, nil
}

func (p *spiffeProvider) ServerTLSConfig() (*tls.Config, error) {
	return p.client.TLSConfig(), nil
}

func (p *spiffeProvider) ClientTLSConfig() (*tls.Config, error) {
	return p.client.ClientTLSConfig(), nil
}

func (p *spiffeProvider) HTTPClient() *http.Client {
	return p.client.HTTPClient()
}

func (p *spiffeProvider) Close() error {
	return p.client.Close()
}
