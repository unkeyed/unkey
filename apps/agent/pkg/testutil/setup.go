package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

type resources struct {
	UnkeyWorkspace entities.Workspace
	UnkeyApi       entities.Api
	UnkeyKeyAuth   entities.KeyAuth
	UnkeyKey       string
	UserWorkspace  entities.Workspace
	UserApi        entities.Api
	UserKeyAuth    entities.KeyAuth
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

	r.UnkeyKeyAuth = entities.KeyAuth{
		Id:          uid.KeyAuth(),
		WorkspaceId: r.UnkeyWorkspace.Id,
	}
	r.UserKeyAuth = entities.KeyAuth{
		Id:          uid.KeyAuth(),
		WorkspaceId: r.UserWorkspace.Id,
	}

	r.UnkeyApi = entities.Api{
		Id:          uid.Api(),
		Name:        "name",
		WorkspaceId: r.UnkeyWorkspace.Id,
		AuthType:    entities.AuthTypeKey,
		KeyAuthId:   r.UnkeyKeyAuth.Id,
	}
	r.UserApi = entities.Api{
		Id:          uid.Api(),
		Name:        "name",
		WorkspaceId: r.UserWorkspace.Id,
		AuthType:    entities.AuthTypeKey,
		KeyAuthId:   r.UserKeyAuth.Id,
	}

	require.NoError(t, db.CreateWorkspace(ctx, r.UnkeyWorkspace))
	require.NoError(t, db.CreateKeyAuth(ctx, r.UnkeyKeyAuth))
	require.NoError(t, db.InsertApi(ctx, r.UnkeyApi))
	require.NoError(t, db.CreateWorkspace(ctx, r.UserWorkspace))
	require.NoError(t, db.CreateKeyAuth(ctx, r.UserKeyAuth))
	require.NoError(t, db.InsertApi(ctx, r.UserApi))

	r.UnkeyKey = uid.New(16, string(uid.UnkeyPrefix))

	rootKeyEntity := entities.Key{
		Id:             uid.Key(),
		KeyAuthId:      r.UnkeyKeyAuth.Id,
		WorkspaceId:    r.UnkeyWorkspace.Id,
		ForWorkspaceId: r.UserWorkspace.Id,
		Hash:           hash.Sha256(r.UnkeyKey),
		CreatedAt:      time.Now(),
	}

	require.NoError(t, db.CreateKey(ctx, rootKeyEntity))

	return r
}
