package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_update_policy"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func makeRequest(env seededEnv, policyID string) handler.Request {
	var req handler.Request
	req.Project = env.projectID
	req.App = env.appID
	req.Environment = env.environmentID
	req.PolicyId = policyID
	return req
}

type seededEnv struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

func seedEnvironment(t *testing.T, h *testutil.Harness) seededEnv {
	t.Helper()

	workspace := h.Resources().UserWorkspace

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
	})

	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
		DefaultBranch: "main",
	})

	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
		Description: "Production environment",
	})

	return seededEnv{
		workspaceID:   workspace.ID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}

// seedSentinelConfig overwrites the seeded runtime settings row's policy blob
// directly, bypassing the write handler, so tests can set up pre-existing
// state. The environment seeder always
// creates the row (with the legacy "{}" blob).
func seedSentinelConfig(t *testing.T, h *testutil.Harness, env seededEnv, config *frontlinev1.Config) {
	t.Helper()
	blob, err := protojson.Marshal(config)
	require.NoError(t, err)
	ctx := context.Background()
	now := h.Clock.Now().UnixMilli()
	require.NoError(t, db.Query.UpsertAppRuntimeSettingsPolicyConfig(ctx, h.DB.RW(), db.UpsertAppRuntimeSettingsPolicyConfigParams{
		WorkspaceID:    env.workspaceID,
		AppID:          env.appID,
		EnvironmentID:  env.environmentID,
		SentinelConfig: blob,
		CreatedAt:      now,
		UpdatedAt:      sql.NullInt64{Int64: now, Valid: true},
	}))

	stored, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
		AppID:         env.appID,
		EnvironmentID: env.environmentID,
	})
	require.NoError(t, err)
	require.Equal(t, blob, stored.AppRuntimeSetting.SentinelConfig)
}

// seedFirewallPolicies stores n firewall policies and returns their ids in
// stored order.
func seedFirewallPolicies(t *testing.T, h *testutil.Harness, env seededEnv, n int) []string {
	t.Helper()
	ids := make([]string, 0, n)
	policies := make([]*frontlinev1.Policy, 0, n)
	for i := range n {
		id := uid.New(uid.PolicyPrefix)
		ids = append(ids, id)
		policies = append(policies, &frontlinev1.Policy{
			Id:      id,
			Name:    fmt.Sprintf("KEBAP %d", i),
			Enabled: proto.Bool(true),
			Config: &frontlinev1.Policy_Firewall{Firewall: &frontlinev1.Firewall{
				Action: frontlinev1.Action_ACTION_DENY,
			}},
		})
	}
	seedSentinelConfig(t, h, env, &frontlinev1.Config{Policies: policies})
	return ids
}

func readStoredBlob(t *testing.T, h *testutil.Harness, env seededEnv) string {
	t.Helper()
	stored, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(context.Background(), h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
		AppID:         env.appID,
		EnvironmentID: env.environmentID,
	})
	require.NoError(t, err)
	return string(stored.AppRuntimeSetting.SentinelConfig)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
