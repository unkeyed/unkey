package clidevice

import (
	"context"
	"encoding/json"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	principalpermissions "github.com/unkeyed/unkey/svc/api/internal/principal"
)

func decodePermissions(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func intersectPermissions(ctx context.Context, workspaceID string, roles []string, stored []string) ([]string, error) {
	caller := &principal.Principal{
		Version:               principal.Version,
		Type:                  principal.TypeJWT,
		Subject:               principal.Subject{ID: "", Name: "", Type: ""},
		Source:                nil,
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           workos.PermissionsForRoles(workspaceID, roles),
	}

	kept := make([]string, 0, len(stored))
	for _, permission := range stored {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		validated, err := principalpermissions.ValidateDelegatedPermissions(ctx, caller, []string{permission})
		if err != nil {
			continue
		}
		kept = append(kept, validated...)
	}
	return kept, nil
}
