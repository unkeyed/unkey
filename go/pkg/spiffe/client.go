// Package spiffe provides SPIFFE-based mTLS configuration for HTTP clients.
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
)

// Client provides SPIFFE-based mTLS configuration using X.509 SVIDs.
type Client struct {
	source *workloadapi.X509Source
	id     spiffeid.ID
}

// Options configures SPIFFE client creation.
type Options struct {
	// SocketPath is the SPIRE agent socket path.
	SocketPath string
}

// New creates a SPIFFE client using the default SPIRE agent socket.
// It connects to unix:///var/lib/spire/agent/agent.sock with a 30-second timeout.
func New(ctx context.Context) (*Client, error) {
	return NewWithOptions(ctx, Options{
		SocketPath: "unix:///var/lib/spire/agent/agent.sock",
	})
}

// NewWithOptions creates a SPIFFE client with custom options.
// It establishes a connection to the SPIRE agent and retrieves the workload SVID.
// NewWithOptions returns an error if the agent is unreachable or SVID retrieval fails.
func NewWithOptions(ctx context.Context, opts Options) (*Client, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	source, err := workloadapi.NewX509Source(
		connectCtx,
		workloadapi.WithClientOptions(
			workloadapi.WithAddr(opts.SocketPath),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create X509 source: %w", err)
	}

	svidCtx, svidCancel := context.WithTimeout(ctx, 5*time.Second)
	defer svidCancel()

	svid, err := source.GetX509SVID()
	if err != nil {
		source.Close()
		return nil, fmt.Errorf("get SVID: %w", err)
	}
	_ = svidCtx

	return &Client{
		source: source,
		id:     svid.ID,
	}, nil
}

// ServiceName returns the service name extracted from the SPIFFE ID path.
// For SPIFFE IDs with path "/service/name", it returns "name".
// ServiceName returns "unknown" if the path format is unexpected.
func (c *Client) ServiceName() string {
	path := c.id.Path()
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(segments) >= 2 && segments[0] == "service" {
		return segments[1]
	}
	return "unknown"
}

// TLSConfig returns a TLS configuration for mTLS servers.
// The configuration validates client certificates from the same trust domain.
func (c *Client) TLSConfig() *tls.Config {
	return tlsconfig.MTLSServerConfig(c.source, c.source, tlsconfig.AuthorizeMemberOf(c.id.TrustDomain()))
}

// ClientTLSConfig returns a TLS configuration for mTLS clients.
// The configuration validates server certificates from the same trust domain.
func (c *Client) ClientTLSConfig() *tls.Config {
	return tlsconfig.MTLSClientConfig(c.source, c.source, tlsconfig.AuthorizeMemberOf(c.id.TrustDomain()))
}

// HTTPClient returns an HTTP client configured with mTLS and security timeouts.
func (c *Client) HTTPClient() *http.Client {
	transport := &http.Transport{
		TLSClientConfig: c.ClientTLSConfig(),
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

// AuthorizeService returns an authorizer that validates certificates from the same trust domain.
// The allowedServices parameter is currently unused but reserved for future authorization logic.
func (c *Client) AuthorizeService(allowedServices ...string) tlsconfig.Authorizer {
	return tlsconfig.AuthorizeMemberOf(c.id.TrustDomain())
}

// Close closes the underlying X509Source and releases associated resources.
func (c *Client) Close() error {
	return c.source.Close()
}
