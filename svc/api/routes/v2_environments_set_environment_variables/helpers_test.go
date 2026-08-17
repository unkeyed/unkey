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

// rawVar is a stored environment variable row, read directly so tests can assert
// the encrypted value and type that the response omits.
type rawVar struct {
	value       string
	varType     db.AppEnvironmentVariablesType
	description string
}

func listRawVars(t *testing.T, h *testutil.Harness, env seededEnv) map[string]rawVar {
	t.Helper()
	rows, err := db.Query.ListAppEnvVarsByAppAndEnv(context.Background(), h.DB.RO(), db.ListAppEnvVarsByAppAndEnvParams{
		AppID:         env.appID,
		EnvironmentID: env.environmentID,
		IDCursor:      "",
		Limit:         100,
	})
	require.NoError(t, err)

	out := make(map[string]rawVar)
	for _, row := range rows {
		out[row.Key] = rawVar{
			value:       row.Value,
			varType:     row.Type,
			description: row.Description.String,
		}
	}
	return out
}

// seedVar inserts an existing variable directly, bypassing the handler, so tests
// can set up pre-existing state.
func seedVar(t *testing.T, h *testutil.Harness, env seededEnv, key, value string, varType db.AppEnvironmentVariablesType) {
	t.Helper()
	seedVarFull(t, h, env, key, value, varType, "")
}

// seedVarFull is seedVar with an explicit description (empty string stored as NULL).
func seedVarFull(t *testing.T, h *testutil.Harness, env seededEnv, key, value string, varType db.AppEnvironmentVariablesType, description string) {
	t.Helper()
	require.NoError(t, db.Query.InsertAppEnvironmentVariable(context.Background(), h.DB.RW(), db.InsertAppEnvironmentVariableParams{
		ID:            uid.New(uid.EnvironmentVariablePrefix),
		WorkspaceID:   env.workspaceID,
		AppID:         env.appID,
		EnvironmentID: env.environmentID,
		EnvKey:        key,
		Value:         value,
		Type:          varType,
		Description:   sql.NullString{String: description, Valid: description != ""},
		CreatedAt:     1,
	}))
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
