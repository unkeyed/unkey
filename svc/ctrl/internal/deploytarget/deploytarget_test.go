package deploytarget_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploytarget"
)

// fixture is one workspace holding two projects, each with an app. The second
// project exists so the cross-project cases have a real mismatched id to pass
// rather than a random one, which would fail as "not found" instead.
type fixture struct {
	database db.Database
	seeder   *seed.Seeder

	project   db.Project
	app       db.App
	env       db.Environment
	otherProj db.Project
	otherApp  db.App
}

func newFixture(t *testing.T, ctx context.Context) fixture {
	t.Helper()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID

	newProject := func(name string) db.Project {
		return seeder.CreateProject(ctx, seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspaceID,
			Name:        name,
			Slug:        slug("project"),
		})
	}
	newApp := func(projectID string) db.App {
		return seeder.CreateApp(ctx, seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspaceID,
			ProjectID:     projectID,
			Name:          "KEBAP",
			Slug:          slug("app"),
			DefaultBranch: "main",
		})
	}

	project := newProject("Target")
	app := newApp(project.ID)
	env := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
	})

	otherProj := newProject("Other")

	return fixture{
		database:  database,
		seeder:    seeder,
		project:   project,
		app:       app,
		env:       env,
		otherProj: otherProj,
		otherApp:  newApp(otherProj.ID),
	}
}

func TestLoadResolvesTarget(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)

	target, err := deploytarget.Load(ctx, f.database, f.project.ID, f.app.ID,
		"production", deploytarget.WithoutSecrets)
	require.NoError(t, err)

	require.Equal(t, f.project.WorkspaceID, target.WorkspaceID)
	require.Equal(t, f.project.ID, target.ProjectID)
	require.Equal(t, f.app.ID, target.AppID)
	require.Equal(t, f.env.ID, target.EnvironmentID)
	require.Equal(t, "production", target.EnvironmentSlug)
	require.Equal(t, "main", target.DefaultBranch)

	// Settings the seeder writes for every (app, environment) pair. They are
	// what the create copies onto the deployment row, so a join that dropped
	// one of the settings tables would still pass every id assertion above.
	require.Equal(t, "Dockerfile", target.Dockerfile.String)
	require.Equal(t, ".", target.DockerContext)
	require.Equal(t, int32(8080), target.Port)
	require.Equal(t, int32(250), target.CpuMillicores)
	require.Equal(t, int32(256), target.MemoryMib)
	require.Equal(t, []byte("{}"), target.SentinelConfig)

	require.Empty(t, target.SecretsBlob, "WithoutSecrets must not fetch env vars")

	// A rebuild names the environment by id instead, which must land on the
	// same row: the two queries differ only in that predicate.
	byID, err := deploytarget.Load(ctx, f.database, f.project.ID, f.app.ID,
		f.env.ID, deploytarget.WithoutSecrets)
	require.NoError(t, err)
	require.Equal(t, target, byID)
}

func TestLoadWithSecretsMarshalsEnvVars(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)

	require.NoError(t, f.database.InsertAppEnvironmentVariable(ctx, db.InsertAppEnvironmentVariableParams{
		ID:            uid.New("envvar"),
		WorkspaceID:   f.project.WorkspaceID,
		AppID:         f.app.ID,
		EnvironmentID: f.env.ID,
		EnvKey:        "MEAL",
		Value:         "KEBAP",
		CreatedAt:     time.Now().UnixMilli(),
	}))

	target, err := deploytarget.Load(ctx, f.database, f.project.ID, f.app.ID,
		"production", deploytarget.WithSecrets)
	require.NoError(t, err)
	require.JSONEq(t, `{"secrets":{"MEAL":"KEBAP"}}`, string(target.SecretsBlob))
}

// TestLoadRejectsMismatchedTriples covers every way the triple can fail to
// line up. All of them answer NotFound: the join decides the triple as a
// whole, and a caller must not learn from the error that an app it cannot
// reach exists.
func TestLoadRejectsMismatchedTriples(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t, ctx)
	workspaceID := f.project.WorkspaceID

	// An environment on the right app with no settings rows, reachable only by
	// inserting directly: the seeder always writes settings alongside.
	bareEnvSlug := "bare"
	bareEnvID := uid.New(uid.EnvironmentPrefix)
	require.NoError(t, f.database.InsertEnvironment(ctx, db.InsertEnvironmentParams{
		ID:          bareEnvID,
		WorkspaceID: workspaceID,
		ProjectID:   f.project.ID,
		AppID:       f.app.ID,
		Slug:        bareEnvSlug,
		Description: "",
		Kind:        mysqltype.EnvironmentKindPreview,
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   sql.NullInt64{Valid: false, Int64: 0},
	}))

	// An environment hanging off the right app but stamped with the other
	// project, which only a bug or a half-finished move could produce.
	strayEnvSlug := "stray"
	require.NoError(t, f.database.InsertEnvironment(ctx, db.InsertEnvironmentParams{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   f.otherProj.ID,
		AppID:       f.app.ID,
		Slug:        strayEnvSlug,
		Description: "",
		Kind:        mysqltype.EnvironmentKindPreview,
		CreatedAt:   time.Now().UnixMilli(),
		UpdatedAt:   sql.NullInt64{Valid: false, Int64: 0},
	}))

	unknownProject := uid.New(uid.ProjectPrefix)
	unknownApp := uid.New(uid.AppPrefix)
	unknownEnv := uid.New(uid.EnvironmentPrefix)

	// An environment under the other project's app, to be named by id from a
	// request that claims the first app.
	foreignEnv := f.seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   f.otherProj.ID,
		AppID:       f.otherApp.ID,
		Slug:        "production",
		Kind:        mysqltype.EnvironmentKindProduction,
	})

	tests := []struct {
		name      string
		projectID string
		appID     string
		env       string
	}{
		{name: "unknown project", projectID: unknownProject, appID: f.app.ID, env: "production"},
		{name: "unknown app", projectID: f.project.ID, appID: unknownApp, env: "production"},
		{name: "app in another project", projectID: f.project.ID, appID: f.otherApp.ID, env: "production"},
		{name: "unknown environment slug", projectID: f.project.ID, appID: f.app.ID, env: "staging"},
		{name: "empty environment", projectID: f.project.ID, appID: f.app.ID, env: ""},
		{name: "environment in another project", projectID: f.project.ID, appID: f.app.ID, env: strayEnvSlug},
		{name: "environment without settings", projectID: f.project.ID, appID: f.app.ID, env: bareEnvSlug},
		{name: "unknown environment id", projectID: f.project.ID, appID: f.app.ID, env: unknownEnv},
		{name: "environment id under another app", projectID: f.project.ID, appID: f.app.ID, env: foreignEnv.ID},
		{name: "environment id without settings", projectID: f.project.ID, appID: f.app.ID, env: bareEnvID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := deploytarget.Load(ctx, f.database, tt.projectID, tt.appID, tt.env, deploytarget.WithoutSecrets)
			require.Error(t, err)

			var terminal *deploytarget.TerminalError
			require.True(t, errors.As(err, &terminal), "want TerminalError, got %v", err)
			require.Equal(t, connect.CodeNotFound, terminal.Code)
			require.Contains(t, terminal.Message, tt.projectID)
			require.Contains(t, terminal.Message, tt.appID)
		})
	}
}

func slug(prefix uid.Prefix) string {
	return strings.ToLower(strings.ReplaceAll(uid.New(prefix), "_", "-"))
}
