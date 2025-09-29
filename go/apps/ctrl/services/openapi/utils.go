package openapi

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (s *Service) loadOpenApiSpec(ctx context.Context, deploymentID string) (string, error) {
	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), deploymentID)
	if err != nil {
		return "", err
	}

	s.logger.Info("Deploymet", "raw", deployment)

	if !deployment.OpenapiSpec.Valid {
		return "", fault.New("deployment has no OpenAPI spec stored",
			fault.Public("OpenAPI specification not available for this deployment"),
		)
	}

	return deployment.OpenapiSpec.String, nil
}
