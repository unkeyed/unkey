package privatenetwork

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/unkeyed/unkey/pkg/db"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"golang.org/x/sync/errgroup"
)

type loadShape struct {
	tenants, apps, history, branches, perBranch int
}

var loadShapeBase = loadShape{tenants: 1, apps: 6, history: 10, branches: 3, perBranch: 3}

var loadAxes = []struct {
	name  string
	scale func(loadShape, int) loadShape
}{
	{"history", func(s loadShape, f int) loadShape { s.history *= f; s.perBranch *= f; return s }},
	{"apps", func(s loadShape, f int) loadShape { s.apps *= f; return s }},
	{"branches", func(s loadShape, f int) loadShape { s.branches *= f; return s }},
	{"tenants", func(s loadShape, f int) loadShape { s.tenants *= f; return s }},
}

type loadProbe struct {
	workspaceID, projectID  string
	hubID, callerID, nextID string
	callerProductionID      string
	callerLiveID, pinnedID  string
	callerIDs               []string
	platform                string
}

type loadEnvironment struct {
	id, slug    string
	kind        mysqltype.EnvironmentKind
	deployments []string
	live        []string
}

type loadApp struct {
	id                           string
	production, staging, preview loadEnvironment
}

type loadRule struct {
	id, name, target string
	mode             apidb.ConnectionAppTargetsSelectionMode
	environmentID    string
	deploymentID     string
}

type loadStatus struct {
	id     string
	status mysqltype.DeploymentsStatus
}

type loadRows struct {
	workspaces            []apidb.InsertWorkspaceParams
	projects              []apidb.InsertProjectParams
	apps                  []apidb.InsertAppParams
	environments          []apidb.InsertEnvironmentParams
	deployments           []apidb.InsertDeploymentParams
	topologies            []apidb.InsertDeploymentTopologyParams
	appConnections        []apidb.InsertAppConnectionParams
	connectionTargets     []apidb.InsertConnectionAppTargetParams
	deploymentConnections []apidb.InsertDeploymentConnectionParams
	deploymentTargets     []apidb.InsertDeploymentConnectionAppTargetParams
	statuses              []loadStatus
	live                  []db.SetAppCurrentDeploymentParams
	created               int64
}

func seedLoad(t *testing.T, database *sql.DB, shape loadShape) loadProbe {
	t.Helper()
	ctx := t.Context()
	platform := "load"
	regionID := uid.New(uid.RegionPrefix)
	require.NoError(t, db.NewQueries(database).UpsertRegion(ctx, db.UpsertRegionParams{ID: regionID, Name: "load-region", Platform: platform}))

	var rows loadRows
	var probe loadProbe
	for tenant := range shape.tenants {
		p := rows.addWorkspace(shape, regionID)
		if tenant == 0 {
			probe = p
		}
	}
	probe.platform = platform

	bulk := apidb.BulkQuery
	require.NoError(t, bulkInsert(ctx, database, rows.workspaces, bulk.InsertWorkspaces))
	require.NoError(t, bulkInsert(ctx, database, rows.projects, bulk.InsertProjects))
	require.NoError(t, bulkInsert(ctx, database, rows.apps, bulk.InsertApps))
	require.NoError(t, bulkInsert(ctx, database, rows.environments, bulk.InsertEnvironments))
	require.NoError(t, bulkInsert(ctx, database, rows.deployments, bulk.InsertDeployments))
	require.NoError(t, bulkInsert(ctx, database, rows.topologies, bulk.InsertDeploymentTopologies))
	require.NoError(t, bulkInsert(ctx, database, rows.appConnections, bulk.InsertAppConnections))
	require.NoError(t, bulkInsert(ctx, database, rows.connectionTargets, bulk.InsertConnectionAppTargets))
	require.NoError(t, bulkInsert(ctx, database, rows.deploymentConnections, bulk.InsertDeploymentConnections))
	require.NoError(t, bulkInsert(ctx, database, rows.deploymentTargets, bulk.InsertDeploymentConnectionAppTargets))

	queries := db.NewQueries(database)
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(16)
	for _, s := range rows.statuses {
		group.Go(func() error {
			return queries.UpdateDeploymentStatus(groupCtx, db.UpdateDeploymentStatusParams{Status: s.status, UpdatedAt: sql.NullInt64{}, ID: s.id})
		})
	}
	for _, live := range rows.live {
		group.Go(func() error { return queries.SetAppCurrentDeployment(groupCtx, live) })
	}
	require.NoError(t, group.Wait())
	// InnoDB refreshes index statistics asynchronously after bulk writes, so
	// without this the optimizer picks plans from whichever stats exist when a
	// sample runs, and plans differ between runs on identical data.
	_, err := database.ExecContext(ctx, "ANALYZE TABLE workspaces, projects, apps, environments, deployments, deployment_topology, app_connections, connection_app_targets, deployment_connections, deployment_connection_app_targets")
	require.NoError(t, err)
	t.Logf("seeded %d workspaces, %d apps, %d deployments, %d deployment connections",
		len(rows.workspaces), len(rows.apps), len(rows.deployments), len(rows.deploymentConnections))
	return probe
}

func bulkInsert[P any](ctx context.Context, database *sql.DB, rows []P, insert func(context.Context, apidb.DBTX, []P) error) error {
	for chunk := range slices.Chunk(rows, 500) {
		if err := insert(ctx, database, chunk); err != nil {
			return err
		}
	}
	return nil
}

func (r *loadRows) addWorkspace(shape loadShape, regionID string) loadProbe {
	workspaceID, projectID := uid.New(uid.WorkspacePrefix), uid.New(uid.ProjectPrefix)
	r.workspaces = append(r.workspaces, apidb.InsertWorkspaceParams{
		ID: workspaceID, OrgID: uid.New(uid.OrgPrefix), Name: workspaceID, Slug: workspaceID, CreatedAt: 1, K8sNamespace: uid.DNS1035(),
	})
	r.projects = append(r.projects, apidb.InsertProjectParams{ID: projectID, WorkspaceID: workspaceID, Name: projectID, Slug: projectID, CreatedAt: 1})

	apps := make([]loadApp, shape.apps)
	for i := range apps {
		app := &apps[i]
		app.id = uid.New(uid.AppPrefix)
		r.apps = append(r.apps, apidb.InsertAppParams{
			ID: app.id, WorkspaceID: workspaceID, ProjectID: projectID, Name: app.id, Slug: fmt.Sprintf("app-%d", i),
			SourceType: apidb.AppsSourceTypeGit, CreatedAt: 1,
		})
		app.production = loadEnvironment{id: uid.New(uid.EnvironmentPrefix), slug: "production", kind: mysqltype.EnvironmentKindProduction}
		app.staging = loadEnvironment{id: uid.New(uid.EnvironmentPrefix), slug: "staging", kind: mysqltype.EnvironmentKindPreview}
		app.preview = loadEnvironment{id: uid.New(uid.EnvironmentPrefix), slug: "preview", kind: mysqltype.EnvironmentKindPreview}
		for _, env := range []*loadEnvironment{&app.production, &app.staging, &app.preview} {
			r.environments = append(r.environments, apidb.InsertEnvironmentParams{
				ID: env.id, WorkspaceID: workspaceID, ProjectID: projectID, AppID: app.id, Slug: env.slug, Kind: env.kind, CreatedAt: 1,
			})
		}
		r.deploy(workspaceID, projectID, app.id, &app.production, "main", shape.history, regionID)
		r.deploy(workspaceID, projectID, app.id, &app.staging, "staging", shape.history, regionID)
		for b := range shape.branches {
			r.deploy(workspaceID, projectID, app.id, &app.preview, fmt.Sprintf("feature-%d", b), shape.perBranch, regionID)
		}
		r.live = append(r.live, db.SetAppCurrentDeploymentParams{
			DeploymentID: sql.NullString{String: app.production.live[0], Valid: true}, AppID: app.id,
		})
	}

	hub := apps[0]
	for i := 1; i < len(apps); i++ {
		app, next := apps[i], apps[1+i%(len(apps)-1)]
		pin := loadRule{id: uid.New(uid.ConnectionPrefix), name: "next", target: next.id, mode: apidb.ConnectionAppTargetsSelectionModeDeployment, deploymentID: next.production.live[0]}
		production := []loadRule{{id: uid.New(uid.ConnectionPrefix), name: "hub", target: hub.id, mode: apidb.ConnectionAppTargetsSelectionModeAutomatic}, pin}
		legacy := loadRule{id: uid.New(uid.ConnectionPrefix), name: "legacy", target: hub.id, mode: apidb.ConnectionAppTargetsSelectionModeAutomatic}
		staging := []loadRule{{id: uid.New(uid.ConnectionPrefix), name: "hub", target: hub.id, mode: apidb.ConnectionAppTargetsSelectionModeEnvironment, environmentID: hub.staging.id}}
		preview := []loadRule{
			{id: uid.New(uid.ConnectionPrefix), name: "hub", target: hub.id, mode: apidb.ConnectionAppTargetsSelectionModeAutomatic},
			{id: uid.New(uid.ConnectionPrefix), name: "next", target: next.id, mode: apidb.ConnectionAppTargetsSelectionModeAutomatic},
		}
		r.connect(workspaceID, projectID, app.id, app.production, production, legacy)
		r.connect(workspaceID, projectID, app.id, app.staging, staging, loadRule{})
		r.connect(workspaceID, projectID, app.id, app.preview, preview, loadRule{})
	}

	var running []string
	for _, app := range apps {
		running = append(running, app.production.live...)
		running = append(running, app.staging.live...)
		running = append(running, app.preview.live...)
	}
	random := rand.New(rand.NewPCG(1, 2))
	random.Shuffle(len(running), func(i, j int) { running[i], running[j] = running[j], running[i] })
	caller := apps[1]
	return loadProbe{
		workspaceID: workspaceID, projectID: projectID,
		hubID: hub.id, callerID: caller.id, nextID: apps[2].id,
		callerProductionID: caller.production.id,
		callerLiveID:       caller.production.live[0],
		pinnedID:           apps[2].production.live[0],
		callerIDs:          running[:min(len(running), callerChunkSize)],
		platform:           "",
	}
}

func (r *loadRows) deploy(workspaceID, projectID, appID string, env *loadEnvironment, branch string, count int, regionID string) {
	for i := range count {
		r.created++
		id := uid.New(uid.DeploymentPrefix)
		latest := i == count-1
		status, desired := mysqltype.DeploymentsStatusReady, apidb.DeploymentTopologyDesiredStatusStopped
		switch {
		case latest:
			r.statuses = append(r.statuses, loadStatus{id: id, status: mysqltype.DeploymentsStatusReady})
			desired = apidb.DeploymentTopologyDesiredStatusRunning
			env.live = append(env.live, id)
		case i%5 == 2:
			status = mysqltype.DeploymentsStatusFailed
		default:
			r.statuses = append(r.statuses, loadStatus{id: id, status: mysqltype.DeploymentsStatusStopped})
		}
		r.deployments = append(r.deployments, apidb.InsertDeploymentParams{
			ID: id, K8sName: uid.New("k8s"), WorkspaceID: workspaceID, ProjectID: projectID, AppID: appID, EnvironmentID: env.id,
			Source: apidb.DeploymentsSourceGit, GitBranch: sql.NullString{String: branch, Valid: true},
			SentinelConfig: []byte("{}"), EncryptedEnvironmentVariables: []byte("{}"), Status: status,
			CpuMillicores: 250, MemoryMib: 256, Port: 8080,
			ShutdownSignal: apidb.DeploymentsShutdownSignalSIGINT, UpstreamProtocol: apidb.DeploymentsUpstreamProtocolHttp1,
			Capabilities: mysqltype.DeploymentCapabilities{PrivateNetworking: true}, DeploymentTrigger: apidb.DeploymentsTriggerUnknown,
			CreatedAt: r.created,
		})
		r.topologies = append(r.topologies, apidb.InsertDeploymentTopologyParams{
			WorkspaceID: workspaceID, DeploymentID: id, RegionID: regionID, AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1,
			DesiredStatus: desired, CreatedAt: r.created,
		})
		env.deployments = append(env.deployments, id)
	}
}

func (r *loadRows) connect(workspaceID, projectID, appID string, env loadEnvironment, rules []loadRule, legacy loadRule) {
	for _, rule := range rules {
		r.appConnections = append(r.appConnections, apidb.InsertAppConnectionParams{
			ID: rule.id, WorkspaceID: workspaceID, ProjectID: projectID, AppID: appID, EnvironmentID: env.id,
			ResourceType: apidb.AppConnectionsResourceTypeApp, ResourceID: rule.target, Name: rule.name, CreatedAt: 1,
		})
		r.connectionTargets = append(r.connectionTargets, apidb.InsertConnectionAppTargetParams{
			ConnectionID: rule.id, SelectionMode: rule.mode,
			TargetEnvironmentID: sql.NullString{String: rule.environmentID, Valid: rule.environmentID != ""},
			TargetDeploymentID:  sql.NullString{String: rule.deploymentID, Valid: rule.deploymentID != ""},
		})
	}
	for i, deploymentID := range env.deployments {
		saved := rules
		if legacy.id != "" && i < len(env.deployments)/2 {
			saved = append(slices.Clone(rules), legacy)
		}
		for _, rule := range saved {
			r.deploymentConnections = append(r.deploymentConnections, apidb.InsertDeploymentConnectionParams{
				DeploymentID: deploymentID, ConnectionID: rule.id, WorkspaceID: workspaceID, ProjectID: projectID, AppID: appID,
				EnvironmentID: env.id, ResourceType: apidb.DeploymentConnectionsResourceTypeApp, ResourceID: rule.target,
				Name: rule.name, CreatedAt: 1,
			})
			r.deploymentTargets = append(r.deploymentTargets, apidb.InsertDeploymentConnectionAppTargetParams{
				DeploymentID: deploymentID, ConnectionID: rule.id,
				SelectionMode:       apidb.DeploymentConnectionAppTargetsSelectionMode(rule.mode),
				TargetEnvironmentID: sql.NullString{String: rule.environmentID, Valid: rule.environmentID != ""},
				TargetDeploymentID:  sql.NullString{String: rule.deploymentID, Valid: rule.deploymentID != ""},
			})
		}
	}
}
