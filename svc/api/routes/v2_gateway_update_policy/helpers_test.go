package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/policyconfig"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_gateway_update_policy"
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

// seedSentinelConfig overwrites the seeded runtime settings row's blob
// directly, bypassing the write handler, so tests can set up pre-existing
// state. Policies are encoded with the same policyconfig codec the handler
// uses for reading. The environment seeder always creates the row (with the
// legacy "{}" blob).
func seedSentinelConfig(t *testing.T, h *testutil.Harness, env seededEnv, policies ...*frontlinev1.Policy) {
	t.Helper()
	blob, err := policyconfig.Marshal(policies)
	require.NoError(t, err)

	_, err = h.DB.RW().ExecContext(context.Background(),
		"UPDATE app_runtime_settings SET sentinel_config = ? WHERE app_id = ? AND environment_id = ?",
		blob, env.appID, env.environmentID)
	require.NoError(t, err)

	// MySQL reports 0 affected rows when the value is unchanged, so verify by
	// reading back instead.
	var stored []byte
	err = h.DB.RO().QueryRowContext(context.Background(),
		"SELECT sentinel_config FROM app_runtime_settings WHERE app_id = ? AND environment_id = ?",
		env.appID, env.environmentID).Scan(&stored)
	require.NoError(t, err)
	require.Equal(t, string(blob), string(stored))
}

// firewallPolicy returns an enabled deny-all firewall policy with a fresh id
// and the given match conditions.
func firewallPolicy(name string, match ...*frontlinev1.MatchExpr) *frontlinev1.Policy {
	return &frontlinev1.Policy{
		Id:      uid.New(uid.PolicyPrefix),
		Name:    name,
		Enabled: ptr.P(true),
		Match:   match,
		Config: &frontlinev1.Policy_Firewall{
			Firewall: &frontlinev1.Firewall{Action: frontlinev1.Action_ACTION_DENY},
		},
	}
}

// pathPrefixMatch builds a match expression for paths starting with prefix.
func pathPrefixMatch(prefix string) *frontlinev1.MatchExpr {
	return &frontlinev1.MatchExpr{
		Expr: &frontlinev1.MatchExpr_Path{
			Path: &frontlinev1.PathMatch{
				Path: &frontlinev1.StringMatch{
					Match: &frontlinev1.StringMatch_Prefix{Prefix: prefix},
				},
			},
		},
	}
}

// seedFirewallPolicies stores n firewall policies named "KEBAP 0",
// "KEBAP 1", ... and returns their generated ids in stored order.
func seedFirewallPolicies(t *testing.T, h *testutil.Harness, env seededEnv, n int) []string {
	t.Helper()
	ids := make([]string, 0, n)
	policies := make([]*frontlinev1.Policy, 0, n)
	for i := range n {
		policy := firewallPolicy(fmt.Sprintf("KEBAP %d", i))
		ids = append(ids, policy.GetId())
		policies = append(policies, policy)
	}
	seedSentinelConfig(t, h, env, policies...)
	return ids
}

func readStoredBlob(t *testing.T, h *testutil.Harness, env seededEnv) string {
	t.Helper()
	var blob []byte
	err := h.DB.RO().QueryRowContext(context.Background(),
		"SELECT sentinel_config FROM app_runtime_settings WHERE app_id = ? AND environment_id = ?",
		env.appID, env.environmentID).Scan(&blob)
	require.NoError(t, err)
	return string(blob)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
