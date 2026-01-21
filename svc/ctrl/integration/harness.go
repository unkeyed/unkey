package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

// Harness provides a test environment for ctrl service integration tests.
// It sets up MySQL connection and seeded data for testing the sync functionality.
type Harness struct {
	t              *testing.T
	ctx            context.Context
	Seed           *seed.Seeder
	DB             db.Database
	versionCounter uint64
}

// New creates a new integration test harness.
func New(t *testing.T) *Harness {
	t.Helper()

	ctx := context.Background()

	mysqlHostCfg := containers.MySQL(t)
	mysqlHostCfg.DBName = "unkey"
	mysqlHostDSN := mysqlHostCfg.FormatDSN()

	database, err := db.New(db.Config{
		Logger:      logging.NewNoop(),
		PrimaryDSN:  mysqlHostDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	h := &Harness{
		t:              t,
		ctx:            ctx,
		Seed:           seed.New(t, database, nil),
		DB:             database,
		versionCounter: 0,
	}

	h.Seed.Seed(ctx)

	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	return h
}

// Context returns the test context.
func (h *Harness) Context() context.Context {
	return h.ctx
}

// Resources returns the seeded resources.
func (h *Harness) Resources() seed.Resources {
	return h.Seed.Resources
}

// InsertStateChange inserts a state change record for testing.
// Returns the auto-generated sequence number.
func (h *Harness) InsertStateChange(ctx context.Context, params db.InsertStateChangeParams) int64 {
	seq, err := db.Query.InsertStateChange(ctx, h.DB.RW(), params)
	require.NoError(h.t, err)
	return seq
}

// Now returns current time in milliseconds.
func (h *Harness) Now() int64 {
	return time.Now().UnixMilli()
}

// CreateDeploymentRequest contains parameters for creating a test deployment.
type CreateDeploymentRequest struct {
	Region       string
	DesiredState db.DeploymentsDesiredState
}

// CreateDeploymentResult contains the created deployment and topology.
type CreateDeploymentResult struct {
	Deployment db.Deployment
	Topology   db.DeploymentTopology
}

// CreateDeployment creates a deployment with topology for testing.
func (h *Harness) CreateDeployment(ctx context.Context, req CreateDeploymentRequest) CreateDeploymentResult {
	workspaceID := h.Seed.Resources.UserWorkspace.ID

	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      workspaceID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		GitRepositoryURL: "",
		DefaultBranch:    "",
		DeleteProtection: false,
	})

	env := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New("env"),
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		Slug:             "production",
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	deploymentID := uid.New("dep")
	k8sName := uid.New("k8s")

	err := db.Query.InsertDeployment(ctx, h.DB.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       k8sName,
		WorkspaceID:                   workspaceID,
		ProjectID:                     project.ID,
		EnvironmentID:                 env.ID,
		GitCommitSha:                  sql.NullString{Valid: false},
		GitBranch:                     sql.NullString{Valid: false},
		SentinelConfig:                []byte("{}"),
		GitCommitMessage:              sql.NullString{Valid: false},
		GitCommitAuthorHandle:         sql.NullString{Valid: false},
		GitCommitAuthorAvatarUrl:      sql.NullString{Valid: false},
		GitCommitTimestamp:            sql.NullInt64{Valid: false},
		OpenapiSpec:                   sql.NullString{Valid: false},
		EncryptedEnvironmentVariables: []byte(""),
		Status:                        db.DeploymentsStatusReady,
		CpuMillicores:                 100,
		MemoryMib:                     128,
		CreatedAt:                     h.Now(),
		UpdatedAt:                     sql.NullInt64{Valid: false},
		Command:                       json.RawMessage([]byte("[]")),
	})
	require.NoError(h.t, err)

	// Update desired_state (insert doesn't set it, but it defaults to running)
	if req.DesiredState != "" && req.DesiredState != db.DeploymentsDesiredStateRunning {
		_, err = h.DB.RW().ExecContext(ctx, "UPDATE deployments SET desired_state = ? WHERE id = ?", req.DesiredState, deploymentID)
		require.NoError(h.t, err)
	}

	// Set image (required for streaming)
	_, err = h.DB.RW().ExecContext(ctx, "UPDATE deployments SET image = ? WHERE id = ?", "nginx:1.19", deploymentID)
	require.NoError(h.t, err)

	h.versionCounter++
	err = db.Query.InsertDeploymentTopology(ctx, h.DB.RW(), db.InsertDeploymentTopologyParams{
		WorkspaceID:     workspaceID,
		DeploymentID:    deploymentID,
		Region:          req.Region,
		DesiredReplicas: 1,
		DesiredStatus:   db.DeploymentTopologyDesiredStatusStarted,
		Version:         h.versionCounter,
		CreatedAt:       h.Now(),
	})
	require.NoError(h.t, err)

	deployment, err := db.Query.FindDeploymentById(ctx, h.DB.RO(), deploymentID)
	require.NoError(h.t, err)

	return CreateDeploymentResult{
		Deployment: deployment,
		Topology: db.DeploymentTopology{
			Pk:              0,
			WorkspaceID:     workspaceID,
			DeploymentID:    deploymentID,
			Region:          req.Region,
			DesiredReplicas: 1,
			DesiredStatus:   db.DeploymentTopologyDesiredStatusStarted,
			Version:         h.versionCounter,
			CreatedAt:       h.Now(),
			UpdatedAt:       sql.NullInt64{Valid: false},
		},
	}
}

// CreateSentinelRequest contains parameters for creating a test sentinel.
type CreateSentinelRequest struct {
	Region       string
	DesiredState db.SentinelsDesiredState
}

// CreateSentinel creates a sentinel for testing.
func (h *Harness) CreateSentinel(ctx context.Context, req CreateSentinelRequest) db.Sentinel {
	workspaceID := h.Seed.Resources.UserWorkspace.ID

	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      workspaceID,
		Name:             "test-project-sentinel",
		Slug:             uid.New("slug"),
		GitRepositoryURL: "",
		DefaultBranch:    "",
		DeleteProtection: false,
	})

	env := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New("env"),
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		Slug:             "production",
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	sentinelID := uid.New("sen")
	k8sName := uid.New("k8s")

	desiredState := req.DesiredState
	if desiredState == "" {
		desiredState = db.SentinelsDesiredStateRunning
	}

	h.versionCounter++
	err := db.Query.InsertSentinel(ctx, h.DB.RW(), db.InsertSentinelParams{
		ID:                sentinelID,
		WorkspaceID:       workspaceID,
		EnvironmentID:     env.ID,
		ProjectID:         project.ID,
		K8sAddress:        "http://localhost:8080",
		K8sName:           k8sName,
		Region:            req.Region,
		Image:             "sentinel:1.0",
		Health:            db.SentinelsHealthHealthy,
		DesiredReplicas:   1,
		AvailableReplicas: 1,
		CpuMillicores:     100,
		MemoryMib:         128,
		Version:           h.versionCounter,
		CreatedAt:         h.Now(),
	})
	require.NoError(h.t, err)

	// Update desired_state if needed
	if desiredState != db.SentinelsDesiredStateRunning {
		_, err = h.DB.RW().ExecContext(ctx, "UPDATE sentinels SET desired_state = ? WHERE id = ?", desiredState, sentinelID)
		require.NoError(h.t, err)
	}

	sentinel, err := db.Query.FindSentinelByID(ctx, h.DB.RO(), sentinelID)
	require.NoError(h.t, err)

	return sentinel
}
