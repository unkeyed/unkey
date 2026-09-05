package api

import (
	"database/sql"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type mockDeployService struct {
	hydrav1.UnimplementedDeployServiceServer
	requests chan *hydrav1.DeployRequest
}

func (m *mockDeployService) Deploy(ctx restate.ObjectContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	m.requests <- req
	return &hydrav1.DeployResponse{}, nil
}

// mockDeploymentService stubs the DeploymentService (state-change RPCs) for tests
// that need the service registered in Restate but don't exercise its methods.
type mockDeploymentService struct {
	hydrav1.UnimplementedDeploymentServiceServer
	requests chan *hydrav1.DeployRequest
}

func TestDeployment_Create_UsesOCIAppDefault(t *testing.T) {
	requests := make(chan *hydrav1.DeployRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(&mockDeployService{requests: requests})},
	})

	ctx := harness.RequestContext()
	workspaceID := harness.Seed.Resources.UserWorkspace.ID
	require.NoError(t, harness.DB.SetWorkspaceDeployPlan(ctx, db.SetWorkspaceDeployPlanParams{
		Plan:        sql.NullString{String: "pro", Valid: true},
		WorkspaceID: workspaceID,
	}))
	project := harness.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      workspaceID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})

	envID := uid.New("env")

	app := harness.CreateAppWithSettings(ctx, seed.CreateAppRequest{
		ID:          uid.New("app"),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	}, envID)
	_, err := harness.DB.RW().ExecContext(ctx, "UPDATE apps SET source_type = ? WHERE id = ?", db.AppsSourceTypeOci, app.ID)
	require.NoError(t, err)
	_, err = harness.DB.RW().ExecContext(
		ctx,
		"INSERT INTO app_source_oci (workspace_id, app_id, image_reference, created_at) VALUES (?, ?, ?, ?)",
		workspaceID,
		app.ID,
		"nginx:latest",
		time.Now().UnixMilli(),
	)
	require.NoError(t, err)

	environment := harness.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               envID,
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "production",
		Kind:             mysqltype.EnvironmentKindProduction,
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	// Seed a schedulable region and regional settings so the environment passes
	// the deployability gate in CreateDeployment, which requires at least one
	// schedulable region.
	region := harness.Seed.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     "us-east-1",
		Platform: "test",
	})
	require.NoError(t, harness.DB.UpsertAppRegionalSettings(ctx, db.UpsertAppRegionalSettingsParams{
		WorkspaceID:   workspaceID,
		AppID:         app.ID,
		EnvironmentID: environment.ID,
		RegionID:      region.ID,
		Replicas:      1,
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     sql.NullInt64{Valid: false},
	}))

	client := ctrlv1connect.NewDeployServiceClient(harness.ConnectClient(), harness.CtrlURL, harness.ConnectOptions()...)
	resp, err := client.CreateDeployment(ctx, connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
		ProjectId:       project.ID,
		AppId:           app.ID,
		EnvironmentSlug: environment.Slug,
	}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetDeploymentId())
	require.Equal(t, ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING, resp.Msg.GetStatus())

	select {
	case req := <-requests:
		require.Equal(t, resp.Msg.GetDeploymentId(), req.GetDeploymentId())
		ociImage, ok := req.GetSource().(*hydrav1.DeployRequest_OciImage)
		require.True(t, ok, "expected OCI image source")
		require.Equal(t, "index.docker.io/library/nginx:latest", ociImage.OciImage.GetImage())
	case <-time.After(10 * time.Second):
		t.Fatal("expected deployment workflow invocation")
	}

	deployment, err := harness.DB.FindDeploymentById(ctx, resp.Msg.GetDeploymentId())
	require.NoError(t, err)
	require.Equal(t, project.ID, deployment.ProjectID)
	require.Equal(t, mysqltype.DeploymentsStatusPending, deployment.Status)
}
