package db

import (
	"database/sql"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
)

type catalogIDs struct {
	workspace, otherWorkspace, unscheduledWorkspace string
	project, otherProject, foreignProject, unscheduledProject string
	caller, target, otherTarget, foreign, unscheduledCaller, unscheduledTarget string
	callerProd, callerCanary, callerStaging, callerPreview, callerManual, callerPin, callerRollback string
	targetProd, targetCanary, targetPreview, targetManual, foreignProd, unscheduledEnv string
	region, region2, otherRegion string
	callerCanaryOCI, callerPreviewOwn, callerPreviewFork, callerPreviewMissing, callerUnapproved, callerWrongPlatform string
	callerManualDep, callerPinDep, callerRollbackDep, callerStagingGit, callerProdLive, callerProdDeploying, callerProdDisabled, callerNew string
	targetCanaryReady, targetCanaryFeature, targetNewProd, targetPreviewOld, targetPreviewNever, targetPreviewAwaiting, targetPreviewFork string
	targetManualOld, targetManualLatestFailed, targetPinnedStopped, targetLive, targetCanaryRecreated, foreignLive string
	unscheduledCaller1, unscheduledCaller2 string
	productionConnection, selfProdConnection, crossWorkspaceConnection, otherProjectConnection, canaryConnection, canaryAutoConnection string
	selfCanaryToProdConnection, previewConnection, selfPreviewConnection, manualConnection, reservedPrefixConnection, pinnedConnection string
	reservedOwnSlugConnection, explicitProdConnection, stagingAutoConnection, unscheduledConnection string
}

func newCatalogIDs() catalogIDs {
	return catalogIDs{
		workspace: uid.New(uid.WorkspacePrefix), otherWorkspace: uid.New(uid.WorkspacePrefix), unscheduledWorkspace: uid.New(uid.WorkspacePrefix),
		project: uid.New(uid.ProjectPrefix), otherProject: uid.New(uid.ProjectPrefix), foreignProject: uid.New(uid.ProjectPrefix), unscheduledProject: uid.New(uid.ProjectPrefix),
		caller: uid.New(uid.AppPrefix), target: uid.New(uid.AppPrefix), otherTarget: uid.New(uid.AppPrefix), foreign: uid.New(uid.AppPrefix), unscheduledCaller: uid.New(uid.AppPrefix), unscheduledTarget: uid.New(uid.AppPrefix),
		callerProd: uid.New(uid.EnvironmentPrefix), callerCanary: uid.New(uid.EnvironmentPrefix), callerStaging: uid.New(uid.EnvironmentPrefix), callerPreview: uid.New(uid.EnvironmentPrefix), callerManual: uid.New(uid.EnvironmentPrefix), callerPin: uid.New(uid.EnvironmentPrefix), callerRollback: uid.New(uid.EnvironmentPrefix),
		targetProd: uid.New(uid.EnvironmentPrefix), targetCanary: uid.New(uid.EnvironmentPrefix), targetPreview: uid.New(uid.EnvironmentPrefix), targetManual: uid.New(uid.EnvironmentPrefix), foreignProd: uid.New(uid.EnvironmentPrefix), unscheduledEnv: uid.New(uid.EnvironmentPrefix),
		region: uid.New(uid.RegionPrefix), region2: uid.New(uid.RegionPrefix), otherRegion: uid.New(uid.RegionPrefix),
		callerCanaryOCI: uid.New(uid.DeploymentPrefix), callerPreviewOwn: uid.New(uid.DeploymentPrefix), callerPreviewFork: uid.New(uid.DeploymentPrefix), callerPreviewMissing: uid.New(uid.DeploymentPrefix), callerUnapproved: uid.New(uid.DeploymentPrefix), callerWrongPlatform: uid.New(uid.DeploymentPrefix),
		callerManualDep: uid.New(uid.DeploymentPrefix), callerPinDep: uid.New(uid.DeploymentPrefix), callerRollbackDep: uid.New(uid.DeploymentPrefix), callerStagingGit: uid.New(uid.DeploymentPrefix), callerProdLive: uid.New(uid.DeploymentPrefix), callerProdDeploying: uid.New(uid.DeploymentPrefix), callerProdDisabled: uid.New(uid.DeploymentPrefix), callerNew: uid.New(uid.DeploymentPrefix),
		targetCanaryReady: uid.New(uid.DeploymentPrefix), targetCanaryFeature: uid.New(uid.DeploymentPrefix), targetNewProd: uid.New(uid.DeploymentPrefix), targetPreviewOld: uid.New(uid.DeploymentPrefix), targetPreviewNever: uid.New(uid.DeploymentPrefix), targetPreviewAwaiting: uid.New(uid.DeploymentPrefix), targetPreviewFork: uid.New(uid.DeploymentPrefix),
		targetManualOld: uid.New(uid.DeploymentPrefix), targetManualLatestFailed: uid.New(uid.DeploymentPrefix), targetPinnedStopped: uid.New(uid.DeploymentPrefix), targetLive: uid.New(uid.DeploymentPrefix), targetCanaryRecreated: uid.New(uid.DeploymentPrefix), foreignLive: uid.New(uid.DeploymentPrefix),
		unscheduledCaller1: uid.New(uid.DeploymentPrefix), unscheduledCaller2: uid.New(uid.DeploymentPrefix),
		productionConnection: uid.New(uid.ConnectionPrefix), selfProdConnection: uid.New(uid.ConnectionPrefix), crossWorkspaceConnection: uid.New(uid.ConnectionPrefix), otherProjectConnection: uid.New(uid.ConnectionPrefix), canaryConnection: uid.New(uid.ConnectionPrefix), canaryAutoConnection: uid.New(uid.ConnectionPrefix),
		selfCanaryToProdConnection: uid.New(uid.ConnectionPrefix), previewConnection: uid.New(uid.ConnectionPrefix), selfPreviewConnection: uid.New(uid.ConnectionPrefix), manualConnection: uid.New(uid.ConnectionPrefix), reservedPrefixConnection: uid.New(uid.ConnectionPrefix), pinnedConnection: uid.New(uid.ConnectionPrefix),
		reservedOwnSlugConnection: uid.New(uid.ConnectionPrefix), explicitProdConnection: uid.New(uid.ConnectionPrefix), stagingAutoConnection: uid.New(uid.ConnectionPrefix), unscheduledConnection: uid.New(uid.ConnectionPrefix),
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

func (ids catalogIDs) seedCatalogDeployments(t *testing.T, q *Queries, platform string) {
	t.Helper()
	insertTestWorkspace(t, q, ids.workspace, "namespace")
	insertTestWorkspace(t, q, ids.otherWorkspace, "other-namespace")
	insertTestWorkspace(t, q, ids.unscheduledWorkspace, "")
	insertTestProject(t, q, ids.workspace, ids.project)
	insertTestProject(t, q, ids.workspace, ids.otherProject)
	insertTestProject(t, q, ids.otherWorkspace, ids.foreignProject)
	insertTestProject(t, q, ids.unscheduledWorkspace, ids.unscheduledProject)
	insertTestApp(t, q, ids.workspace, ids.project, ids.caller, "caller")
	insertTestApp(t, q, ids.workspace, ids.project, ids.target, "target")
	insertTestApp(t, q, ids.workspace, ids.project, ids.otherTarget, "other-target")
	insertTestApp(t, q, ids.otherWorkspace, ids.foreignProject, ids.foreign, "caller")
	insertTestApp(t, q, ids.unscheduledWorkspace, ids.unscheduledProject, ids.unscheduledCaller, "caller")
	insertTestApp(t, q, ids.unscheduledWorkspace, ids.unscheduledProject, ids.unscheduledTarget, "target")

	production, preview := mysqltype.EnvironmentKindProduction, mysqltype.EnvironmentKindPreview
	for _, environment := range []struct {
		app, id, slug string
		kind          mysqltype.EnvironmentKind
	}{
		{ids.caller, ids.callerProd, "production", production},
		{ids.caller, ids.callerCanary, "canary", production},
		{ids.caller, ids.callerStaging, "staging", production},
		{ids.caller, ids.callerPreview, "preview", preview},
		{ids.caller, ids.callerManual, "manual", preview},
		{ids.caller, ids.callerPin, "pin", preview},
		{ids.caller, ids.callerRollback, "rollback", preview},
		{ids.target, ids.targetProd, "production", production},
		{ids.target, ids.targetCanary, "canary", production},
		{ids.target, ids.targetPreview, "preview", preview},
		{ids.target, ids.targetManual, "manual", preview},
	} {
		insertTestEnvironment(t, q, ids.workspace, ids.project, environment.app, environment.id, environment.slug, environment.kind)
	}
	insertTestEnvironment(t, q, ids.otherWorkspace, ids.foreignProject, ids.foreign, ids.foreignProd, "production", production)
	insertTestEnvironment(t, q, ids.unscheduledWorkspace, ids.unscheduledProject, ids.unscheduledCaller, ids.unscheduledEnv, "preview", preview)

	for _, region := range []UpsertRegionParams{
		{ID: ids.region, Name: "connection-region", Platform: platform},
		{ID: ids.region2, Name: "connection-region-2", Platform: platform},
		{ID: ids.otherRegion, Name: "connection-region-other", Platform: "other-" + platform},
	} {
		require.NoError(t, q.UpsertRegion(t.Context(), region))
	}

	ready := sql.NullInt64{Int64: 1, Valid: true}
	never := sql.NullInt64{}
	insertDeployment := func(id, app, environment string, source DeploymentsSource, branch, fork string, status mysqltype.DeploymentsStatus, desired mysqltype.DeploymentsDesiredState, firstReady sql.NullInt64, created int64, regions ...string) {
		t.Helper()
		insertTestDeployment(t, q, testDeployment{
			ID: id, WorkspaceID: ids.workspace, ProjectID: ids.project, AppID: app, EnvironmentID: environment,
			Source: source, GitBranch: branch, ForkRepository: fork, Status: status, DesiredState: desired,
			FirstReadyAt: firstReady, PrivateNetworking: true, CreatedAt: created, RegionIDs: regions,
		})
	}
	git, oci := DeploymentsSourceGit, DeploymentsSourceOci
	running, stopped := mysqltype.DeploymentsDesiredStateRunning, mysqltype.DeploymentsDesiredStateStopped
	readyStatus := mysqltype.DeploymentsStatusReady

	insertDeployment(ids.callerCanaryOCI, ids.caller, ids.callerCanary, oci, "", "", readyStatus, running, ready, 11, ids.region)
	insertDeployment(ids.callerPreviewOwn, ids.caller, ids.callerPreview, git, "feature", "", readyStatus, running, ready, 12, ids.region)
	insertDeployment(ids.callerPreviewFork, ids.caller, ids.callerPreview, git, "feature", "fork/repo", readyStatus, running, ready, 13, ids.region)
	insertDeployment(ids.callerPreviewMissing, ids.caller, ids.callerPreview, git, "isolated", "", readyStatus, running, ready, 13, ids.region)
	insertDeployment(ids.callerUnapproved, ids.caller, ids.callerPreview, git, "feature", "", mysqltype.DeploymentsStatusAwaitingApproval, running, never, 14, ids.region)
	insertDeployment(ids.callerWrongPlatform, ids.caller, ids.callerPreview, git, "feature", "", readyStatus, running, ready, 15, ids.otherRegion)
	insertDeployment(ids.callerManualDep, ids.caller, ids.callerManual, git, "feature", "", readyStatus, running, ready, 16, ids.region)
	insertDeployment(ids.callerPinDep, ids.caller, ids.callerPin, git, "feature", "", readyStatus, running, ready, 17, ids.region)
	insertDeployment(ids.callerRollbackDep, ids.caller, ids.callerRollback, oci, "", "", readyStatus, running, ready, 18, ids.region)
	insertDeployment(ids.callerStagingGit, ids.caller, ids.callerStaging, git, "feature", "", readyStatus, running, ready, 19, ids.region)

	insertDeployment(ids.targetCanaryReady, ids.target, ids.targetCanary, oci, "", "", readyStatus, running, ready, 21, ids.region)
	insertDeployment(ids.targetCanaryFeature, ids.target, ids.targetCanary, git, "feature", "", readyStatus, running, ready, 35, ids.region)
	insertDeployment(ids.targetNewProd, ids.target, ids.targetProd, git, "main", "", readyStatus, running, ready, 99, ids.region)
	insertDeployment(ids.targetPreviewOld, ids.target, ids.targetPreview, git, "feature", "", readyStatus, running, ready, 30, ids.region)
	insertDeployment(ids.targetPreviewNever, ids.target, ids.targetPreview, git, "feature", "", readyStatus, running, never, 31, ids.region)
	insertDeployment(ids.targetPreviewAwaiting, ids.target, ids.targetPreview, git, "feature", "", mysqltype.DeploymentsStatusAwaitingApproval, running, never, 32, ids.region)
	insertDeployment(ids.targetPreviewFork, ids.target, ids.targetPreview, git, "feature", "fork/repo", readyStatus, running, ready, 33, ids.region)
	insertDeployment(ids.targetManualOld, ids.target, ids.targetManual, git, "manual", "", readyStatus, running, ready, 40, ids.region)
	insertDeployment(ids.targetManualLatestFailed, ids.target, ids.targetManual, git, "manual", "", mysqltype.DeploymentsStatusFailed, running, ready, 41, ids.region)
	insertDeployment(ids.targetPinnedStopped, ids.target, ids.targetPreview, git, "pin", "", mysqltype.DeploymentsStatusStopped, stopped, ready, 50, ids.region)

	insertTestDeployment(t, q, testDeployment{
		ID: ids.callerProdLive, WorkspaceID: ids.workspace, ProjectID: ids.project, AppID: ids.caller, EnvironmentID: ids.callerProd,
		GitBranch: "main", Status: readyStatus, FirstReadyAt: ready, Port: 4000, CreatedAt: 9,
		PrivateNetworking: true, RegionIDs: []string{ids.region, ids.region2},
	})
	insertTestDeployment(t, q, testDeployment{
		ID: ids.callerProdDeploying, WorkspaceID: ids.workspace, ProjectID: ids.project, AppID: ids.caller, EnvironmentID: ids.callerProd,
		GitBranch: "main", Status: mysqltype.DeploymentsStatusDeploying, Port: 4000, CreatedAt: 10,
		PrivateNetworking: true, RegionIDs: []string{ids.region},
	})
	insertTestDeployment(t, q, testDeployment{
		ID: ids.callerProdDisabled, WorkspaceID: ids.workspace, ProjectID: ids.project, AppID: ids.caller, EnvironmentID: ids.callerProd,
		GitBranch: "main", Status: readyStatus, FirstReadyAt: ready, CreatedAt: 8, RegionIDs: []string{ids.region},
	})
	insertTestDeployment(t, q, testDeployment{
		ID: ids.targetLive, WorkspaceID: ids.workspace, ProjectID: ids.project, AppID: ids.target, EnvironmentID: ids.targetProd,
		GitBranch: "main", Status: readyStatus, FirstReadyAt: ready, Port: 7946, CreatedAt: 20,
		PrivateNetworking: true, RegionIDs: []string{ids.region},
	})
	insertTestDeployment(t, q, testDeployment{
		ID: ids.foreignLive, WorkspaceID: ids.otherWorkspace, ProjectID: ids.foreignProject, AppID: ids.foreign, EnvironmentID: ids.foreignProd,
		GitBranch: "main", Status: readyStatus, FirstReadyAt: ready, CreatedAt: 60, RegionIDs: []string{ids.region},
	})
	for _, id := range []string{ids.unscheduledCaller1, ids.unscheduledCaller2} {
		insertTestDeployment(t, q, testDeployment{
			ID: id, WorkspaceID: ids.unscheduledWorkspace, ProjectID: ids.unscheduledProject, AppID: ids.unscheduledCaller,
			EnvironmentID: ids.unscheduledEnv, Status: readyStatus, PrivateNetworking: true, CreatedAt: 1, RegionIDs: []string{ids.region},
		})
	}
	setTestLiveDeployment(t, q, ids.caller, ids.callerProdLive)
	setTestLiveDeployment(t, q, ids.target, ids.targetLive)
	setTestLiveDeployment(t, q, ids.foreign, ids.foreignLive)
}

func (ids catalogIDs) seedCatalogConnections(t *testing.T, q *Queries) {
	t.Helper()
	callers := map[string][]string{
		ids.callerProd: {ids.callerProdDeploying, ids.callerProdLive, ids.callerProdDisabled},
		ids.callerCanary: {ids.callerCanaryOCI},
		ids.callerPreview: {ids.callerPreviewOwn, ids.callerPreviewFork, ids.callerPreviewMissing, ids.callerUnapproved, ids.callerWrongPlatform},
		ids.callerManual: {ids.callerManualDep},
		ids.callerPin: {ids.callerPinDep},
		ids.callerRollback: {ids.callerRollbackDep},
		ids.callerStaging: {ids.callerStagingGit},
	}
	connection := func(id, name, environment, target, mode, targetEnvironment, targetDeployment string) testConnection {
		return testConnection{
			ID: id, WorkspaceID: ids.workspace, ProjectID: ids.project, AppID: ids.caller, EnvironmentID: environment,
			ResourceID: target, Name: name, Mode: mode, TargetEnvironmentID: targetEnvironment, TargetDeploymentID: targetDeployment,
		}
	}
	leak := connection(ids.otherProjectConnection, "leak", ids.callerProd, ids.otherTarget, "automatic", "", "")
	leak.ProjectID = ids.otherProject
	for _, c := range []testConnection{
		connection(ids.productionConnection, "target", ids.callerProd, ids.target, "automatic", "", ""),
		connection(ids.selfProdConnection, "caller", ids.callerProd, ids.caller, "automatic", "", ""),
		connection(ids.crossWorkspaceConnection, "foreign", ids.callerProd, ids.foreign, "automatic", "", ""),
		leak,
		connection(ids.canaryConnection, "target", ids.callerCanary, ids.target, "environment", ids.targetCanary, ""),
		connection(ids.canaryAutoConnection, "other-target", ids.callerCanary, ids.otherTarget, "automatic", "", ""),
		connection(ids.selfCanaryToProdConnection, "caller", ids.callerCanary, ids.caller, "environment", ids.callerProd, ""),
		connection(ids.previewConnection, "target", ids.callerPreview, ids.target, "automatic", "", ""),
		connection(ids.selfPreviewConnection, "caller", ids.callerPreview, ids.caller, "automatic", "", ""),
		connection(ids.manualConnection, "target", ids.callerManual, ids.target, "environment", ids.targetManual, ""),
		connection(ids.reservedPrefixConnection, "unkey-internal", ids.callerManual, ids.otherTarget, "automatic", "", ""),
		connection(ids.pinnedConnection, "target", ids.callerPin, ids.target, "deployment", "", ids.targetPinnedStopped),
		connection(ids.reservedOwnSlugConnection, "caller", ids.callerPin, ids.otherTarget, "automatic", "", ""),
		connection(ids.explicitProdConnection, "target", ids.callerRollback, ids.target, "environment", ids.targetProd, ""),
		connection(ids.stagingAutoConnection, "target", ids.callerStaging, ids.target, "automatic", "", ""),
	} {
		for _, caller := range callers[c.EnvironmentID] {
			insertTestSavedConnection(t, q, caller, c)
		}
	}
	for _, caller := range []string{ids.unscheduledCaller1, ids.unscheduledCaller2} {
		insertTestSavedConnection(t, q, caller, testConnection{
			ID: ids.unscheduledConnection, WorkspaceID: ids.unscheduledWorkspace, ProjectID: ids.unscheduledProject, AppID: ids.unscheduledCaller,
			EnvironmentID: ids.unscheduledEnv, ResourceID: ids.unscheduledTarget, Name: "target", Mode: "automatic",
		})
	}
}

func setTestLiveDeployment(t *testing.T, q *Queries, appID, deploymentID string) {
	t.Helper()
	require.NoError(t, q.UpdateAppDeployments(t.Context(), UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{String: deploymentID, Valid: deploymentID != ""}, AppID: appID,
	}))
}

func (ids catalogIDs) catalogReplicas(t *testing.T, q *Queries, platform string) []ListPrivateNetworkReplicasRow {
	t.Helper()
	deploymentIDs := []string{
		ids.callerCanaryOCI, ids.callerPreviewOwn, ids.callerPreviewFork, ids.callerPreviewMissing,
		ids.callerUnapproved, ids.callerWrongPlatform, ids.callerManualDep, ids.callerPinDep,
		ids.callerRollbackDep, ids.callerStagingGit, ids.callerProdLive, ids.callerProdDeploying,
		ids.callerProdDisabled, ids.callerNew,
		ids.targetCanaryReady, ids.targetCanaryFeature, ids.targetNewProd, ids.targetPreviewOld,
		ids.targetPreviewNever, ids.targetPreviewAwaiting, ids.targetPreviewFork, ids.targetManualOld,
		ids.targetManualLatestFailed, ids.targetPinnedStopped, ids.targetLive, ids.targetCanaryRecreated,
		ids.foreignLive, ids.unscheduledCaller1, ids.unscheduledCaller2,
	}
	var replicas []ListPrivateNetworkReplicasRow
	for chunk := range slices.Chunk(deploymentIDs, 3) {
		rows, err := q.ListPrivateNetworkReplicas(t.Context(), ListPrivateNetworkReplicasParams{DeploymentIds: chunk, Platform: platform})
		require.NoError(t, err)
		replicas = append(replicas, rows...)
	}
	return replicas
}

func (ids catalogIDs) catalogReplicaIDs(t *testing.T, q *Queries, platform string) []string {
	t.Helper()
	replicas := ids.catalogReplicas(t, q, platform)
	deploymentIDs := make([]string, 0, len(replicas))
	for _, replica := range replicas {
		require.Equal(t, ids.workspace, replica.WorkspaceID, "a workspace without a Kubernetes namespace publishes no replicas")
		require.NotEmpty(t, replica.AppSlug)
		deploymentIDs = append(deploymentIDs, replica.DeploymentID)
	}
	return deploymentIDs
}

func (ids catalogIDs) catalogConnections(t *testing.T, q *Queries, platform string) map[string]ListPrivateNetworkConnectionsRow {
	t.Helper()
	var callerIDs []string
	for _, caller := range ids.catalogReplicas(t, q, platform) {
		callerIDs = append(callerIDs, caller.DeploymentID)
	}
	var rows []ListPrivateNetworkConnectionsRow
	for callerPage := range slices.Chunk(callerIDs, 3) {
		for _, page := range catalogConnectionPages(t, q, platform, callerPage, 1) {
			rows = append(rows, page...)
		}
	}
	byConnectionCaller := make(map[string]ListPrivateNetworkConnectionsRow, len(rows))
	for _, row := range rows {
		key := row.ConnectionID + "/" + row.CallerDeploymentID
		require.NotContains(t, byConnectionCaller, key, "one row per connection and caller deployment, even across regions")
		byConnectionCaller[key] = row
		require.Equal(t, row.DeploymentID == "", row.Port == 0, "port is known exactly when %s is resolved", key)
		require.Equal(t, ids.workspace, row.WorkspaceID)
		require.Equal(t, ids.project, row.ProjectID)
	}
	return byConnectionCaller
}

func catalogConnectionPages(t *testing.T, q *Queries, platform string, callerIDs []string, pageSize int32) [][]ListPrivateNetworkConnectionsRow {
	t.Helper()
	params := ListPrivateNetworkConnectionsParams{
		CallerDeploymentIds: callerIDs, AfterCallerDeploymentID: "", AfterConnectionID: "", Limit: pageSize, Platform: platform,
	}
	var pages [][]ListPrivateNetworkConnectionsRow
	for {
		page, err := q.ListPrivateNetworkConnections(t.Context(), params)
		require.NoError(t, err)
		require.LessOrEqual(t, len(page), int(pageSize))
		pages = append(pages, page)
		if len(page) < int(pageSize) {
			return pages
		}
		last := page[len(page)-1]
		params.AfterCallerDeploymentID, params.AfterConnectionID = last.CallerDeploymentID, last.ConnectionID
	}
}

// TestListPrivateNetworkConnections guarantees that the private network
// catalog publishes exactly the connections saved by active caller deployments
// created with private networking, each resolved to a ready target on the same
// platform under the rules of its saved selection, and that mutable connection
// defaults never change what a running deployment reaches.
type catalogFixture struct {
	*Queries
	catalogIDs
}

func TestListPrivateNetworkConnections(t *testing.T) {
	database := openTestDatabase(t)
	platform := strings.ToLower(uid.New("pf"))
	newCatalog := func(t *testing.T) *catalogFixture {
		t.Helper()
		q := &catalogFixture{Queries: NewQueries(beginRollbackTx(t, database)), catalogIDs: newCatalogIDs()}
		q.seedCatalogDeployments(t, q.Queries, platform)
		q.seedCatalogConnections(t, q.Queries)
		return q
	}
	list := func(t *testing.T, q *catalogFixture) map[string]ListPrivateNetworkConnectionsRow {
		t.Helper()
		return q.catalogConnections(t, q.Queries, platform)
	}

	t.Run("replicas follow the decision persisted at creation", func(t *testing.T) {
		q := &catalogFixture{Queries: NewQueries(beginRollbackTx(t, database)), catalogIDs: newCatalogIDs()}
		q.seedCatalogDeployments(t, q.Queries, platform)
		require.ElementsMatch(t, q.enabledReplicas(), q.catalogReplicaIDs(t, q.Queries, platform), "replicas follow the decision persisted at creation, even before the workspace has connections")
		q.seedCatalogConnections(t, q.Queries)
		require.ElementsMatch(t, q.enabledReplicas(), q.catalogReplicaIDs(t, q.Queries, platform), "every active deployment created with private networking publishes its replicas")
	})

	t.Run("one row per connection and caller", func(t *testing.T) {
		q := newCatalog(t)
		require.Len(t, list(t, q), 11)
	})

	t.Run("connection pages split one caller without losing or repeating rows", func(t *testing.T) {
		q := newCatalog(t)
		callers := []string{q.callerCanaryOCI}
		whole := catalogConnectionPages(t, q.Queries, platform, callers, 100)
		require.Len(t, whole, 1)
		require.Len(t, whole[0], 2, "the caller saved two connections")

		pages := catalogConnectionPages(t, q.Queries, platform, callers, 1)
		require.Len(t, pages, 3, "two full pages for the same caller, then an empty page")
		var split []ListPrivateNetworkConnectionsRow
		for _, page := range pages {
			split = append(split, page...)
		}
		require.Equal(t, whole[0], split)
	})

	t.Run("callers publish from deploying through ready", func(t *testing.T) {
		q := newCatalog(t)
		for _, status := range []mysqltype.DeploymentsStatus{
			mysqltype.DeploymentsStatusDeploying, mysqltype.DeploymentsStatusNetwork,
			mysqltype.DeploymentsStatusFinalizing, mysqltype.DeploymentsStatusReady,
		} {
			setTestDeploymentStatus(t, q.Queries, q.callerProdDeploying, status, sql.NullInt64{})
			current := list(t, q)
			require.Contains(t, current, q.productionConnection+"/"+q.callerProdDeploying, status)
			require.Equal(t, q.targetLive, current[q.productionConnection+"/"+q.callerProdDeploying].DeploymentID, status)
			require.Contains(t, q.catalogReplicaIDs(t, q.Queries, platform), q.callerProdDeploying, status)
		}
	})

	t.Run("targets resolve only when ready", func(t *testing.T) {
		q := newCatalog(t)
		for _, status := range []mysqltype.DeploymentsStatus{
			mysqltype.DeploymentsStatusDeploying, mysqltype.DeploymentsStatusNetwork, mysqltype.DeploymentsStatusFinalizing,
		} {
			setTestDeploymentStatus(t, q.Queries, q.targetLive, status, sql.NullInt64{})
			current := list(t, q)
			require.Contains(t, current, q.productionConnection+"/"+q.callerProdLive, status)
			require.Empty(t, current[q.productionConnection+"/"+q.callerProdLive].DeploymentID, status)
		}
	})

	t.Run("saved selections resolve their targets", func(t *testing.T) {
		q := newCatalog(t)
		selected := list(t, q)
		for key, want := range map[string]string{
			q.productionConnection+"/"+q.callerProdDeploying: q.targetLive,
			q.productionConnection+"/"+q.callerProdLive: q.targetLive,
			q.canaryAutoConnection+"/"+q.callerCanaryOCI: "",
			q.previewConnection+"/"+q.callerPreviewFork: q.targetPreviewFork,
			q.previewConnection+"/"+q.callerPreviewMissing: "",
			q.manualConnection+"/"+q.callerManualDep: "",
			q.pinnedConnection+"/"+q.callerPinDep: "",
			q.explicitProdConnection+"/"+q.callerRollbackDep: q.targetLive,
		} {
			require.Contains(t, selected, key)
			require.Equal(t, want, selected[key].DeploymentID, key)
		}
		require.Equal(t, q.targetLive, selected[q.stagingAutoConnection+"/"+q.callerStagingGit].DeploymentID, "an automatic caller in any production-kind environment follows the live pointer, not its git branch")
		require.Equal(t, q.targetPreviewOld, selected[q.previewConnection+"/"+q.callerPreviewOwn].DeploymentID, "branch matching selects only preview-kind deployments, never a production-kind custom environment")
		require.Contains(t, selected, q.canaryConnection+"/"+q.callerCanaryOCI)
		require.Empty(t, selected[q.canaryConnection+"/"+q.callerCanaryOCI].DeploymentID, "an explicit production-kind environment without the live deployment must not fall back to its newest ready build")
		require.Equal(t, int32(7946), selected[q.productionConnection+"/"+q.callerProdLive].Port, "port comes from the target deployment")
		for _, excluded := range []string{
			q.crossWorkspaceConnection+"/"+q.callerProdLive, q.otherProjectConnection+"/"+q.callerProdLive,
			q.previewConnection+"/"+q.callerUnapproved, q.previewConnection+"/"+q.callerWrongPlatform,
			q.selfPreviewConnection+"/"+q.callerUnapproved, q.selfPreviewConnection+"/"+q.callerWrongPlatform,
			q.reservedOwnSlugConnection+"/"+q.callerPinDep, q.reservedPrefixConnection+"/"+q.callerManualDep,
			q.productionConnection+"/"+q.callerProdDisabled,
		} {
			require.NotContains(t, selected, excluded)
		}
	})

	t.Run("production selection follows the live pointer and rollbacks", func(t *testing.T) {
		q := newCatalog(t)
		setTestLiveDeployment(t, q.Queries, q.target, q.targetNewProd)
		selected := list(t, q)
		require.Equal(t, q.targetNewProd, selected[q.productionConnection+"/"+q.callerProdLive].DeploymentID)
		require.Equal(t, q.targetNewProd, selected[q.explicitProdConnection+"/"+q.callerRollbackDep].DeploymentID)
		require.Equal(t, q.targetNewProd, selected[q.stagingAutoConnection+"/"+q.callerStagingGit].DeploymentID)
		setTestLiveDeployment(t, q.Queries, q.target, q.targetLive)
		selected = list(t, q)
		require.Equal(t, q.targetLive, selected[q.productionConnection+"/"+q.callerProdLive].DeploymentID, "automatic production follows a rollback")
		require.Equal(t, q.targetLive, selected[q.explicitProdConnection+"/"+q.callerRollbackDep].DeploymentID, "an explicit production environment follows a rollback, not the newest ready deployment")
		require.Equal(t, q.targetLive, selected[q.stagingAutoConnection+"/"+q.callerStagingGit].DeploymentID, "a custom production-kind caller follows a rollback")

		setTestLiveDeployment(t, q.Queries, q.target, q.targetCanaryReady)
		selected = list(t, q)
		require.Equal(t, q.targetCanaryReady, selected[q.canaryConnection+"/"+q.callerCanaryOCI].DeploymentID, "an explicit production-kind environment follows the live pointer into that environment")
		require.Equal(t, q.targetCanaryReady, selected[q.stagingAutoConnection+"/"+q.callerStagingGit].DeploymentID)
		require.Empty(t, selected[q.explicitProdConnection+"/"+q.callerRollbackDep].DeploymentID, "an explicit environment ignores a live deployment in another environment")

		setTestLiveDeployment(t, q.Queries, q.target, "")
		selected = list(t, q)
		require.Empty(t, selected[q.productionConnection+"/"+q.callerProdLive].DeploymentID)
		require.Empty(t, selected[q.stagingAutoConnection+"/"+q.callerStagingGit].DeploymentID)
		require.Empty(t, selected[q.explicitProdConnection+"/"+q.callerRollbackDep].DeploymentID, "production must not fall back to a non-live build")
		require.Empty(t, selected[q.canaryConnection+"/"+q.callerCanaryOCI].DeploymentID, "a production-kind custom environment must not fall back to a non-live build")

		setTestLiveDeployment(t, q.Queries, q.target, q.targetLive)
		setTestDeploymentStatus(t, q.Queries, q.targetLive, mysqltype.DeploymentsStatusFailed, sql.NullInt64{})
		selected = list(t, q)
		require.Empty(t, selected[q.productionConnection+"/"+q.callerProdLive].DeploymentID)
		require.Empty(t, selected[q.stagingAutoConnection+"/"+q.callerStagingGit].DeploymentID, "an unavailable live deployment must not select a branch match")
		require.Empty(t, selected[q.explicitProdConnection+"/"+q.callerRollbackDep].DeploymentID, "an unavailable live deployment must not select another production build")
	})

	t.Run("explicit preview environment falls back to its newest first-ready deployment", func(t *testing.T) {
		q := newCatalog(t)
		setTestDeploymentStatus(t, q.Queries, q.targetManualLatestFailed, mysqltype.DeploymentsStatusReady, sql.NullInt64{})
		require.Equal(t, q.targetManualLatestFailed, list(t, q)[q.manualConnection+"/"+q.callerManualDep].DeploymentID)
	})

	t.Run("defaults do not change saved connections across caller stop and restart", func(t *testing.T) {
		q := newCatalog(t)
		selected := list(t, q)
		insertTestDefaultConnection(t, q.Queries, testConnection{
			ID: q.productionConnection, WorkspaceID: q.workspace, ProjectID: q.project, AppID: q.caller, EnvironmentID: q.callerProd,
			ResourceID: q.target, Name: "target", Mode: "automatic",
		})
		require.NoError(t, q.UpdateAppConnectionName(t.Context(), UpdateAppConnectionNameParams{
			ID: q.productionConnection, Name: "changed", UpdatedAt: sql.NullInt64{Int64: 2, Valid: true},
		}))
		require.NoError(t, q.UpdateConnectionAppTarget(t.Context(), UpdateConnectionAppTargetParams{
			ConnectionID: q.productionConnection, SelectionMode: ConnectionAppTargetsSelectionModeEnvironment,
			TargetEnvironmentID: sql.NullString{String: q.targetManual, Valid: true},
		}))
		require.Equal(t, selected, list(t, q), "retargeting and renaming must leave existing deployments unchanged")

		insertTestDeployment(t, q.Queries, testDeployment{
			ID: q.callerNew, WorkspaceID: q.workspace, ProjectID: q.project, AppID: q.caller, EnvironmentID: q.callerProd,
			GitBranch: "main", Status: mysqltype.DeploymentsStatusReady, FirstReadyAt: sql.NullInt64{Int64: 1, Valid: true},
			PrivateNetworking: true, CreatedAt: 100, RegionIDs: []string{q.region},
		})
		require.Equal(t, selected, list(t, q), "a deployment without snapshots must not fall back to mutable defaults")
		defaults, err := q.ListAppConnectionsByApp(t.Context(), ListAppConnectionsByAppParams{
			WorkspaceID: q.workspace, ProjectID: q.project, AppID: q.caller, EnvironmentID: q.callerProd,
		})
		require.NoError(t, err)
		require.Len(t, defaults, 1)
		current := defaults[0]
		require.Equal(t, q.productionConnection, current.ID)
		require.True(t, current.SelectionMode.Valid)
		insertTestSavedConnection(t, q.Queries, q.callerNew, testConnection{
			ID: current.ID, WorkspaceID: q.workspace, ProjectID: q.project, AppID: q.caller, EnvironmentID: q.callerProd,
			ResourceType: current.ResourceType, ResourceID: current.ResourceID, Name: current.Name,
		})
		require.Equal(t, selected, list(t, q), "a snapshot without app target settings must not borrow another deployment's settings")
		require.NoError(t, q.InsertDeploymentConnectionAppTarget(t.Context(), InsertDeploymentConnectionAppTargetParams{
			DeploymentID: q.callerNew, ConnectionID: current.ID,
			SelectionMode:       DeploymentConnectionAppTargetsSelectionMode(current.SelectionMode.ConnectionAppTargetsSelectionMode),
			TargetEnvironmentID: current.TargetEnvironmentID,
			TargetDeploymentID:  current.TargetDeploymentID,
		}))
		updated := list(t, q)
		require.Equal(t, "changed", updated[q.productionConnection+"/"+q.callerNew].ConnectionName)
		require.Empty(t, updated[q.productionConnection+"/"+q.callerNew].DeploymentID, "the new snapshot selects the failed manual target instead of live production")
		require.Equal(t, q.targetLive, updated[q.productionConnection+"/"+q.callerProdLive].DeploymentID)

		deleted, err := q.DeleteAppConnectionById(t.Context(), q.productionConnection)
		require.NoError(t, err)
		require.Equal(t, int64(2), deleted, "the default and its app target are deleted together")
		require.Equal(t, updated, list(t, q), "deleting defaults must leave saved connections unchanged")

		setTestDesiredState(t, q.Queries, q.callerProdLive, mysqltype.DeploymentsDesiredStateStopped)
		stopped := list(t, q)
		require.NotContains(t, stopped, q.productionConnection+"/"+q.callerProdLive)
		require.Contains(t, stopped, q.productionConnection+"/"+q.callerProdDeploying)
		require.Contains(t, stopped, q.productionConnection+"/"+q.callerNew)
		require.Contains(t, stopped, q.previewConnection+"/"+q.callerPreviewOwn)
		setTestDesiredState(t, q.Queries, q.callerProdLive, mysqltype.DeploymentsDesiredStateRunning)
		require.Equal(t, updated, list(t, q), "restarting a caller restores its saved connections, not the current defaults")
	})

	t.Run("catalog is scoped to the platform", func(t *testing.T) {
		q := newCatalog(t)
		require.Empty(t, q.catalogConnections(t, q.Queries, "missing"))
		selected := list(t, q)
		require.NotContains(t, selected, q.previewConnection+"/"+q.callerWrongPlatform)
		require.NotContains(t, selected, q.selfPreviewConnection+"/"+q.callerWrongPlatform)
	})

	t.Run("deleting and recreating a target environment fails closed", func(t *testing.T) {
		q := newCatalog(t)
		setTestLiveDeployment(t, q.Queries, q.target, q.targetCanaryReady)
		require.Equal(t, q.targetCanaryReady, list(t, q)[q.canaryConnection+"/"+q.callerCanaryOCI].DeploymentID)
		require.NoError(t, q.DeleteEnvironmentById(t.Context(), q.targetCanary))
		selected := list(t, q)
		require.Contains(t, selected, q.canaryConnection+"/"+q.callerCanaryOCI)
		require.Empty(t, selected[q.canaryConnection+"/"+q.callerCanaryOCI].DeploymentID, "a deleted target environment fails closed even while its deployments linger")

		recreatedEnvironmentID := uid.New(uid.EnvironmentPrefix)
		insertTestEnvironment(t, q.Queries, q.workspace, q.project, q.target, recreatedEnvironmentID, "canary", mysqltype.EnvironmentKindProduction)
		insertTestDeployment(t, q.Queries, testDeployment{
			ID: q.targetCanaryRecreated, WorkspaceID: q.workspace, ProjectID: q.project, AppID: q.target, EnvironmentID: recreatedEnvironmentID,
			Source: DeploymentsSourceOci, Status: mysqltype.DeploymentsStatusReady, FirstReadyAt: sql.NullInt64{Int64: 1, Valid: true},
			PrivateNetworking: true, CreatedAt: 70, RegionIDs: []string{q.region},
		})
		setTestLiveDeployment(t, q.Queries, q.target, q.targetCanaryRecreated)
		require.Empty(t, list(t, q)[q.canaryConnection+"/"+q.callerCanaryOCI].DeploymentID, "recreating the slug must not revive the connection")
	})

	t.Run("deleting a caller environment removes its saved connections", func(t *testing.T) {
		q := newCatalog(t)
		require.NoError(t, q.DeleteDeploymentConnectionsByEnvironmentId(t.Context(), DeleteDeploymentConnectionsByEnvironmentIdParams{AppID: q.caller, EnvironmentID: q.callerProd}))
		require.NoError(t, q.DeleteEnvironmentById(t.Context(), q.callerProd))
		require.Len(t, list(t, q), 9)
	})

	t.Run("non-app resources are excluded even with a matching app ID", func(t *testing.T) {
		q := newCatalog(t)
		selected := list(t, q)
		for _, resourceType := range []string{"queue", "vault"} {
			insertTestSavedConnection(t, q.Queries, q.callerCanaryOCI, testConnection{
				ID: uid.New(uid.ConnectionPrefix), WorkspaceID: q.workspace, ProjectID: q.project, AppID: q.caller, EnvironmentID: q.callerCanary,
				ResourceType: resourceType, ResourceID: q.target, Name: resourceType, Mode: "automatic",
			})
		}
		require.Equal(t, selected, list(t, q))
	})
}
