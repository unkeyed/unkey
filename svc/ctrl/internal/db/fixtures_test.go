package db

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("mysql", containers.MySQL(t).DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	return database
}

func beginRollbackTx(t *testing.T, database *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := database.BeginTx(t.Context(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, tx.Rollback()) })
	return tx
}

func insertTestWorkspace(t *testing.T, q *Queries, id, k8sNamespace string) {
	t.Helper()
	require.NoError(t, q.InsertWorkspace(t.Context(), InsertWorkspaceParams{
		ID: id, OrgID: uid.New(uid.OrgPrefix), Name: id, Slug: id, CreatedAt: 1, K8sNamespace: k8sNamespace,
	}))
}

func insertTestProject(t *testing.T, q *Queries, workspaceID, id string) {
	t.Helper()
	require.NoError(t, q.InsertProject(t.Context(), InsertProjectParams{
		ID: id, WorkspaceID: workspaceID, Name: id, Slug: id, CreatedAt: 1,
	}))
}

func insertTestApp(t *testing.T, q *Queries, workspaceID, projectID, id, slug string) {
	t.Helper()
	require.NoError(t, q.InsertApp(t.Context(), InsertAppParams{
		ID: id, WorkspaceID: workspaceID, ProjectID: projectID, Name: id, Slug: slug,
		SourceType: AppsSourceTypeGit, CreatedAt: 1,
	}))
}

func insertTestEnvironment(t *testing.T, q *Queries, workspaceID, projectID, appID, id, slug string, kind mysqltype.EnvironmentKind) {
	t.Helper()
	require.NoError(t, q.InsertEnvironment(t.Context(), InsertEnvironmentParams{
		ID: id, WorkspaceID: workspaceID, ProjectID: projectID, AppID: appID, Slug: slug, Kind: kind, CreatedAt: 1,
	}))
}

type testDeployment struct {
	ID                string
	WorkspaceID       string
	ProjectID         string
	AppID             string
	EnvironmentID     string
	Source            DeploymentsSource
	GitBranch         string
	ForkRepository    string
	Status            mysqltype.DeploymentsStatus
	DesiredState      mysqltype.DeploymentsDesiredState
	FirstReadyAt      sql.NullInt64
	PrivateNetworking bool
	Port              int32
	CreatedAt         int64
	UpdatedAt         sql.NullInt64
	RegionIDs         []string
}

func insertTestDeployment(t *testing.T, q *Queries, d testDeployment) {
	t.Helper()
	source := d.Source
	if source == "" {
		source = DeploymentsSourceGit
	}
	port := d.Port
	if port == 0 {
		port = 8080
	}
	status := d.Status
	if d.FirstReadyAt.Valid {
		status = mysqltype.DeploymentsStatusPending
	}
	emptyConfig, err := json.Marshal(struct{}{})
	require.NoError(t, err)
	require.NoError(t, q.InsertDeployment(t.Context(), InsertDeploymentParams{
		ID:                            d.ID,
		K8sName:                       d.ID,
		WorkspaceID:                   d.WorkspaceID,
		ProjectID:                     d.ProjectID,
		AppID:                         d.AppID,
		EnvironmentID:                 d.EnvironmentID,
		Source:                        source,
		GitBranch:                     sql.NullString{String: d.GitBranch, Valid: d.GitBranch != ""},
		ForkRepositoryFullName:        sql.NullString{String: d.ForkRepository, Valid: d.ForkRepository != ""},
		SentinelConfig:                emptyConfig,
		EncryptedEnvironmentVariables: emptyConfig,
		Status:                        status,
		CpuMillicores:                 100,
		MemoryMib:                     128,
		Port:                          port,
		ShutdownSignal:                DeploymentsShutdownSignalSIGTERM,
		UpstreamProtocol:              DeploymentsUpstreamProtocolHttp1,
		Capabilities:                  mysqltype.DeploymentCapabilities{PrivateNetworking: d.PrivateNetworking},
		DeploymentTrigger:             DeploymentsTriggerUnknown,
		CreatedAt:                     d.CreatedAt,
		UpdatedAt:                     d.UpdatedAt,
	}))
	if d.FirstReadyAt.Valid {
		setTestDeploymentStatus(t, q, d.ID, mysqltype.DeploymentsStatusReady, d.FirstReadyAt)
		setTestDeploymentStatus(t, q, d.ID, d.Status, d.UpdatedAt)
	}
	if d.DesiredState != "" && d.DesiredState != mysqltype.DeploymentsDesiredStateRunning {
		require.NoError(t, q.UpdateDeploymentDesiredState(t.Context(), UpdateDeploymentDesiredStateParams{
			DesiredState: d.DesiredState, UpdatedAt: d.UpdatedAt, ID: d.ID,
		}))
	}
	for _, region := range d.RegionIDs {
		require.NoError(t, q.InsertDeploymentTopology(t.Context(), InsertDeploymentTopologyParams{
			WorkspaceID: d.WorkspaceID, DeploymentID: d.ID, RegionID: region,
			AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1,
			DesiredStatus: DeploymentTopologyDesiredStatusRunning, CreatedAt: 1,
		}))
	}
}

func setTestDeploymentStatus(t *testing.T, q *Queries, id string, status mysqltype.DeploymentsStatus, updatedAt sql.NullInt64) {
	t.Helper()
	require.NoError(t, q.UpdateDeploymentStatus(t.Context(), UpdateDeploymentStatusParams{ID: id, Status: status, UpdatedAt: updatedAt}))
}

func setTestDesiredState(t *testing.T, q *Queries, id string, state mysqltype.DeploymentsDesiredState) {
	t.Helper()
	require.NoError(t, q.UpdateDeploymentDesiredState(t.Context(), UpdateDeploymentDesiredStateParams{DesiredState: state, ID: id}))
}

type testConnection struct {
	ID                  string
	WorkspaceID         string
	ProjectID           string
	AppID               string
	EnvironmentID       string
	ResourceType        string
	ResourceID          string
	Name                string
	Mode                string
	TargetEnvironmentID string
	TargetDeploymentID  string
}

func (c testConnection) resourceType() string {
	if c.ResourceType == "" {
		return "app"
	}
	return c.ResourceType
}

func insertTestDefaultConnection(t *testing.T, q *Queries, c testConnection) {
	t.Helper()
	require.NoError(t, q.InsertAppConnection(t.Context(), InsertAppConnectionParams{
		ID: c.ID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, AppID: c.AppID, EnvironmentID: c.EnvironmentID,
		ResourceType: c.resourceType(), ResourceID: c.ResourceID, Name: c.Name, CreatedAt: 1,
	}))
	if c.Mode == "" {
		return
	}
	require.NoError(t, q.InsertConnectionAppTarget(t.Context(), InsertConnectionAppTargetParams{
		ConnectionID:        c.ID,
		SelectionMode:       ConnectionAppTargetsSelectionMode(c.Mode),
		TargetEnvironmentID: sql.NullString{String: c.TargetEnvironmentID, Valid: c.TargetEnvironmentID != ""},
		TargetDeploymentID:  sql.NullString{String: c.TargetDeploymentID, Valid: c.TargetDeploymentID != ""},
	}))
}

func insertTestSavedConnection(t *testing.T, q *Queries, deploymentID string, c testConnection) {
	t.Helper()
	require.NoError(t, q.InsertDeploymentConnection(t.Context(), InsertDeploymentConnectionParams{
		DeploymentID: deploymentID, ConnectionID: c.ID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID,
		AppID: c.AppID, EnvironmentID: c.EnvironmentID, ResourceType: c.resourceType(), ResourceID: c.ResourceID,
		Name: c.Name, CreatedAt: 1,
	}))
	if c.Mode == "" {
		return
	}
	require.NoError(t, q.InsertDeploymentConnectionAppTarget(t.Context(), InsertDeploymentConnectionAppTargetParams{
		DeploymentID:        deploymentID,
		ConnectionID:        c.ID,
		SelectionMode:       DeploymentConnectionAppTargetsSelectionMode(c.Mode),
		TargetEnvironmentID: sql.NullString{String: c.TargetEnvironmentID, Valid: c.TargetEnvironmentID != ""},
		TargetDeploymentID:  sql.NullString{String: c.TargetDeploymentID, Valid: c.TargetDeploymentID != ""},
	}))
}
