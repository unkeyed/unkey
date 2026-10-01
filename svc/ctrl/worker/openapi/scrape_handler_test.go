package openapi

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestPersistOpenAPISpecSkipsDeletingEnvironment(t *testing.T) {
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	ctx := t.Context()
	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID
	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "OpenAPI guard",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New(uid.ProjectPrefix), "_", "-")),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "OpenAPI guard",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New(uid.AppPrefix), "_", "-")),
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "preview",
		Kind:        mysqltype.EnvironmentKindPreview,
	})
	deployment := seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		Status:        mysqltype.DeploymentsStatusReady,
	})
	require.NoError(t, database.MarkEnvironmentDeleting(ctx, db.MarkEnvironmentDeletingParams{
		ID:         environment.ID,
		DeletingAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}))

	persisted, err := New(Config{DB: database}).persistOpenAPISpec(ctx, deployment, []byte(`{"openapi":"3.0.0"}`))
	require.NoError(t, err)
	require.False(t, persisted)

	_, err = database.FindOpenApiSpecByDeploymentID(ctx, sql.NullString{Valid: true, String: deployment.ID})
	require.True(t, db.IsNotFound(err))
}

func TestValidateSpecPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "valid simple path", path: "/openapi.json", wantErr: false},
		{name: "valid nested path", path: "/api/v1/openapi.json", wantErr: false},
		{name: "valid path with query", path: "/openapi.json?format=yaml", wantErr: false},

		// Authority-confusion SSRF payloads.
		{name: "at-sign authority confusion", path: "@attacker.com/openapi.json", wantErr: true},
		{name: "at-sign with port", path: "@127.0.0.1:8080/openapi.json", wantErr: true},

		// Scheme-based payloads.
		{name: "absolute URL with https", path: "https://attacker.com/openapi.json", wantErr: true},
		{name: "absolute URL with http", path: "http://attacker.com/openapi.json", wantErr: true},

		// Authority reference payloads.
		{name: "double-slash authority", path: "//attacker.com/openapi.json", wantErr: true},

		// Relative paths (no leading slash).
		{name: "relative path", path: "openapi.json", wantErr: true},
		{name: "empty string", path: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := validateSpecPath(tt.path)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, parsed)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, parsed)
		})
	}
}

// TestReadOpenAPISpecBodyRejectsOversizedBody guarantees tenant-controlled specs
// cannot force ctrl to buffer more than the configured OpenAPI spec limit.
func TestReadOpenAPISpecBodyRejectsOversizedBody(t *testing.T) {
	body := bytes.NewReader(bytes.Repeat([]byte("a"), maxOpenAPISpecBytes+1))

	_, err := readOpenAPISpecBody(body)

	require.ErrorIs(t, err, errOpenAPISpecTooLarge)
}

func TestHTTPClientRefusesRedirects(t *testing.T) {
	internalHit := false
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		internalHit = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(internal.Close)

	deployment := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Redirect(w, &http.Request{}, internal.URL, http.StatusFound)
	}))
	t.Cleanup(deployment.Close)

	client := New(Config{}).httpClient

	resp, err := client.Get(deployment.URL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.False(t, internalHit, "client must not follow redirect to internal host")
}
