package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"sync"
	"time"
)

type CertCache struct {
	mu    sync.RWMutex
	certs map[string]cachedCert
	ca    *x509.Certificate
	caKey *ecdsa.PrivateKey
}

type cachedCert struct {
	cert      *tls.Certificate
	expiresAt time.Time
}

func NewCertCache(ca *x509.Certificate, caKey *ecdsa.PrivateKey) *CertCache {
	//nolint:exhaustruct
	cc := &CertCache{
		certs: make(map[string]cachedCert),
		ca:    ca,
		caKey: caKey,
	}
	go cc.sweepLoop()
	return cc
}

func (c *CertCache) sweepLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for host, entry := range c.certs {
			if now.After(entry.expiresAt) {
				delete(c.certs, host)
			}
		}
		c.mu.Unlock()
	}
}

func (c *CertCache) GetOrCreate(host string) (*tls.Certificate, error) {
	now := time.Now()

	c.mu.RLock()
	cached, ok := c.certs[host]
	c.mu.RUnlock()

	if ok && now.Before(cached.expiresAt) {
		return cached.cert, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock.
	cached, ok = c.certs[host]
	if ok && now.Before(cached.expiresAt) {
		return cached.cert, nil
	}

	cert, err := c.generate(host, now)
	if err != nil {
		return nil, err
	}

	c.certs[host] = cachedCert{
		cert:      cert,
		expiresAt: now.Add(23 * time.Hour),
	}

	return cert, nil
}

func (c *CertCache) generate(host string, now time.Time) (*tls.Certificate, error) {
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate leaf key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	//nolint:exhaustruct
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: host}, //nolint:exhaustruct
		DNSNames:     []string{host},
		NotBefore:    now.Add(-5 * time.Minute),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, template, c.ca, &leafKey.PublicKey, c.caKey)
	if err != nil {
		return nil, fmt.Errorf("sign leaf cert: %w", err)
	}

	leafCert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return nil, fmt.Errorf("parse leaf cert: %w", err)
	}

	//nolint:exhaustruct
	tlsCert := &tls.Certificate{
		Certificate: [][]byte{leafDER},
		PrivateKey:  leafKey,
		Leaf:        leafCert,
	}

	return tlsCert, nil
}
