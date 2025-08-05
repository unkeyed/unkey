package certmanager

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var _ Service = (*service)(nil)

// service provides a basic certificate manager.
type service struct {
	mu       sync.RWMutex
	certs    map[string]*tls.Certificate
	logger   logging.Logger
	fallback *tls.Certificate
}

// New creates a new certificate manager.
func New(logger logging.Logger) *service {
	return &service{
		certs:  make(map[string]*tls.Certificate),
		logger: logger,
	}
}

// LoadCertificate loads a certificate/key pair for a specific domain.
func (cm *service) LoadCertificate(domain, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate for %s: %w", domain, err)
	}

	cm.mu.Lock()
	cm.certs[domain] = &cert
	cm.mu.Unlock()

	cm.logger.Info("loaded certificate", "domain", domain)
	return nil
}

// GetCertificate implements the CertManager interface.
func (cm *service) GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Try to find exact domain match
	if cert, ok := cm.certs[domain]; ok {
		cm.logger.Debug("serving certificate", "domain", domain)
		return cert, nil
	}

	// Use fallback if available
	if cm.fallback != nil {
		cm.logger.Debug("serving fallback certificate", "requested", domain)
		return cm.fallback, nil
	}

	cm.logger.Warn("no certificate found", "domain", domain)
	return nil, fmt.Errorf("no certificate available for %s", domain)
}
