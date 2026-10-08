package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_remove_environment_variables"
)

// makeRequest builds a remove request targeting a seeded environment.
func makeRequest(env seededEnv, variables []string) handler.Request {
	return handler.Request{
		Project:     env.projectID,
		App:         env.appID,
		Environment: env.environmentID,
		Variables:   variables,
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
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Payments API",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
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

func listRawVars(t *testing.T, h *testutil.Harness, env seededEnv) map[string]db.ListAppEnvVarsByAppAndEnvRow {
	t.Helper()
	rows, err := db.Query.ListAppEnvVarsByAppAndEnv(context.Background(), h.DB.RO(), db.ListAppEnvVarsByAppAndEnvParams{
		AppID:         env.appID,
		EnvironmentID: env.environmentID,
		IDCursor:      "",
		Limit:         1000,
	})
	require.NoError(t, err)

	out := make(map[string]db.ListAppEnvVarsByAppAndEnvRow, len(rows))
	for _, row := range rows {
		out[row.Key] = row
	}
	return out
}

// seedVar inserts an existing variable directly, bypassing the handler, so tests
// can set up pre-existing state.
func seedVar(t *testing.T, h *testutil.Harness, env seededEnv, key, value string, varType db.AppEnvironmentVariablesType) {
	t.Helper()
	err := db.Query.InsertAppEnvironmentVariable(context.Background(), h.DB.RW(), db.InsertAppEnvironmentVariableParams{
		ID:            uid.New(uid.EnvironmentVariablePrefix),
		WorkspaceID:   env.workspaceID,
		AppID:         env.appID,
		EnvironmentID: env.environmentID,
		EnvKey:        key,
		Value:         value,
		Type:          varType,
		Description:   sql.NullString{},
		CreatedAt:     1,
	})
	require.NoError(t, err)
}

// seedLegacyProtectedVar writes a delete-protected row with raw SQL because no
// query sets delete_protection on environment variables anymore.
func seedLegacyProtectedVar(t *testing.T, h *testutil.Harness, env seededEnv, key, value string, varType db.AppEnvironmentVariablesType) {
	t.Helper()
	_, err := h.DB.RW().ExecContext(context.Background(),
		"INSERT INTO app_environment_variables (id, workspace_id, app_id, environment_id, `key`, value, `type`, delete_protection, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, true, ?)",
		uid.New(uid.EnvironmentVariablePrefix), env.workspaceID, env.appID, env.environmentID, key, value, varType, 1)
	require.NoError(t, err)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
