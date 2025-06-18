package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/deploy/pkg/spiffe"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// AIDEV-NOTE: Opt-in TLS provider that supports multiple backends
// Allows gradual migration from no-TLS → file-based TLS → SPIFFE

// Provider abstracts TLS configuration source
type Provider interface {
	// Server-side TLS config
	ServerTLSConfig() (*tls.Config, error)

	// Client-side TLS config
	ClientTLSConfig() (*tls.Config, error)

	// HTTP client with appropriate transport
	HTTPClient() *http.Client

	// gRPC dial options
	GRPCDialOption() grpc.DialOption

	// Cleanup resources
	Close() error
}

// Mode defines the TLS mode
type Mode string

const (
	ModeDisabled Mode = "disabled" // No TLS (development)
	ModeFile     Mode = "file"     // Traditional file-based certificates
	ModeSPIFFE   Mode = "spiffe"   // SPIFFE/SPIRE (when ready)
)

// Config for TLS provider
type Config struct {
	Mode Mode `json:"mode" default:"disabled"`

	// File-based TLS options
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"-"` // AIDEV-NOTE: Never serialize private key paths
	CAFile   string `json:"ca_file,omitempty"`

	// SPIFFE options
	SPIFFESocketPath string `json:"spiffe_socket_path,omitempty"`
	SPIFFETimeout    string `json:"spiffe_timeout,omitempty"`

	// Performance options
	EnableCertCaching bool          `json:"enable_cert_caching,omitempty"`
	CertCacheTTL      time.Duration `json:"cert_cache_ttl,omitempty"`
}

// NewProvider creates a TLS provider based on config
func NewProvider(ctx context.Context, cfg Config) (Provider, error) {
	switch cfg.Mode {
	case ModeDisabled:
		return &disabledProvider{}, nil

	case ModeFile:
		// AIDEV-NOTE: Use cached provider for better performance if enabled
		// Default cache TTL is 5 seconds - frequent enough for rotation but reduces I/O
		if cfg.EnableCertCaching {
			cacheTTL := cfg.CertCacheTTL
			if cacheTTL == 0 {
				cacheTTL = 5 * time.Second
			}
			return newCachedFileProvider(cfg, cacheTTL)
		}
		return newFileProvider(cfg)

	case ModeSPIFFE:
		// Check if SPIFFE is actually available
		if _, err := os.Stat(cfg.SPIFFESocketPath); os.IsNotExist(err) {
			// Fallback to disabled if SPIFFE not available
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
	return nil, nil
}

func (p *disabledProvider) ClientTLSConfig() (*tls.Config, error) {
	return nil, nil
}

func (p *disabledProvider) HTTPClient() *http.Client {
	return &http.Client{}
}

func (p *disabledProvider) GRPCDialOption() grpc.DialOption {
	return grpc.WithTransportCredentials(insecure.NewCredentials())
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

// validateFilePath ensures the path is safe and doesn't contain directory traversal
func validateFilePath(path string) error {
	if path == "" {
		return nil // Empty paths are allowed (optional files)
	}

	// Clean the path to resolve any .. or . elements
	cleanPath := filepath.Clean(path)

	// Ensure the path is absolute or relative without traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path contains directory traversal: %s", path)
	}

	// Additional validation: ensure path doesn't start with system directories
	if strings.HasPrefix(cleanPath, "/etc/") ||
		strings.HasPrefix(cleanPath, "/usr/") ||
		strings.HasPrefix(cleanPath, "/var/") {
		// Allow specific certificate directories
		if !strings.HasPrefix(cleanPath, "/etc/ssl/") &&
			!strings.HasPrefix(cleanPath, "/etc/pki/") &&
			!strings.HasPrefix(cleanPath, "/etc/unkey/") &&
			!strings.HasPrefix(cleanPath, "/var/lib/unkey/") {
			return fmt.Errorf("path points to system directory: %s", path)
		}
	}

	return nil
}

// loadTLSConfig loads certificates fresh from disk to support rotation
func (p *fileProvider) loadTLSConfig() (*tls.Config, error) {
	if p.certFile == "" || p.keyFile == "" {
		return nil, nil
	}

	// AIDEV-NOTE: GetCertificate callback loads cert/key on each connection
	// This enables certificate rotation without restart
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13, // AIDEV-NOTE: Enforce TLS 1.3 for best security
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
			if err != nil {
				return nil, fmt.Errorf("load cert/key: %w", err)
			}
			return &cert, nil
		},
		// AIDEV-NOTE: Restrict to secure cipher suites
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	// Load CA for mutual TLS if provided
	if p.caFile != "" {
		caCert, err := os.ReadFile(p.caFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("invalid CA certificate")
		}

		// Enable mutual TLS
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		tlsConfig.RootCAs = caCertPool
	}

	// AIDEV-NOTE: Clone the config to prevent caller mutations
	return tlsConfig.Clone(), nil
}

func (p *fileProvider) ServerTLSConfig() (*tls.Config, error) {
	return p.loadTLSConfig()
}

func (p *fileProvider) ClientTLSConfig() (*tls.Config, error) {
	return p.loadTLSConfig()
}

func (p *fileProvider) HTTPClient() *http.Client {
	// AIDEV-NOTE: Load TLS config when creating client to get latest certs
	tlsConfig, _ := p.loadTLSConfig()

	// Configure transport with security timeouts
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
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

func (p *fileProvider) GRPCDialOption() grpc.DialOption {
	tlsConfig, _ := p.loadTLSConfig()
	if tlsConfig != nil {
		return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	}
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}

func (p *fileProvider) Close() error {
	return nil
}

// spiffeProvider uses SPIFFE/SPIRE
type spiffeProvider struct {
	client *spiffe.Client
}

func newSPIFFEProvider(ctx context.Context, cfg Config) (Provider, error) {
	// Use configured socket path or default
	socketPath := cfg.SPIFFESocketPath
	if socketPath == "" {
		socketPath = "/run/spire/sockets/agent.sock"
	}

	// AIDEV-NOTE: Ensure socket path has unix:// scheme prefix
	// The SPIFFE library requires the URI scheme, but configs often omit it
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

func (p *spiffeProvider) GRPCDialOption() grpc.DialOption {
	return p.client.GRPCDialOption()
}

func (p *spiffeProvider) Close() error {
	return p.client.Close()
}
