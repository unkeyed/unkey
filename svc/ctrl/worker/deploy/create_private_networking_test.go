package deploy_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestCreatePrivateNetworkingDecision guarantees that every created deployment
// joins private DNS while a skipped one does not, and that other apps'
// connections give a deployment no access.
func TestCreatePrivateNetworkingDecision(t *testing.T) {
	for _, test := range []struct {
		name        string
		connections bool
		skip        bool
		wantEnabled bool
	}{
		{name: "no workspace connections still enables DNS", connections: false, skip: false, wantEnabled: true},
		{name: "connections on other apps grant no access", connections: true, skip: false, wantEnabled: true},
		{name: "skipped deployment stays disabled", connections: true, skip: true, wantEnabled: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			h := newCreateHarness(t, ctx)
			if test.connections {
				h.bindOtherApps(t, ctx)
			}
			req := h.imageRequest()
			if test.skip {
				req.Decision = hydrav1.CreateDecision_CREATE_DECISION_SKIP
			}
			deploymentID := uid.New(uid.DeploymentPrefix)
			h.create(t, ctx, deploymentID, req)
			row := h.deployment(t, ctx, deploymentID)
			require.Equal(t, test.wantEnabled, row.Capabilities.PrivateNetworking)
			require.NotContains(t, string(row.EncryptedEnvironmentVariables), "DATABASE_HOST")
			require.Empty(t, h.savedConnections(t, ctx, deploymentID))
		})
	}
}

// TestCreateApprovalReusesPrivateNetworkingDecision guarantees that approving a
// deployment keeps the private networking decision, secrets, and connections
// saved when it was pushed, even after the defaults change.
func TestCreateApprovalReusesPrivateNetworkingDecision(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		name := "legacy disabled deployment stays disabled after a connection is added"
		if enabled {
			name = "keeps enabled and saved connection after defaults change"
		}
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			h := newCreateHarness(t, ctx)
			target := h.newApp(t, ctx)
			deploymentID := uid.New(uid.DeploymentPrefix)
			var connectionID string
			if enabled {
				connectionID = h.bind(t, ctx, h.appID, h.environmentID, target.appID)
				push := h.imageRequest()
				push.Decision = hydrav1.CreateDecision_CREATE_DECISION_AWAIT_APPROVAL
				h.create(t, ctx, deploymentID, push)
			} else {
				h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
					ID: deploymentID, WorkspaceID: h.workspaceID, ProjectID: h.projectID, AppID: h.appID,
					EnvironmentID: h.environmentID, Status: mysqltype.DeploymentsStatusAwaitingApproval,
					Capabilities: mysqltype.DeploymentCapabilities{PrivateNetworking: false},
				})
			}
			pushed := h.deployment(t, ctx, deploymentID)
			require.Equal(t, enabled, pushed.Capabilities.PrivateNetworking)

			if enabled {
				require.Contains(t, string(pushed.EncryptedEnvironmentVariables), "ciphertext-for-DATABASE_HOST")
				require.NoError(t, h.database.UpdateAppConnectionName(ctx, db.UpdateAppConnectionNameParams{
					ID: connectionID, Name: "unkey-invalid", UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				}))
				require.NoError(t, h.database.DeleteConnectionAppTargetByConnectionId(ctx, connectionID))
			} else {
				h.bind(t, ctx, h.appID, h.environmentID, target.appID)
			}
			h.approve(t, ctx, deploymentID)
			resp := h.create(t, ctx, deploymentID, h.imageRequest())
			require.Equal(t, hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, resp.GetOutcome())
			h.awaitDeploy(t, deploymentID)
			approved := h.deployment(t, ctx, deploymentID)
			require.Equal(t, enabled, approved.Capabilities.PrivateNetworking)
			require.Equal(t, pushed.EncryptedEnvironmentVariables, approved.EncryptedEnvironmentVariables)
			if enabled {
				require.Equal(t, []db.ListDeploymentConnectionsByDeploymentIdRow{
					automaticConnection(connectionID, "database", target.appID),
				}, h.savedConnections(t, ctx, deploymentID))
			} else {
				require.NotContains(t, string(approved.EncryptedEnvironmentVariables), "DATABASE_HOST")
				require.Empty(t, h.savedConnections(t, ctx, deploymentID))
			}
		})
	}
}

// TestCreateRebuildDecidesPrivateNetworkingForNewDeployment guarantees that a
// rebuild makes its own private networking decision instead of copying the
// source deployment's.
func TestCreateRebuildDecidesPrivateNetworkingForNewDeployment(t *testing.T) {
	ctx := t.Context()
	h := newCreateHarness(t, ctx)
	target := h.newApp(t, ctx)
	h.bind(t, ctx, h.appID, h.environmentID, target.appID)
	source := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID: h.workspaceID, ProjectID: h.projectID, AppID: h.appID,
		EnvironmentID: h.environmentID, Status: mysqltype.DeploymentsStatusReady,
		Capabilities: mysqltype.DeploymentCapabilities{PrivateNetworking: false},
	})
	require.NoError(t, h.database.UpdateDeploymentImage(ctx, db.UpdateDeploymentImageParams{
		ImageResolved: sql.NullString{Valid: true, String: fixtureImage},
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:            source.ID,
	}))
	deploymentID := uid.New(uid.DeploymentPrefix)
	h.create(t, ctx, deploymentID, h.existingRequest(source.ID, false))
	rebuilt := h.deployment(t, ctx, deploymentID)
	require.True(t, rebuilt.Capabilities.PrivateNetworking)
	require.Contains(t, string(rebuilt.EncryptedEnvironmentVariables), "DATABASE_HOST")
	require.False(t, h.deployment(t, ctx, source.ID).Capabilities.PrivateNetworking)
}

// TestCreateSnapshotsConnections guarantees that each deployment saves the
// connection defaults of its creation time, and that later edits and
// deletions of the defaults leave saved deployments unchanged.
func TestCreateSnapshotsConnections(t *testing.T) {
	ctx := t.Context()
	h := newCreateHarness(t, ctx)
	target := h.newApp(t, ctx)
	connectionID := h.bind(t, ctx, h.appID, h.environmentID, target.appID)

	first := uid.New(uid.DeploymentPrefix)
	h.create(t, ctx, first, h.imageRequest())
	firstSecrets := h.deployment(t, ctx, first).EncryptedEnvironmentVariables
	h.awaitDeploy(t, first)

	pinned := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID: h.workspaceID, ProjectID: h.projectID, AppID: target.appID,
		EnvironmentID: target.environmentID, Status: mysqltype.DeploymentsStatusReady,
	})
	require.NoError(t, h.database.UpdateAppConnectionName(ctx, db.UpdateAppConnectionNameParams{
		ID: connectionID, Name: "pinned-db", UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}))
	require.NoError(t, h.database.UpdateConnectionAppTarget(ctx, db.UpdateConnectionAppTargetParams{
		ConnectionID: connectionID, SelectionMode: db.ConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{Valid: true, String: pinned.ID},
	}))
	second := uid.New(uid.DeploymentPrefix)
	h.create(t, ctx, second, h.imageRequest())
	h.awaitDeploy(t, second)
	require.Contains(t, string(h.deployment(t, ctx, second).EncryptedEnvironmentVariables), "PINNED_DB_HOST")
	require.NotContains(t, string(h.deployment(t, ctx, second).EncryptedEnvironmentVariables), "DATABASE_HOST")

	deleted, err := h.database.DeleteAppConnectionById(ctx, connectionID)
	require.NoError(t, err)
	require.Equal(t, int64(2), deleted, "the default and its app target are deleted together")
	third := uid.New(uid.DeploymentPrefix)
	h.create(t, ctx, third, h.imageRequest())
	h.awaitDeploy(t, third)
	require.True(t, h.deployment(t, ctx, third).Capabilities.PrivateNetworking, "retained snapshots still need new targets to join discovery after every default is removed")
	require.Empty(t, h.savedConnections(t, ctx, third))
	require.NotContains(t, string(h.deployment(t, ctx, third).EncryptedEnvironmentVariables), "_HOST")
	require.Equal(t, firstSecrets, h.deployment(t, ctx, first).EncryptedEnvironmentVariables)

	require.Equal(t, []db.ListDeploymentConnectionsByDeploymentIdRow{
		automaticConnection(connectionID, "database", target.appID),
	}, h.savedConnections(t, ctx, first))
	require.Equal(t, []db.ListDeploymentConnectionsByDeploymentIdRow{
		pinnedConnection(connectionID, "pinned-db", target.appID, pinned.ID),
	}, h.savedConnections(t, ctx, second))
}

func TestCreateSnapshotsMixedConnectionTargets(t *testing.T) {
	ctx := t.Context()
	h := newCreateHarness(t, ctx)
	automatic, environment, pinned := h.newApp(t, ctx), h.newApp(t, ctx), h.newApp(t, ctx)
	target := h.readyDeployment(t, ctx, pinned)
	automaticID := h.bind(t, ctx, h.appID, h.environmentID, automatic.appID)
	environmentID := h.seeder.CreateAppConnection(ctx, seed.CreateAppConnectionRequest{
		WorkspaceID: h.workspaceID, ProjectID: h.projectID, CallerAppID: h.appID, CallerEnvironmentID: h.environmentID,
		TargetAppID: environment.appID, Name: "events",
		SelectionMode:       db.ConnectionAppTargetsSelectionModeEnvironment,
		TargetEnvironmentID: sql.NullString{Valid: true, String: environment.environmentID},
	})
	pinnedID := h.pin(t, ctx, "primary", pinned.appID, target)

	deploymentID := uid.New(uid.DeploymentPrefix)
	h.create(t, ctx, deploymentID, h.imageRequest())
	h.awaitDeploy(t, deploymentID)

	require.Equal(t, []db.ListDeploymentConnectionsByDeploymentIdRow{
		automaticConnection(automaticID, "database", automatic.appID),
		{
			ConnectionID: environmentID, Name: "events", ResourceID: environment.appID,
			SelectionMode:       db.NullDeploymentConnectionAppTargetsSelectionMode{Valid: true, DeploymentConnectionAppTargetsSelectionMode: db.DeploymentConnectionAppTargetsSelectionModeEnvironment},
			TargetEnvironmentID: sql.NullString{Valid: true, String: environment.environmentID},
			TargetDeploymentID:  sql.NullString{Valid: false, String: ""},
		},
		pinnedConnection(pinnedID, "primary", pinned.appID, target),
	}, h.savedConnections(t, ctx, deploymentID))
}

// TestCreateRejectsInvalidConnectionTargetsAtomically guarantees that one
// connection whose target cannot be saved rejects the whole deployment, so no
// row, connection, or pipeline is left behind for the others.
func TestCreateRejectsInvalidConnectionTargetsAtomically(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(t *testing.T, ctx context.Context, h *createHarness, target deployFixture)
	}{
		{name: "stopped pinned target", setup: func(t *testing.T, ctx context.Context, h *createHarness, target deployFixture) {
			stopped := h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
				WorkspaceID: h.workspaceID, ProjectID: h.projectID, AppID: target.appID,
				EnvironmentID: target.environmentID, Status: mysqltype.DeploymentsStatusStopped,
			})
			h.pin(t, ctx, "stopped", target.appID, stopped.ID)
		}},
		{name: "missing pinned target", setup: func(t *testing.T, ctx context.Context, h *createHarness, target deployFixture) {
			h.pin(t, ctx, "missing", target.appID, uid.New(uid.DeploymentPrefix))
		}},
		{name: "pinned target of another app", setup: func(t *testing.T, ctx context.Context, h *createHarness, target deployFixture) {
			h.pin(t, ctx, "other-app", target.appID, h.readyDeployment(t, ctx, h.newApp(t, ctx)))
		}},
		{name: "pinned target in another project", setup: func(t *testing.T, ctx context.Context, h *createHarness, _ deployFixture) {
			other := h.appInNewProject(t, ctx)
			h.pin(t, ctx, "other-project", other.appID, h.readyDeployment(t, ctx, other))
		}},
		{name: "one target pinned for two apps", setup: func(t *testing.T, ctx context.Context, h *createHarness, target deployFixture) {
			shared := h.readyDeployment(t, ctx, target)
			h.pin(t, ctx, "owner", target.appID, shared)
			h.pin(t, ctx, "impostor", h.newApp(t, ctx).appID, shared)
		}},
		{name: "missing app target settings", setup: func(t *testing.T, ctx context.Context, h *createHarness, target deployFixture) {
			connectionID := h.bind(t, ctx, h.appID, h.environmentID, target.appID)
			require.NoError(t, h.database.DeleteConnectionAppTargetByConnectionId(ctx, connectionID))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			h := newCreateHarness(t, ctx)
			h.seeder.CreateAppConnection(ctx, seed.CreateAppConnectionRequest{
				WorkspaceID: h.workspaceID, ProjectID: h.projectID, CallerAppID: h.appID,
				CallerEnvironmentID: h.environmentID, TargetAppID: h.newApp(t, ctx).appID, Name: "valid",
			})
			test.setup(t, ctx, h, h.newApp(t, ctx))

			id := uid.New(uid.DeploymentPrefix)
			_, err := h.tryCreate(ctx, id, h.imageRequest())
			require.Error(t, err)
			_, err = h.database.FindDeploymentById(ctx, id)
			require.True(t, db.IsNotFound(err), "the deployment row must roll back, got %v", err)
			require.Empty(t, h.savedConnections(t, ctx, id))
			h.requireNoDeploy(t, id)
		})
	}
}

func (h *createHarness) approve(t *testing.T, ctx context.Context, deploymentID string) {
	t.Helper()
	result, err := h.database.CompareAndSwapDeploymentStatus(ctx, db.CompareAndSwapDeploymentStatusParams{
		NewStatus: mysqltype.DeploymentsStatusPending, UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID: deploymentID, ExpectedStatus: mysqltype.DeploymentsStatusAwaitingApproval,
	})
	require.NoError(t, err)
	rows, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), rows)
}

type connectionVault struct {
	vault.VaultServiceClient
}

func (connectionVault) EncryptBulk(_ context.Context, req *vaultv1.EncryptBulkRequest) (*vaultv1.EncryptBulkResponse, error) {
	items := make(map[string]*vaultv1.EncryptBulkResponseItem, len(req.GetItems()))
	for key := range req.GetItems() {
		items[key] = &vaultv1.EncryptBulkResponseItem{Encrypted: "ciphertext-for-" + key}
	}
	return &vaultv1.EncryptBulkResponse{Items: items}, nil
}

func (h *createHarness) bindOtherApps(t *testing.T, ctx context.Context) {
	t.Helper()
	caller, target := h.newApp(t, ctx), h.newApp(t, ctx)
	h.bind(t, ctx, caller.appID, caller.environmentID, target.appID)
}

func (h *createHarness) bind(t *testing.T, ctx context.Context, callerAppID, callerEnvironmentID, targetAppID string) string {
	t.Helper()
	return h.seeder.CreateAppConnection(ctx, seed.CreateAppConnectionRequest{
		WorkspaceID: h.workspaceID, ProjectID: h.projectID, CallerAppID: callerAppID,
		CallerEnvironmentID: callerEnvironmentID, TargetAppID: targetAppID, Name: "database",
	})
}

func (h *createHarness) pin(t *testing.T, ctx context.Context, name, targetAppID, deploymentID string) string {
	t.Helper()
	return h.seeder.CreateAppConnection(ctx, seed.CreateAppConnectionRequest{
		WorkspaceID: h.workspaceID, ProjectID: h.projectID, CallerAppID: h.appID, CallerEnvironmentID: h.environmentID,
		TargetAppID: targetAppID, Name: name,
		SelectionMode:      db.ConnectionAppTargetsSelectionModeDeployment,
		TargetDeploymentID: sql.NullString{Valid: true, String: deploymentID},
	})
}

func (h *createHarness) readyDeployment(t *testing.T, ctx context.Context, app deployFixture) string {
	t.Helper()
	return h.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		WorkspaceID: app.workspaceID, ProjectID: app.projectID, AppID: app.appID,
		EnvironmentID: app.environmentID, Status: mysqltype.DeploymentsStatusReady,
	}).ID
}

func (h *createHarness) appInNewProject(t *testing.T, ctx context.Context) deployFixture {
	t.Helper()
	project := h.seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: h.workspaceID, Name: "other", Slug: deploySlug(uid.ProjectPrefix),
	})
	app := h.seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID: uid.New(uid.AppPrefix), WorkspaceID: h.workspaceID, ProjectID: project.ID, Name: "other", Slug: deploySlug(uid.AppPrefix),
	})
	environment := h.seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID: uid.New(uid.EnvironmentPrefix), WorkspaceID: h.workspaceID, ProjectID: project.ID, AppID: app.ID,
		Slug: "production", Kind: mysqltype.EnvironmentKindProduction,
	})
	return deployFixture{seeder: h.seeder, workspaceID: h.workspaceID, projectID: project.ID, appID: app.ID, environmentID: environment.ID}
}

func (h *createHarness) savedConnections(t *testing.T, ctx context.Context, deploymentID string) []db.ListDeploymentConnectionsByDeploymentIdRow {
	t.Helper()
	saved, err := h.database.ListDeploymentConnectionsByDeploymentId(ctx, deploymentID)
	require.NoError(t, err)
	return saved
}

func automaticConnection(connectionID, name, targetAppID string) db.ListDeploymentConnectionsByDeploymentIdRow {
	return db.ListDeploymentConnectionsByDeploymentIdRow{
		ConnectionID: connectionID, Name: name, ResourceID: targetAppID,
		SelectionMode:       db.NullDeploymentConnectionAppTargetsSelectionMode{Valid: true, DeploymentConnectionAppTargetsSelectionMode: db.DeploymentConnectionAppTargetsSelectionModeAutomatic},
		TargetEnvironmentID: sql.NullString{Valid: false, String: ""},
		TargetDeploymentID:  sql.NullString{Valid: false, String: ""},
	}
}

func pinnedConnection(connectionID, name, targetAppID, deploymentID string) db.ListDeploymentConnectionsByDeploymentIdRow {
	return db.ListDeploymentConnectionsByDeploymentIdRow{
		ConnectionID: connectionID, Name: name, ResourceID: targetAppID,
		SelectionMode:       db.NullDeploymentConnectionAppTargetsSelectionMode{Valid: true, DeploymentConnectionAppTargetsSelectionMode: db.DeploymentConnectionAppTargetsSelectionModeDeployment},
		TargetEnvironmentID: sql.NullString{Valid: false, String: ""},
		TargetDeploymentID:  sql.NullString{Valid: true, String: deploymentID},
	}
}
