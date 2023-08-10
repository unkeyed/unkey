package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) InsertApi(ctx context.Context, api entities.Api) error {

	req := gen.InsertApiParams{
		ID:          api.Id,
		WorkspaceID: api.WorkspaceId,
		Name:        api.Name,
	}

	if len(api.IpWhitelist) > 0 {
		req.IpWhitelist = sql.NullString{String: strings.Join(api.IpWhitelist, ","), Valid: true}
	}
	switch api.AuthType {
	case entities.AuthTypeKey:
		req.AuthType = gen.NullApisAuthType{ApisAuthType: gen.ApisAuthTypeKey, Valid: true}
		req.KeyAuthID = sql.NullString{String: api.KeyAuthId, Valid: true}
	case entities.AuthTypeJWT:
		req.AuthType = gen.NullApisAuthType{ApisAuthType: gen.ApisAuthTypeJwt}
		// TODO: add jwt id here once it exists
	default:
		return fmt.Errorf("unknown auth type: %s", api.AuthType)
	}

	return db.write().InsertApi(ctx, req)

}
