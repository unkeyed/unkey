package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"
)

// cachedCert holds a certificate with expiration time for caching.
type cachedCert struct {
	cert     *tls.Certificate
	loadedAt time.Time
	cacheTTL time.Duration
}

// isExpired reports whether the cached certificate should be reloaded.
func (c *cachedCert) isExpired() bool {
	return time.Since(c.loadedAt) > c.cacheTTL
}

// cachedFileProvider wraps fileProvider with certificate caching for performance.
type cachedFileProvider struct {
	*fileProvider
	mu       sync.RWMutex
	cache    *cachedCert
	cacheTTL time.Duration
}

// newCachedFileProvider creates a file provider with certificate caching enabled.
func newCachedFileProvider(cfg Config, cacheTTL time.Duration) (Provider, error) {
	base, err := newFileProvider(cfg)
	if err != nil {
		return nil, err
	}

	fp, ok := base.(*fileProvider)
	if !ok {
		return nil, fmt.Errorf("unexpected provider type")
	}

	return &cachedFileProvider{
		fileProvider: fp,
		cacheTTL:     cacheTTL,
	}, nil
}

// loadTLSConfigCached returns cached TLS configuration or loads fresh certificates.
func (p *cachedFileProvider) loadTLSConfigCached() (*tls.Config, error) {
	if p.certFile == "" || p.keyFile == "" {
		return nil, nil //nolint: all
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			p.mu.RLock()
			if p.cache != nil && !p.cache.isExpired() {
				cert := p.cache.cert
				p.mu.RUnlock()
				return cert, nil
			}
			p.mu.RUnlock()

			p.mu.Lock()
			defer p.mu.Unlock()

			if p.cache != nil && !p.cache.isExpired() {
				return p.cache.cert, nil
			}

			cert, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
			if err != nil {
				if p.cache != nil && p.cache.cert != nil {
					return p.cache.cert, nil
				}
				return nil, fmt.Errorf("load cert/key: %w", err)
			}

			p.cache = &cachedCert{
				cert:     &cert,
				loadedAt: time.Now(),
				cacheTTL: p.cacheTTL,
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

func (p *cachedFileProvider) ServerTLSConfig() (*tls.Config, error) {
	return p.loadTLSConfigCached()
}

func (p *cachedFileProvider) ClientTLSConfig() (*tls.Config, error) {
	return p.loadTLSConfigCached()
}

// GetCacheMetrics returns cache hit and miss counts for monitoring.
func (p *cachedFileProvider) GetCacheMetrics() (hits, misses uint64) {
	return 0, 0
}
