package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// config collects the option values applied by [New] and [ValidateEndpoint].
// The zero value is the safe default: guards on, no timeout, no redirects.
type config struct {
	timeout        time.Duration
	unsafeAllowAll bool
	redirectsMax   int
}

// Option configures [New] and [ValidateEndpoint].
type Option func(*config)

// WithTimeout bounds the whole request, including connect, redirects, and
// reading the response body. Zero means no timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *config) {
		c.timeout = timeout
	}
}

// UnsafeAllowAll disables every guard: the SSRF check on resolved IPs, the
// https requirement on endpoints, and the https requirement on redirect
// targets. Development and tests only.
func UnsafeAllowAll() Option {
	return func(c *config) {
		c.unsafeAllowAll = true
	}
}

// FollowRedirects lets the client follow up to redirectsMax redirects.
// Redirect targets must use https unless [UnsafeAllowAll] is set. Without
// this option the client does not follow redirects: the first redirect
// response fails the request.
func FollowRedirects(redirectsMax int) Option {
	return func(c *config) {
		c.redirectsMax = redirectsMax
	}
}

// New returns an HTTP client hardened for customer-supplied endpoints.
func New(opts ...Option) *http.Client {
	cfg := applyOptions(opts)
	dialer := new(net.Dialer)
	dialer.Timeout = 30 * time.Second
	dialer.KeepAlive = 30 * time.Second
	transport := &http.Transport{
		DialContext:            dialContext(dialer, cfg),
		ForceAttemptHTTP2:      false,
		MaxIdleConns:           100,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ExpectContinueTimeout:  time.Second,
		ResponseHeaderTimeout:  30 * time.Second,
		MaxResponseHeaderBytes: 1 << 20,
	}
	return &http.Client{Transport: transport, Timeout: cfg.timeout, CheckRedirect: checkRedirect(cfg)}
}

// ValidateEndpoint rejects endpoints that are not absolute https URLs.
// [UnsafeAllowAll] additionally permits plain http, for development and
// tests only.
func ValidateEndpoint(raw string, opts ...Option) error {
	cfg := applyOptions(opts)
	endpoint, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse endpoint: %w", err)
	}
	if endpoint.Host == "" || (endpoint.Scheme != "https" && !(cfg.unsafeAllowAll && endpoint.Scheme == "http")) {
		return errors.New("endpoint must be an absolute https URL")
	}
	return nil
}

// IsForbiddenIP reports whether the SSRF guard rejects the resolved address.
// It rejects nil, loopback, private (RFC 1918 and ULA), link-local,
// unspecified, and multicast addresses.
func IsForbiddenIP(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

// applyOptions folds opts into a config, starting from the safe zero value.
func applyOptions(opts []Option) config {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// dialContext returns the transport dial function that enforces the SSRF
// guard. It resolves the host itself and dials resolved IPs directly, so the
// address that passed the check is exactly the address that is dialed.
func dialContext(dialer *net.Dialer, cfg config) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("split dial address: %w", err)
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve endpoint host: %w", err)
		}
		var dialErr error
		for _, resolved := range ips {
			if !cfg.unsafeAllowAll && IsForbiddenIP(resolved.IP) {
				continue
			}

			// We must dial the IP directly because the attacker controls the DNS server
			// for their own hostname, so they can answer the first lookup with a harmless
			// public IP and the next lookup with a private one (a TTL of zero forces the
			// re-lookup). If we checked the first answer and then handed the hostname to
			// a separate dial step, that step would resolve again and could connect to
			// the private address that was never checked. Resolving once and dialing the
			// checked IP leaves no second lookup to exploit.
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.IP.String(), port))
			if err == nil {
				return conn, nil
			}
			dialErr = err
		}
		if dialErr != nil {
			return nil, fmt.Errorf("dial endpoint: %w", dialErr)
		}
		return nil, errors.New("endpoint host resolved only to forbidden IP addresses")
	}
}

// checkRedirect returns the client redirect policy for cfg: refuse all
// redirects by default, and with [FollowRedirects] allow up to the configured
// number of https-only hops. The dial guard still applies to every hop.
func checkRedirect(cfg config) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if cfg.redirectsMax <= 0 {
			return errors.New("redirects are not allowed")
		}
		if !cfg.unsafeAllowAll && req.URL.Scheme != "https" {
			return errors.New("redirect endpoint must use https")
		}
		if len(via) > cfg.redirectsMax {
			return fmt.Errorf("stopped after %d redirects", cfg.redirectsMax)
		}
		return nil
	}
}
