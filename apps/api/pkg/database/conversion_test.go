package database

import (
	"database/sql"
	"testing"

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
	}

	m := apiEntityToModel(e)

	require.Equal(t, e.Id, m.ID)
	require.Equal(t, e.Name, m.Name)
	require.Equal(t, e.WorkspaceId, m.WorkspaceID)
	require.Equal(t, false, m.IPWhitelist.Valid)
	require.Equal(t, "", m.IPWhitelist.String)

}

func Test_apiModelToEntity(t *testing.T) {

	m := &models.API{
		ID:          uid.Api(),
		Name:        "test",
		WorkspaceID: uid.Workspace(),
	}

	e := apiModelToEntity(m)

	require.Equal(t, m.ID, e.Id)
	require.Equal(t, m.Name, e.Name)
	require.Equal(t, m.WorkspaceID, e.WorkspaceId)
	require.Equal(t, 0, len(e.IpWhitelist))
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
