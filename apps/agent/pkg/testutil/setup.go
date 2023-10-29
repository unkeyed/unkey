package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/database"

	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type resources struct {
	UnkeyWorkspace *workspacesv1.Workspace
	UnkeyApi       *apisv1.Api
	UnkeyKeyAuth   *authenticationv1.KeyAuth
	UserRootKey    string
	UserWorkspace  *workspacesv1.Workspace
	UserApi        *apisv1.Api
	UserKeyAuth    *authenticationv1.KeyAuth
	Database       database.Database
	DatabaseDSN    string
}

func SetupResources(t *testing.T) resources {
	t.Helper()
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		var err error
		dsn, err = createMysqlDatabase(t)
		require.NoError(t, err)
	}

	db, err := database.New(database.Config{
		Logger:    logging.NewNoop(),
		PrimaryUs: dsn,
	})

	require.NoError(t, err)

	r := resources{
		Database:    db,
		DatabaseDSN: dsn,
	}

	r.UnkeyWorkspace = &workspacesv1.Workspace{
		WorkspaceId: uid.Workspace(),
		Name:        "unkey",
		TenantId:    uid.New(16, "tenant"),
		Plan:        workspacesv1.Plan_PLAN_FREE,
	}

	r.UserWorkspace = &workspacesv1.Workspace{
		WorkspaceId: uid.Workspace(),
		Name:        "user",
		TenantId:    uid.New(16, "tenant"),
	}

	r.UnkeyKeyAuth = &authenticationv1.KeyAuth{
		KeyAuthId:   uid.KeyAuth(),
		WorkspaceId: r.UnkeyWorkspace.WorkspaceId,
	}
	r.UserKeyAuth = &authenticationv1.KeyAuth{
		KeyAuthId:   uid.KeyAuth(),
		WorkspaceId: r.UserWorkspace.WorkspaceId,
	}

	r.UnkeyApi = &apisv1.Api{
		ApiId:       uid.Api(),
		Name:        "name",
		WorkspaceId: r.UnkeyWorkspace.WorkspaceId,
		AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
		KeyAuthId:   &r.UnkeyKeyAuth.KeyAuthId,
	}
	r.UserApi = &apisv1.Api{
		ApiId:       uid.Api(),
		Name:        "name",
		WorkspaceId: r.UserWorkspace.WorkspaceId,
		AuthType:    apisv1.AuthType_AUTH_TYPE_KEY,
		KeyAuthId:   &r.UserKeyAuth.KeyAuthId,
	}

	require.NoError(t, db.InsertWorkspace(ctx, r.UnkeyWorkspace))
	require.NoError(t, db.InsertKeyAuth(ctx, r.UnkeyKeyAuth))
	require.NoError(t, db.InsertApi(ctx, r.UnkeyApi))
	require.NoError(t, db.InsertWorkspace(ctx, r.UserWorkspace))
	require.NoError(t, db.InsertKeyAuth(ctx, r.UserKeyAuth))
	require.NoError(t, db.InsertApi(ctx, r.UserApi))

	r.UserRootKey = uid.New(16, string(uid.UnkeyPrefix))

	rootKey := &authenticationv1.Key{
		KeyId:          uid.Key(),
		KeyAuthId:      r.UnkeyKeyAuth.KeyAuthId,
		WorkspaceId:    r.UnkeyWorkspace.WorkspaceId,
		ForWorkspaceId: util.Pointer(r.UserWorkspace.WorkspaceId),
		Hash:           hash.Sha256(r.UserRootKey),
		CreatedAt:      time.Now().UnixMilli(),
	}

	require.NoError(t, db.InsertKey(ctx, rootKey))

	return r
}
