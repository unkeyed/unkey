package database

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/logging"
	"github.com/unkeyed/unkey/apps/api/pkg/uid"
)

func TestApiSetGet_Simple(t *testing.T) {
	ctx := context.Background()
	db, err := New(Config{
		Logger:    logging.NewNoopLogger(),
		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})

	require.NoError(t, err)

	api := entities.Api{
		Id:          uid.Api(),
		Name:        "test",
		WorkspaceId: uid.Workspace(),
	}

	err = db.CreateApi(ctx, api)
	require.NoError(t, err)

	found, err := db.GetApi(ctx, api.Id)
	require.NoError(t, err)

	require.Equal(t, api.Id, found.Id)
	require.Equal(t, api.Name, found.Name)
	require.Equal(t, api.WorkspaceId, found.WorkspaceId)
	require.Equal(t, 0, len(found.IpWhitelist))

}

func TestApiSetGet_WithIpWhitelist(t *testing.T) {
	ctx := context.Background()
	db, err := New(Config{
		Logger:    logging.NewNoopLogger(),
		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})

	require.NoError(t, err)

	api := entities.Api{
		Id:          uid.Api(),
		Name:        "test",
		WorkspaceId: uid.Workspace(),
		IpWhitelist: []string{"1.1.1.1", "2.2.2.2"},
	}

	err = db.CreateApi(ctx, api)
	require.NoError(t, err)

	found, err := db.GetApi(ctx, api.Id)
	require.NoError(t, err)

	require.Equal(t, api.Id, found.Id)
	require.Equal(t, api.Name, found.Name)
	require.Equal(t, api.WorkspaceId, found.WorkspaceId)
	require.Equal(t, api.IpWhitelist, found.IpWhitelist)

}
