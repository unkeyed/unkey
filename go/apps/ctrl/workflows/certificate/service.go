package certificate

import (
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

// Service handles ACME certificate challenge workflows.
//
// This service orchestrates the complete certificate issuance process including
// domain validation, challenge claiming, ACME protocol communication, and certificate
// storage. It implements the hydrav1.CertificateServiceServer interface.
//
// The service uses Restate virtual objects keyed by domain name to ensure that only
// one certificate challenge runs per domain at any time, preventing duplicate requests
// and rate limit violations.
type Service struct {
	hydrav1.UnimplementedCertificateServiceServer
	db          db.Database
	partitionDB db.Database
	vault       *vault.Service
	logger      logging.Logger
}

var _ hydrav1.CertificateServiceServer = (*Service)(nil)

// Config holds the configuration for creating a certificate service.
type Config struct {
	// DB is the main database connection for workspace and domain data.
	DB db.Database

	// PartitionDB is the partition database connection for certificate storage.
	PartitionDB db.Database

	// Vault provides encryption services for private key storage.
	Vault *vault.Service

	// Logger for structured logging.
	Logger logging.Logger
}

// New creates a new certificate service instance.
func New(cfg Config) *Service {
	return &Service{
		db:          cfg.DB,
		partitionDB: cfg.PartitionDB,
		vault:       cfg.Vault,
		logger:      cfg.Logger,
	}
}
