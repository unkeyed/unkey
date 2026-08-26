package deployment

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploytarget"
)

// createCallTimeout caps how long CreateDeployment blocks waiting for the
// create worker. Nothing else caps it: the SDK ingress client uses
// http.DefaultClient (no timeout) and Restate holds the connection open
// while the invocation retries, so during a database outage the RPC would
// block until the caller gives up. On expiry the caller gets Unavailable,
// the invocation keeps running, and a retry with the same key attaches to it.
const createCallTimeout = 30 * time.Second

// maxIdempotencyKeyBytes bounds the caller-supplied idempotency key. The key
// travels as an HTTP header on the ingress call, so it must stay within what
// a header value can carry. Matches the public API boundary's limit in
// svc/api v2_deployments_create_deployment.
const maxIdempotencyKeyBytes = 255

// CreateDeployment creates a new deployment record and initiates an async Restate
// workflow. When source is omitted, the handler auto-detects: git-connected
// apps deploy HEAD of their default branch, non-git apps reuse the live
// deployment's Docker image.
//
// The workflow runs asynchronously keyed by deployment id, so deployments
// build in parallel. Workspace-wide build concurrency is enforced separately
// via BuildSlotService. Returns the deployment ID and initial status.
func (s *Service) CreateDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateDeploymentRequest],
) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	if req.Msg.GetProjectId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("project_id is required"))
	}

	appID := req.Msg.GetAppId()
	if appID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("app_id is required"))
	}

	target, err := s.loadDeploymentContext(ctx, req.Msg.GetProjectId(), appID, req.Msg.GetEnvironmentSlug())
	if err != nil {
		return nil, err
	}

	res, err := s.createAndDeploy(ctx, createParams{
		context:        target,
		action:         actionCreate,
		actor:          req.Msg.GetActor(),
		dockerImage:    req.Msg.GetDockerImage(),
		gitCommit:      req.Msg.GetGitCommit(),
		command:        req.Msg.GetCommand(),
		trigger:        req.Msg.GetTrigger(),
		triggeredBy:    req.Msg.GetTriggeredBy(),
		triggerReason:  req.Msg.GetTriggerReason(),
		idempotencyKey: req.Msg.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{
		DeploymentId: res.deploymentID,
		Status:       convertDbStatusToProto(res.status),
		Replayed:     res.replayed,
	}), nil
}

// loadDeploymentContext resolves the deployment target, translating load
// refusals into the connect codes this RPC's callers expect.
func (s *Service) loadDeploymentContext(
	ctx context.Context,
	projectID, appID, envSlug string,
) (deploytarget.Target, error) {
	target, err := deploytarget.Load(ctx, s.db, projectID, appID, envSlug)
	if err != nil {
		var terminal *deploytarget.TerminalError
		if errors.As(err, &terminal) {
			return deploytarget.Target{}, connect.NewError(terminal.Code, terminal)
		}
		return deploytarget.Target{}, connect.NewError(connect.CodeInternal, err)
	}
	return target, nil
}

// ensureEnvironmentDeployable rejects an environment whose runtime or
// regional settings would fail the deploy pipeline (e.g. no schedulable
// region, port out of bounds) before anything is enqueued. Every caller
// passes through here; the worker keeps the same checks as a backstop.
func (s *Service) ensureEnvironmentDeployable(ctx context.Context, target deploytarget.Target) error {
	messages := make([]string, 0)
	for _, v := range deployfail.RuntimeViolations(
		target.AppRuntimeSettings.Port,
		target.AppRuntimeSettings.CpuMillicores,
		target.AppRuntimeSettings.MemoryMib,
	) {
		messages = append(messages, v.Message)
	}

	regional, err := s.db.FindAppRegionalSettingsByAppAndEnv(ctx, db.FindAppRegionalSettingsByAppAndEnvParams{
		AppID:         target.App.ID,
		EnvironmentID: target.Env.Environment.ID,
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load regional settings: %w", err))
	}
	if !slices.ContainsFunc(regional, func(r db.FindAppRegionalSettingsByAppAndEnvRow) bool { return r.RegionCanSchedule }) {
		messages = append(messages, deployfail.MsgNoSchedulableRegions)
	}

	if len(messages) == 0 {
		return nil
	}
	return connect.NewError(connect.CodeFailedPrecondition,
		fmt.Errorf("environment %q is not deployable: %s", target.Env.Environment.Slug, strings.Join(messages, "; ")))
}

// createParams carries everything createAndDeploy needs from a caller.
type createParams struct {
	context deploytarget.Target
	action  createAction

	// actor attributed on the deployment.create audit log. Nil falls back to
	// the system actor. Unused outside the create action.
	actor *ctrlv1.ActorInfo

	// Source overrides. dockerImage wins if set; otherwise we auto-detect
	// from git repo connection (using gitCommit.commit_sha if provided) or
	// fall back to the live deployment's image.
	dockerImage string
	gitCommit   *ctrlv1.GitCommitInfo
	command     []string

	// Attribution persisted on the deployment row.
	trigger       ctrlv1.DeploymentTrigger
	triggeredBy   string
	triggerReason string

	// Caller-supplied deduplication key: a retry carrying the same key
	// attaches to the original create instead of deploying twice. Empty
	// disables dedup.
	idempotencyKey string
}

// createAction is why a deployment row is being created. Only a create writes
// the deployment.create audit log; a rebuild records its own event.
type createAction string

const (
	actionCreate  createAction = "create"
	actionRebuild createAction = "rebuild"
)

// createResult is the answer to a create: which deployment the caller gets,
// and how that answer was produced.
type createResult struct {
	deploymentID string

	// status at answer time: pending for a fresh create, the row's current
	// status for a replay.
	status mysqltype.DeploymentsStatus

	// replayed: answered with the journaled response of an earlier request
	// that carried the same idempotency key, instead of creating anything.
	replayed bool
}

// createAndDeploy is the shared path of CreateDeployment and Rebuild. It runs
// the gates and source resolution here, where a failure consumes nothing,
// then hands the durable part (insert, audit, send, sibling dedup) to the
// DeploymentCreateService worker, keyed by the caller's idempotency key so a
// retry attaches to the original invocation instead of creating again.
func (s *Service) createAndDeploy(ctx context.Context, p createParams) (createResult, error) {
	c := p.context

	p.idempotencyKey = strings.TrimSpace(p.idempotencyKey)

	if len(p.idempotencyKey) > maxIdempotencyKeyBytes {
		return createResult{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("idempotency_key must be at most %d bytes", maxIdempotencyKeyBytes))
	}

	// Pre-key gates: a rejection here never consumes the idempotency key, so
	// the dashboard can resubmit a corrected form with the same key.
	if err := s.ensureWorkspaceCanDeploy(ctx, c.WorkspaceID, string(p.action)); err != nil {
		return createResult{}, err
	}
	if err := s.ensureEnvironmentDeployable(ctx, c); err != nil {
		return createResult{}, err
	}

	// Per-request command override (e.g. `unkey deploy --command`) wins over
	// the app's stored default, so the row records what actually runs.
	command := c.AppRuntimeSettings.Command
	if len(p.command) > 0 {
		command = p.command
	}

	deploymentID := uid.New(uid.DeploymentPrefix)

	commit := commitFromRequest(p.gitCommit)
	deployReq, commit, err := s.resolveSource(ctx, c, deploymentID, command, commit, p.dockerImage, p.gitCommit != nil)
	if err != nil {
		return createResult{}, err
	}

	// Replay detection: the handler echoes the executing request's nonce, so
	// a foreign nonce means a journaled answer. Restate exposes no replay
	// indicator of its own.
	nonce := uid.New("nonce")

	action := hydrav1.DeploymentCreateAction_DEPLOYMENT_CREATE_ACTION_CREATE
	if p.action == actionRebuild {
		action = hydrav1.DeploymentCreateAction_DEPLOYMENT_CREATE_ACTION_REBUILD
	}

	createReq := &hydrav1.DeploymentCreateRequest{
		Nonce:           nonce,
		ProjectId:       c.Project.ID,
		AppId:           c.App.ID,
		EnvironmentSlug: c.Env.Environment.Slug,
		DeployRequest:   deployReq,
		GitCommit: &ctrlv1.GitCommitInfo{
			CommitSha:       commit.SHA,
			Branch:          commit.Branch,
			CommitMessage:   commit.Message,
			AuthorHandle:    commit.AuthorHandle,
			AuthorAvatarUrl: commit.AuthorAvatarURL,
			Timestamp:       commit.Timestamp,
			ForkRepository:  commit.ForkRepository,
		},
		Trigger:       p.trigger,
		TriggeredBy:   p.triggeredBy,
		TriggerReason: p.triggerReason,
		Actor:         p.actor,
		Action:        action,
	}

	logger.Info(
		"starting deployment create",
		"deployment_id", deploymentID,
		"workspace_id", c.WorkspaceID,
		"project_id", c.Project.ID,
		"app_id", c.App.ID,
		"environment", c.Env.Environment.ID,
		"keyed", p.idempotencyKey != "",
	)

	callCtx, cancel := context.WithTimeout(ctx, createCallTimeout)
	defer cancel()

	// Restate scopes idempotency keys per handler, so the prefix scopes them
	// to the target: without the workspace two customers both sending
	// "deploy-1" would share one invocation, and without app and environment
	// one key could not deploy the same commit to prod and staging. An empty
	// key sends no header at all.
	scopedIdempotencyKey := ""
	if p.idempotencyKey != "" {
		scopedIdempotencyKey = strings.Join(
			[]string{c.WorkspaceID, c.App.ID, c.Env.Environment.ID, p.idempotencyKey}, "/",
		)
	}
	resp, callErr := hydrav1.NewDeploymentCreateServiceIngressClient(s.restate).
		Create().
		Request(callCtx, createReq, restate.WithIdempotencyKey(scopedIdempotencyKey))
	if callErr != nil {
		if errors.Is(callErr, context.DeadlineExceeded) {
			if p.idempotencyKey != "" {
				return createResult{}, connect.NewError(connect.CodeUnavailable,
					fmt.Errorf("deployment create is still in progress; retry with the same idempotency key"))
			}
			// Without a key a retry cannot attach; it would create a second
			// deployment while this one completes in the background.
			return createResult{}, connect.NewError(connect.CodeUnavailable,
				fmt.Errorf("deployment create is still in progress and will complete in the background; check deployment %s before retrying", deploymentID))
		}
		return createResult{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("deployment create failed: %w", callErr))
	}

	replayed := resp.GetNonce() != nonce
	status := mysqltype.DeploymentsStatusPending
	if replayed {
		// The journaled response predates this request; answer with the row's
		// current status. A missing row is unreachable (deleting a deployment
		// deletes its target too, which fails the pre-call load), so it lands
		// in the generic error like any broken read.
		row, findErr := s.db.FindDeploymentById(ctx, resp.GetDeploymentId())
		if findErr != nil {
			return createResult{}, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to look up replayed deployment: %w", findErr))
		}
		status = row.Status

		logger.Info(
			"replayed idempotent deployment create",
			"deployment_id", resp.GetDeploymentId(),
			"workspace_id", c.WorkspaceID,
			"status", string(status),
		)
	}

	return createResult{deploymentID: resp.GetDeploymentId(), status: status, replayed: replayed}, nil
}
