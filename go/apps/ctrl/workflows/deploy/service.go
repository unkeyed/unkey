package deploy

import (
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

const hardcodedNamespace = "unkey" // TODO change to workspace scope

// Workflow orchestrates deployment lifecycle operations
type Workflow struct {
	hydrav1.UnimplementedDeploymentServiceServer
	db            db.Database
	partitionDB   db.Database
	logger        logging.Logger
	krane         kranev1connect.DeploymentServiceClient
	defaultDomain string
}

var _ hydrav1.DeploymentServiceServer = (*Workflow)(nil)

type Config struct {
	Logger        logging.Logger
	DB            db.Database
	PartitionDB   db.Database
	Krane         kranev1connect.DeploymentServiceClient
	DefaultDomain string
}

// New creates a new deploy workflow instance
func New(cfg Config) *Workflow {
	return &Workflow{
		db:            cfg.DB,
		partitionDB:   cfg.PartitionDB,
		logger:        cfg.Logger,
		krane:         cfg.Krane,
		defaultDomain: cfg.DefaultDomain,
	}
}
