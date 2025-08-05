package certmanager

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/partition/db"
)

var _ Service = (*service)(nil)

// service provides a basic certificate manager.
type service struct {
	// Logger is the logger used to log messages.
	logger logging.Logger

	// DB is the database used to store certificates.
	db db.Database
}

// New creates a new certificate manager.
func New(cfg Config) *service {
	return &service{
		logger: cfg.Logger,
		db:     cfg.DB,
	}
}

// GetCertificate implements the CertManager interface.
func (cm *service) GetCertificate(ctx context.Context, domain string) (*tls.Certificate, error) {
	return nil, fmt.Errorf("no certificate available for %s", domain)
}
