package routing

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) AssignIngressRoutes(ctx restate.ObjectContext, req *hydrav1.AssignIngressRoutesRequest) (*hydrav1.AssignIngressRoutesResponse, error) {
	s.logger.Info("assigning domains",
		"deployment_id", req.GetDeploymentId(),
		"ingress_route_count", len(req.GetIngressRouteIds()),
	)

	// Upsert each domain in the database
	for _, ingressRouteID := range req.GetIngressRouteIds() {
		_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.ReassignIngressRoute(stepCtx, s.db.RW(), db.ReassignIngressRouteParams{
				ID:           ingressRouteID,
				DeploymentID: req.GetDeploymentId(),
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})

		}, restate.WithName(fmt.Sprintf("reassign-%s", ingressRouteID)))
		if err != nil {
			return nil, err
		}

	}

	return &hydrav1.AssignIngressRoutesResponse{}, nil
}
