package openapi

import (
	"context"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

func (s *Service) loadOpenApiSpec(ctx context.Context, deploymentID string) (string, error) {
	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), deploymentID)
	if err != nil {
		return "", err
	}

	// Consider: logger.Debug("Deployment fetched", "id", deployment.ID, "hasSpec", deployment.OpenapiSpec.Valid)
	if !deployment.OpenapiSpec.Valid {
		return "", fault.New("deployment has no OpenAPI spec stored",
			fault.Public("OpenAPI specification not available for this deployment"),
		)
	}

	return deployment.OpenapiSpec.String, nil
}
