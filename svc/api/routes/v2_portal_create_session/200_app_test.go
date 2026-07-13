package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_portal_create_session"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestCreateSessionAppMapped verifies that an app-mapped portal config resolves
// its keyspaces from the app's current deployment sentinel config (the keyauth
// policies' keySpaceIds) rather than from the public request.
func TestCreateSessionAppMapped(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:            h.DB,
		Auditlogs:     h.Auditlogs,
		PortalBaseURL: "https://portal.unkey.com",
	}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	now := time.Now().UnixMilli()

	// A keyspace the app's sentinel config verifies keys against at the gateway.
	keySpaceID := uid.New(uid.KeySpacePrefix)
	require.NoError(t, db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:            keySpaceID,
		WorkspaceID:   workspaceID,
		CreatedAtM:    now,
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	}))

	// A project + app + environment + deployment, with the deployment carrying a
	// sentinel config whose keyauth policy references the keyspace above.
	project := h.CreateProject(seed.CreateProjectRequest{
		WorkspaceID: workspaceID,
		Name:        "portal-app-project",
		ID:          uid.New(uid.ProjectPrefix),
		Slug:        "portal-app-project",
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "Portal App",
		Slug:          "portal-app",
		DefaultBranch: "main",
	})
	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Description: "production environment",
	})

	sentinelConfig, err := protojson.Marshal(&frontlinev1.Config{
		Policies: []*frontlinev1.Policy{
			{
				Id:      "pol_keyauth",
				Name:    "keyauth",
				Enabled: true,
				Config: &frontlinev1.Policy_Keyauth{
					Keyauth: &frontlinev1.KeyAuth{
						KeySpaceIds: []string{keySpaceID},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	deploymentID := uid.New(uid.DeploymentPrefix)
	require.NoError(t, db.Query.InsertDeployment(ctx, h.DB.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       "test-" + deploymentID,
		WorkspaceID:                   workspaceID,
		ProjectID:                     project.ID,
		AppID:                         app.ID,
		EnvironmentID:                 environment.ID,
		SentinelConfig:                sentinelConfig,
		EncryptedEnvironmentVariables: []byte{},
		Status:                        db.DeploymentsStatusReady,
		CpuMillicores:                 100,
		MemoryMib:                     128,
		Port:                          8080,
		ShutdownSignal:                db.DeploymentsShutdownSignalSIGTERM,
		UpstreamProtocol:              db.DeploymentsUpstreamProtocolHttp1,
		DeploymentTrigger:             db.DeploymentsTriggerUnknown,
		CreatedAt:                     now,
	}))

	// The app must point at this deployment so createSession can find its config.
	require.NoError(t, db.Query.UpdateAppDeployments(ctx, h.DB.RW(), db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{Valid: true, String: deploymentID},
		IsRolledBack:        false,
		UpdatedAt:           sql.NullInt64{Valid: true, Int64: now},
		AppID:               app.ID,
	}))

	// App-mapped portal config: app_id set, key_auth_id left null.
	require.NoError(t, db.Query.InsertPortalConfig(ctx, h.DB.RW(), db.InsertPortalConfigParams{
		ID:          uid.New(uid.PortalConfigPrefix),
		WorkspaceID: workspaceID,
		Slug:        "app-portal",
		AppID:       sql.NullString{Valid: true, String: app.ID},
		Enabled:     true,
		CreatedAt:   now,
	}))

	rootKey := h.CreateRootKey(workspaceID)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Slug:        "app-portal",
		ExternalId:  "user_app",
		Permissions: []openapi.V2PortalCreateSessionRequestBodyPermissions{"keys:read"},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Data.SessionId)

	// The persisted grant must be scoped to the keyspace resolved from the app's
	// sentinel config, not anything in the request.
	token, err := db.Query.FindValidPortalSessionToken(ctx, h.DB.RO(), db.FindValidPortalSessionTokenParams{
		ID:  res.Body.Data.SessionId,
		Now: time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	var grant struct {
		KeyspaceIDs []string `json:"keyspaceIds"`
		Permissions []string `json:"permissions"`
	}
	require.NoError(t, json.Unmarshal(token.Permissions, &grant))
	require.Equal(t, []string{keySpaceID}, grant.KeyspaceIDs)
}
