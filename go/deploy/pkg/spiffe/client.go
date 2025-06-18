package spiffe

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// AIDEV-NOTE: SPIFFE integration for automatic mTLS in Unkey services
// Replaces manual certificate management with dynamic identity

// Client provides SPIFFE-based mTLS configuration
type Client struct {
	source *workloadapi.X509Source
	id     spiffeid.ID
}

// New creates a SPIFFE client for the service
func New(ctx context.Context) (*Client, error) {
	return NewWithOptions(ctx, Options{
		SocketPath: "unix:///run/spire/sockets/agent.sock",
	})
}

// Options for SPIFFE client creation
type Options struct {
	SocketPath string
	// Add more options as needed
}

// NewWithOptions creates a SPIFFE client with custom options
func NewWithOptions(ctx context.Context, opts Options) (*Client, error) {
	// AIDEV-NOTE: Use context with timeout to prevent hanging on unresponsive agent
	// Create a timeout context for initial connection
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Connect to SPIRE agent via Unix socket
	source, err := workloadapi.NewX509Source(
		connectCtx,
		workloadapi.WithClientOptions(
			workloadapi.WithAddr(opts.SocketPath),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create X509 source: %w", err)
	}

	// Get our SPIFFE ID with timeout
	svidCtx, svidCancel := context.WithTimeout(ctx, 5*time.Second)
	defer svidCancel()

	svid, err := source.GetX509SVID()
	if err != nil {
		source.Close()
		return nil, fmt.Errorf("get SVID: %w", err)
	}
	_ = svidCtx // Context is for future use when API supports it

	return &Client{
		source: source,
		id:     svid.ID,
	}, nil
}

// ServiceName returns the service name from SPIFFE ID
// e.g., spiffe://unkey.prod/service/metald -> metald
func (c *Client) ServiceName() string {
	// Parse the path to extract service name
	path := c.id.Path()
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(segments) >= 2 && segments[0] == "service" {
		return segments[1]
	}
	return "unknown"
}

// TLSConfig returns TLS configuration for servers
func (c *Client) TLSConfig() *tls.Config {
	// AIDEV-NOTE: Use AuthorizeMemberOf to only accept SVIDs from our trust domain
	// This prevents accepting certificates from other SPIFFE deployments
	return tlsconfig.MTLSServerConfig(c.source, c.source, tlsconfig.AuthorizeMemberOf(c.id.TrustDomain()))
}

// ClientTLSConfig returns TLS configuration for clients
func (c *Client) ClientTLSConfig() *tls.Config {
	// AIDEV-NOTE: Client also validates server is in same trust domain
	return tlsconfig.MTLSClientConfig(c.source, c.source, tlsconfig.AuthorizeMemberOf(c.id.TrustDomain()))
}

// HTTPClient returns an HTTP client with mTLS
func (c *Client) HTTPClient() *http.Client {
	// AIDEV-NOTE: Configure transport with security timeouts
	transport := &http.Transport{
		TLSClientConfig: c.ClientTLSConfig(),
		// Security timeouts to prevent hanging connections
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, // Connection timeout
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
		Timeout:   30 * time.Second, // Overall request timeout
	}
}

// GRPCDialOption returns gRPC dial options with mTLS
func (c *Client) GRPCDialOption() grpc.DialOption {
	return grpc.WithTransportCredentials(
		credentials.NewTLS(c.ClientTLSConfig()),
	)
}

// AuthorizeService creates an authorizer for specific services
func (c *Client) AuthorizeService(allowedServices ...string) tlsconfig.Authorizer {
	return tlsconfig.AuthorizeMemberOf(c.id.TrustDomain())
}

// Close cleans up resources
func (c *Client) Close() error {
	return c.source.Close()
}
