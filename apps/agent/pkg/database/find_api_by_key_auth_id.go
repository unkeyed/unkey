package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"errors"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func (db *database) FindApiByKeyAuthId(ctx context.Context, keyAuthId string) (*apisv1.Api, bool, error) {

	model, err := db.read().FindApiByKeyAuthId(ctx, sql.NullString{String: keyAuthId, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("unable to find api: %w", err)
	}

	api, err := transformApiModelToEntity(model)
	if err != nil {
		return nil, true, fmt.Errorf("unable to transform key model to entity: %w", err)
	}
	return api, true, nil
}

func transformApiModelToEntity(m gen.Api) (*apisv1.Api, error) {
	api := &apisv1.Api{
		ApiId:       m.ID,
		Name:        m.Name,
		WorkspaceId: m.WorkspaceID,
	}
	if m.IpWhitelist.Valid {
		api.IpWhitelist = strings.Split(m.IpWhitelist.String, ",")
	}

	if m.AuthType.Valid {
		authType, err := m.AuthType.Value()
		if err != nil {
			return nil, fmt.Errorf("unable to determine auth type: %w", err)
		}

		switch gen.ApisAuthType(authType.(string)) {
		case gen.ApisAuthTypeKey:
			api.AuthType = apisv1.AuthType_AUTH_TYPE_KEY
			if !m.KeyAuthID.Valid {
				return nil, fmt.Errorf("auth type is 'key' but keyAuthId is empty")
			}
			api.KeyAuthId = util.Pointer(m.KeyAuthID.String)
		case gen.ApisAuthTypeJwt:
			api.AuthType = apisv1.AuthType_AUTH_TYPE_JWT
		default:
			return nil, fmt.Errorf("unknown auth type: '%s'", authType)
		}
	}

	return api, nil
}
