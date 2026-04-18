package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

// Harness provides a test environment for ctrl service integration tests.
// It sets up MySQL connection and seeded data for testing the sync functionality.
type Harness struct {
	t    *testing.T
	ctx  context.Context
	Seed *seed.Seeder
	DB   db.Database
}

// New creates a new integration test harness.
func New(t *testing.T) *Harness {
	t.Helper()

	ctx := context.Background()

	mysqlCfg := dockertest.MySQL(t)
	mysqlHostDSN := mysqlCfg.DSN

	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlHostDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	h := &Harness{
		t:    t,
		ctx:  ctx,
		Seed: seed.New(t, database, nil),
		DB:   database,
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
		ID:          uid.New("prj"),
		WorkspaceID: workspaceID,
		Name:        "test-project",
		Slug:        uid.New("slug"),

		DeleteProtection: false,
	})

	app := h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "default",
		Slug:          "default",
		DefaultBranch: "main",
	})

	env := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New("env"),
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		AppID:            app.ID,
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
		AppID:                         app.ID,
		EnvironmentID:                 env.ID,
		GitCommitSha:                  sql.NullString{Valid: false},
		GitBranch:                     sql.NullString{Valid: false},
		SentinelConfig:                []byte("{}"),
		GitCommitMessage:              sql.NullString{Valid: false},
		GitCommitAuthorHandle:         sql.NullString{Valid: false},
		GitCommitAuthorAvatarUrl:      sql.NullString{Valid: false},
		GitCommitTimestamp:            sql.NullInt64{Valid: false},
		EncryptedEnvironmentVariables: []byte(""),
		Status:                        db.DeploymentsStatusReady,
		CpuMillicores:                 100,
		MemoryMib:                     128,
		StorageMib:                    0,
		Port:                          8080,
		ShutdownSignal:                db.DeploymentsShutdownSignalSIGTERM,
		UpstreamProtocol:              db.DeploymentsUpstreamProtocolHttp1,
		Healthcheck:                   dbtype.NullHealthcheck{Healthcheck: nil, Valid: false},
		PrNumber:                      sql.NullInt64{Int64: 0, Valid: false},
		ForkRepositoryFullName:        sql.NullString{String: "", Valid: false},
		CreatedAt:                     h.Now(),
		UpdatedAt:                     sql.NullInt64{Valid: false},
		Command:                       nil,
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

	// Ensure the region exists
	regionID := uid.New(uid.RegionPrefix)
	err = db.Query.UpsertRegion(ctx, h.DB.RW(), db.UpsertRegionParams{
		ID:       regionID,
		Name:     req.Region,
		Platform: "test",
	})
	require.NoError(h.t, err)

	region, err := db.Query.FindRegionByNameAndPlatform(ctx, h.DB.RO(), db.FindRegionByNameAndPlatformParams{
		Name:     req.Region,
		Platform: "test",
	})
	require.NoError(h.t, err)

	err = db.Query.InsertDeploymentTopology(ctx, h.DB.RW(), db.InsertDeploymentTopologyParams{
		WorkspaceID:                workspaceID,
		DeploymentID:               deploymentID,
		RegionID:                   region.ID,
		AutoscalingReplicasMin:     1,
		AutoscalingReplicasMax:     1,
		AutoscalingThresholdCpu:    sql.NullInt16{Valid: false},
		AutoscalingThresholdMemory: sql.NullInt16{Valid: false},
		DesiredStatus:              db.DeploymentTopologyDesiredStatusRunning,
		CreatedAt:                  h.Now(),
	})
	require.NoError(h.t, err)

	deployment, err := db.Query.FindDeploymentById(ctx, h.DB.RO(), deploymentID)
	require.NoError(h.t, err)

	return CreateDeploymentResult{
		Deployment: deployment,
		Topology: db.DeploymentTopology{
			Pk:                         0,
			WorkspaceID:                workspaceID,
			DeploymentID:               deploymentID,
			RegionID:                   regionID,
			AutoscalingReplicasMin:     1,
			AutoscalingReplicasMax:     1,
			AutoscalingThresholdCpu:    sql.NullInt16{Valid: false},
			AutoscalingThresholdMemory: sql.NullInt16{Valid: false},
			DesiredStatus:              db.DeploymentTopologyDesiredStatusRunning,
			CreatedAt:                  h.Now(),
			UpdatedAt:                  sql.NullInt64{Valid: false},
		},
	}
}

// CreateSentinelRequest contains parameters for creating a test sentinel.
type CreateSentinelRequest struct {
	RegionID     string
	DesiredState db.SentinelsDesiredState
}

// CreateSentinel creates a sentinel for testing.
func (h *Harness) CreateSentinel(ctx context.Context, req CreateSentinelRequest) db.Sentinel {
	workspaceID := h.Seed.Resources.UserWorkspace.ID

	project := h.Seed.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New("prj"),
		WorkspaceID: workspaceID,
		Name:        "test-project-sentinel",
		Slug:        uid.New("slug"),

		DeleteProtection: false,
	})

	sentinelApp := h.Seed.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "default",
		Slug:          "default",
		DefaultBranch: "main",
	})

	env := h.Seed.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New("env"),
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		AppID:            sentinelApp.ID,
		Slug:             "production",
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	sentinelID := uid.New("sen")
	k8sName := uid.New("k8s")
	subscriptionID := uid.New(uid.SentinelSubscriptionPrefix)

	desiredState := req.DesiredState
	if desiredState == "" {
		desiredState = db.SentinelsDesiredStateRunning
	}

	err := db.Tx(ctx, h.DB.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if err := h.seedSentinelTier(txCtx, tx); err != nil {
			return err
		}
		if err := db.Query.InsertSentinelSubscription(txCtx, tx, db.InsertSentinelSubscriptionParams{
			ID:             subscriptionID,
			SentinelID:     sentinelID,
			WorkspaceID:    workspaceID,
			RegionID:       req.RegionID,
			TierID:         integrationSentinelTierID,
			TierVersion:    integrationSentinelTierVersion,
			CpuMillicores:  100,
			MemoryMib:      128,
			Replicas:       1,
			PricePerSecond: "0",
			CreatedAt:      h.Now(),
		}); err != nil {
			return err
		}
		return db.Query.InsertSentinel(txCtx, tx, db.InsertSentinelParams{
			ID:              sentinelID,
			WorkspaceID:     workspaceID,
			EnvironmentID:   env.ID,
			ProjectID:       project.ID,
			SubscriptionID:  subscriptionID,
			K8sAddress:      "http://localhost:8080",
			K8sName:         k8sName,
			RegionID:        req.RegionID,
			Image:           "sentinel:1.0",
			DesiredReplicas: 1,
			CreatedAt:       h.Now(),
		})
	})
	require.NoError(h.t, err)

	// Seed observed state so tests can route to the sentinel.
	err = db.Query.UpdateSentinelObservedState(ctx, h.DB.RW(), db.UpdateSentinelObservedStateParams{
		K8sName:           k8sName,
		RunningImage:      "sentinel:1.0",
		AvailableReplicas: 1,
		Health:            db.SentinelsHealthHealthy,
		UpdatedAt:         sql.NullInt64{Valid: true, Int64: h.Now()},
	})
	require.NoError(h.t, err)

	// Update desired_state if needed
	if desiredState != db.SentinelsDesiredStateRunning {
		_, err = h.DB.RW().ExecContext(ctx, "UPDATE sentinels SET desired_state = ? WHERE id = ?", desiredState, sentinelID)
		require.NoError(h.t, err)
	}

	joined, err := db.Query.FindSentinelByID(ctx, h.DB.RO(), sentinelID)
	require.NoError(h.t, err)

	return joined.Sentinel
}

const (
	integrationSentinelTierID      = "mx-test"
	integrationSentinelTierVersion = "test"
)

// seedSentinelTier inserts the fixed tier row used by integration-test
// sentinels. INSERT IGNORE via InsertSentinelTier makes repeated calls safe.
func (h *Harness) seedSentinelTier(ctx context.Context, tx db.DBTX) error {
	return db.Query.InsertSentinelTier(ctx, tx, db.InsertSentinelTierParams{
		ID:             "tier_integration",
		TierID:         integrationSentinelTierID,
		Version:        integrationSentinelTierVersion,
		CpuMillicores:  100,
		MemoryMib:      128,
		PricePerSecond: "0",
		EffectiveFrom:  h.Now(),
	})
}
