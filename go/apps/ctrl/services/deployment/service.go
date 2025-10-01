package deployment

import (
	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlrestate "github.com/unkeyed/unkey/go/apps/ctrl/restate"
	deploymentworkflow "github.com/unkeyed/unkey/go/apps/ctrl/workflows/deployment"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Service struct {
	ctrlv1connect.UnimplementedDeploymentServiceHandler
	db             db.Database
	partitionDB    db.Database
	restate        *restateingress.Client
	deployWorkflow *ctrlrestate.Runner[deploymentworkflow.DeployRequest]
	logger         logging.Logger
}

func New(database db.Database, partitionDB db.Database, c *restateingress.Client, deployWorkflow *deploymentworkflow.DeployWorkflow, logger logging.Logger) *Service {

	return &Service{
		UnimplementedDeploymentServiceHandler: ctrlv1connect.UnimplementedDeploymentServiceHandler{},
		db:                                    database,
		partitionDB:                           partitionDB,
		restate:                               c,
		deployWorkflow:                        ctrlrestate.CreateRunner[deploymentworkflow.DeployRequest](c, deployWorkflow),
		logger:                                logger,
	}
}
