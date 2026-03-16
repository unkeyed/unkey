package openapi

import (
	"context"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

func (s *Service) loadOpenApiSpec(ctx context.Context, deploymentID string) ([]byte, error) {
	row, err := db.Query.FindOpenApiSpecByDeploymentID(ctx, s.db.RO(), deploymentID)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to find openapi spec"),
			fault.Public("Could not load OpenAPI spec for deployment."))
	}
	if len(row.Spec) == 0 {
		return nil, fault.New("deployment has no OpenAPI spec stored",
			fault.Internal("openapi spec is empty"),
			fault.Public("No OpenAPI spec found for this deployment."))
	}
	return row.Spec, nil
}
