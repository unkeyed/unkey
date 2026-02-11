package api

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
)

type mockDeploymentService struct {
	hydrav1.UnimplementedDeploymentServiceServer
	requests chan *hydrav1.DeployRequest
}

func (m *mockDeploymentService) Deploy(ctx restate.WorkflowSharedContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	m.requests <- req
	return &hydrav1.DeployResponse{}, nil
}

func TestDeployment_Create_TriggersWorkflow(t *testing.T) {
	requests := make(chan *hydrav1.DeployRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeploymentServiceServer(&mockDeploymentService{requests: requests})},
	})

	ctx := harness.RequestContext()
	workspaceID := harness.Seed.Resources.UserWorkspace.ID
	project := harness.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      workspaceID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DefaultBranch:    "main",
		DeleteProtection: false,
	})
	environment := harness.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New("env"),
		WorkspaceID:      workspaceID,
		ProjectID:        project.ID,
		Slug:             "production",
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	client := ctrlv1connect.NewDeploymentServiceClient(harness.ConnectClient(), harness.CtrlURL, harness.ConnectOptions()...)
	resp, err := client.CreateDeployment(ctx, connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
		ProjectId:       project.ID,
		EnvironmentSlug: environment.Slug,
		DockerImage:     "nginx:latest",
	}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetDeploymentId())
	require.Equal(t, ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING, resp.Msg.GetStatus())

	select {
	case req := <-requests:
		require.Equal(t, resp.Msg.GetDeploymentId(), req.GetDeploymentId())
		dockerImage, ok := req.GetSource().(*hydrav1.DeployRequest_DockerImage)
		require.True(t, ok, "expected DockerImage source")
		require.Equal(t, "nginx:latest", dockerImage.DockerImage.GetImage())
	case <-time.After(10 * time.Second):
		t.Fatal("expected deployment workflow invocation")
	}

	deployment, err := db.Query.FindDeploymentById(ctx, harness.DB.RO(), resp.Msg.GetDeploymentId())
	require.NoError(t, err)
	require.Equal(t, project.ID, deployment.ProjectID)
	require.Equal(t, db.DeploymentsStatusPending, deployment.Status)
}
