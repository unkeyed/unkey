package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/hash"
	"github.com/unkeyed/unkey/apps/api/pkg/logging"
	"github.com/unkeyed/unkey/apps/api/pkg/uid"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type resources struct {
	UnkeyWorkspace entities.Workspace
	UnkeyApi       entities.Api
	UnkeyKey       string
	UserWorkspace  entities.Workspace
	UserApi        entities.Api
	Database       database.Database
}

func SetupResources(t *testing.T) resources {
	t.Helper()
	ctx := context.Background()
	db, err := database.New(database.Config{
		Logger:    logging.NewNoopLogger(),
		PrimaryUs: os.Getenv("DATABASE_DSN"),
	})
	require.NoError(t, err)

	r := resources{
		Database: db,
	}

	r.UnkeyWorkspace = entities.Workspace{
		Id:       uid.Workspace(),
		Name:     "unkey",
		Slug:     uuid.NewString(),
		TenantId: uuid.NewString(),
	}

	r.UserWorkspace = entities.Workspace{
		Id:       uid.Workspace(),
		Name:     "user",
		Slug:     uuid.NewString(),
		TenantId: uuid.NewString(),
	}

	r.UnkeyApi = entities.Api{
		Id:          uid.Api(),
		Name:        "name",
		WorkspaceId: r.UnkeyWorkspace.Id,
	}
	r.UserApi = entities.Api{
		Id:          uid.Api(),
		Name:        "name",
		WorkspaceId: r.UserWorkspace.Id,
	}

	require.NoError(t, db.CreateWorkspace(ctx, r.UnkeyWorkspace))
	require.NoError(t, db.CreateApi(ctx, r.UnkeyApi))
	require.NoError(t, db.CreateWorkspace(ctx, r.UserWorkspace))
	require.NoError(t, db.CreateApi(ctx, r.UserApi))

	r.UnkeyKey = uid.New(16, string(uid.UnkeyPrefix))

	rootKeyEntity := entities.Key{
		Id:             uid.Key(),
		ApiId:          r.UnkeyApi.Id,
		WorkspaceId:    r.UnkeyWorkspace.Id,
		ForWorkspaceId: r.UserWorkspace.Id,
		Hash:           hash.Sha256(r.UnkeyKey),
		CreatedAt:      time.Now(),
	}

	require.NoError(t, db.CreateKey(ctx, rootKeyEntity))

	return r
}
