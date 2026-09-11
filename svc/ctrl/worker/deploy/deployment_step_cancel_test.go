package deploy_test

import (
	"context"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
)

// TestDeploymentStepDoesNotReviveACancelledDeployment pins that a step started
// after a user cancel refuses to run and leaves the row cancelled instead of
// moving it back to a progressing status.
func TestDeploymentStepDoesNotReviveACancelledDeployment(t *testing.T) {
	ctx := context.Background()

	database, fixture := newDeployFixture(t, ctx)

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	workflow, err := deploy.New(deploy.Config{
		DB:            database,
		Auditlogs:     auditlogSvc,
		DefaultDomain: "test.example.com",
		DashboardURL:  "https://app.unkey.local",
		Vault:         nil,
		GitHub:        githubclient.NewNoop(),
		Build: deploy.BuildConfig{
			Backend:    deploy.BuildBackendDepot,
			Depot:      deploy.DepotConfig{APIUrl: "", ProjectRegion: "", ProjectPrefix: "builds-test"},
			Kubernetes: deploy.KubernetesBuildConfig{Namespace: "", Image: ""},
		},
		K8s:                             nil,
		RegistryConfig:                  deploy.RegistryConfig{Repository: "", Username: "", Password: "", Insecure: false},
		BuildPlatform:                   deploy.BuildPlatform{Platform: "", Architecture: ""},
		Clickhouse:                      nil,
		BuildSteps:                      batch.NewNoop[schema.BuildStepV1](),
		BuildStepLogs:                   batch.NewNoop[schema.BuildStepLogV1](),
		AllowUnauthenticatedDeployments: false,
		RestateAdmin:                    nil,
	})
	require.NoError(t, err)

	cancelled := fixture.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   fixture.workspaceID,
		ProjectID:     fixture.projectID,
		AppID:         fixture.appID,
		EnvironmentID: fixture.environmentID,
		Status:        mysqltype.DeploymentsStatusCancelled,
	})

	cfg := containers.Restate(t, restate.Reflect(StepProbe{workflow: workflow, db: database}))

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err = ingress.Object[string, string](cfg.IngressClient, "StepProbe", cancelled.ID, "Start").
		Request(callCtx, cancelled.ID)
	require.Error(t, err, "the step must refuse to start on a cancelled deployment")

	after, err := database.FindDeploymentById(ctx, cancelled.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusCancelled, after.Status,
		"a step start must not move a cancelled deployment back to %s", after.Status)
}

// StepProbe hosts DeploymentStep behind a Restate object so the test can hand
// it a real ObjectContext.
type StepProbe struct {
	workflow *deploy.Workflow
	db       db.Database
}

func (p StepProbe) Start(ctx restate.ObjectContext, deploymentID string) (string, error) {
	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Deployment, error) {
		return p.db.FindDeploymentById(runCtx, deploymentID)
	})
	if err != nil {
		return "", err
	}

	err = p.workflow.DeploymentStep(ctx, db.DeploymentStepsStepBuilding, deployment, func(restate.ObjectContext) error {
		return nil
	})
	if err != nil {
		return "", err
	}
	return deployment.ID, nil
}
