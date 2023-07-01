package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/chronark/unkey/apps/api/pkg/database/models"
	"github.com/chronark/unkey/apps/api/pkg/entities"
)

func keyModelToEntity(model *models.Key) (entities.Key, error) {

	key := entities.Key{}
	key.Id = model.ID
	key.ApiId = model.APIID
	key.WorkspaceId = model.WorkspaceID
	key.Hash = model.Hash
	key.Start = model.Start

	key.CreatedAt = model.CreatedAt

	if model.OwnerID.Valid {
		key.OwnerId = model.OwnerID.String
	}

	if model.Name.Valid {
		key.Name = model.Name.String
	}

	if model.Expires.Valid {
		key.Expires = model.Expires.Time
	}

	if model.ForWorkspaceID.Valid {
		key.ForWorkspaceId = model.ForWorkspaceID.String
	}

	if model.Meta.Valid {
		err := json.Unmarshal([]byte(model.Meta.String), &key.Meta)
		if err != nil {
			return entities.Key{}, fmt.Errorf("unable to unmarshal meta: %w", err)
		}
	}

	if model.RatelimitType.Valid {
		key.Ratelimit = &entities.Ratelimit{
			Type:           model.RatelimitType.String,
			Limit:          model.RatelimitLimit.Int64,
			RefillRate:     model.RatelimitRefillRate.Int64,
			RefillInterval: model.RatelimitRefillInterval.Int64,
		}
	}
	return key, nil
}

func keyEntityToModel(e entities.Key) (*models.Key, error) {
	metaBuf, err := json.Marshal(e.Meta)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal meta: %w", err)
	}

	key := &models.Key{
		ID:          e.Id,
		APIID:       e.ApiId,
		Start:       e.Start,
		WorkspaceID: e.WorkspaceId,
		Hash:        e.Hash,
		OwnerID: sql.NullString{
			String: e.OwnerId,
			Valid:  e.OwnerId != "",
		},
		Name: sql.NullString{
			String: e.Name,
			Valid:  e.Name != "",
		},
		Meta:      sql.NullString{String: string(metaBuf), Valid: len(metaBuf) > 0},
		CreatedAt: e.CreatedAt,
		Expires: sql.NullTime{
			Time:  e.Expires,
			Valid: !e.Expires.IsZero(),
		},

		ForWorkspaceID: sql.NullString{String: e.ForWorkspaceId, Valid: e.ForWorkspaceId != ""},
	}
	if e.Ratelimit != nil {
		key.RatelimitType = sql.NullString{String: e.Ratelimit.Type, Valid: e.Ratelimit.Type != ""}
		key.RatelimitLimit = sql.NullInt64{Int64: e.Ratelimit.Limit, Valid: e.Ratelimit.Limit > 0}
		key.RatelimitRefillRate = sql.NullInt64{Int64: e.Ratelimit.RefillRate, Valid: e.Ratelimit.RefillRate > 0}
		key.RatelimitRefillInterval = sql.NullInt64{Int64: e.Ratelimit.RefillInterval, Valid: e.Ratelimit.RefillRate > 0}
	}

	return key, nil

}

func workspaceEntityToModel(w entities.Workspace) *models.Workspace {

	return &models.Workspace{
		ID:                 w.Id,
		Name:               w.Name,
		Slug:               w.Slug,
		TenantID:           w.TenantId,
		Internal:           w.Internal,
		EnableBetaFeatures: w.EnableBetaFeatures,
	}

}

func workspaceModelToEntity(model *models.Workspace) entities.Workspace {
	return entities.Workspace{
		Id:                 model.ID,
		Name:               model.Name,
		Slug:               model.Slug,
		TenantId:           model.TenantID,
		Internal:           model.Internal,
		EnableBetaFeatures: model.EnableBetaFeatures,
	}

}

func apiEntityToModel(a entities.Api) *models.API {

	return &models.API{
		ID:          a.Id,
		Name:        a.Name,
		WorkspaceID: a.WorkspaceId,
	}

}

func apiModelToEntity(model *models.API) entities.Api {
	return entities.Api{
		Id:          model.ID,
		Name:        model.Name,
		WorkspaceId: model.WorkspaceID,
	}

}
