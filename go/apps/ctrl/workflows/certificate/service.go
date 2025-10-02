package certificate

import (
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
)

// Service handles ACME certificate challenge workflows
type Service struct {
	hydrav1.UnimplementedCertificateServiceServer
	db          db.Database
	partitionDB db.Database
	vault       *vault.Service
	logger      logging.Logger
}

var _ hydrav1.CertificateServiceServer = (*Service)(nil)

type Config struct {
	DB          db.Database
	PartitionDB db.Database
	Vault       *vault.Service
	Logger      logging.Logger
}

func New(cfg Config) *Service {
	return &Service{
		db:          cfg.DB,
		partitionDB: cfg.PartitionDB,
		vault:       cfg.Vault,
		logger:      cfg.Logger,
	}
}
