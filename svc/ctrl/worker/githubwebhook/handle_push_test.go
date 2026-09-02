package githubwebhook_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
	"github.com/unkeyed/unkey/svc/ctrl/worker/githubwebhook"
	"google.golang.org/protobuf/encoding/protojson"
)

// The build and runtime settings every fixture app carries. They differ from
// the seeder's defaults, so an assertion on a deployment row proves the row was
// filled from this app's settings and not from a default that happened to match.
const (
	fixtureDockerfile               = "docker/KEBAP.Dockerfile"
	fixtureDockerContext            = "services/kebap"
	fixtureBuildCommand             = "make KEBAP"
	fixturePort              int32  = 9091
	fixtureCPUMillicores     int32  = 500
	fixtureMemoryMiB         int32  = 512
	fixtureSender                   = "kebap-chef"
	fixtureAvatarURL                = "https://github.com/kebap-chef.png"
	fixtureProductionEnvSlug        = "production"
	fixturePreviewEnvSlug           = "preview"
	fixtureDefaultBranch            = "main"
	fixtureMatchingFile             = "services/kebap/main.go"
	fixtureMatchingWatchPath        = "services/kebap/**"
	fixtureOtherWatchPath           = "services/lahmacun/**"
	fixtureStorageMiB        uint32 = 0
)

var fixtureCommand = mysqltype.StringSlice{"./KEBAP", "serve"}

// TestHandlePushSkipsWhenNotDeployable pins the two reasons a matched app
// records the push but builds nothing. Both leave a row behind: the dashboard
// reads deployments to show that a commit arrived, so a silent drop would make
// the commit invisible to the user.
func TestHandlePushSkipsWhenNotDeployable(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	t.Run("auto deploy disabled", func(t *testing.T) {
		target := h.newTarget(t, ctx, targetOptions{})
		app := h.newApp(t, ctx, target, appOptions{disableAutoDeploy: true})

		h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

		row := h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusSkipped))
		require.Equal(t, app.productionEnvID, row.environmentID,
			"a push to the default branch belongs to the production environment")

		// auto_deploy off is the user asking for manual control, so a build here
		// would spend their money.
		h.requireNoDeploy(t, row.id)
	})

	t.Run("watch paths do not match changed files", func(t *testing.T) {
		target := h.newTarget(t, ctx, targetOptions{})
		app := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureOtherWatchPath}})

		h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

		row := h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusSkipped))
		require.Equal(t, app.productionEnvID, row.environmentID)
		h.requireNoDeploy(t, row.id)
	})
}

// TestHandlePushQueuesGitDeployment pins the successful path. The part easiest
// to lose is the invocation id landing back on the row: a cancel needs it, so a
// deployment created without one can never be stopped.
func TestHandlePushQueuesGitDeployment(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	t.Run("pending row, queued step and deploy invocation", func(t *testing.T) {
		target := h.newTarget(t, ctx, targetOptions{})
		app := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureMatchingWatchPath}})

		push := target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile})
		h.push(t, ctx, push)

		row := h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusPending))
		require.Equal(t, app.productionEnvID, row.environmentID)

		// A rebuild resolves the source from these columns, so a lost field is a
		// deployment nobody can trace back to a commit.
		require.Equal(t, push.GetAfter(), row.commitSHA.String)
		require.Equal(t, push.GetBranch(), row.branch.String)
		require.Equal(t, push.GetCommitMessage(), row.commitMessage.String)
		require.Equal(t, push.GetCommitAuthorHandle(), row.authorHandle.String)
		require.Equal(t, push.GetCommitAuthorAvatarUrl(), row.authorAvatar.String)
		require.Equal(t, push.GetCommitTimestamp(), row.commitTimestamp.Int64)

		// A branch push is neither a PR nor a fork, and leaving either field
		// set would send the build to the wrong ref.
		require.False(t, row.prNumber.Valid)
		require.False(t, row.forkRepository.Valid)

		// Billing and the audit trail both distinguish a webhook deployment from
		// a CLI or dashboard one.
		require.Equal(t, string(db.DeploymentsTriggerGithub), row.trigger)
		require.Equal(t, push.GetSenderLogin(), row.triggeredBy.String)

		// The row, not the app settings, provisions the container, so the runtime
		// shape is copied at creation time. A later edit to the app must not
		// change a running deployment.
		require.Equal(t, fixtureCPUMillicores, row.cpuMillicores)
		require.Equal(t, fixtureMemoryMiB, row.memoryMib)
		require.Equal(t, fixturePort, row.port)
		require.Equal(t, fixtureCommand, row.command)

		// The dashboard reads deployment_steps for progress. Without an open
		// queued step a freshly created deployment looks stalled, because
		// Deploy only ends the queued step and never inserts one.
		h.requireOpenQueuedStep(t, ctx, row.id)

		// The Deploy payload is the whole build instruction. The workflow gets no
		// other input, so a field missing here is a field the builder never sees.
		sent := h.awaitDeploy(t, row.id)
		require.Equal(t, row.id, sent.GetDeploymentId())
		git := sent.GetGit()
		require.NotNil(t, git, "a webhook deployment must carry a git source, not an image")
		require.Equal(t, target.installationID, git.GetInstallationId())
		require.Equal(t, target.repoFullName, git.GetRepository())
		require.Equal(t, push.GetAfter(), git.GetCommitSha())
		require.Equal(t, fixtureDockerContext, git.GetContextPath())
		require.Equal(t, fixtureDockerfile, git.GetDockerfilePath())
		require.Equal(t, fixtureBuildCommand, git.GetBuildCommand())
		require.Zero(t, git.GetPrNumber())
		require.Empty(t, git.GetForkRepository())

		withInvocation := h.awaitDeployment(t, ctx, app.id, func(row deploymentRow) bool {
			return row.invocationID.Valid
		})
		require.NotEmpty(t, withInvocation.invocationID.String,
			"a deployment with no invocation id can never be cancelled")
	})

	t.Run("environment variables land on the row", func(t *testing.T) {
		target := h.newTarget(t, ctx, targetOptions{})
		app := h.newApp(t, ctx, target, appOptions{
			productionEnvVars: map[string]string{"MEAL": "KEBAP", "SIDE": "ayran"},
		})

		h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

		// The blob is the only copy of the secrets the container starts with. The
		// deploy workflow never re-reads app_environment_variables, so a variable
		// missing here is a variable missing at runtime.
		row := h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusPending))
		requireSecrets(t, row.envVars, map[string]string{"MEAL": "KEBAP", "SIDE": "ayran"})
	})
}

// TestHandlePushForkPRAwaitsApproval pins the security boundary. A fork PR runs
// code written by someone with no write access to the repository, so it must
// reach a project member for approval before anything builds, and it must land
// in preview with preview's secrets rather than production's.
func TestHandlePushForkPRAwaitsApproval(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx, targetOptions{})
	app := h.newApp(t, ctx, target, appOptions{
		productionEnvVars: map[string]string{"MEAL": "production-KEBAP"},
		previewEnvVars:    map[string]string{"MEAL": "preview-KEBAP"},
	})

	// The head ref is the default branch: a fork PR opened from the contributor's
	// own main must still resolve to preview, so the fork flag and not the branch
	// name decides the environment.
	push := target.newPush(fixtureDefaultBranch, nil)
	push.IsForkPr = true
	push.PrNumber = nextGitHubID()%9000 + 1000
	push.ForkRepositoryFullName = "kebap-chef/" + testSlug(uid.TestPrefix)

	// A fork PR arrives through the pull_request webhook, which carries no
	// commit list, so the changed files come from the GitHub API instead.
	h.github.setCommitFiles([]string{fixtureMatchingFile})

	h.push(t, ctx, push)

	row := h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusAwaitingApproval))

	// Both fields are what lets the build fetch refs/pull/<n>/head from the
	// base repo and clone the contributor's fork.
	require.Equal(t, push.GetPrNumber(), row.prNumber.Int64)
	require.Equal(t, push.GetForkRepositoryFullName(), row.forkRepository.String)

	require.Equal(t, app.previewEnvID, row.environmentID,
		"external code must never run against the production environment")
	requireSecrets(t, row.envVars, map[string]string{"MEAL": "preview-KEBAP"})

	// The approval gate is worthless if the build starts anyway.
	h.requireNoDeploy(t, row.id)
}

// TestHandlePushDropsIneligibleWorkspaces pins the three ways a push is dropped
// before any row is written. All three answer success: the handler runs inside a
// Restate invocation, and failing it would make Restate retry a permanently
// ineligible workspace forever and stall every later push for that repository.
func TestHandlePushDropsIneligibleWorkspaces(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	t.Run("workspace has no compute plan", func(t *testing.T) {
		target := h.newTarget(t, ctx, targetOptions{noComputePlan: true})
		app := h.newApp(t, ctx, target, appOptions{})

		h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

		// Not even a skipped row: an unentitled workspace must leave no trace
		// that would count against usage or show up as a deployment.
		h.requireNoDeployment(t, ctx, app.id)
	})

	t.Run("workspace is spend suspended", func(t *testing.T) {
		// Entitled and suspended, so the drop is attributable to the spend cap
		// rather than to a missing plan.
		target := h.newTarget(t, ctx, targetOptions{spendSuspended: true})
		app := h.newApp(t, ctx, target, appOptions{})

		h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

		h.requireNoDeployment(t, ctx, app.id)
	})

	t.Run("no repo connection matches", func(t *testing.T) {
		target := h.newTarget(t, ctx, targetOptions{})
		app := h.newApp(t, ctx, target, appOptions{})

		// A push from a repository nobody connected: the installation is real,
		// the repository is not one of ours.
		push := target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile})
		push.RepositoryId = nextGitHubID()

		h.push(t, ctx, push)

		h.requireNoDeployment(t, ctx, app.id)
	})
}

// TestHandlePushDecidesEachMatchedAppSeparately pins the monorepo case: one
// repository feeds several apps, and watch paths are what keeps a commit in one
// service from rebuilding all of them.
func TestHandlePushDecidesEachMatchedAppSeparately(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx, targetOptions{})
	matching := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureMatchingWatchPath}})
	other := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureOtherWatchPath}})

	h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

	deployed := h.awaitDeployment(t, ctx, matching.id, hasStatus(mysqltype.DeploymentsStatusPending))
	skipped := h.awaitDeployment(t, ctx, other.id, hasStatus(mysqltype.DeploymentsStatusSkipped))

	h.awaitDeploy(t, deployed.id)
	h.requireNoDeploy(t, skipped.id)
}

// pushHarness is one MySQL database and one Restate server hosting the real
// GitHubWebhookService next to stand-ins for everything it calls out to.
type pushHarness struct {
	database db.Database
	seeder   *seed.Seeder
	ingress  containers.RestateConfig
	deploys  *deployRecorder
	github   *fakeGitHub

	// region is what every environment schedules onto. A create refuses an
	// environment with none.
	region db.Region
}

func newPushHarness(t *testing.T, ctx context.Context) *pushHarness {
	t.Helper()

	// The approval decision reads this from the process environment, so a
	// developer who left it set locally would otherwise see every push here
	// come back awaiting approval.
	t.Setenv("FORCE_DEPLOYMENT_APPROVAL", "false")

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	gh := &fakeGitHub{Noop: githubclient.NewNoop()}

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	// The real Create, so a push produces a real deployment row. It applies the
	// same deploy gate the webhook does, and both are enforced here.
	workflow, err := deploy.New(deploy.Config{
		DB:            database,
		Auditlogs:     auditlogSvc,
		DefaultDomain: "test.example.com",
		DashboardURL:  "https://app.unkey.local",
		Vault:         nil,
		GitHub:        gh,
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

	deploys := &deployRecorder{
		Workflow: workflow,
		requests: make(map[string]*hydrav1.DeployRequest),
	}

	svc := githubwebhook.New(githubwebhook.Config{
		DB:                              database,
		GitHub:                          gh,
		AllowUnauthenticatedDeployments: false,
	})

	// Restate retries a call to an unregistered service indefinitely, so every
	// service reachable from this push has to be bound, or a test would hang
	// rather than fail. Create posts the awaiting-authorization commit status
	// through the GitHub client above, not through another service.
	ingressCfg := containers.Restate(t,
		hydrav1.NewGitHubWebhookServiceServer(svc),
		hydrav1.NewDeployServiceServer(deploys),
	)

	seeder := seed.New(t, database, nil)

	return &pushHarness{
		database: database,
		seeder:   seeder,
		ingress:  ingressCfg,
		deploys:  deploys,
		github:   gh,
		region:   seeder.CreateRegion(ctx, seed.CreateRegionRequest{Name: "kebap-1", Platform: "k8s"}),
	}
}

// deployTarget is one seeded workspace, project and GitHub repository. Each
// scenario gets its own so the handler's repository lookup only ever sees that
// scenario's connections.
type deployTarget struct {
	workspaceID    string
	projectID      string
	installationID int64
	repositoryID   int64
	repoFullName   string
}

type targetOptions struct {
	// noComputePlan leaves workspace_billing without a plan, which is what an
	// unentitled workspace looks like to the deploy gate.
	noComputePlan bool
	// spendSuspended is the state the spend cap leaves behind after it tears a
	// workspace's compute down.
	spendSuspended bool
}

func (h *pushHarness) newTarget(t *testing.T, ctx context.Context, opts targetOptions) deployTarget {
	t.Helper()

	workspace := h.seeder.CreateWorkspace(ctx)
	project := h.seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "KEBAP",
		Slug:             testSlug(uid.ProjectPrefix),
		DeleteProtection: false,
	})

	if !opts.noComputePlan {
		// plan_override is the manual-comp column the gate accepts alongside a
		// Stripe-synced plan. No generated query writes it.
		_, err := h.database.RW().ExecContext(ctx,
			"UPDATE workspace_billing SET plan_override = ? WHERE workspace_id = ?",
			"pro", workspace.ID)
		require.NoError(t, err)
	}

	if opts.spendSuspended {
		require.NoError(t, h.database.SetWorkspaceDeploySpendSuspended(ctx, db.SetWorkspaceDeploySpendSuspendedParams{
			Suspended: true,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ID:        workspace.ID,
		}))
	}

	return deployTarget{
		workspaceID:    workspace.ID,
		projectID:      project.ID,
		installationID: nextGitHubID(),
		repositoryID:   nextGitHubID(),
		repoFullName:   "unkeyed/" + testSlug(uid.TestPrefix),
	}
}

// targetApp is one deployable app wired to its target's repository, with both
// environments a push can resolve to.
type targetApp struct {
	id              string
	productionEnvID string
	previewEnvID    string
}

type appOptions struct {
	watchPaths        []string
	disableAutoDeploy bool
	productionEnvVars map[string]string
	previewEnvVars    map[string]string
}

// newApp seeds an app with both a production and a preview environment. Both
// always exist so that the environment recorded on a deployment row is a real
// choice the handler made and not the only row available.
func (h *pushHarness) newApp(t *testing.T, ctx context.Context, target deployTarget, opts appOptions) targetApp {
	t.Helper()

	app := h.seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   target.workspaceID,
		ProjectID:     target.projectID,
		Name:          "KEBAP",
		Slug:          testSlug(uid.AppPrefix),
		DefaultBranch: fixtureDefaultBranch,
	})

	result := targetApp{
		id:              app.ID,
		productionEnvID: uid.New(uid.EnvironmentPrefix),
		previewEnvID:    uid.New(uid.EnvironmentPrefix),
	}
	environments := []struct {
		id      string
		slug    string
		kind    mysqltype.EnvironmentKind
		envVars map[string]string
	}{
		{
			id:      result.productionEnvID,
			slug:    fixtureProductionEnvSlug,
			kind:    mysqltype.EnvironmentKindProduction,
			envVars: opts.productionEnvVars,
		},
		{
			id:      result.previewEnvID,
			slug:    fixturePreviewEnvSlug,
			kind:    mysqltype.EnvironmentKindPreview,
			envVars: opts.previewEnvVars,
		},
	}

	now := time.Now().UnixMilli()
	for _, env := range environments {
		h.seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
			ID:               env.id,
			WorkspaceID:      target.workspaceID,
			ProjectID:        target.projectID,
			AppID:            app.ID,
			Slug:             env.slug,
			Description:      "",
			Kind:             env.kind,
			SentinelConfig:   nil,
			DeleteProtection: false,
		})

		require.NoError(t, h.database.UpsertAppBuildSettings(ctx, db.UpsertAppBuildSettingsParams{
			WorkspaceID:   target.workspaceID,
			AppID:         app.ID,
			EnvironmentID: env.id,
			Dockerfile:    sql.NullString{Valid: true, String: fixtureDockerfile},
			DockerContext: fixtureDockerContext,
			BuildCommand:  sql.NullString{Valid: true, String: fixtureBuildCommand},
			WatchPaths:    opts.watchPaths,
			AutoDeploy:    !opts.disableAutoDeploy,
			CreatedAt:     now,
			UpdatedAt:     sql.NullInt64{Valid: false, Int64: 0},
		}))

		require.NoError(t, h.database.UpsertAppRuntimeSettings(ctx, db.UpsertAppRuntimeSettingsParams{
			WorkspaceID:      target.workspaceID,
			AppID:            app.ID,
			EnvironmentID:    env.id,
			Port:             fixturePort,
			CpuMillicores:    fixtureCPUMillicores,
			MemoryMib:        fixtureMemoryMiB,
			StorageMib:       fixtureStorageMiB,
			Command:          fixtureCommand,
			Healthcheck:      mysqltype.NullHealthcheck{Healthcheck: nil, Valid: false},
			ShutdownSignal:   db.AppRuntimeSettingsShutdownSignalSIGTERM,
			UpstreamProtocol: db.AppRuntimeSettingsUpstreamProtocolHttp1,
			SentinelConfig:   []byte("{}"),
			OpenapiSpecPath:  sql.NullString{Valid: false, String: ""},
			CreatedAt:        now,
			UpdatedAt:        sql.NullInt64{Valid: false, Int64: 0},
		}))

		// A create refuses an environment with nowhere to schedule, so every
		// environment carries the one region a deployable app has.
		require.NoError(t, h.database.UpsertAppRegionalSettings(ctx, db.UpsertAppRegionalSettingsParams{
			WorkspaceID:   target.workspaceID,
			AppID:         app.ID,
			EnvironmentID: env.id,
			RegionID:      h.region.ID,
			Replicas:      1,
			CreatedAt:     now,
			UpdatedAt:     sql.NullInt64{Valid: false, Int64: 0},
		}))

		for key, value := range env.envVars {
			require.NoError(t, h.database.InsertAppEnvironmentVariable(ctx, db.InsertAppEnvironmentVariableParams{
				ID:            uid.New(uid.EnvironmentVariablePrefix),
				WorkspaceID:   target.workspaceID,
				AppID:         app.ID,
				EnvironmentID: env.id,
				EnvKey:        key,
				Value:         value,
				CreatedAt:     now,
			}))
		}
	}

	// The connection is per app, keyed on the target's installation and
	// repository, which is how one monorepo feeds several apps.
	require.NoError(t, h.database.InsertGithubRepoConnection(ctx, db.InsertGithubRepoConnectionParams{
		WorkspaceID:        target.workspaceID,
		ProjectID:          target.projectID,
		AppID:              app.ID,
		InstallationID:     target.installationID,
		RepositoryID:       target.repositoryID,
		RepositoryFullName: target.repoFullName,
		CreatedAt:          now,
		UpdatedAt:          sql.NullInt64{Valid: false, Int64: 0},
	}))

	return result
}

// newPush builds the request ctrl-api forwards for a branch push. Every call
// invents a fresh commit so an assertion can only pass on values this push
// carried.
func (target deployTarget) newPush(branch string, changedFiles []string) *hydrav1.HandlePushRequest {
	return &hydrav1.HandlePushRequest{
		InstallationId:         target.installationID,
		RepositoryId:           target.repositoryID,
		RepositoryFullName:     target.repoFullName,
		Branch:                 branch,
		After:                  newCommitSHA(),
		CommitMessage:          "feat: serve " + uid.New(uid.TestPrefix),
		CommitAuthorHandle:     fixtureSender,
		CommitAuthorAvatarUrl:  fixtureAvatarURL,
		CommitTimestamp:        time.Now().UnixMilli(),
		DeliveryId:             uid.New(uid.TestPrefix),
		ChangedFiles:           changedFiles,
		SenderLogin:            fixtureSender,
		IsForkPr:               false,
		PrNumber:               0,
		ForkRepositoryFullName: "",
	}
}

// push invokes HandlePush through the Restate ingress the way the webhook
// endpoint does, and requires success: a rejected policy decision is a
// successful no-op, and any error here would be retried forever.
func (h *pushHarness) push(t *testing.T, ctx context.Context, req *hydrav1.HandlePushRequest) {
	t.Helper()

	// Restate reads a slash in an object key as a path separator, so ctrl-api
	// joins the installation and repository ids with a colon.
	key := fmt.Sprintf("%d:%d", req.GetInstallationId(), req.GetRepositoryId())
	_, err := hydrav1.NewGitHubWebhookServiceIngressClient(h.ingress.IngressClient, key).
		HandlePush().
		Request(ctx, req)
	require.NoError(t, err)
}

// deploymentRow is the part of a deployments row a push is responsible for
// filling in.
type deploymentRow struct {
	id              string
	environmentID   string
	status          string
	commitSHA       sql.NullString
	branch          sql.NullString
	commitMessage   sql.NullString
	authorHandle    sql.NullString
	authorAvatar    sql.NullString
	commitTimestamp sql.NullInt64
	prNumber        sql.NullInt64
	forkRepository  sql.NullString
	trigger         string
	triggeredBy     sql.NullString
	cpuMillicores   int32
	memoryMib       int32
	port            int32
	command         mysqltype.StringSlice
	envVars         []byte
	invocationID    sql.NullString
}

func (h *pushHarness) listDeployments(ctx context.Context, appID string) ([]deploymentRow, error) {
	rows, err := h.database.RO().QueryContext(ctx,
		"SELECT id, environment_id, status, git_commit_sha, git_branch, git_commit_message, "+
			"git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp, "+
			"pr_number, fork_repository_full_name, `trigger`, triggered_by, "+
			"cpu_millicores, memory_mib, port, command, encrypted_environment_variables, invocation_id "+
			"FROM deployments WHERE app_id = ? ORDER BY pk", appID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []deploymentRow
	for rows.Next() {
		var row deploymentRow
		if scanErr := rows.Scan(
			&row.id, &row.environmentID, &row.status, &row.commitSHA, &row.branch, &row.commitMessage,
			&row.authorHandle, &row.authorAvatar, &row.commitTimestamp,
			&row.prNumber, &row.forkRepository, &row.trigger, &row.triggeredBy,
			&row.cpuMillicores, &row.memoryMib, &row.port, &row.command, &row.envVars, &row.invocationID,
		); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// awaitDeployment polls for the single deployments row of an app that satisfies
// ready. The push only sends to Create, so the row is not there yet when the
// push returns.
func (h *pushHarness) awaitDeployment(
	t *testing.T,
	ctx context.Context,
	appID string,
	ready func(deploymentRow) bool,
) deploymentRow {
	t.Helper()

	var found deploymentRow
	require.Eventually(t, func() bool {
		rows, err := h.listDeployments(ctx, appID)
		if err != nil || len(rows) != 1 || !ready(rows[0]) {
			return false
		}
		found = rows[0]
		return true
	}, 60*time.Second, 100*time.Millisecond, "no deployment row reached the expected state for app %s", appID)

	return found
}

// hasStatus is the poll predicate for a deployment that reached a status.
func hasStatus(want mysqltype.DeploymentsStatus) func(deploymentRow) bool {
	return func(row deploymentRow) bool { return row.status == string(want) }
}

// requireNoDeployment holds the window open long enough that a row from an
// asynchronous Create would have landed, then fails if one appeared.
func (h *pushHarness) requireNoDeployment(t *testing.T, ctx context.Context, appID string) {
	t.Helper()

	require.Never(t, func() bool {
		rows, err := h.listDeployments(ctx, appID)
		return err == nil && len(rows) > 0
	}, 5*time.Second, 100*time.Millisecond, "a dropped push must leave no deployment row for app %s", appID)
}

func (h *pushHarness) requireOpenQueuedStep(t *testing.T, ctx context.Context, deploymentID string) {
	t.Helper()

	require.Eventually(t, func() bool {
		var endedAt sql.NullInt64
		err := h.database.RO().QueryRowContext(ctx,
			"SELECT ended_at FROM deployment_steps WHERE deployment_id = ? AND step = ?",
			deploymentID, string(db.DeploymentStepsStepQueued),
		).Scan(&endedAt)
		return err == nil && !endedAt.Valid
	}, 60*time.Second, 100*time.Millisecond,
		"deployment %s needs an open queued step to show progress", deploymentID)
}

// requireSecrets decodes the blob the deploy workflow hands to the container.
func requireSecrets(t *testing.T, blob []byte, want map[string]string) {
	t.Helper()

	var secrets ctrlv1.SecretsConfig
	require.NoError(t, protojson.Unmarshal(blob, &secrets))
	require.Equal(t, want, secrets.GetSecrets())
}

// deployRecorder is the real DeployService with only Deploy stubbed out, since
// building is not what these tests observe. Every other handler, Create above
// all, keeps its real behavior through the embedded workflow.
//
// Invocations are indexed by deployment id rather than queued: a Send lands
// asynchronously, so a Deploy from an earlier scenario can arrive during a
// later one and must not be mistaken for it.
type deployRecorder struct {
	*deploy.Workflow
	mu       sync.Mutex
	requests map[string]*hydrav1.DeployRequest
}

func (r *deployRecorder) Deploy(_ restate.ObjectContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests[req.GetDeploymentId()] = req
	return &hydrav1.DeployResponse{}, nil
}

func (r *deployRecorder) get(deploymentID string) *hydrav1.DeployRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requests[deploymentID]
}

func (h *pushHarness) awaitDeploy(t *testing.T, deploymentID string) *hydrav1.DeployRequest {
	t.Helper()

	var sent *hydrav1.DeployRequest
	require.Eventually(t, func() bool {
		sent = h.deploys.get(deploymentID)
		return sent != nil
	}, 60*time.Second, 100*time.Millisecond, "no Deploy invocation arrived for deployment %s", deploymentID)

	return sent
}

// requireNoDeploy holds the window open long enough that a send issued during
// the invocation would have been delivered.
func (h *pushHarness) requireNoDeploy(t *testing.T, deploymentID string) {
	t.Helper()

	require.Never(t, func() bool {
		return h.deploys.get(deploymentID) != nil
	}, 5*time.Second, 100*time.Millisecond, "deployment %s must not have been handed to Deploy", deploymentID)
}

// fakeGitHub answers the GitHub calls a push reaches. Embedding Noop covers the
// rest of the interface with methods that return errors, so an unexpected call
// fails loudly instead of passing silently.
type fakeGitHub struct {
	*githubclient.Noop
	mu          sync.Mutex
	commitFiles []string
}

func (f *fakeGitHub) setCommitFiles(files []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commitFiles = files
}

func (f *fakeGitHub) ListCommitFiles(_ int64, _ string, _ string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.commitFiles, nil
}

func (f *fakeGitHub) CreateCommitStatus(_ int64, _ string, _ string, _ string, _ string, _ string, _ string) error {
	return nil
}

// githubIDs hands out installation and repository ids. github_repo_connections
// rows outlive the seeder's workspace cleanup and MySQL is shared across runs,
// so the ids must not repeat between runs either.
var githubIDs = func() *atomic.Int64 {
	var counter atomic.Int64
	counter.Store(time.Now().UnixMicro())
	return &counter
}()

func nextGitHubID() int64 {
	return githubIDs.Add(1)
}

// newCommitSHA derives a commit-shaped value from a fresh id: git_commit_sha is
// only 40 characters wide, so a prefixed id does not fit.
func newCommitSHA() string {
	sum := sha256.Sum256([]byte(uid.New(uid.TestPrefix)))
	return hex.EncodeToString(sum[:])[:40]
}

func testSlug(prefix uid.Prefix) string {
	return strings.ToLower(strings.ReplaceAll(uid.New(prefix), "_", "-"))
}
