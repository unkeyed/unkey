package routing

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

func (s *Service) AssignFrontlineRoutes(ctx restate.ObjectContext, req *hydrav1.AssignFrontlineRoutesRequest) (*hydrav1.AssignFrontlineRoutesResponse, error) {
	s.logger.Info("assigning domains",
		"deployment_id", req.GetDeploymentId(),
		"frontline_route_count", len(req.GetFrontlineRouteIds()),
	)

	// Upsert each domain in the database
	for _, frontlineRouteID := range req.GetFrontlineRouteIds() {
		_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.ReassignFrontlineRoute(stepCtx, s.db.RW(), db.ReassignFrontlineRouteParams{
				ID:           frontlineRouteID,
				DeploymentID: req.GetDeploymentId(),
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})

		}, restate.WithName(fmt.Sprintf("reassign-%s", frontlineRouteID)))
		if err != nil {
			return nil, err
		}

	}

	return &hydrav1.AssignFrontlineRoutesResponse{}, nil
}
