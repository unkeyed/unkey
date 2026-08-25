package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_set_policies"
	"google.golang.org/protobuf/encoding/protojson"
)

func makeRequest(env seededEnv, policies []openapi.Policy) handler.Request {
	return handler.Request{
		Project:     env.projectID,
		App:         env.appID,
		Environment: env.environmentID,
		Policies:    policies,
	}
}

func firewallPolicy(name string, enabled bool) openapi.Policy {
	return openapi.Policy{
		Name:     name,
		Enabled:  enabled,
		Firewall: &openapi.FirewallPolicy{Action: "ACTION_DENY"},
	}
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
// directly, bypassing the handler, so tests can set up pre-existing state
// including policy variants the API cannot create. The environment seeder
// always creates the row (with the legacy "{}" blob).
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

	require.Equal(t, string(blob), readStoredBlob(t, h, env))
}

// readStoredBlob returns the environment's raw sentinel_config bytes.
func readStoredBlob(t *testing.T, h *testutil.Harness, env seededEnv) string {
	t.Helper()
	stored, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(context.Background(), h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
		AppID:         env.appID,
		EnvironmentID: env.environmentID,
	})
	require.NoError(t, err)
	return string(stored.AppRuntimeSetting.SentinelConfig)
}

// readStoredPolicies returns the raw policy documents currently stored for the
// environment, so tests can assert exact wire bytes.
func readStoredPolicies(t *testing.T, h *testutil.Harness, env seededEnv) []json.RawMessage {
	t.Helper()
	var envelope struct {
		Policies []json.RawMessage `json:"policies"`
	}
	require.NoError(t, json.Unmarshal([]byte(readStoredBlob(t, h, env)), &envelope))
	return envelope.Policies
}

// storedPolicyIDs returns the ids of the stored policies in stored order.
func storedPolicyIDs(t *testing.T, h *testutil.Harness, env seededEnv) []string {
	t.Helper()
	stored := readStoredPolicies(t, h, env)
	ids := make([]string, 0, len(stored))
	for _, raw := range stored {
		var doc struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal(raw, &doc))
		ids = append(ids, doc.ID)
	}
	return ids
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}

// jsonContentKeys returns every object key in doc whose value carries
// content. Zero values (false, "", 0, empty array or object) are skipped
// because protojson normalizes them out of stored blobs.
func jsonContentKeys(t *testing.T, doc []byte) map[string]struct{} {
	t.Helper()
	var root any
	require.NoError(t, json.Unmarshal(doc, &root))

	keys := map[string]struct{}{}
	var walk func(node any)
	walk = func(node any) {
		switch v := node.(type) {
		case map[string]any:
			for key, value := range v {
				if !isJSONZero(value) {
					keys[key] = struct{}{}
				}
				walk(value)
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		}
	}
	walk(root)
	return keys
}

func isJSONZero(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case bool:
		return !v
	case string:
		return v == ""
	case float64:
		return v == 0
	case map[string]any:
		return len(v) == 0
	case []any:
		return len(v) == 0
	default:
		return false
	}
}
