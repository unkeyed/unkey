package githubwebhook_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
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
)

const (
	fixtureDockerfile               = "docker/KEBAP.Dockerfile"
	fixtureDockerContext            = "services/kebap"
	fixtureBuildCommand             = "make KEBAP"
	fixturePort              int32  = 9091
	fixtureCPUMillicores     int32  = 500
	fixtureMemoryMiB         int32  = 512
	fixtureStorageMiB        uint32 = 0
	fixtureSender                   = "kebap-chef"
	fixtureAvatarURL                = "https://github.com/kebap-chef.png"
	fixtureProductionEnvSlug        = "production"
	fixturePreviewEnvSlug           = "preview"
	fixtureDefaultBranch            = "main"
	fixtureMatchingFile             = "services/kebap/main.go"
	fixtureMatchingWatchPath        = "services/kebap/**"
	fixtureOtherWatchPath           = "services/lahmacun/**"
)

var fixtureCommand = mysqltype.StringSlice{"./KEBAP", "serve"}

// TestHandlePushSkipsWhenNotDeployable pins the reasons a matched app records
// the push but builds nothing. All leave a row behind: the dashboard reads
// deployments to show that a commit arrived, so a silent drop would make the
// commit invisible to the user.
func TestHandlePushSkipsWhenNotDeployable(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	t.Run("auto deploy disabled", func(t *testing.T) {
		target := h.newTarget(t, ctx)
		app := h.newApp(t, ctx, target, appOptions{disableAutoDeploy: true})

		h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

		row := h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusSkipped))
		require.Equal(t, app.productionEnvID, row.environmentID,
			"a push to the default branch belongs to the production environment")
		require.Equal(t, "Auto deploy is disabled for this environment.", row.triggerReason.String)
	})

	t.Run("watch paths do not match changed files", func(t *testing.T) {
		target := h.newTarget(t, ctx)
		app := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureOtherWatchPath}})

		h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

		row := h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusSkipped))
		require.Equal(t, app.productionEnvID, row.environmentID)
		require.Equal(t, "Watch paths did not match any changed files.", row.triggerReason.String)
	})

	// The lookup is retried for changedFilesRetryDuration, so a GitHub 5xx or
	// rate limit is ridden out rather than handed to the handler as an error.
	// That is why a push survives a GitHub wobble instead of being recorded as
	// skipped, and why the failure below needs a terminal error to reach the
	// handler at all.
	t.Run("a transient github failure is retried, not skipped", func(t *testing.T) {
		target := h.newTarget(t, ctx)
		app := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureMatchingWatchPath}})

		h.github.setCommitFiles([]string{fixtureMatchingFile})
		before := h.github.commitFilesCallCount()
		h.github.failCommitFilesTimes(2)

		h.push(t, ctx, target.newPush(fixtureDefaultBranch, nil))

		h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusPending))
		require.Greater(t, h.github.commitFilesCallCount()-before, 1,
			"the lookup has to be retried, not answered with an empty file list")
	})

	// Without a file list every watch path misses, so proceeding would record
	// the push as deliberately skipped over a failure of ours. The handler has
	// to fail instead and leave no row behind.
	t.Run("changed files could not be fetched", func(t *testing.T) {
		target := h.newTarget(t, ctx)
		app := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureOtherWatchPath}})

		h.github.setCommitFilesErr(restate.TerminalError(errors.New("KEBAP")))
		t.Cleanup(func() { h.github.setCommitFilesErr(nil) })

		key := fmt.Sprintf("%d:%d", target.installationID, target.repositoryID)
		_, err := hydrav1.NewGitHubWebhookServiceIngressClient(h.ingress.IngressClient, key).
			HandlePush().
			Request(ctx, target.newPush(fixtureDefaultBranch, nil))
		require.Error(t, err, "a push whose files cannot be read must not be answered with a skip")

		h.requireNoDeployment(t, ctx, app.id)
	})

	t.Run("watch path is not a valid glob", func(t *testing.T) {
		target := h.newTarget(t, ctx)
		app := h.newApp(t, ctx, target, appOptions{watchPaths: []string{"services/[kebap"}})

		h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

		// A broken pattern is indistinguishable from a miss, so the reason has to
		// name the pattern for the user to find it.
		row := h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusSkipped))
		require.Contains(t, row.triggerReason.String, "services/[kebap")
	})
}

// TestHandlePushQueuesGitDeployment pins what the push hands to Create. Create
// owns the rest of the row and is tested on its own; this only checks that the
// commit the webhook saw is the commit the row describes.
func TestHandlePushQueuesGitDeployment(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx)
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

	require.Equal(t, string(db.DeploymentsTriggerGithub), row.trigger)
	require.Equal(t, push.GetSenderLogin(), row.triggeredBy.String)
	require.False(t, row.triggerReason.Valid)
}

// TestHandlePushRoutesOtherBranchesToPreview pins the other half of the
// branch to environment rule: anything but the default branch is preview.
func TestHandlePushRoutesOtherBranchesToPreview(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx)
	app := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureMatchingWatchPath}})

	push := target.newPush("feat/"+testSlug(uid.TestPrefix), []string{fixtureMatchingFile})
	h.push(t, ctx, push)

	row := h.awaitDeployment(t, ctx, app.id, hasStatus(mysqltype.DeploymentsStatusPending))
	require.Equal(t, app.previewEnvID, row.environmentID)
	require.Equal(t, push.GetBranch(), row.branch.String)
}

// TestHandlePushForkPRAwaitsApproval pins the security boundary. A fork PR runs
// code written by someone with no write access to the repository, so it must
// reach a project member for approval before anything builds, and it must land
// in preview rather than production.
func TestHandlePushForkPRAwaitsApproval(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx)
	app := h.newApp(t, ctx, target, appOptions{})

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
}

// TestHandlePushIgnoresUnconnectedRepositories pins the early return: the
// installation is real, the repository is not one of ours, and the handler
// answers success without touching Create. Failing it would make Restate retry
// forever and stall every later push for that repository.
func TestHandlePushIgnoresUnconnectedRepositories(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx)
	app := h.newApp(t, ctx, target, appOptions{})

	push := target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile})
	push.RepositoryId = nextGitHubID()

	h.push(t, ctx, push)

	h.requireNoDeployment(t, ctx, app.id)
}

// TestHandlePushSurvivesARejectedCreate uses an environment with nowhere to
// schedule: the push is eligible, the watch paths match, and Create refuses.
//
// The push must still succeed. A rejection is a successful answer, and failing
// the whole delivery over one app would leave Restate retrying a repository
// that can never make progress.
func TestHandlePushSurvivesARejectedCreate(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx)
	app := h.newApp(t, ctx, target, appOptions{})

	// Every environment is seeded with a schedulable region, which is what makes
	// a create possible at all. Without one, Create rejects with
	// ENVIRONMENT_NOT_DEPLOYABLE.
	for _, environmentID := range []string{app.productionEnvID, app.previewEnvID} {
		require.NoError(t, h.database.DeleteAppRegionalSettingsByEnvironmentId(ctx, environmentID))
	}

	// require.NoError inside push is the assertion that matters: a rejected
	// create is logged and skipped, never propagated as a handler failure.
	push := target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile})
	h.push(t, ctx, push)

	h.requireNoDeployment(t, ctx, app.id)

	// No row means nothing in the dashboard, so the commit itself has to say why.
	statuses := h.github.commitStatuses()
	require.Len(t, statuses, 1)
	require.Equal(t, commitStatus{
		repo:        push.GetRepositoryFullName(),
		sha:         push.GetAfter(),
		state:       "error",
		description: deployfail.MsgNoSchedulableRegions,
		context:     githubclient.DeployRejectedContext,
	}, statuses[0])
}

// TestHandlePushKeepsInternalIdsOffTheCommitStatus pins the allowlist in
// postRejectedStatus. Everyone with read access to the repository reads a
// commit status, a fork PR's outside author included, so only an outcome whose
// detail describes the push carries that detail. A workspace-level refusal
// names the workspace, so it gets the fixed line instead.
func TestHandlePushKeepsInternalIdsOffTheCommitStatus(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx)
	app := h.newApp(t, ctx, target, appOptions{})

	// Dropping the granted plan is what makes Create answer NO_COMPUTE_PLAN,
	// whose detail names the workspace.
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE workspace_billing SET plan_override = NULL WHERE workspace_id = ?", target.workspaceID)
	require.NoError(t, err)

	push := target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile})
	h.push(t, ctx, push)

	h.requireNoDeployment(t, ctx, app.id)

	statuses := h.github.commitStatuses()
	require.Len(t, statuses, 1)
	require.NotContains(t, statuses[0].description, target.workspaceID,
		"the workspace id must never reach the commit status")
	require.Equal(t,
		"Unkey did not deploy this commit. Open the Unkey dashboard for the reason.",
		statuses[0].description)
}

// TestHandlePushReportsACreateThatNeverAnswered pins the commit status for a
// create that failed instead of answering. It writes no row, so the status is
// the only place the push is visible.
func TestHandlePushReportsACreateThatNeverAnswered(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx)
	app := h.newApp(t, ctx, target, appOptions{})

	// A key no validation would ever accept is corrupt stored data, which Create
	// answers with a terminal error rather than an outcome.
	require.NoError(t, h.database.InsertAppEnvironmentVariable(ctx, db.InsertAppEnvironmentVariableParams{
		ID:            uid.New(uid.EnvironmentVariablePrefix),
		WorkspaceID:   target.workspaceID,
		AppID:         app.id,
		EnvironmentID: app.productionEnvID,
		EnvKey:        "KEBAP-INVALID",
		Value:         "KEBAP",
		CreatedAt:     time.Now().UnixMilli(),
	}))

	h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

	h.requireNoDeployment(t, ctx, app.id)

	statuses := h.github.commitStatuses()
	require.Len(t, statuses, 1)
	require.Equal(t,
		"Unkey did not deploy this commit. Open the Unkey dashboard for the reason.",
		statuses[0].description)
}

// TestHandlePushDecidesEachMatchedAppSeparately pins the monorepo case: one
// repository feeds several apps, and watch paths are what keeps a commit in one
// service from rebuilding all of them. Two apps match so that their creates
// actually run concurrently.
func TestHandlePushDecidesEachMatchedAppSeparately(t *testing.T) {
	ctx := context.Background()
	h := newPushHarness(t, ctx)

	target := h.newTarget(t, ctx)
	matching := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureMatchingWatchPath}})
	alsoMatching := h.newApp(t, ctx, target, appOptions{watchPaths: []string{"services/**"}})
	other := h.newApp(t, ctx, target, appOptions{watchPaths: []string{fixtureOtherWatchPath}})

	h.push(t, ctx, target.newPush(fixtureDefaultBranch, []string{fixtureMatchingFile}))

	h.awaitDeployment(t, ctx, matching.id, hasStatus(mysqltype.DeploymentsStatusPending))
	h.awaitDeployment(t, ctx, alsoMatching.id, hasStatus(mysqltype.DeploymentsStatusPending))
	h.awaitDeployment(t, ctx, other.id, hasStatus(mysqltype.DeploymentsStatusSkipped))
}

// pushHarness is one MySQL database and one Restate server hosting the real
// GitHubWebhookService next to stand-ins for everything it calls out to.
type pushHarness struct {
	database db.Database
	seeder   *seed.Seeder
	ingress  containers.RestateConfig
	github   *fakeGitHub

	// Every environment is given this region. Create rejects an environment
	// with no region.
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

	// The real Create, so a push produces a real deployment row and the
	// entitlement gate is enforced.
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
		hydrav1.NewDeployServiceServer(&deployStub{Workflow: workflow}),
	)

	seeder := seed.New(t, database, nil)

	return &pushHarness{
		database: database,
		seeder:   seeder,
		ingress:  ingressCfg,
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

func (h *pushHarness) newTarget(t *testing.T, ctx context.Context) deployTarget {
	t.Helper()

	workspace := h.seeder.CreateWorkspace(ctx)
	project := h.seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      workspace.ID,
		Name:             "KEBAP",
		Slug:             testSlug(uid.ProjectPrefix),
		DeleteProtection: false,
	})

	// plan_override is a manually granted plan the entitlement gate accepts
	// alongside the Stripe-synced one. No generated query writes it.
	_, err := h.database.RW().ExecContext(ctx,
		"UPDATE workspace_billing SET plan_override = ? WHERE workspace_id = ?",
		"pro", workspace.ID)
	require.NoError(t, err)

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
}

// newApp seeds an app with both a production and a preview environment. Both
// always exist so that the environment recorded on a deployment row is a real
// choice the handler made and not the only row available.
func (h *pushHarness) newApp(t *testing.T, ctx context.Context, target deployTarget, opts appOptions) targetApp {
	t.Helper()

	app := h.seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: target.workspaceID,
		ProjectID:   target.projectID,
		Name:        "KEBAP",
		Slug:        testSlug(uid.AppPrefix),
	})

	result := targetApp{
		id:              app.ID,
		productionEnvID: uid.New(uid.EnvironmentPrefix),
		previewEnvID:    uid.New(uid.EnvironmentPrefix),
	}
	environments := []struct {
		id   string
		slug string
		kind mysqltype.EnvironmentKind
	}{
		{id: result.productionEnvID, slug: fixtureProductionEnvSlug, kind: mysqltype.EnvironmentKindProduction},
		{id: result.previewEnvID, slug: fixturePreviewEnvSlug, kind: mysqltype.EnvironmentKindPreview},
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
	triggerReason   sql.NullString
}

func (h *pushHarness) listDeployments(ctx context.Context, appID string) ([]deploymentRow, error) {
	rows, err := h.database.RO().QueryContext(ctx,
		"SELECT id, environment_id, status, git_commit_sha, git_branch, git_commit_message, "+
			"git_commit_author_handle, git_commit_author_avatar_url, git_commit_timestamp, "+
			"pr_number, fork_repository_full_name, `trigger`, triggered_by, trigger_reason "+
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
			&row.prNumber, &row.forkRepository, &row.trigger, &row.triggeredBy, &row.triggerReason,
		); scanErr != nil {
			return nil, scanErr
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// awaitDeployment polls for the single deployments row of an app that satisfies
// ready. Create writes the row before the push returns, but Deploy updates it
// asynchronously, so a status can still be in flight.
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

func hasStatus(want mysqltype.DeploymentsStatus) func(deploymentRow) bool {
	return func(row deploymentRow) bool { return row.status == string(want) }
}

// requireNoDeployment fails if any deployments row appears for the app within
// the window.
func (h *pushHarness) requireNoDeployment(t *testing.T, ctx context.Context, appID string) {
	t.Helper()

	require.Never(t, func() bool {
		rows, err := h.listDeployments(ctx, appID)
		return err == nil && len(rows) > 0
	}, 5*time.Second, 100*time.Millisecond, "a dropped push must leave no deployment row for app %s", appID)
}

// deployStub is the real DeployService with only Deploy stubbed out, since
// building is not what these tests observe. Every other handler, Create above
// all, keeps its real behavior through the embedded workflow.
type deployStub struct {
	*deploy.Workflow
}

func (s *deployStub) Deploy(_ restate.ObjectContext, _ *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	return &hydrav1.DeployResponse{}, nil
}

// fakeGitHub answers the GitHub calls a push reaches. Embedding Noop covers the
// rest of the interface with methods that return errors, so an unexpected call
// fails loudly instead of passing silently.
type fakeGitHub struct {
	*githubclient.Noop
	mu                  sync.Mutex
	commitFiles         []string
	commitFilesErr      error
	commitFilesFailures int
	commitFilesCalls    int
	statuses            []commitStatus
}

type commitStatus struct {
	repo, sha, state, description, context string
}

func (f *fakeGitHub) commitStatuses() []commitStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]commitStatus(nil), f.statuses...)
}

func (f *fakeGitHub) setCommitFiles(files []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commitFiles = files
}

// setCommitFilesErr makes the lookup fail terminally. A retryable failure would
// be retried by Restate forever and hang the test instead of failing it.
func (f *fakeGitHub) setCommitFilesErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commitFilesErr = err
}

// failCommitFilesTimes makes the next n lookups fail retryably, the way a
// GitHub 5xx or rate limit arrives.
func (f *fakeGitHub) failCommitFilesTimes(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commitFilesFailures = n
}

func (f *fakeGitHub) commitFilesCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.commitFilesCalls
}

func (f *fakeGitHub) ListCommitFiles(_ int64, _ string, _ string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commitFilesCalls++
	if f.commitFilesFailures > 0 {
		f.commitFilesFailures--
		return nil, errors.New("KEBAP is temporarily unavailable")
	}
	return f.commitFiles, f.commitFilesErr
}

func (f *fakeGitHub) CreateCommitStatus(_ int64, repo, sha, state, _ string, description, context string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.statuses = append(f.statuses, commitStatus{repo: repo, sha: sha, state: state, description: description, context: context})
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
