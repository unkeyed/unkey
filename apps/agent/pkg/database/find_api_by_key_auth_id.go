package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"errors"

	gen "github.com/unkeyed/unkey/apps/agent/gen/database"

	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
)

func (db *database) FindApiByKeyAuthId(ctx context.Context, keyAuthId string) (entities.Api, bool, error) {

	model, err := db.read().FindApiByKeyAuthId(ctx, sql.NullString{String: keyAuthId, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entities.Api{}, false, nil
		}
		return entities.Api{}, false, fmt.Errorf("unable to find api: %w", err)
	}

	api, err := transformApiModelToEntity(model)
	if err != nil {
		return entities.Api{}, true, fmt.Errorf("unable to transform key model to entity: %w", err)
	}
	return api, true, nil
}

func transformApiModelToEntity(m gen.Api) (entities.Api, error) {
	api := entities.Api{
		Id:          m.ID,
		Name:        m.Name,
		WorkspaceId: m.WorkspaceID,
	}
	if m.IpWhitelist.Valid {
		api.IpWhitelist = strings.Split(m.IpWhitelist.String, ",")
	}

	if m.AuthType.Valid {
		authType, err := m.AuthType.Value()
		if err != nil {
			return entities.Api{}, fmt.Errorf("unable to determine auth type: %w", err)
		}

		switch gen.ApisAuthType(authType.(string)) {
		case gen.ApisAuthTypeKey:
			api.AuthType = entities.AuthTypeKey
			if !m.KeyAuthID.Valid {
				return entities.Api{}, fmt.Errorf("auth type is 'key' but keyAuthId is empty")
			}
			api.KeyAuthId = m.KeyAuthID.String
		case gen.ApisAuthTypeJwt:
			api.AuthType = entities.AuthTypeJWT
		default:
			return entities.Api{}, fmt.Errorf("unknown auth type: '%s'", authType)
		}
	}

	return api, nil
}
