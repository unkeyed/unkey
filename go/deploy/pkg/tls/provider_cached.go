package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"
)

// AIDEV-NOTE: Cached certificate provider for performance optimization
// Balances security (frequent rotation checks) with performance (avoiding disk I/O on every connection)

// cachedCert holds a certificate with expiration time
type cachedCert struct {
	cert     *tls.Certificate
	loadedAt time.Time
	cacheTTL time.Duration
}

// isExpired checks if the cached certificate should be reloaded
func (c *cachedCert) isExpired() bool {
	return time.Since(c.loadedAt) > c.cacheTTL
}

// cachedFileProvider wraps fileProvider with caching
type cachedFileProvider struct {
	*fileProvider
	mu    sync.RWMutex
	cache *cachedCert
	// Configuration
	cacheTTL time.Duration
}

// newCachedFileProvider creates a file provider with certificate caching
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

// loadTLSConfigCached returns cached config or loads fresh one
func (p *cachedFileProvider) loadTLSConfigCached() (*tls.Config, error) {
	if p.certFile == "" || p.keyFile == "" {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			// Fast path - check cache with read lock
			p.mu.RLock()
			if p.cache != nil && !p.cache.isExpired() {
				cert := p.cache.cert
				p.mu.RUnlock()
				return cert, nil
			}
			p.mu.RUnlock()

			// Slow path - load certificate with write lock
			p.mu.Lock()
			defer p.mu.Unlock()

			// Double-check after acquiring write lock
			if p.cache != nil && !p.cache.isExpired() {
				return p.cache.cert, nil
			}

			// Load fresh certificate
			cert, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
			if err != nil {
				// AIDEV-NOTE: If cert loading fails but we have cached cert, use it
				// This provides resilience during brief filesystem issues
				if p.cache != nil && p.cache.cert != nil {
					// Log warning but continue with stale cert
					// In production, this should trigger an alert
					return p.cache.cert, nil
				}
				return nil, fmt.Errorf("load cert/key: %w", err)
			}

			// Update cache
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

	// Handle CA and mutual TLS same as before
	if p.caFile != "" {
		// This part doesn't change frequently, so we can load it once
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

// Optional: Add metrics for cache hit/miss ratio
type cacheMetrics struct {
	hits   uint64
	misses uint64
}

// GetCacheMetrics returns cache performance metrics
func (p *cachedFileProvider) GetCacheMetrics() (hits, misses uint64) {
	// Implementation would track hits/misses in GetCertificate
	return 0, 0
}
