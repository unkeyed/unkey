package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
)

func (db *database) InsertApi(ctx context.Context, api *apisv1.Api) error {

	req := gen.InsertApiParams{
		ID:          api.ApiId,
		WorkspaceID: api.WorkspaceId,
		Name:        api.Name,
	}

	if len(api.IpWhitelist) > 0 {
		req.IpWhitelist = sql.NullString{String: strings.Join(api.IpWhitelist, ","), Valid: true}
	}
	switch api.AuthType {
	case apisv1.AuthType_AUTH_TYPE_KEY:
		req.AuthType = gen.NullApisAuthType{ApisAuthType: gen.ApisAuthTypeKey, Valid: true}
		req.KeyAuthID = sql.NullString{String: api.GetKeyAuthId(), Valid: true}
	case apisv1.AuthType_AUTH_TYPE_JWT:
		req.AuthType = gen.NullApisAuthType{ApisAuthType: gen.ApisAuthTypeJwt}
		// TODO: add jwt id here once it exists
	default:
		return fmt.Errorf("unknown auth type: %s", api.AuthType)
	}

	return db.write().InsertApi(ctx, req)

}
