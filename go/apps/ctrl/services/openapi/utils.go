package openapi

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (s *Service) loadVersionSpec(ctx context.Context, versionID string) (string, error) {
	version, err := db.Query.FindVersionById(ctx, s.db.RO(), versionID)
	if err != nil {
		return "", err
	}

	if !version.OpenapiSpec.Valid {
		return "", fault.New("version has no OpenAPI spec stored",
			fault.Public("OpenAPI specification not available for this version"),
		)
	}

	return version.OpenapiSpec.String, nil
}
