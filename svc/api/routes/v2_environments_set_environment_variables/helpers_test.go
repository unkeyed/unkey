package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_set_environment_variables"
)

// makeRequest builds a set request targeting a seeded environment.
func makeRequest(env seededEnv, vars []openapi.EnvironmentVariableInput) handler.Request {
	return handler.Request{
		Project:     env.projectID,
		App:         env.appID,
		Environment: env.environmentID,
		Variables:   vars,
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
		Description: "Production environment",
	})

	return seededEnv{
		workspaceID:   workspace.ID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}

// rawVar is a stored environment variable row, read directly so tests can assert
// the encrypted value, type, and delete protection that the response omits.
type rawVar struct {
	value            string
	varType          db.AppEnvironmentVariablesType
	description      string
	deleteProtection bool
}

func listRawVars(t *testing.T, h *testutil.Harness, environmentID string) map[string]rawVar {
	t.Helper()
	rows, err := h.DB.RO().QueryContext(context.Background(),
		"SELECT `key`, value, `type`, COALESCE(description, ''), COALESCE(delete_protection, false) FROM app_environment_variables WHERE environment_id = ?",
		environmentID)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	out := make(map[string]rawVar)
	for rows.Next() {
		var key string
		var v rawVar
		require.NoError(t, rows.Scan(&key, &v.value, &v.varType, &v.description, &v.deleteProtection))
		out[key] = v
	}
	require.NoError(t, rows.Err())
	return out
}

// seedVar inserts an existing variable directly, bypassing the handler, so tests
// can set up pre-existing state (including delete-protected rows).
func seedVar(t *testing.T, h *testutil.Harness, env seededEnv, key, value string, varType db.AppEnvironmentVariablesType, deleteProtection bool) {
	t.Helper()
	seedVarFull(t, h, env, key, value, varType, "", deleteProtection)
}

// seedVarFull is seedVar with an explicit description (empty string stored as NULL).
func seedVarFull(t *testing.T, h *testutil.Harness, env seededEnv, key, value string, varType db.AppEnvironmentVariablesType, description string, deleteProtection bool) {
	t.Helper()
	var desc any
	if description != "" {
		desc = description
	}
	_, err := h.DB.RW().ExecContext(context.Background(),
		"INSERT INTO app_environment_variables (id, workspace_id, app_id, environment_id, `key`, value, `type`, description, delete_protection, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		uid.New(uid.EnvironmentVariablePrefix), env.workspaceID, env.appID, env.environmentID, key, value, varType, desc, deleteProtection, 1)
	require.NoError(t, err)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}

func ptr[T any](v T) *T {
	return &v
}
