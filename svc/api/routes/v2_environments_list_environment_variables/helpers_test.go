package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_list_environment_variables"
)

type seededEnv struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

// makeRequest builds a list request targeting a seeded environment.
func makeRequest(env seededEnv, limit *int, cursor *string) handler.Request {
	return handler.Request{
		Project:     env.projectID,
		App:         env.appID,
		Environment: env.environmentID,
		Limit:       limit,
		Cursor:      cursor,
	}
}

func ptr[T any](v T) *T {
	return &v
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

// seedVar encrypts a value through the real test vault and inserts the row
// directly, so the handler's DecryptBulk round-trips the same keyring.
func seedVar(t *testing.T, h *testutil.Harness, env seededEnv, key, value string, varType db.AppEnvironmentVariablesType, description string) {
	t.Helper()
	ctx := context.Background()

	id := uid.New(uid.EnvironmentVariablePrefix)
	encrypted, err := h.Vault.EncryptBulk(ctx, &vaultv1.EncryptBulkRequest{
		Keyring: env.environmentID,
		Items:   map[string]string{id: value},
	})
	require.NoError(t, err)
	item, ok := encrypted.GetItems()[id]
	require.True(t, ok)

	var desc any
	if description != "" {
		desc = description
	}
	_, err = h.DB.RW().ExecContext(ctx,
		"INSERT INTO app_environment_variables (id, workspace_id, app_id, environment_id, `key`, value, `type`, description, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, env.workspaceID, env.appID, env.environmentID, key, item.GetEncrypted(), varType, desc, 1)
	require.NoError(t, err)
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
