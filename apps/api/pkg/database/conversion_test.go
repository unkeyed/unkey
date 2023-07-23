package database

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/pkg/database/models"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/uid"
)

func Test_apiEntityToModel(t *testing.T) {
	e := entities.Api{
		Id:          uid.Api(),
		Name:        "test",
		WorkspaceId: uid.Workspace(),
		KeyAuthId:   "key_auth_123",
	}

	m := apiEntityToModel(e)

	require.Equal(t, e.Id, m.ID)
	require.Equal(t, e.Name, m.Name)
	require.Equal(t, e.WorkspaceId, m.WorkspaceID)
	require.Equal(t, false, m.IPWhitelist.Valid)
	require.False(t, m.IPWhitelist.Valid)
	require.True(t, m.KeyAuthID.Valid)
	require.Equal(t, "key_auth_123", m.KeyAuthID.String)

}

func Test_apiModelToEntity(t *testing.T) {

	m := &models.API{
		ID:          uid.Api(),
		Name:        "test",
		WorkspaceID: uid.Workspace(),
		KeyAuthID:   sql.NullString{String: "key_auth_123", Valid: true},
	}

	e := apiModelToEntity(m)

	require.Equal(t, m.ID, e.Id)
	require.Equal(t, m.Name, e.Name)
	require.Equal(t, m.WorkspaceID, e.WorkspaceId)
	require.Equal(t, 0, len(e.IpWhitelist))
	require.Equal(t, "key_auth_123", e.KeyAuthId)
}

func Test_apiEntityToModel_WithIpWithlist(t *testing.T) {
	e := entities.Api{
		Id:          uid.Api(),
		Name:        "test",
		WorkspaceId: uid.Workspace(),
		IpWhitelist: []string{"1.1.1.1", "2.2.2.2"},
	}

	m := apiEntityToModel(e)

	require.Equal(t, e.Id, m.ID)
	require.Equal(t, e.Name, m.Name)
	require.Equal(t, e.WorkspaceId, m.WorkspaceID)
	require.Equal(t, true, m.IPWhitelist.Valid)
	require.Equal(t, "1.1.1.1,2.2.2.2", m.IPWhitelist.String)

}

func Test_apiModelToEntity_WithIpWithlist(t *testing.T) {

	m := &models.API{
		ID:          uid.Api(),
		Name:        "test",
		WorkspaceID: uid.Workspace(),
		IPWhitelist: sql.NullString{
			Valid:  true,
			String: "1.1.1.1,2.2.2.2",
		},
	}

	e := apiModelToEntity(m)

	require.Equal(t, m.ID, e.Id)
	require.Equal(t, m.Name, e.Name)
	require.Equal(t, m.WorkspaceID, e.WorkspaceId)
	require.Equal(t, []string{"1.1.1.1", "2.2.2.2"}, e.IpWhitelist)
}

func Test_keyModelToEntity(t *testing.T) {

	m := &models.Key{
		ID:          uid.Key(),
		KeyAuthID:   sql.NullString{String: uid.KeyAuth(), Valid: true},
		WorkspaceID: uid.Workspace(),
		Hash:        "abc",
		Start:       "abc",
		CreatedAt:   time.Now(),
	}

	e, err := keyModelToEntity(m)
	require.NoError(t, err)
	require.Equal(t, m.ID, e.Id)
	require.Equal(t, m.KeyAuthID.String, e.KeyAuthId)
	require.Equal(t, m.WorkspaceID, e.WorkspaceId)
	require.Equal(t, m.Hash, e.Hash)
	require.Equal(t, m.Start, e.Start)
	require.Equal(t, m.CreatedAt, e.CreatedAt)
	require.Nil(t, e.Ratelimit)
}

func Test_keyModelToEntity_WithNullFields(t *testing.T) {

	m := &models.Key{
		ID:                      uid.Key(),
		KeyAuthID:               sql.NullString{String: uid.KeyAuth(), Valid: true},
		WorkspaceID:             uid.Workspace(),
		Hash:                    "abc",
		Start:                   "abc",
		CreatedAt:               time.Now(),
		RemainingRequests:       sql.NullInt64{Int64: 99, Valid: true},
		RatelimitType:           sql.NullString{String: "fast", Valid: true},
		RatelimitLimit:          sql.NullInt64{Int64: 10, Valid: true},
		RatelimitRefillRate:     sql.NullInt64{Int64: 1, Valid: true},
		RatelimitRefillInterval: sql.NullInt64{Int64: 1000, Valid: true},
	}

	e, err := keyModelToEntity(m)
	require.NoError(t, err)
	require.Equal(t, m.ID, e.Id)
	require.Equal(t, m.KeyAuthID.String, e.KeyAuthId)
	require.Equal(t, m.WorkspaceID, e.WorkspaceId)
	require.Equal(t, m.Hash, e.Hash)
	require.Equal(t, m.Start, e.Start)
	require.Equal(t, m.CreatedAt, e.CreatedAt)
	require.Equal(t, true, e.Remaining.Enabled)
	require.Equal(t, int64(99), e.Remaining.Remaining)
	require.Equal(t, "fast", e.Ratelimit.Type)
	require.Equal(t, int64(10), e.Ratelimit.Limit)
	require.Equal(t, int64(1), e.Ratelimit.RefillRate)
	require.Equal(t, int64(1000), e.Ratelimit.RefillInterval)
}
