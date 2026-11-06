package privatenetwork

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type catalogIDs struct {
	workspace, otherWorkspace, unscheduledWorkspace                                                                                        string
	project, otherProject, foreignProject, unscheduledProject                                                                              string
	caller, target, otherTarget, foreign, unscheduledCaller, unscheduledTarget                                                             string
	callerProd, callerCanary, callerStaging, callerPreview, callerManual, callerPin, callerRollback                                        string
	targetProd, targetCanary, targetPreview, targetManual, foreignProd, unscheduledEnv                                                     string
	callerCanaryOCI, callerPreviewOwn, callerPreviewFork, callerPreviewMissing, callerUnapproved, callerWrongPlatform                      string
	callerManualDep, callerPinDep, callerRollbackDep, callerStagingGit, callerProdLive, callerProdDeploying, callerProdDisabled, callerNew string
	targetCanaryReady, targetCanaryFeature, targetNewProd, targetPreviewOld, targetPreviewNever, targetPreviewAwaiting, targetPreviewFork  string
	targetManualOld, targetManualLatestFailed, targetPinnedStopped, targetLive, targetCanaryRecreated, foreignLive                         string
	unscheduledCaller1, unscheduledCaller2                                                                                                 string
	productionConnection, selfProdConnection, crossWorkspaceConnection, otherProjectConnection, canaryConnection, canaryAutoConnection     string
	selfCanaryToProdConnection, previewConnection, selfPreviewConnection, manualConnection, reservedPrefixConnection, pinnedConnection     string
	reservedSelfConnection, explicitProdConnection, stagingAutoConnection, unscheduledConnection                                           string
}

func newCatalogIDs() catalogIDs {
	return catalogIDs{
		workspace: "", otherWorkspace: "", unscheduledWorkspace: "",
		project: uid.New(uid.ProjectPrefix), otherProject: uid.New(uid.ProjectPrefix), foreignProject: uid.New(uid.ProjectPrefix), unscheduledProject: uid.New(uid.ProjectPrefix),
		caller: uid.New(uid.AppPrefix), target: uid.New(uid.AppPrefix), otherTarget: uid.New(uid.AppPrefix), foreign: uid.New(uid.AppPrefix), unscheduledCaller: uid.New(uid.AppPrefix), unscheduledTarget: uid.New(uid.AppPrefix),
		callerProd: uid.New(uid.EnvironmentPrefix), callerCanary: uid.New(uid.EnvironmentPrefix), callerStaging: uid.New(uid.EnvironmentPrefix), callerPreview: uid.New(uid.EnvironmentPrefix), callerManual: uid.New(uid.EnvironmentPrefix), callerPin: uid.New(uid.EnvironmentPrefix), callerRollback: uid.New(uid.EnvironmentPrefix),
		targetProd: uid.New(uid.EnvironmentPrefix), targetCanary: uid.New(uid.EnvironmentPrefix), targetPreview: uid.New(uid.EnvironmentPrefix), targetManual: uid.New(uid.EnvironmentPrefix), foreignProd: uid.New(uid.EnvironmentPrefix), unscheduledEnv: uid.New(uid.EnvironmentPrefix),
		callerCanaryOCI: uid.New(uid.DeploymentPrefix), callerPreviewOwn: uid.New(uid.DeploymentPrefix), callerPreviewFork: uid.New(uid.DeploymentPrefix), callerPreviewMissing: uid.New(uid.DeploymentPrefix), callerUnapproved: uid.New(uid.DeploymentPrefix), callerWrongPlatform: uid.New(uid.DeploymentPrefix),
		callerManualDep: uid.New(uid.DeploymentPrefix), callerPinDep: uid.New(uid.DeploymentPrefix), callerRollbackDep: uid.New(uid.DeploymentPrefix), callerStagingGit: uid.New(uid.DeploymentPrefix), callerProdLive: uid.New(uid.DeploymentPrefix), callerProdDeploying: uid.New(uid.DeploymentPrefix), callerProdDisabled: uid.New(uid.DeploymentPrefix), callerNew: uid.New(uid.DeploymentPrefix),
		targetCanaryReady: uid.New(uid.DeploymentPrefix), targetCanaryFeature: uid.New(uid.DeploymentPrefix), targetNewProd: uid.New(uid.DeploymentPrefix), targetPreviewOld: uid.New(uid.DeploymentPrefix), targetPreviewNever: uid.New(uid.DeploymentPrefix), targetPreviewAwaiting: uid.New(uid.DeploymentPrefix), targetPreviewFork: uid.New(uid.DeploymentPrefix),
		targetManualOld: uid.New(uid.DeploymentPrefix), targetManualLatestFailed: uid.New(uid.DeploymentPrefix), targetPinnedStopped: uid.New(uid.DeploymentPrefix), targetLive: uid.New(uid.DeploymentPrefix), targetCanaryRecreated: uid.New(uid.DeploymentPrefix), foreignLive: uid.New(uid.DeploymentPrefix),
		unscheduledCaller1: uid.New(uid.DeploymentPrefix), unscheduledCaller2: uid.New(uid.DeploymentPrefix),
		productionConnection: uid.New(uid.ConnectionPrefix), selfProdConnection: uid.New(uid.ConnectionPrefix), crossWorkspaceConnection: uid.New(uid.ConnectionPrefix), otherProjectConnection: uid.New(uid.ConnectionPrefix), canaryConnection: uid.New(uid.ConnectionPrefix), canaryAutoConnection: uid.New(uid.ConnectionPrefix),
		selfCanaryToProdConnection: uid.New(uid.ConnectionPrefix), previewConnection: uid.New(uid.ConnectionPrefix), selfPreviewConnection: uid.New(uid.ConnectionPrefix), manualConnection: uid.New(uid.ConnectionPrefix), reservedPrefixConnection: uid.New(uid.ConnectionPrefix), pinnedConnection: uid.New(uid.ConnectionPrefix),
		reservedSelfConnection: uid.New(uid.ConnectionPrefix), explicitProdConnection: uid.New(uid.ConnectionPrefix), stagingAutoConnection: uid.New(uid.ConnectionPrefix), unscheduledConnection: uid.New(uid.ConnectionPrefix),
	}
}

func (ids catalogIDs) enabledReplicas() []string {
	return []string{
		ids.callerProdDeploying, ids.callerProdLive, ids.callerCanaryOCI,
		ids.callerPreviewOwn, ids.callerPreviewFork, ids.callerPreviewMissing,
		ids.callerManualDep, ids.callerPinDep, ids.callerRollbackDep, ids.callerStagingGit,
		ids.targetLive, ids.targetCanaryReady, ids.targetCanaryFeature, ids.targetNewProd, ids.targetPreviewOld,
		ids.targetPreviewNever, ids.targetPreviewFork, ids.targetManualOld,
	}
}

type catalogRegions struct {
	platform, region, region2, otherRegion string
}

func createCatalogRegions(t *testing.T, database db.Database) catalogRegions {
	t.Helper()
	platform := strings.ToLower(uid.New("pf"))
	regions := catalogRegions{platform: platform, region: uid.New(uid.RegionPrefix), region2: uid.New(uid.RegionPrefix), otherRegion: uid.New(uid.RegionPrefix)}
	for _, region := range []db.UpsertRegionParams{
		{ID: regions.region, Name: "connection-region", Platform: platform},
		{ID: regions.region2, Name: "connection-region-2", Platform: platform},
		{ID: regions.otherRegion, Name: "connection-region-other", Platform: "other-" + platform},
	} {
		require.NoError(t, database.UpsertRegion(t.Context(), region))
	}
	t.Cleanup(func() {
		_, err := database.DeleteDeploymentTopologiesByRegionIds(context.Background(), db.DeleteDeploymentTopologiesByRegionIdsParams{
			RegionIds: []string{regions.region, regions.region2, regions.otherRegion}, Limit: 10_000,
		})
		require.NoError(t, err)
	})
	return regions
}

type catalogFixture struct {
	catalogIDs
	catalogRegions
	seeder *seed.Seeder
}

func newCatalogFixture(t *testing.T, database db.Database, regions catalogRegions) *catalogFixture {
	t.Helper()
	return &catalogFixture{catalogIDs: newCatalogIDs(), catalogRegions: regions, seeder: seed.New(t, database, nil)}
}

func (f *catalogFixture) database() db.Database {
	return f.seeder.DB
}

type catalogDeployment struct {
	id, workspace, project, app, environment string
	source                                   db.DeploymentsSource
	branch, fork                             string
	status                                   mysqltype.DeploymentsStatus
	desired                                  mysqltype.DeploymentsDesiredState
	firstReady                               bool
	disabled                                 bool
	created                                  int64
	port                                     int32
	regions                                  []string
}

func (f *catalogFixture) deploy(t *testing.T, d catalogDeployment) {
	t.Helper()
	ctx := t.Context()
	workspace, project := d.workspace, d.project
	if workspace == "" {
		workspace, project = f.workspace, f.project
	}
	source := d.source
	if source == "" {
		source = db.DeploymentsSourceGit
	}
	f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID: d.id, WorkspaceID: workspace, ProjectID: project, AppID: d.app, EnvironmentID: d.environment,
		Status: d.status, CreatedAt: d.created, FirstReadyAt: sql.NullInt64{Int64: 1, Valid: d.firstReady}, DesiredState: d.desired,
		GitBranch:              sql.NullString{String: d.branch, Valid: d.branch != ""},
		ForkRepositoryFullName: sql.NullString{String: d.fork, Valid: d.fork != ""},
		Source:                 source,
		Capabilities:           mysqltype.DeploymentCapabilities{PrivateNetworking: !d.disabled},
		Port:                   d.port,
	})
	for _, region := range d.regions {
		require.NoError(t, f.database().InsertDeploymentTopology(ctx, db.InsertDeploymentTopologyParams{
			WorkspaceID: workspace, DeploymentID: d.id, RegionID: region,
			AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1,
			DesiredStatus: db.DeploymentTopologyDesiredStatusRunning, CreatedAt: 1,
		}))
	}
}

func (f *catalogFixture) seedDeployments(t *testing.T) {
	t.Helper()
	ctx := t.Context()
	f.workspace = f.seeder.CreateWorkspace(ctx).ID
	f.otherWorkspace = f.seeder.CreateWorkspace(ctx).ID
	f.unscheduledWorkspace = f.seeder.CreateWorkspaceInNamespace(ctx, "").ID
	for _, project := range []struct{ workspace, id string }{
		{f.workspace, f.project}, {f.workspace, f.otherProject},
		{f.otherWorkspace, f.foreignProject}, {f.unscheduledWorkspace, f.unscheduledProject},
	} {
		f.seeder.CreateProject(ctx, seed.CreateProjectRequest{ID: project.id, WorkspaceID: project.workspace, Name: project.id, Slug: project.id})
	}
	for _, app := range []struct{ workspace, project, id, slug string }{
		{f.workspace, f.project, f.caller, "caller"},
		{f.workspace, f.project, f.target, "target"},
		{f.workspace, f.project, f.otherTarget, "other-target"},
		{f.otherWorkspace, f.foreignProject, f.foreign, "caller"},
		{f.unscheduledWorkspace, f.unscheduledProject, f.unscheduledCaller, "caller"},
		{f.unscheduledWorkspace, f.unscheduledProject, f.unscheduledTarget, "target"},
	} {
		f.seeder.CreateApp(ctx, seed.CreateAppRequest{ID: app.id, WorkspaceID: app.workspace, ProjectID: app.project, Name: app.slug, Slug: app.slug, SourceType: db.AppsSourceTypeGit})
	}

	production, preview := mysqltype.EnvironmentKindProduction, mysqltype.EnvironmentKindPreview
	for _, environment := range []struct {
		workspace, project, app, id, slug string
		kind                              mysqltype.EnvironmentKind
	}{
		{f.workspace, f.project, f.caller, f.callerProd, "production", production},
		{f.workspace, f.project, f.caller, f.callerCanary, "canary", production},
		{f.workspace, f.project, f.caller, f.callerStaging, "staging", production},
		{f.workspace, f.project, f.caller, f.callerPreview, "preview", preview},
		{f.workspace, f.project, f.caller, f.callerManual, "manual", preview},
		{f.workspace, f.project, f.caller, f.callerPin, "pin", preview},
		{f.workspace, f.project, f.caller, f.callerRollback, "rollback", preview},
		{f.workspace, f.project, f.target, f.targetProd, "production", production},
		{f.workspace, f.project, f.target, f.targetCanary, "canary", production},
		{f.workspace, f.project, f.target, f.targetPreview, "preview", preview},
		{f.workspace, f.project, f.target, f.targetManual, "manual", preview},
		{f.otherWorkspace, f.foreignProject, f.foreign, f.foreignProd, "production", production},
		{f.unscheduledWorkspace, f.unscheduledProject, f.unscheduledCaller, f.unscheduledEnv, "preview", preview},
	} {
		f.createEnvironment(t, environment.workspace, environment.project, environment.app, environment.id, environment.slug, environment.kind)
	}

	oci := db.DeploymentsSourceOci
	running, stopped := mysqltype.DeploymentsDesiredStateRunning, mysqltype.DeploymentsDesiredStateStopped
	ready, awaiting := mysqltype.DeploymentsStatusReady, mysqltype.DeploymentsStatusAwaitingApproval
	for _, d := range []catalogDeployment{
		{id: f.callerCanaryOCI, app: f.caller, environment: f.callerCanary, source: oci, status: ready, desired: running, firstReady: true, created: 11},
		{id: f.callerPreviewOwn, app: f.caller, environment: f.callerPreview, branch: "feature", status: ready, desired: running, firstReady: true, created: 12},
		{id: f.callerPreviewFork, app: f.caller, environment: f.callerPreview, branch: "feature", fork: "fork/repo", status: ready, desired: running, firstReady: true, created: 13},
		{id: f.callerPreviewMissing, app: f.caller, environment: f.callerPreview, branch: "isolated", status: ready, desired: running, firstReady: true, created: 13},
		{id: f.callerUnapproved, app: f.caller, environment: f.callerPreview, branch: "feature", status: awaiting, desired: running, created: 14},
		{id: f.callerWrongPlatform, app: f.caller, environment: f.callerPreview, branch: "feature", status: ready, desired: running, firstReady: true, created: 15, regions: []string{f.otherRegion}},
		{id: f.callerManualDep, app: f.caller, environment: f.callerManual, branch: "feature", status: ready, desired: running, firstReady: true, created: 16},
		{id: f.callerPinDep, app: f.caller, environment: f.callerPin, branch: "feature", status: ready, desired: running, firstReady: true, created: 17},
		{id: f.callerRollbackDep, app: f.caller, environment: f.callerRollback, source: oci, status: ready, desired: running, firstReady: true, created: 18},
		{id: f.callerStagingGit, app: f.caller, environment: f.callerStaging, branch: "feature", status: ready, desired: running, firstReady: true, created: 19},

		{id: f.targetCanaryReady, app: f.target, environment: f.targetCanary, source: oci, status: ready, desired: running, firstReady: true, created: 21},
		{id: f.targetCanaryFeature, app: f.target, environment: f.targetCanary, branch: "feature", status: ready, desired: running, firstReady: true, created: 35},
		{id: f.targetNewProd, app: f.target, environment: f.targetProd, branch: "main", status: ready, desired: running, firstReady: true, created: 99},
		{id: f.targetPreviewOld, app: f.target, environment: f.targetPreview, branch: "feature", status: ready, desired: running, firstReady: true, created: 30},
		{id: f.targetPreviewNever, app: f.target, environment: f.targetPreview, branch: "feature", status: ready, desired: running, created: 31},
		{id: f.targetPreviewAwaiting, app: f.target, environment: f.targetPreview, branch: "feature", status: awaiting, desired: running, created: 32},
		{id: f.targetPreviewFork, app: f.target, environment: f.targetPreview, branch: "feature", fork: "fork/repo", status: ready, desired: running, firstReady: true, created: 33},
		{id: f.targetManualOld, app: f.target, environment: f.targetManual, branch: "manual", status: ready, desired: running, firstReady: true, created: 40},
		{id: f.targetManualLatestFailed, app: f.target, environment: f.targetManual, branch: "manual", status: mysqltype.DeploymentsStatusFailed, desired: running, firstReady: true, created: 41},
		{id: f.targetPinnedStopped, app: f.target, environment: f.targetPreview, branch: "pin", status: mysqltype.DeploymentsStatusStopped, desired: stopped, firstReady: true, created: 50},

		{id: f.callerProdLive, app: f.caller, environment: f.callerProd, branch: "main", status: ready, firstReady: true, created: 9, port: 4000, regions: []string{f.region, f.region2}},
		{id: f.callerProdDeploying, app: f.caller, environment: f.callerProd, branch: "main", status: mysqltype.DeploymentsStatusDeploying, created: 10, port: 4000},
		{id: f.callerProdDisabled, app: f.caller, environment: f.callerProd, branch: "main", status: ready, firstReady: true, disabled: true, created: 8},
		{id: f.targetLive, app: f.target, environment: f.targetProd, branch: "main", status: ready, firstReady: true, created: 20, port: 7946},
		{id: f.foreignLive, workspace: f.otherWorkspace, project: f.foreignProject, app: f.foreign, environment: f.foreignProd, branch: "main", status: ready, firstReady: true, disabled: true, created: 60},
		{id: f.unscheduledCaller1, workspace: f.unscheduledWorkspace, project: f.unscheduledProject, app: f.unscheduledCaller, environment: f.unscheduledEnv, status: ready, created: 1},
		{id: f.unscheduledCaller2, workspace: f.unscheduledWorkspace, project: f.unscheduledProject, app: f.unscheduledCaller, environment: f.unscheduledEnv, status: ready, created: 1},
	} {
		if d.regions == nil {
			d.regions = []string{f.region}
		}
		f.deploy(t, d)
	}
	f.setLive(t, f.caller, f.callerProdLive)
	f.setLive(t, f.target, f.targetLive)
	f.setLive(t, f.foreign, f.foreignLive)
}

func (f *catalogFixture) createEnvironment(t *testing.T, workspace, project, app, id, slug string, kind mysqltype.EnvironmentKind) {
	t.Helper()
	f.seeder.CreateEnvironment(t.Context(), seed.CreateEnvironmentRequest{
		ID: id, WorkspaceID: workspace, ProjectID: project, AppID: app, Slug: slug, Kind: kind,
	})
}

type savedConnection struct {
	id, name, environment, target string
	mode                          db.DeploymentConnectionAppTargetsSelectionMode
	targetEnvironment             string
	targetDeployment              string
}

func (f *catalogFixture) save(t *testing.T, deploymentID string, c savedConnection) {
	t.Helper()
	f.seeder.CreateDeploymentConnection(t.Context(), seed.CreateDeploymentConnectionRequest{
		DeploymentID: deploymentID, ConnectionID: c.id, WorkspaceID: f.workspace, ProjectID: f.project,
		CallerAppID: f.caller, CallerEnvironmentID: c.environment, TargetAppID: c.target, Name: c.name,
		SelectionMode:       c.mode,
		TargetEnvironmentID: sql.NullString{String: c.targetEnvironment, Valid: c.targetEnvironment != ""},
		TargetDeploymentID:  sql.NullString{String: c.targetDeployment, Valid: c.targetDeployment != ""},
	})
}

func (f *catalogFixture) seedConnections(t *testing.T) {
	t.Helper()
	callers := map[string][]string{
		f.callerProd:     {f.callerProdDeploying, f.callerProdLive, f.callerProdDisabled},
		f.callerCanary:   {f.callerCanaryOCI},
		f.callerPreview:  {f.callerPreviewOwn, f.callerPreviewFork, f.callerPreviewMissing, f.callerUnapproved, f.callerWrongPlatform},
		f.callerManual:   {f.callerManualDep},
		f.callerPin:      {f.callerPinDep},
		f.callerRollback: {f.callerRollbackDep},
		f.callerStaging:  {f.callerStagingGit},
	}
	automatic := db.DeploymentConnectionAppTargetsSelectionModeAutomatic
	environment := db.DeploymentConnectionAppTargetsSelectionModeEnvironment
	for _, c := range []savedConnection{
		{id: f.productionConnection, name: "target", environment: f.callerProd, target: f.target, mode: automatic},
		{id: f.selfProdConnection, name: "caller", environment: f.callerProd, target: f.caller, mode: automatic},
		{id: f.crossWorkspaceConnection, name: "foreign", environment: f.callerProd, target: f.foreign, mode: automatic},
		{id: f.canaryConnection, name: "target", environment: f.callerCanary, target: f.target, mode: environment, targetEnvironment: f.targetCanary},
		{id: f.canaryAutoConnection, name: "other-target", environment: f.callerCanary, target: f.otherTarget, mode: automatic},
		{id: f.selfCanaryToProdConnection, name: "caller", environment: f.callerCanary, target: f.caller, mode: environment, targetEnvironment: f.callerProd},
		{id: f.previewConnection, name: "target", environment: f.callerPreview, target: f.target, mode: automatic},
		{id: f.selfPreviewConnection, name: "caller", environment: f.callerPreview, target: f.caller, mode: automatic},
		{id: f.manualConnection, name: "target", environment: f.callerManual, target: f.target, mode: environment, targetEnvironment: f.targetManual},
		{id: f.reservedPrefixConnection, name: "unkey-internal", environment: f.callerManual, target: f.otherTarget, mode: automatic},
		{id: f.pinnedConnection, name: "target", environment: f.callerPin, target: f.target, mode: db.DeploymentConnectionAppTargetsSelectionModeDeployment, targetDeployment: f.targetPinnedStopped},
		{id: f.reservedSelfConnection, name: "self", environment: f.callerPin, target: f.otherTarget, mode: automatic},
		{id: f.explicitProdConnection, name: "target", environment: f.callerRollback, target: f.target, mode: environment, targetEnvironment: f.targetProd},
		{id: f.stagingAutoConnection, name: "caller", environment: f.callerStaging, target: f.target, mode: automatic},
	} {
		for _, caller := range callers[c.environment] {
			f.save(t, caller, c)
		}
	}
	for _, caller := range callers[f.callerProd] {
		f.seeder.CreateDeploymentConnection(t.Context(), seed.CreateDeploymentConnectionRequest{
			DeploymentID: caller, ConnectionID: f.otherProjectConnection, WorkspaceID: f.workspace, ProjectID: f.otherProject,
			CallerAppID: f.caller, CallerEnvironmentID: f.callerProd, TargetAppID: f.otherTarget, Name: "leak",
			SelectionMode: automatic,
		})
	}
	for _, caller := range []string{f.unscheduledCaller1, f.unscheduledCaller2} {
		f.seeder.CreateDeploymentConnection(t.Context(), seed.CreateDeploymentConnectionRequest{
			DeploymentID: caller, ConnectionID: f.unscheduledConnection, WorkspaceID: f.unscheduledWorkspace, ProjectID: f.unscheduledProject,
			CallerAppID: f.unscheduledCaller, CallerEnvironmentID: f.unscheduledEnv, TargetAppID: f.unscheduledTarget, Name: "target",
			SelectionMode: automatic,
		})
	}
}

func (f *catalogFixture) setLive(t *testing.T, appID, deploymentID string) {
	t.Helper()
	require.NoError(t, f.database().UpdateAppDeployments(t.Context(), db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{String: deploymentID, Valid: deploymentID != ""}, AppID: appID,
	}))
}

func (f *catalogFixture) setStatus(t *testing.T, deploymentID string, status mysqltype.DeploymentsStatus) {
	t.Helper()
	require.NoError(t, f.database().UpdateDeploymentStatus(t.Context(), db.UpdateDeploymentStatusParams{Status: status, UpdatedAt: sql.NullInt64{}, ID: deploymentID}))
}

func (f *catalogFixture) setDesiredState(t *testing.T, deploymentID string, state mysqltype.DeploymentsDesiredState) {
	t.Helper()
	require.NoError(t, f.database().UpdateDeploymentDesiredState(t.Context(), db.UpdateDeploymentDesiredStateParams{DesiredState: state, UpdatedAt: sql.NullInt64{}, ID: deploymentID}))
}

func (f *catalogFixture) replicas(t *testing.T, platform string) []db.ListPrivateNetworkReplicasRow {
	t.Helper()
	deploymentIDs := []string{
		f.callerCanaryOCI, f.callerPreviewOwn, f.callerPreviewFork, f.callerPreviewMissing,
		f.callerUnapproved, f.callerWrongPlatform, f.callerManualDep, f.callerPinDep,
		f.callerRollbackDep, f.callerStagingGit, f.callerProdLive, f.callerProdDeploying,
		f.callerProdDisabled, f.callerNew,
		f.targetCanaryReady, f.targetCanaryFeature, f.targetNewProd, f.targetPreviewOld,
		f.targetPreviewNever, f.targetPreviewAwaiting, f.targetPreviewFork, f.targetManualOld,
		f.targetManualLatestFailed, f.targetPinnedStopped, f.targetLive, f.targetCanaryRecreated,
		f.foreignLive, f.unscheduledCaller1, f.unscheduledCaller2,
	}
	var replicas []db.ListPrivateNetworkReplicasRow
	for chunk := range slices.Chunk(deploymentIDs, 3) {
		rows, err := f.database().ListPrivateNetworkReplicas(t.Context(), db.ListPrivateNetworkReplicasParams{DeploymentIds: chunk, Platform: platform})
		require.NoError(t, err)
		replicas = append(replicas, rows...)
	}
	return replicas
}

func (f *catalogFixture) replicaIDs(t *testing.T) []string {
	t.Helper()
	replicas := f.replicas(t, f.platform)
	deploymentIDs := make([]string, 0, len(replicas))
	for _, replica := range replicas {
		require.Equal(t, f.workspace, replica.WorkspaceID, "a workspace without a Kubernetes namespace publishes no replicas")
		require.NotEmpty(t, replica.AppSlug)
		deploymentIDs = append(deploymentIDs, replica.DeploymentID)
	}
	return deploymentIDs
}

func (f *catalogFixture) connectionsOn(t *testing.T, platform string) map[string]*ctrlv1.PrivateNetworkConnection {
	t.Helper()
	var callerIDs []string
	for _, caller := range f.replicas(t, platform) {
		callerIDs = append(callerIDs, caller.DeploymentID)
	}
	var rows []*ctrlv1.PrivateNetworkConnection
	for callerPage := range slices.Chunk(callerIDs, 3) {
		for _, page := range f.pages(t, platform, callerPage, 1) {
			rows = append(rows, page...)
		}
	}
	byConnectionCaller := make(map[string]*ctrlv1.PrivateNetworkConnection, len(rows))
	for _, row := range rows {
		key := row.GetConnectionId() + "/" + row.GetCallerDeploymentId()
		require.NotContains(t, byConnectionCaller, key, "one row per connection and caller deployment, even across regions")
		byConnectionCaller[key] = row
		require.Equal(t, row.GetTargetDeploymentId() == "", row.GetTargetPort() == 0, "port is known exactly when %s is resolved", key)
		require.Equal(t, f.workspace, row.GetWorkspaceId())
		require.Equal(t, f.project, row.GetProjectId())
	}
	return byConnectionCaller
}

func (f *catalogFixture) connections(t *testing.T) map[string]*ctrlv1.PrivateNetworkConnection {
	t.Helper()
	return f.connectionsOn(t, f.platform)
}

func (f *catalogFixture) pages(t *testing.T, platform string, callerIDs []string, pageSize int32) [][]*ctrlv1.PrivateNetworkConnection {
	t.Helper()
	params := db.ListPrivateNetworkConnectionSelectionsParams{
		CallerDeploymentIds: callerIDs, AfterCallerDeploymentID: "", AfterConnectionID: "", Limit: pageSize,
	}
	var pages [][]*ctrlv1.PrivateNetworkConnection
	for {
		page, err := listConnections(t.Context(), db.NewQueries(f.database().RW()), params, platform)
		require.NoError(t, err)
		require.LessOrEqual(t, len(page), int(pageSize))
		pages = append(pages, page)
		if len(page) < int(pageSize) {
			return pages
		}
		last := page[len(page)-1]
		params.AfterCallerDeploymentID, params.AfterConnectionID = last.GetCallerDeploymentId(), last.GetConnectionId()
	}
}

// TestListConnections guarantees that the private network catalog publishes
// exactly the connections saved by active caller deployments created with
// private networking, each resolved to a ready target on the same platform
// under the rules of its saved selection, and that mutable connection defaults
// never change what a running deployment reaches.
func TestListConnections(t *testing.T) {
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	regions := createCatalogRegions(t, database)
	newCatalog := func(t *testing.T) *catalogFixture {
		t.Helper()
		f := newCatalogFixture(t, database, regions)
		f.seedDeployments(t)
		f.seedConnections(t)
		return f
	}

	t.Run("replicas follow the decision persisted at creation", func(t *testing.T) {
		f := newCatalogFixture(t, database, regions)
		f.seedDeployments(t)
		require.ElementsMatch(t, f.enabledReplicas(), f.replicaIDs(t), "replicas follow the decision persisted at creation, even before the workspace has connections")
		f.seedConnections(t)
		require.ElementsMatch(t, f.enabledReplicas(), f.replicaIDs(t), "every active deployment created with private networking publishes its replicas")
	})

	t.Run("one row per connection and caller", func(t *testing.T) {
		f := newCatalog(t)
		require.Len(t, f.connections(t), 11)
	})

	t.Run("connection pages split one caller without losing or repeating rows", func(t *testing.T) {
		f := newCatalog(t)
		callers := []string{f.callerCanaryOCI}
		whole := f.pages(t, f.platform, callers, 100)
		require.Len(t, whole, 1)
		require.Len(t, whole[0], 2, "the caller saved two connections")

		pages := f.pages(t, f.platform, callers, 1)
		require.Len(t, pages, 3, "two full pages for the same caller, then an empty page")
		var split []*ctrlv1.PrivateNetworkConnection
		for _, page := range pages {
			split = append(split, page...)
		}
		require.Equal(t, whole[0], split)
	})

	t.Run("one page resolves each selection like a page of one connection", func(t *testing.T) {
		f := newCatalog(t)
		whole := f.pages(t, f.platform, f.replicaIDs(t), 100)
		require.Len(t, whole, 1)
		byConnectionCaller := make(map[string]*ctrlv1.PrivateNetworkConnection, len(whole[0]))
		for _, row := range whole[0] {
			byConnectionCaller[row.GetConnectionId()+"/"+row.GetCallerDeploymentId()] = row
		}
		require.Equal(t, f.connections(t), byConnectionCaller, "same-branch callers on different forks and callers without a match must not share or shift targets")
	})

	t.Run("callers publish from deploying through ready", func(t *testing.T) {
		f := newCatalog(t)
		for _, status := range []mysqltype.DeploymentsStatus{
			mysqltype.DeploymentsStatusDeploying, mysqltype.DeploymentsStatusNetwork,
			mysqltype.DeploymentsStatusFinalizing, mysqltype.DeploymentsStatusReady,
		} {
			f.setStatus(t, f.callerProdDeploying, status)
			current := f.connections(t)
			require.Contains(t, current, f.productionConnection+"/"+f.callerProdDeploying, status)
			require.Equal(t, f.targetLive, current[f.productionConnection+"/"+f.callerProdDeploying].GetTargetDeploymentId(), status)
			require.Contains(t, f.replicaIDs(t), f.callerProdDeploying, status)
		}
	})

	t.Run("targets resolve only when ready", func(t *testing.T) {
		f := newCatalog(t)
		for _, status := range []mysqltype.DeploymentsStatus{
			mysqltype.DeploymentsStatusDeploying, mysqltype.DeploymentsStatusNetwork, mysqltype.DeploymentsStatusFinalizing,
		} {
			f.setStatus(t, f.targetLive, status)
			current := f.connections(t)
			require.Contains(t, current, f.productionConnection+"/"+f.callerProdLive, status)
			require.Empty(t, current[f.productionConnection+"/"+f.callerProdLive].GetTargetDeploymentId(), status)
		}
	})

	t.Run("saved selections resolve their targets", func(t *testing.T) {
		f := newCatalog(t)
		selected := f.connections(t)
		for key, want := range map[string]string{
			f.productionConnection + "/" + f.callerProdDeploying: f.targetLive,
			f.productionConnection + "/" + f.callerProdLive:      f.targetLive,
			f.canaryAutoConnection + "/" + f.callerCanaryOCI:     "",
			f.previewConnection + "/" + f.callerPreviewFork:      f.targetPreviewFork,
			f.previewConnection + "/" + f.callerPreviewMissing:   "",
			f.manualConnection + "/" + f.callerManualDep:         "",
			f.pinnedConnection + "/" + f.callerPinDep:            "",
			f.explicitProdConnection + "/" + f.callerRollbackDep: f.targetLive,
		} {
			require.Contains(t, selected, key)
			require.Equal(t, want, selected[key].GetTargetDeploymentId(), key)
		}
		require.Equal(t, f.targetLive, selected[f.stagingAutoConnection+"/"+f.callerStagingGit].GetTargetDeploymentId(), "an automatic caller in any production-kind environment follows the live pointer, not its git branch")
		require.Equal(t, f.targetPreviewOld, selected[f.previewConnection+"/"+f.callerPreviewOwn].GetTargetDeploymentId(), "branch matching selects only preview-kind deployments, never a production-kind custom environment")
		require.Contains(t, selected, f.canaryConnection+"/"+f.callerCanaryOCI)
		require.Empty(t, selected[f.canaryConnection+"/"+f.callerCanaryOCI].GetTargetDeploymentId(), "an explicit production-kind environment without the live deployment must not fall back to its newest ready build")
		require.Equal(t, int32(7946), selected[f.productionConnection+"/"+f.callerProdLive].GetTargetPort(), "port comes from the target deployment")
		for _, excluded := range []string{
			f.crossWorkspaceConnection + "/" + f.callerProdLive, f.otherProjectConnection + "/" + f.callerProdLive,
			f.previewConnection + "/" + f.callerUnapproved, f.previewConnection + "/" + f.callerWrongPlatform,
			f.selfPreviewConnection + "/" + f.callerUnapproved, f.selfPreviewConnection + "/" + f.callerWrongPlatform,
			f.reservedSelfConnection + "/" + f.callerPinDep, f.reservedPrefixConnection + "/" + f.callerManualDep,
			f.productionConnection + "/" + f.callerProdDisabled,
		} {
			require.NotContains(t, selected, excluded)
		}
	})

	t.Run("production selection follows the live pointer and rollbacks", func(t *testing.T) {
		f := newCatalog(t)
		f.setLive(t, f.target, f.targetNewProd)
		selected := f.connections(t)
		require.Equal(t, f.targetNewProd, selected[f.productionConnection+"/"+f.callerProdLive].GetTargetDeploymentId())
		require.Equal(t, f.targetNewProd, selected[f.explicitProdConnection+"/"+f.callerRollbackDep].GetTargetDeploymentId())
		require.Equal(t, f.targetNewProd, selected[f.stagingAutoConnection+"/"+f.callerStagingGit].GetTargetDeploymentId())
		f.setLive(t, f.target, f.targetLive)
		selected = f.connections(t)
		require.Equal(t, f.targetLive, selected[f.productionConnection+"/"+f.callerProdLive].GetTargetDeploymentId(), "automatic production follows a rollback")
		require.Equal(t, f.targetLive, selected[f.explicitProdConnection+"/"+f.callerRollbackDep].GetTargetDeploymentId(), "an explicit production environment follows a rollback, not the newest ready deployment")
		require.Equal(t, f.targetLive, selected[f.stagingAutoConnection+"/"+f.callerStagingGit].GetTargetDeploymentId(), "a custom production-kind caller follows a rollback")

		f.setLive(t, f.target, f.targetCanaryReady)
		selected = f.connections(t)
		require.Equal(t, f.targetCanaryReady, selected[f.canaryConnection+"/"+f.callerCanaryOCI].GetTargetDeploymentId(), "an explicit production-kind environment follows the live pointer into that environment")
		require.Equal(t, f.targetCanaryReady, selected[f.stagingAutoConnection+"/"+f.callerStagingGit].GetTargetDeploymentId())
		require.Empty(t, selected[f.explicitProdConnection+"/"+f.callerRollbackDep].GetTargetDeploymentId(), "an explicit environment ignores a live deployment in another environment")

		f.setLive(t, f.target, "")
		selected = f.connections(t)
		require.Empty(t, selected[f.productionConnection+"/"+f.callerProdLive].GetTargetDeploymentId())
		require.Empty(t, selected[f.stagingAutoConnection+"/"+f.callerStagingGit].GetTargetDeploymentId())
		require.Empty(t, selected[f.explicitProdConnection+"/"+f.callerRollbackDep].GetTargetDeploymentId(), "production must not fall back to a non-live build")
		require.Empty(t, selected[f.canaryConnection+"/"+f.callerCanaryOCI].GetTargetDeploymentId(), "a production-kind custom environment must not fall back to a non-live build")

		f.setLive(t, f.target, f.targetLive)
		f.setStatus(t, f.targetLive, mysqltype.DeploymentsStatusFailed)
		selected = f.connections(t)
		require.Empty(t, selected[f.productionConnection+"/"+f.callerProdLive].GetTargetDeploymentId())
		require.Empty(t, selected[f.stagingAutoConnection+"/"+f.callerStagingGit].GetTargetDeploymentId(), "an unavailable live deployment must not select a branch match")
		require.Empty(t, selected[f.explicitProdConnection+"/"+f.callerRollbackDep].GetTargetDeploymentId(), "an unavailable live deployment must not select another production build")
	})

	t.Run("explicit preview environment falls back to its newest first-ready deployment", func(t *testing.T) {
		f := newCatalog(t)
		f.setStatus(t, f.targetManualLatestFailed, mysqltype.DeploymentsStatusReady)
		require.Equal(t, f.targetManualLatestFailed, f.connections(t)[f.manualConnection+"/"+f.callerManualDep].GetTargetDeploymentId())
	})

	t.Run("defaults do not change saved connections across caller stop and restart", func(t *testing.T) {
		f := newCatalog(t)
		ctx := t.Context()
		selected := f.connections(t)
		f.seeder.CreateAppConnection(ctx, seed.CreateAppConnectionRequest{
			ID: f.productionConnection, WorkspaceID: f.workspace, ProjectID: f.project, CallerAppID: f.caller, CallerEnvironmentID: f.callerProd,
			TargetAppID: f.target, Name: "target",
		})
		require.NoError(t, f.database().UpdateAppConnectionName(ctx, db.UpdateAppConnectionNameParams{
			ID: f.productionConnection, Name: "changed", UpdatedAt: sql.NullInt64{Int64: 2, Valid: true},
		}))
		require.NoError(t, f.database().UpdateConnectionAppTarget(ctx, db.UpdateConnectionAppTargetParams{
			ConnectionID: f.productionConnection, SelectionMode: db.ConnectionAppTargetsSelectionModeEnvironment,
			TargetEnvironmentID: sql.NullString{String: f.targetManual, Valid: true},
		}))
		require.Equal(t, selected, f.connections(t), "retargeting and renaming must leave existing deployments unchanged")

		f.deploy(t, catalogDeployment{
			id: f.callerNew, app: f.caller, environment: f.callerProd, branch: "main",
			status: mysqltype.DeploymentsStatusReady, firstReady: true, created: 100, regions: []string{f.region},
		})
		require.Equal(t, selected, f.connections(t), "a deployment without snapshots must not fall back to mutable defaults")
		defaults, err := f.database().ListAppConnectionsByApp(ctx, db.ListAppConnectionsByAppParams{
			WorkspaceID: f.workspace, ProjectID: f.project, AppID: f.caller, EnvironmentID: f.callerProd,
		})
		require.NoError(t, err)
		require.Len(t, defaults, 1)
		current := defaults[0]
		require.Equal(t, f.productionConnection, current.ID)
		require.True(t, current.SelectionMode.Valid)
		f.save(t, f.callerNew, savedConnection{id: current.ID, name: current.Name, environment: f.callerProd, target: current.ResourceID})
		require.Equal(t, selected, f.connections(t), "a snapshot without app target settings must not borrow another deployment's settings")
		require.NoError(t, f.database().InsertDeploymentConnectionAppTarget(ctx, db.InsertDeploymentConnectionAppTargetParams{
			DeploymentID: f.callerNew, ConnectionID: current.ID,
			SelectionMode:       db.DeploymentConnectionAppTargetsSelectionMode(current.SelectionMode.ConnectionAppTargetsSelectionMode),
			TargetEnvironmentID: current.TargetEnvironmentID,
			TargetDeploymentID:  current.TargetDeploymentID,
		}))
		updated := f.connections(t)
		require.Equal(t, "changed", updated[f.productionConnection+"/"+f.callerNew].GetConnectionName())
		require.Empty(t, updated[f.productionConnection+"/"+f.callerNew].GetTargetDeploymentId(), "the new snapshot selects the failed manual target instead of live production")
		require.Equal(t, f.targetLive, updated[f.productionConnection+"/"+f.callerProdLive].GetTargetDeploymentId())

		deleted, err := f.database().DeleteAppConnectionById(ctx, f.productionConnection)
		require.NoError(t, err)
		require.Equal(t, int64(2), deleted, "the default and its app target are deleted together")
		require.Equal(t, updated, f.connections(t), "deleting defaults must leave saved connections unchanged")

		f.setDesiredState(t, f.callerProdLive, mysqltype.DeploymentsDesiredStateStopped)
		stopped := f.connections(t)
		require.NotContains(t, stopped, f.productionConnection+"/"+f.callerProdLive)
		require.Contains(t, stopped, f.productionConnection+"/"+f.callerProdDeploying)
		require.Contains(t, stopped, f.productionConnection+"/"+f.callerNew)
		require.Contains(t, stopped, f.previewConnection+"/"+f.callerPreviewOwn)
		f.setDesiredState(t, f.callerProdLive, mysqltype.DeploymentsDesiredStateRunning)
		require.Equal(t, updated, f.connections(t), "restarting a caller restores its saved connections, not the current defaults")
	})

	t.Run("catalog is scoped to the platform", func(t *testing.T) {
		f := newCatalog(t)
		require.Empty(t, f.connectionsOn(t, strings.ToLower(uid.New("pf"))))
		selected := f.connections(t)
		require.NotContains(t, selected, f.previewConnection+"/"+f.callerWrongPlatform)
		require.NotContains(t, selected, f.selfPreviewConnection+"/"+f.callerWrongPlatform)
	})

	t.Run("deleting and recreating a target environment fails closed", func(t *testing.T) {
		f := newCatalog(t)
		f.setLive(t, f.target, f.targetCanaryReady)
		require.Equal(t, f.targetCanaryReady, f.connections(t)[f.canaryConnection+"/"+f.callerCanaryOCI].GetTargetDeploymentId())
		require.NoError(t, f.database().DeleteEnvironmentById(t.Context(), f.targetCanary))
		selected := f.connections(t)
		require.Contains(t, selected, f.canaryConnection+"/"+f.callerCanaryOCI)
		require.Empty(t, selected[f.canaryConnection+"/"+f.callerCanaryOCI].GetTargetDeploymentId(), "a deleted target environment fails closed even while its deployments linger")

		recreatedEnvironmentID := uid.New(uid.EnvironmentPrefix)
		f.createEnvironment(t, f.workspace, f.project, f.target, recreatedEnvironmentID, "canary", mysqltype.EnvironmentKindProduction)
		f.deploy(t, catalogDeployment{
			id: f.targetCanaryRecreated, app: f.target, environment: recreatedEnvironmentID, source: db.DeploymentsSourceOci,
			status: mysqltype.DeploymentsStatusReady, firstReady: true, created: 70, regions: []string{f.region},
		})
		f.setLive(t, f.target, f.targetCanaryRecreated)
		require.Empty(t, f.connections(t)[f.canaryConnection+"/"+f.callerCanaryOCI].GetTargetDeploymentId(), "recreating the slug must not revive the connection")
	})

	t.Run("deleting a caller environment removes its saved connections", func(t *testing.T) {
		f := newCatalog(t)
		require.NoError(t, f.database().DeleteDeploymentConnectionsByEnvironmentId(t.Context(), db.DeleteDeploymentConnectionsByEnvironmentIdParams{AppID: f.caller, EnvironmentID: f.callerProd}))
		require.NoError(t, f.database().DeleteEnvironmentById(t.Context(), f.callerProd))
		require.Len(t, f.connections(t), 9)
	})

	t.Run("deployment regions are the sorted platform regions each one runs in", func(t *testing.T) {
		f := newCatalog(t)
		rows, err := f.database().ListPrivateNetworkDeploymentRegions(t.Context(), db.ListPrivateNetworkDeploymentRegionsParams{
			DeploymentIds: []string{f.callerProdLive, f.callerCanaryOCI, f.callerWrongPlatform, uid.New(uid.DeploymentPrefix)},
			Platform:      f.platform,
		})
		require.NoError(t, err)
		regions := map[string][]string{}
		for _, row := range rows {
			regions[row.DeploymentID] = append(regions[row.DeploymentID], row.Region)
		}
		require.Equal(t, map[string][]string{
			f.callerCanaryOCI: {"connection-region"},
			f.callerProdLive:  {"connection-region", "connection-region-2"},
		}, regions)
	})
}
