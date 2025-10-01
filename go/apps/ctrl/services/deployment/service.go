package deployment

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service struct {
	ctrlv1connect.UnimplementedDeploymentServiceHandler
	db          db.Database
	partitionDB db.Database

	restate *restateingress.Client

	logger logging.Logger
}

type Config struct {
	Database    db.Database
	PartitionDB db.Database
	Restate     *restateingress.Client
	Logger      logging.Logger
}

func New(cfg Config) *Service {

	return &Service{
		UnimplementedDeploymentServiceHandler: ctrlv1connect.UnimplementedDeploymentServiceHandler{},
		db:                                    cfg.Database,
		partitionDB:                           cfg.PartitionDB,
		restate:                               cfg.Restate,
		logger:                                cfg.Logger,
	}
}
