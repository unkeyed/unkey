package openapi

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (s *Service) loadVersionSpec(ctx context.Context, versionID string) (string, error) {
	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), versionID)
	if err != nil {
		return "", err
	}

	if !deployment.OpenapiSpec.Valid {
		return "", fault.New("deployment has no OpenAPI spec stored",
			fault.Public("OpenAPI specification not available for this deployment"),
		)
	}

	return deployment.OpenapiSpec.String, nil
}
