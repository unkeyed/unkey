package openapi

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

func (s *Service) loadOpenApiSpec(ctx context.Context, deploymentID string) ([]byte, error) {
	row, err := db.Query.FindOpenApiSpecByDeploymentID(ctx, s.db.RO(), sql.NullString{Valid: true, String: deploymentID})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fault.Wrap(err,
				fault.Public("OpenAPI specification not available for this deployment"),
			)
		}
		return nil, fault.Wrap(err)
	}

	if len(row.Content) == 0 {
		return nil, fault.New("deployment has no OpenAPI spec stored",
			fault.Internal("openapi spec is empty"),
			fault.Public("No OpenAPI specification found for this deployment."))
	}

	return row.Content, nil
}
