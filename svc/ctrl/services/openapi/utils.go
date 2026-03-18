package openapi

import (
	"context"
	"database/sql"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

func (s *Service) loadOpenApiSpec(ctx context.Context, deploymentID string) (string, error) {
	spec, err := db.Query.FindOpenApiSpecByDeploymentID(ctx, s.db.RO(), sql.NullString{String: deploymentID, Valid: true})
	if err != nil {
		if db.IsNotFound(err) {
			return "", fault.Wrap(err,
				fault.Public("OpenAPI specification not available for this deployment"),
			)
		}
		return "", fault.Wrap(err)
	}

	return string(spec.Content), nil
}
