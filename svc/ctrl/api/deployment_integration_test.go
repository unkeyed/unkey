package api

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploymentcreate"
)

type mockDeployService struct {
	hydrav1.UnimplementedDeployServiceServer
	requests chan *hydrav1.DeployRequest
}

func (m *mockDeployService) Deploy(ctx restate.ObjectContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	m.requests <- req
	return &hydrav1.DeployResponse{}, nil
}

func TestDeployment_Create_TriggersWorkflow(t *testing.T) {
	requests := make(chan *hydrav1.DeployRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(&mockDeployService{requests: requests})},
	})

	ctx := harness.RequestContext()
	workspaceID := harness.Seed.Resources.UserWorkspace.ID
	target := seedDeployTarget(ctx, t, harness, workspaceID)

	client := ctrlv1connect.NewDeployServiceClient(harness.ConnectClient(), harness.CtrlURL, harness.ConnectOptions()...)
	resp, err := client.CreateDeployment(ctx, connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
		ProjectId:       target.project.ID,
		AppId:           target.app.ID,
		EnvironmentSlug: target.environment.Slug,
		DockerImage:     "nginx:latest",
	}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetDeploymentId())
	require.Equal(t, ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING, resp.Msg.GetStatus())
	require.False(t, resp.Msg.GetReplayed())

	select {
	case req := <-requests:
		require.Equal(t, resp.Msg.GetDeploymentId(), req.GetDeploymentId())
		dockerImage, ok := req.GetSource().(*hydrav1.DeployRequest_DockerImage)
		require.True(t, ok, "expected DockerImage source")
		require.Equal(t, "nginx:latest", dockerImage.DockerImage.GetImage())
	case <-time.After(10 * time.Second):
		t.Fatal("expected deployment workflow invocation")
	}

	deployment, err := harness.DB.FindDeploymentById(ctx, resp.Msg.GetDeploymentId())
	require.NoError(t, err)
	require.Equal(t, target.project.ID, deployment.ProjectID)
	require.Equal(t, mysqltype.DeploymentsStatusPending, deployment.Status)

	// The create worker records the invocation id so the deployment can be
	// cancelled, and its create audit commits with the insert.
	require.Eventually(t, func() bool {
		row, findErr := harness.DB.FindDeploymentById(ctx, resp.Msg.GetDeploymentId())
		return findErr == nil && row.InvocationID.Valid && row.InvocationID.String != ""
	}, 10*time.Second, 100*time.Millisecond, "the create must record the workflow invocation id")
	require.Equal(t, 1, countCreateAuditLogs(ctx, t, harness, workspaceID, resp.Msg.GetDeploymentId()))
}

func TestDeployment_Create_IdempotencyKey(t *testing.T) {
	requests := make(chan *hydrav1.DeployRequest, 16)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(&mockDeployService{requests: requests})},
	})

	ctx := harness.RequestContext()
	workspaceID := harness.Seed.Resources.UserWorkspace.ID
	target := seedDeployTarget(ctx, t, harness, workspaceID)

	client := ctrlv1connect.NewDeployServiceClient(harness.ConnectClient(), harness.CtrlURL, harness.ConnectOptions()...)

	create := func(tgt deployTarget, key string) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
		// nolint: exhaustruct // only the fields the create path reads matter here
		req := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       tgt.project.ID,
			AppId:           tgt.app.ID,
			EnvironmentSlug: tgt.environment.Slug,
			DockerImage:     "nginx:latest",
		}
		if key != "" {
			req.IdempotencyKey = &key
		}
		return client.CreateDeployment(ctx, connect.NewRequest(req))
	}

	// expectWorkflows asserts exactly n Deploy invocations arrive, then silence.
	expectWorkflows := func(t *testing.T, n int) {
		t.Helper()
		for i := range n {
			select {
			case <-requests:
			case <-time.After(10 * time.Second):
				t.Fatalf("expected %d deployment workflow invocations, got %d", n, i)
			}
		}
		// Short silence window: an unwanted duplicate send is enqueued before
		// the create RPC returns, so it lands within milliseconds of the
		// expected ones. Every subtest pays this wait in full.
		select {
		case req := <-requests:
			t.Fatalf("unexpected extra workflow invocation for deployment %s", req.GetDeploymentId())
		case <-time.After(500 * time.Millisecond):
		}
	}

	t.Run("same key twice returns the original deployment", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, key)
		require.NoError(t, err)
		second, err := create(target, key)
		require.NoError(t, err)

		require.Equal(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		require.False(t, first.Msg.GetReplayed(), "the creating request must not report a replay")
		require.True(t, second.Msg.GetReplayed(), "the retry must report a replay")

		expectWorkflows(t, 1)
	})

	// The key is scoped per (workspace, app, environment), so reusing it for
	// a different target dedupes nothing and creates its own deployment.
	t.Run("the same key deploys independently per app", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, key)
		require.NoError(t, err)

		otherTarget := seedDeployTarget(ctx, t, harness, workspaceID)
		second, err := create(otherTarget, key)
		require.NoError(t, err)

		require.NotEqual(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		require.False(t, second.Msg.GetReplayed())
		expectWorkflows(t, 2)
	})

	t.Run("the same key deploys independently per environment", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, key)
		require.NoError(t, err)

		staging := addDeployableEnvironment(ctx, t, harness, target, uid.New("slug"))
		second, err := create(deployTarget{project: target.project, app: target.app, environment: staging}, key)
		require.NoError(t, err)

		require.NotEqual(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		require.False(t, second.Msg.GetReplayed())
		expectWorkflows(t, 2)
	})

	t.Run("same key in another workspace creates its own deployment", func(t *testing.T) {
		otherWorkspace := harness.Seed.CreateWorkspaceWithLimits(ctx, seed.CreateWorkspaceWithLimitsRequest{
			RequestsPerMonth:       1_000_000,
			LogsRetentionDays:      30,
			AuditLogsRetentionDays: 30,
			Team:                   false,
		})
		otherTarget := seedDeployTarget(ctx, t, harness, otherWorkspace.ID)

		key := uid.New(uid.TestPrefix)
		first, err := create(target, key)
		require.NoError(t, err)
		second, err := create(otherTarget, key)
		require.NoError(t, err)

		require.NotEqual(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		expectWorkflows(t, 2)
	})

	t.Run("concurrent creates with one key produce one deployment", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		const workers = 4
		ids := make([]string, workers)
		replays := make([]bool, workers)
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := range ids {
			wg.Go(func() {
				resp, err := create(target, key)
				errs[i] = err
				if err == nil {
					ids[i] = resp.Msg.GetDeploymentId()
					replays[i] = resp.Msg.GetReplayed()
				}
			})
		}
		wg.Wait()

		created := 0
		for i := range ids {
			require.NoError(t, errs[i])
			require.Equal(t, ids[0], ids[i])
			if !replays[i] {
				created++
			}
		}
		require.Equal(t, 1, created, "exactly one request may report creating the deployment")

		// One invocation ran, so exactly one create audit exists.
		require.Equal(t, 1, countCreateAuditLogs(ctx, t, harness, workspaceID, ids[0]))
		expectWorkflows(t, 1)
	})

	// The journaled response predates the retry, so it reports the status the
	// deployment had at create time. The retry must answer with the row's
	// current status instead.
	t.Run("a replay answers with the deployment's current status", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, key)
		require.NoError(t, err)
		deploymentID := first.Msg.GetDeploymentId()

		require.NoError(t, harness.DB.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
			ID:        deploymentID,
			Status:    mysqltype.DeploymentsStatusReady,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		}))

		second, err := create(target, key)
		require.NoError(t, err)
		require.Equal(t, deploymentID, second.Msg.GetDeploymentId())
		require.True(t, second.Msg.GetReplayed())
		require.Equal(t, ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY, second.Msg.GetStatus(),
			"a replay must answer with the deployment's actual status, not a hardcoded pending")
		expectWorkflows(t, 1)
	})

	// Gates and source resolution run in ctrl on every request, before the
	// durable call. A retry that no longer passes them is rejected even
	// though its key is bound; the same retry with a valid body replays.
	t.Run("a retry passes the gates before it can replay", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, key)
		require.NoError(t, err)

		// Same key, broken body: a git commit on an app without a repo
		// connection fails source resolution in ctrl, before the call.
		// nolint: exhaustruct // only the fields the reject path reads matter here
		req := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       target.project.ID,
			AppId:           target.app.ID,
			EnvironmentSlug: target.environment.Slug,
			GitCommit:       &ctrlv1.GitCommitInfo{CommitSha: "0123456789abcdef0123456789abcdef01234567"},
			IdempotencyKey:  &key,
		}
		_, err = client.CreateDeployment(ctx, connect.NewRequest(req))
		require.Error(t, err)
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

		// The same key with a valid body still replays the original create.
		second, err := create(target, key)
		require.NoError(t, err)
		require.Equal(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		require.True(t, second.Msg.GetReplayed())
		expectWorkflows(t, 1)
	})

	// A key only binds once the durable call runs. That is why the dashboard
	// can keep one key across a corrected resubmit instead of rotating on
	// every edit: a request rejected by the gates or source resolution never
	// reaches Restate, so the key is still free.
	t.Run("a rejected attempt does not bind the key", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// A git commit on an app with no repo connection fails source
		// resolution, which runs before the durable call.
		// nolint: exhaustruct // only the fields the reject path reads matter here
		rejected := &ctrlv1.CreateDeploymentRequest{
			ProjectId:       target.project.ID,
			AppId:           target.app.ID,
			EnvironmentSlug: target.environment.Slug,
			GitCommit:       &ctrlv1.GitCommitInfo{CommitSha: "0123456789abcdef0123456789abcdef01234567"},
			IdempotencyKey:  &key,
		}
		_, err := client.CreateDeployment(ctx, connect.NewRequest(rejected))
		require.Error(t, err)

		// Same key, corrected body: this must deploy, not replay.
		resp, err := create(target, key)
		require.NoError(t, err)
		require.False(t, resp.Msg.GetReplayed(), "the corrected resubmit must create, not replay a rejection")
		expectWorkflows(t, 1)
	})

	t.Run("no key creates a deployment per request", func(t *testing.T) {
		first, err := create(target, "")
		require.NoError(t, err)
		second, err := create(target, "")
		require.NoError(t, err)

		require.NotEqual(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		expectWorkflows(t, 2)
	})

	t.Run("a key is trimmed before it keys the invocation", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		first, err := create(target, "  "+key+"\t")
		require.NoError(t, err)
		second, err := create(target, key)
		require.NoError(t, err)

		require.Equal(t, first.Msg.GetDeploymentId(), second.Msg.GetDeploymentId())
		require.True(t, second.Msg.GetReplayed())
		expectWorkflows(t, 1)
	})
}

// The deployment.create audit commits in the same transaction as the row
// insert, inside the create worker. An audit outage must fail the create and
// leave nothing behind.
func TestDeployment_Create_AuditFailureCreatesNothing(t *testing.T) {
	requests := make(chan *hydrav1.DeployRequest, 1)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(&mockDeployService{requests: requests})},
		DBServices: func(database db.Database) []restate.ServiceDefinition {
			return []restate.ServiceDefinition{hydrav1.NewDeploymentCreateServiceServer(
				deploymentcreate.New(deploymentcreate.Config{
					DB:           database,
					Auditlogs:    failingAuditlogs{},
					RestateAdmin: nil,
				}),
				// The insert step fails on every attempt, so kill fast instead
				// of blocking the caller for the full ingress timeout.
				restate.WithInvocationRetryPolicy(
					restate.WithInitialInterval(50*time.Millisecond),
					restate.WithMaxAttempts(2),
					restate.KillOnMaxAttempts(),
				),
			)}
		},
	})

	ctx := harness.RequestContext()
	workspaceID := harness.Seed.Resources.UserWorkspace.ID
	target := seedDeployTarget(ctx, t, harness, workspaceID)

	client := ctrlv1connect.NewDeployServiceClient(harness.ConnectClient(), harness.CtrlURL, harness.ConnectOptions()...)
	key := uid.New(uid.TestPrefix)
	// nolint: exhaustruct // only the fields the create path reads matter here
	_, err := client.CreateDeployment(ctx, connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
		ProjectId:       target.project.ID,
		AppId:           target.app.ID,
		EnvironmentSlug: target.environment.Slug,
		DockerImage:     "nginx:latest",
		IdempotencyKey:  &key,
	}))
	require.Error(t, err, "a create that cannot record its audit must fail")

	// The insert rolls back with the failed audit, so nothing exists: no
	// audit event and no workflow send.
	rows, listErr := harness.DB.ListClickhouseOutboxByWorkspace(ctx, workspaceID)
	require.NoError(t, listErr)
	for _, row := range rows {
		require.NotContains(t, string(row.Payload), string(auditlog.DeploymentCreateEvent))
	}
	select {
	case req := <-requests:
		t.Fatalf("unexpected workflow invocation for deployment %s", req.GetDeploymentId())
	case <-time.After(500 * time.Millisecond):
	}
}

// A resumed create whose insert already committed must feed sibling dedup the
// committed row's created_at. The retry's own clock would make a genuinely
// newer sibling look older than the resumed create and cancel it.
func TestDeployment_Create_DuplicateResumeKeepsRowCreatedAt(t *testing.T) {
	requests := make(chan *hydrav1.DeployRequest, 2)
	harness := newWebhookHarness(t, webhookHarnessConfig{
		Services: []restate.ServiceDefinition{hydrav1.NewDeployServiceServer(&mockDeployService{requests: requests})},
	})

	ctx := harness.RequestContext()
	workspaceID := harness.Seed.Resources.UserWorkspace.ID
	target := seedDeployTarget(ctx, t, harness, workspaceID)

	deploymentID := uid.New(uid.DeploymentPrefix)
	// nolint: exhaustruct // only the fields the create path reads matter here
	createReq := &hydrav1.DeploymentCreateRequest{
		Nonce:           uid.New("nonce"),
		ProjectId:       target.project.ID,
		AppId:           target.app.ID,
		EnvironmentSlug: target.environment.Slug,
		DeployRequest: &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Source:       &hydrav1.DeployRequest_DockerImage{DockerImage: &hydrav1.DockerImage{Image: "nginx:latest"}},
		},
		GitCommit: &ctrlv1.GitCommitInfo{
			Branch:        "main",
			CommitSha:     "0123456789abcdef0123456789abcdef01234567",
			CommitMessage: "KEBAP",
		},
		Trigger: ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_GITHUB,
		Action:  hydrav1.DeploymentCreateAction_DEPLOYMENT_CREATE_ACTION_CREATE,
	}

	workerClient := hydrav1.NewDeploymentCreateServiceIngressClient(restateingress.NewClient(harness.IngressURL))

	_, err := workerClient.Create().Request(ctx, createReq)
	require.NoError(t, err)
	row, err := harness.DB.FindDeploymentById(ctx, deploymentID)
	require.NoError(t, err)

	// A sibling on the same (app, environment, branch), created after the
	// first attempt's insert committed.
	siblingID := uid.New(uid.DeploymentPrefix)
	require.NoError(t, harness.DB.InsertDeployment(ctx, db.InsertDeploymentParams{
		ID:                            siblingID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   workspaceID,
		ProjectID:                     target.project.ID,
		AppID:                         target.app.ID,
		EnvironmentID:                 target.environment.ID,
		GitCommitSha:                  sql.NullString{Valid: false, String: ""},
		GitBranch:                     sql.NullString{Valid: true, String: "main"},
		SentinelConfig:                []byte("{}"),
		GitCommitMessage:              sql.NullString{Valid: true, String: "KEBAP"},
		GitCommitAuthorHandle:         sql.NullString{Valid: false, String: ""},
		GitCommitAuthorAvatarUrl:      sql.NullString{Valid: false, String: ""},
		GitCommitTimestamp:            sql.NullInt64{Valid: false, Int64: 0},
		EncryptedEnvironmentVariables: []byte{},
		Command:                       nil,
		Status:                        mysqltype.DeploymentsStatusPending,
		CpuMillicores:                 250,
		MemoryMib:                     256,
		StorageMib:                    0,
		Port:                          8080,
		ShutdownSignal:                db.DeploymentsShutdownSignalSIGTERM,
		UpstreamProtocol:              db.DeploymentsUpstreamProtocolHttp1,
		Healthcheck:                   mysqltype.NullHealthcheck{Valid: false, Healthcheck: nil},
		PrNumber:                      sql.NullInt64{Valid: false, Int64: 0},
		ForkRepositoryFullName:        sql.NullString{Valid: false, String: ""},
		DeploymentTrigger:             db.DeploymentsTriggerGithub,
		TriggeredBy:                   sql.NullString{Valid: false, String: ""},
		TriggerReason:                 sql.NullString{Valid: false, String: ""},
		CreatedAt:                     row.CreatedAt + 100,
		UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
	}))

	// Re-executing the same request simulates the resumed attempt: the
	// insert hits the id duplicate and must swallow it as success. The pause
	// keeps the retry's clock safely past the sibling's created_at.
	time.Sleep(time.Second)
	_, err = workerClient.Create().Request(ctx, createReq)
	require.NoError(t, err)

	sibling, err := harness.DB.FindDeploymentById(ctx, siblingID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusPending, sibling.Status,
		"a newer sibling must not be superseded by a resumed older create")
}

// failingAuditlogs simulates an audit outbox outage so tests can prove the
// deployment insert and its audit commit atomically.
type failingAuditlogs struct{}

func (failingAuditlogs) Insert(context.Context, db.DBTX, []auditlog.AuditLog) error {
	return errors.New("KEBAP audit outage")
}

type deployTarget struct {
	project     db.Project
	app         db.App
	environment db.Environment
}

// seedDeployTarget creates a project, app, environment, and schedulable
// regional settings so CreateDeployment passes the deployability gate.
func seedDeployTarget(ctx context.Context, t *testing.T, harness *webhookHarness, workspaceID string) deployTarget {
	t.Helper()

	project := harness.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New("prj"),
		WorkspaceID:      workspaceID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})

	envID := uid.New("env")
	app := harness.CreateAppWithSettings(ctx, seed.CreateAppRequest{
		ID:            uid.New("app"),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "default",
		Slug:          "default",
		DefaultBranch: "main",
	}, envID)

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

	target := deployTarget{project: project, app: app, environment: environment}
	seedRegionalSettings(ctx, t, harness, target, envID)
	return target
}

// addDeployableEnvironment attaches a second, fully deployable environment to
// an existing target's app: environment row, build and runtime settings, and
// schedulable regional settings.
func addDeployableEnvironment(ctx context.Context, t *testing.T, harness *webhookHarness, target deployTarget, slug string) db.Environment {
	t.Helper()

	envID := uid.New("env")
	environment := harness.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:               envID,
		WorkspaceID:      target.app.WorkspaceID,
		ProjectID:        target.project.ID,
		AppID:            target.app.ID,
		Slug:             slug,
		Kind:             mysqltype.EnvironmentKindPreview,
		Description:      "",
		SentinelConfig:   []byte("{}"),
		DeleteProtection: false,
	})

	now := time.Now().UnixMilli()
	require.NoError(t, harness.DB.UpsertAppBuildSettings(ctx, db.UpsertAppBuildSettingsParams{
		WorkspaceID:   target.app.WorkspaceID,
		AppID:         target.app.ID,
		EnvironmentID: envID,
		Dockerfile:    sql.NullString{Valid: false, String: ""},
		DockerContext: "",
		BuildCommand:  sql.NullString{Valid: false, String: ""},
		WatchPaths:    nil,
		AutoDeploy:    true,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Valid: false},
	}))
	require.NoError(t, harness.DB.UpsertAppRuntimeSettings(ctx, db.UpsertAppRuntimeSettingsParams{
		WorkspaceID:      target.app.WorkspaceID,
		AppID:            target.app.ID,
		EnvironmentID:    envID,
		Port:             8080,
		CpuMillicores:    250,
		MemoryMib:        256,
		StorageMib:       0,
		Command:          nil,
		Healthcheck:      mysqltype.NullHealthcheck{Healthcheck: nil, Valid: false},
		ShutdownSignal:   db.AppRuntimeSettingsShutdownSignalSIGTERM,
		UpstreamProtocol: db.AppRuntimeSettingsUpstreamProtocolHttp1,
		SentinelConfig:   []byte("{}"),
		OpenapiSpecPath:  sql.NullString{Valid: false},
		CreatedAt:        now,
		UpdatedAt:        sql.NullInt64{Valid: false},
	}))

	seedRegionalSettings(ctx, t, harness, target, envID)
	return environment
}

// seedRegionalSettings gives (app, environment) one schedulable region so the
// deployability gate passes.
func seedRegionalSettings(ctx context.Context, t *testing.T, harness *webhookHarness, target deployTarget, envID string) {
	t.Helper()

	region := harness.Seed.CreateRegion(ctx, seed.CreateRegionRequest{
		Name:     uid.New("rgn"),
		Platform: "test",
	})
	require.NoError(t, harness.DB.UpsertAppRegionalSettings(ctx, db.UpsertAppRegionalSettingsParams{
		WorkspaceID:   target.app.WorkspaceID,
		AppID:         target.app.ID,
		EnvironmentID: envID,
		RegionID:      region.ID,
		Replicas:      1,
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     sql.NullInt64{Valid: false},
	}))
}

// countCreateAuditLogs counts deployment.create outbox events that target the
// given deployment. Audit logs land in the clickhouse_outbox table.
func countCreateAuditLogs(ctx context.Context, t *testing.T, harness *webhookHarness, workspaceID, deploymentID string) int {
	t.Helper()

	rows, err := harness.DB.ListClickhouseOutboxByWorkspace(ctx, workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		payload := string(row.Payload)
		if strings.Contains(payload, string(auditlog.DeploymentCreateEvent)) && strings.Contains(payload, deploymentID) {
			count++
		}
	}
	return count
}
