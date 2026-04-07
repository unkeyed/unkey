package deploy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/compensation"
	"github.com/unkeyed/unkey/pkg/uid"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
	"google.golang.org/protobuf/encoding/protojson"
)

// errInvalidSecretsConfig is returned when the encrypted environment variables
// blob cannot be parsed. This is a permanent error — the data is malformed and
// retrying will not help.
var errInvalidSecretsConfig = errors.New("invalid secrets config")

const (
	// sentinelNamespace isolates sentinel resources from tenant namespaces to
	// simplify RBAC and keep routing infrastructure separate from workloads.
	sentinelNamespace = "sentinel"

	// sentinelPort is the port exposed by sentinel services for frontline traffic
	// and must match the container port and service configuration.
	sentinelPort = 8040

	// regionReadyTimeout is how long to wait for all instances in a region to become
	// ready before considering that region's deployment failed. This is a soft timeout:
	// the workflow continues waiting for other regions and only fails if fewer than
	// [waitForDeployments]'s required minimum become healthy within this window.
	regionReadyTimeout = 15 * time.Minute

	// noInstallationID is the zero value for a GitHub App installation ID.
	// Proto3 omits zero-value int64 fields, so a missing installation ID arrives
	// as 0. When this is the case the repo has no GitHub App connection and we
	// fall back to unauthenticated API access (public repos only).
	noInstallationID = int64(0)
)

// Deploy executes a full deployment workflow for a new application version.
//
// This is a Restate durable workflow, meaning it is idempotent and can safely
// resume from any step after a crash. The workflow orchestrates five phases:
//
//  1. [Workflow.buildImage] — resolve or build the container image
//  2. [Workflow.createTopologies] — provision deployment topologies across regions
//  3. [Workflow.ensureSentinels] and [Workflow.ensureCiliumNetworkPolicy] — set up
//     routing infrastructure and network policies
//  4. [Workflow.configureRouting] — assign domain routes to the deployment
//  5. [Workflow.swapLiveDeployment] — promote to live (production only)
//
// Each phase is wrapped in deployment step tracking so the UI can show progress.
// A compensation stack (executed in reverse on failure) cleans up partial state
// such as inserted topologies and updates the deployment status to failed.
//
// Returns terminal errors for validation failures and retryable errors for
// transient system failures.
func (w *Workflow) Deploy(ctx restate.ObjectContext, req *hydrav1.DeployRequest) (_ *hydrav1.DeployResponse, retErr error) {
	err := assert.All(
		assert.NotEmpty(req.GetDeploymentId(), "deployment_id is required"),
	)
	if err != nil {
		return nil, fault.Wrap(
			restate.TerminalError(err),
			fault.Public("This deployment request is invalid."),
		)
	}

	// compensations are executed in reverse order on failure to clean up any partial state.
	// We use this for steps that have side effects which need to be undone if a later step fails, such as updating deployment status or inserting topologies.
	compensation := compensation.New()

	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, compensation.Execute(ctx))
		}
	}()

	logger.Info("deployment workflow started", "req", fmt.Sprintf("%+v", req))

	compensation.Add("mark deployment as failed", func(runCtx restate.RunContext) error {
		// Use the conditional update so we don't overwrite a status that was
		// set intentionally by the dedup path (superseded) or by a successful
		// completion (ready). Only transitions from active statuses to failed.
		return db.Query.UpdateDeploymentStatusIfActive(runCtx, w.db.RW(), db.UpdateDeploymentStatusIfActiveParams{
			ID:        req.GetDeploymentId(),
			Status:    db.DeploymentsStatusFailed,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	})

	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(runCtx, w.db.RW(), req.GetDeploymentId())
	}, restate.WithName("finding deployment"), restate.WithMaxRetryDuration(time.Minute))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
	}

	// --- Deduplication: skip if a newer deployment is queued for the same app+env+branch ---
	//
	// Because the DeployService VO is keyed by app_id, by the time we run here any
	// subsequent deploys for the same app are already queued in the VO inbox — so a
	// newer-pending check here is race-free.
	if deployment.GitBranch.Valid {
		skipped, skipErr := w.skipIfSuperseded(ctx, deployment)
		if skipErr != nil {
			return nil, skipErr
		}
		if skipped {
			return &hydrav1.DeployResponse{}, nil
		}
	}

	// --- Concurrency gate: acquire a build slot from the workspace's BuildSlotService ---
	//
	// We need to know whether this is a production deployment to decide whether the
	// gate should be bypassed. The Starting step below re-fetches the environment so
	// we don't need to plumb it through — this extra read is cheap compared to a build.
	gateEnvironment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return db.Query.FindEnvironmentById(runCtx, w.db.RO(), deployment.EnvironmentID)
	}, restate.WithName("find environment for concurrency gate"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read environment for build gate."))
	}
	isProduction := gateEnvironment.Slug == "production"

	// Register the slot release as a durable compensation BEFORE calling
	// AcquireOrWait. This ensures the slot is returned on ANY failure path:
	// cancellation, crash, or normal error. Release is idempotent — it
	// handles both active slots and wait_list entries, so calling it when
	// we were never granted a slot is a no-op.
	//
	// Uses AddCtx (not Add) because Release().Send() needs an ObjectContext
	// to dispatch the fire-and-forget call to BuildSlotService.
	compensation.AddCtx(func(ctx restate.ObjectContext) error {
		releaseBuildSlot(ctx, deployment.WorkspaceID, deployment.ID)
		return nil
	})
	if err := w.waitForBuildSlot(ctx, deployment, isProduction); err != nil {
		return nil, err
	}

	// --- Dequeue ---
	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.EndDeploymentStep(runCtx, w.db.RW(), db.EndDeploymentStepParams{
			DeploymentID: req.GetDeploymentId(),
			Step:         db.DeploymentStepsStepQueued,
			EndedAt:      sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			Error:        sql.NullString{Valid: false, String: ""},
		})
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Deployment could not be started."))
	}

	var (
		workspace   db.Workspace
		project     db.Project
		app         db.App
		environment db.Environment
	)

	// --- Starting ---
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepStarting, deployment, func(stepCtx restate.ObjectContext) error {
		workspace, err = restate.Run(ctx, func(runCtx restate.RunContext) (db.Workspace, error) {
			var ws db.Workspace
			err := db.TxRetry(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
				found, err := db.Query.FindWorkspaceByID(txCtx, tx, deployment.WorkspaceID)
				if err != nil {
					if db.IsNotFound(err) {
						return fault.Wrap(
							restate.TerminalError(errors.New("workspace not found")),
							fault.Public("The workspace for this deployment no longer exists."),
						)
					}
					return fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
				}
				ws = found

				if !found.K8sNamespace.Valid {
					ws.K8sNamespace.Valid = true
					ws.K8sNamespace.String = uid.DNS1035()
					return db.Query.SetWorkspaceK8sNamespace(txCtx, tx, db.SetWorkspaceK8sNamespaceParams{
						ID:           ws.ID,
						K8sNamespace: ws.K8sNamespace,
					})
				}
				ws = found

				return nil
			})
			return ws, err
		}, restate.WithName("find workspace"))
		if err != nil {
			return fault.Wrap(err, fault.Public("Workspace settings could not be initialized."))
		}

		project, err = restate.Run(ctx, func(runCtx restate.RunContext) (db.Project, error) {
			return db.Query.FindProjectById(runCtx, w.db.RW(), deployment.ProjectID)
		}, restate.WithName("finding project"))
		if err != nil {
			return fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
		}

		app, err = restate.Run(ctx, func(runCtx restate.RunContext) (db.App, error) {
			return db.Query.FindAppById(runCtx, w.db.RW(), deployment.AppID)
		}, restate.WithName("finding app"))
		if err != nil {
			return fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
		}

		environment, err = restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
			return db.Query.FindEnvironmentById(runCtx, w.db.RW(), deployment.EnvironmentID)
		}, restate.WithName("finding environment"))
		if err != nil {
			return fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// --- Build ---
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepBuilding, deployment, func(stepCtx restate.ObjectContext) error {
		return w.buildImage(stepCtx, req, &deployment)
	})
	if err != nil {
		return nil, err
	}

	// Create the GitHub status reporter V0 after buildImage so that branch-only
	// deploys have a resolved GitCommitSha (buildImage mutates the deployment pointer).
	ghStatus := w.initGitHubStatus(ctx, deployment, project, app, environment, workspace)

	ghStatus.ReportStatus(&hydrav1.GitHubStatusReportRequest{
		State:       hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_IN_PROGRESS,
		Description: "Deploying to regions...",
	})

	// --- Deploy ---
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepDeploying, deployment, func(stepCtx restate.ObjectContext) error {
		topologies, err := w.createTopologies(stepCtx, compensation, workspace, deployment)
		if err != nil {
			return fault.Wrap(err, fault.Public("Regional deployment targets could not be prepared."))
		}

		// Create sentinel DB rows + outbox entries first — these are just
		// inserts, they don't block.
		newSentinelIDs, err := w.ensureSentinelRows(stepCtx, workspace, project, environment, topologies)
		if err != nil {
			return fault.Wrap(err, fault.Public("Sentinels could not be started."))
		}

		if err := w.ensureCiliumNetworkPolicy(stepCtx, workspace, project, environment, topologies, deployment); err != nil {
			return fault.Wrap(err, fault.Public("Applying network policies failed."))
		}

		// Fire off SentinelService.Deploy for each new sentinel WITHOUT
		// waiting, then start the pod-readiness awakeable wait. Krane can
		// work on both in parallel; we drain the sentinel futures after
		// waitForDeployments returns so the two waits overlap.
		sentinelFutures := w.fanOutSentinelDeploys(stepCtx, newSentinelIDs, sentinelReplicasForEnv(environment))

		if err = w.waitForDeployments(stepCtx, compensation, deployment.ID, topologies); err != nil {
			return fault.Wrap(err, fault.Public("Instances did not become healthy in time."))
		}

		if err := w.waitForSentinels(newSentinelIDs, sentinelFutures); err != nil {
			return fault.Wrap(err, fault.Public("Sentinels could not be started."))
		}
		return nil
	})
	if err != nil {
		ghStatus.ReportStatus(&hydrav1.GitHubStatusReportRequest{
			State:       hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_FAILURE,
			Description: "Deployment to regions failed",
		})
		return nil, err
	}

	ghStatus.ReportStatus(&hydrav1.GitHubStatusReportRequest{
		State:       hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_IN_PROGRESS,
		Description: "Configuring routing...",
	})

	// --- Network ---
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepNetwork, deployment, func(stepCtx restate.ObjectContext) error {
		return w.configureRouting(stepCtx, workspace, project, app, environment, deployment)
	})
	if err != nil {
		ghStatus.ReportStatus(&hydrav1.GitHubStatusReportRequest{
			State:       hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_FAILURE,
			Description: "Routing configuration failed",
		})
		return nil, err
	}

	// --- Finalize ---
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepFinalizing, deployment, func(stepCtx restate.ObjectContext) error {
		err = restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
			return db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
				ID:        deployment.ID,
				Status:    db.DeploymentsStatusReady,
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName("updating deployment status to ready"))
		if err != nil {
			return fault.Wrap(err, fault.Public("Deployment completed but final status could not be saved."))
		}

		if err = w.swapLiveDeployment(ctx, deployment, app, environment); err != nil {
			return fault.Wrap(err, fault.Public("Deployment is ready but could not be promoted to live."))
		}
		return nil
	})
	if err != nil {
		ghStatus.ReportStatus(&hydrav1.GitHubStatusReportRequest{
			State:       hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_FAILURE,
			Description: "Finalization failed",
		})
		return nil, err
	}

	ghStatus.ReportStatus(&hydrav1.GitHubStatusReportRequest{
		State:       hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_SUCCESS,
		Description: "Deployment is live",
	})

	logger.Info("deployment workflow completed",
		"deployment_id", deployment.ID,
		"status", "succeeded",
	)

	// Scrape OpenAPI spec asynchronously. The handler reads the configured path
	// from app_runtime_settings; deployment succeeds regardless of scrape outcome.
	hydrav1.NewOpenapiServiceClient(ctx).ScrapeSpec().Send(
		&hydrav1.ScrapeSpecRequest{
			DeploymentId: deployment.ID,
		},
	)

	// Release the build slot on the happy path. The compensation only runs
	// on error, so we release explicitly here. Release is idempotent, so a
	// double-release (if compensation also fires for some reason) is safe.
	releaseBuildSlot(ctx, deployment.WorkspaceID, deployment.ID)

	return &hydrav1.DeployResponse{}, nil
}

// buildImage resolves the container image for a deployment and persists the image
// reference to the database. For a DockerImage source, the image name is used
// directly. For a Git source, the branch HEAD is resolved to a commit SHA (if
// needed), a Docker image is built via Depot using [Workflow.buildDockerImageFromGit],
// and the build ID and git metadata are saved.
//
// The deployment pointer is mutated in place: GitCommitSha and GitBranch are
// updated when a branch is resolved, so the caller sees the resolved values for
// later use in domain generation.
//
// Returns a terminal error for unknown source types and build failures that
// cannot be retried (e.g. bad Dockerfile).
func (w *Workflow) buildImage(ctx restate.ObjectContext, req *hydrav1.DeployRequest, deployment *db.Deployment) error {
	dockerImage := ""

	switch source := req.GetSource().(type) {
	case *hydrav1.DeployRequest_DockerImage:
		dockerImage = source.DockerImage.GetImage()
	case *hydrav1.DeployRequest_Git:
		commitSHA := source.Git.GetCommitSha()

		// Resolve branch→SHA when commit_sha is empty (e.g. CreateDeployment with
		// a GitTarget that specifies only a branch)
		if commitSHA == "" && source.Git.GetBranch() != "" {
			info, resolveErr := restate.Run(ctx, func(runCtx restate.RunContext) (githubclient.CommitInfo, error) {
				if w.allowUnauthenticatedDeployments && source.Git.GetInstallationId() == noInstallationID {
					return w.github.GetBranchHeadCommitPublic(
						source.Git.GetRepository(),
						source.Git.GetBranch(),
					)
				}
				return w.github.GetBranchHeadCommit(
					source.Git.GetInstallationId(),
					source.Git.GetRepository(),
					source.Git.GetBranch(),
				)
			}, restate.WithName("resolve branch head"))
			if resolveErr != nil {
				return fault.Wrap(
					restate.TerminalError(fmt.Errorf("failed to resolve HEAD of branch %q: %w", source.Git.GetBranch(), resolveErr)),
					fault.Public("The configured Git branch could not be resolved. Please check your branch settings."),
				)
			}
			commitSHA = info.SHA

			resolveErr = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
				return db.Query.UpdateDeploymentGitMetadata(runCtx, w.db.RW(), db.UpdateDeploymentGitMetadataParams{
					ID:                       deployment.ID,
					GitCommitSha:             sql.NullString{String: info.SHA, Valid: true},
					GitBranch:                sql.NullString{String: source.Git.GetBranch(), Valid: true},
					GitCommitMessage:         sql.NullString{String: info.Message, Valid: info.Message != ""},
					GitCommitAuthorHandle:    sql.NullString{String: info.AuthorHandle, Valid: info.AuthorHandle != ""},
					GitCommitAuthorAvatarUrl: sql.NullString{String: info.AuthorAvatarURL, Valid: info.AuthorAvatarURL != ""},
					GitCommitTimestamp:       sql.NullInt64{Int64: info.Timestamp.UnixMilli(), Valid: !info.Timestamp.IsZero()},
					UpdatedAt:                sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				})
			}, restate.WithName("update deployment git metadata"))
			if resolveErr != nil {
				return fault.Wrap(
					fmt.Errorf("failed to update deployment git metadata: %w", resolveErr),
					fault.Public("The commit was resolved but metadata could not be saved."),
				)
			}

			deployment.GitCommitSha = sql.NullString{String: info.SHA, Valid: true}
			deployment.GitBranch = sql.NullString{String: source.Git.GetBranch(), Valid: true}
			deployment.GitCommitMessage = sql.NullString{String: info.Message, Valid: info.Message != ""}
			deployment.GitCommitAuthorHandle = sql.NullString{String: info.AuthorHandle, Valid: info.AuthorHandle != ""}
			deployment.GitCommitAuthorAvatarUrl = sql.NullString{String: info.AuthorAvatarURL, Valid: info.AuthorAvatarURL != ""}
			deployment.GitCommitTimestamp = sql.NullInt64{Int64: info.Timestamp.UnixMilli(), Valid: !info.Timestamp.IsZero()}
		}

		// When a SHA is known (either provided directly or just resolved from branch)
		// but the deployment record is still missing git metadata, fetch it from GitHub.
		hasGitHubAuth := !w.allowUnauthenticatedDeployments || source.Git.GetInstallationId() != noInstallationID
		if commitSHA != "" && !deployment.GitCommitMessage.Valid && hasGitHubAuth {
			info, resolveErr := restate.Run(ctx, func(runCtx restate.RunContext) (githubclient.CommitInfo, error) {
				return w.github.GetCommitBySHA(
					source.Git.GetInstallationId(),
					source.Git.GetRepository(),
					commitSHA,
				)
			}, restate.WithName("resolve commit metadata by sha"))
			if resolveErr != nil {
				return fault.Wrap(
					fmt.Errorf("failed to resolve metadata for commit %q: %w", commitSHA, resolveErr),
					fault.Public("Could not retrieve commit information from GitHub."),
				)
			}

			resolveErr = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
				return db.Query.UpdateDeploymentGitMetadata(runCtx, w.db.RW(), db.UpdateDeploymentGitMetadataParams{
					ID:                       deployment.ID,
					GitCommitSha:             sql.NullString{String: info.SHA, Valid: true},
					GitBranch:                deployment.GitBranch,
					GitCommitMessage:         sql.NullString{String: info.Message, Valid: info.Message != ""},
					GitCommitAuthorHandle:    sql.NullString{String: info.AuthorHandle, Valid: info.AuthorHandle != ""},
					GitCommitAuthorAvatarUrl: sql.NullString{String: info.AuthorAvatarURL, Valid: info.AuthorAvatarURL != ""},
					GitCommitTimestamp:       sql.NullInt64{Int64: info.Timestamp.UnixMilli(), Valid: !info.Timestamp.IsZero()},
					UpdatedAt:                sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				})
			}, restate.WithName("update deployment git metadata for sha"))
			if resolveErr != nil {
				return fault.Wrap(
					fmt.Errorf("failed to update deployment git metadata: %w", resolveErr),
					fault.Public("The commit was resolved but metadata could not be saved."),
				)
			}
		}

		build, err := w.buildDockerImageFromGit(ctx, gitBuildParams{
			InstallationID:                source.Git.GetInstallationId(),
			Repository:                    source.Git.GetRepository(),
			CommitSHA:                     commitSHA,
			ContextPath:                   source.Git.GetContextPath(),
			DockerfilePath:                source.Git.GetDockerfilePath(),
			ProjectID:                     deployment.ProjectID,
			AppID:                         deployment.AppID,
			DeploymentID:                  deployment.ID,
			WorkspaceID:                   deployment.WorkspaceID,
			PrNumber:                      source.Git.GetPrNumber(),
			EncryptedEnvironmentVariables: deployment.EncryptedEnvironmentVariables,
			EnvironmentID:                 deployment.EnvironmentID,
		})
		if err != nil {
			// fault.Public set inside buildDockerImageFromGit is lost because
			// restate.Run serialises terminal errors, stripping the fault wrapper.
			// Re-extract the user message on this side of the Restate boundary.
			publicMsg := fault.UserFacingMessage(err)
			if publicMsg == "" {
				publicMsg = extractUserBuildError(err)
			}
			return fault.Wrap(
				fmt.Errorf("failed to build docker image from git: %w", err),
				fault.Public(publicMsg),
			)
		}
		dockerImage = build.ImageName

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Query.UpdateDeploymentBuildID(runCtx, w.db.RW(), db.UpdateDeploymentBuildIDParams{
				ID:        deployment.ID,
				BuildID:   sql.NullString{Valid: true, String: build.DepotBuildID},
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		})
		if err != nil {
			return fault.Wrap(
				fmt.Errorf("failed to update deployment build ID: %w", err),
				fault.Public("Updating build metadata failed."),
			)
		}

	default:
		return fault.Wrap(
			restate.TerminalError(fmt.Errorf("unknown source type: %T", source)),
			fault.Public(fmt.Sprintf("Deployment source %s is not supported.", source)),
		)
	}

	err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.UpdateDeploymentImage(runCtx, w.db.RW(), db.UpdateDeploymentImageParams{
			ID:        deployment.ID,
			Image:     sql.NullString{Valid: true, String: dockerImage},
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("update deployment image"))
	if err != nil {
		return fault.Wrap(err, fault.Public("Unable to save deployment image."))
	}

	return nil
}

// createTopologies determines the target regions and replica counts, bulk-inserts
// the deployment topology records, and writes deployment_changes entries so that
// Watch RPCs pick up the new state.
//
// Region selection uses the environment's runtime settings: if a region config is
// present, only those regions are used with their configured replica counts;
// otherwise the deployment fails with a terminal error.
//
// createTopologies also registers compensations for every inserted
// topology. Compensation deletes by deployment, region, and version so retries
// never remove topologies created by a newer attempt.
func (w *Workflow) createTopologies(
	ctx restate.ObjectContext,
	compensation *compensation.Compensation,
	workspace db.Workspace,
	deployment db.Deployment,
) ([]db.InsertDeploymentTopologyParams, error) {
	// Read regional settings to determine per-region replica counts.
	// If no regional settings exist, fail with a terminal error.
	regionalSettings, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FindAppRegionalSettingsByAppAndEnvRow, error) {
		return db.Query.FindAppRegionalSettingsByAppAndEnv(runCtx, w.db.RO(), db.FindAppRegionalSettingsByAppAndEnvParams{
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
		})
	}, restate.WithName("find regional settings"))
	if err != nil {
		return nil, fault.Wrap(
			fmt.Errorf("failed to find regional settings for environment %s: %w", deployment.EnvironmentID, err),
			fault.Public("Failed to read from database. Please try again."),
		)
	}

	// Filter out regions that are not schedulable and log when we skip one.
	schedulable := make([]db.FindAppRegionalSettingsByAppAndEnvRow, 0, len(regionalSettings))
	for _, rs := range regionalSettings {
		if !rs.RegionCanSchedule {
			logger.Warn("skipping non-schedulable region",
				"region_id", rs.RegionID,
				"region_name", rs.RegionName,
				"app_id", deployment.AppID,
				"environment_id", deployment.EnvironmentID,
			)
			continue
		}
		schedulable = append(schedulable, rs)
	}
	regionalSettings = schedulable

	if len(regionalSettings) == 0 {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("no schedulable regions configured for app %s in environment %s", deployment.AppID, deployment.EnvironmentID), 400),
			fault.Public("No schedulable regions configured. Please configure at least one schedulable region before deploying."),
		)
	}

	// --- Quota check ---
	quota, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Quotas, error) {
		return db.Query.FindQuotaByWorkspaceID(runCtx, w.db.RW(), deployment.WorkspaceID)
	}, restate.WithName("find workspace quota"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
	}

	allocatedResources, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.SumAllocatedResourcesByWorkspaceIDRow, error) {
		return db.Query.SumAllocatedResourcesByWorkspaceID(runCtx, w.db.RW(), workspace.ID)
	}, restate.WithName("sum allocated resources by workspace"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
	}

	for _, rs := range regionalSettings {
		maxReplicas := int32(1)
		if rs.AutoscalingReplicasMax.Valid {
			maxReplicas = rs.AutoscalingReplicasMax.Int32
		}
		allocatedResources.TotalCpuMillicores += int64(deployment.CpuMillicores * maxReplicas)
		allocatedResources.TotalMemoryMib += int64(deployment.MemoryMib * maxReplicas)
		allocatedResources.TotalStorageMib += int64(deployment.StorageMib) * int64(maxReplicas)
	}
	if allocatedResources.TotalCpuMillicores > int64(quota.AllocatedCpuMillicoresTotal) {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("CPU quota exceeded: consumed %d, quota %d", allocatedResources.TotalCpuMillicores, quota.AllocatedCpuMillicoresTotal)),
			fault.Public("We are unable to deploy this application as you have exceeded your CPU quota."),
		)
	}
	if allocatedResources.TotalMemoryMib > int64(quota.AllocatedMemoryMibTotal) {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("Memory quota exceeded: consumed %d, quota %d", allocatedResources.TotalMemoryMib, quota.AllocatedMemoryMibTotal)),
			fault.Public("We are unable to deploy this application as you have exceeded your Memory quota."),
		)
	}
	if allocatedResources.TotalStorageMib > int64(quota.AllocatedStorageMibTotal) {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("Storage quota exceeded: consumed %d, quota %d", allocatedResources.TotalStorageMib, quota.AllocatedStorageMibTotal)),
			fault.Public("We are unable to deploy this application as you have exceeded your Storage quota."),
		)
	}

	topologies := make([]db.InsertDeploymentTopologyParams, 0, len(regionalSettings))

	for _, rs := range regionalSettings {

		// Snapshot autoscaling policy values. When no policy is attached,
		// default to min=1, max=1 (single replica). Once all regional settings
		// have an autoscaling policy this fallback can be removed.
		autoscalingMin := uint32(1)
		autoscalingMax := uint32(1)
		if rs.AutoscalingReplicasMin.Valid {
			autoscalingMin = uint32(rs.AutoscalingReplicasMin.Int32)
		}
		if rs.AutoscalingReplicasMax.Valid {
			autoscalingMax = uint32(rs.AutoscalingReplicasMax.Int32)
		}

		// Clamp to satisfy HPA invariants: min >= 1 and max >= min.
		if autoscalingMin < 1 {
			autoscalingMin = 1
		}
		if autoscalingMax < autoscalingMin {
			autoscalingMax = autoscalingMin
		}

		topologies = append(topologies, db.InsertDeploymentTopologyParams{
			WorkspaceID:                workspace.ID,
			DeploymentID:               deployment.ID,
			RegionID:                   rs.RegionID,
			AutoscalingReplicasMin:     autoscalingMin,
			AutoscalingReplicasMax:     autoscalingMax,
			AutoscalingThresholdCpu:    rs.AutoscalingThresholdCpu,
			AutoscalingThresholdMemory: rs.AutoscalingThresholdMemory,
			DesiredStatus:              db.DeploymentTopologyDesiredStatusRunning,
			CreatedAt:                  time.Now().UnixMilli(),
		})
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			err := db.BulkQuery.InsertDeploymentTopologies(txCtx, tx, topologies)
			if err != nil {
				return err
			}
			now := time.Now().UnixMilli()
			for _, topo := range topologies {
				err := db.Query.InsertDeploymentChange(txCtx, tx, db.InsertDeploymentChangeParams{
					ResourceType: db.DeploymentChangesResourceTypeDeploymentTopology,
					ResourceID:   topo.DeploymentID,
					RegionID:     topo.RegionID,
					CreatedAt:    now,
				})
				if err != nil {
					return err
				}
			}
			return nil
		})
	}, restate.WithName("insert deployment topologies"))
	if err != nil {
		return nil, fault.Wrap(
			fmt.Errorf("failed to insert deployment topologies: %w", err),
			fault.Public("Deployment targets could not be saved."),
		)
	}

	// On failure, mark each topology desired_status=stopped so krane scales
	// the pods to zero immediately via the streaming change feed. Deleting
	// the row instead would also work (the 60s reconciliation safety net
	// catches it) but updating gives an explicit signal with no cleanup
	// lag, and preserves the topology row for debugging.
	for _, topo := range topologies {
		compensation.Add(
			fmt.Sprintf("stop deployment topology %s/%s", topo.DeploymentID, topo.RegionID),
			func(runCtx restate.RunContext) error {
				return db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
					now := time.Now().UnixMilli()
					err := db.Query.UpdateDeploymentTopologyDesiredStatus(txCtx, tx, db.UpdateDeploymentTopologyDesiredStatusParams{
						DeploymentID:  topo.DeploymentID,
						RegionID:      topo.RegionID,
						DesiredStatus: db.DeploymentTopologyDesiredStatusStopped,
						UpdatedAt:     sql.NullInt64{Valid: true, Int64: now},
					})
					if err != nil {
						return err
					}
					return db.Query.InsertDeploymentChange(txCtx, tx, db.InsertDeploymentChangeParams{
						ResourceType: db.DeploymentChangesResourceTypeDeploymentTopology,
						ResourceID:   topo.DeploymentID,
						RegionID:     topo.RegionID,
						CreatedAt:    now,
					})
				})
			},
		)
	}

	return topologies, nil
}

// sentinelReplicasForEnv returns the desired sentinel replica count based
// on the environment slug. Production gets extra replicas for availability.
func sentinelReplicasForEnv(environment db.Environment) int32 {
	if environment.Slug == "production" {
		return 3
	}
	return 1
}

// ensureSentinelRows inserts a sentinel DB row (and outbox entry) for every
// region in topologies that doesn't already have one for this environment.
// Returns the IDs of the newly inserted sentinels.
//
// The insert relies on a unique index on (environment_id, region) to be
// idempotent — if a concurrent workflow already created the sentinel, the
// duplicate key error is silently ignored. This is the DB-mutating half of
// what used to be ensureSentinels; the waiting half is now
// [Workflow.fanOutSentinelDeploys] + [Workflow.waitForSentinels] so
// that the sentinel Deploys can be fired off in parallel with the pod
// readiness wait.
func (w *Workflow) ensureSentinelRows(
	ctx restate.ObjectContext,
	workspace db.Workspace,
	project db.Project,
	environment db.Environment,
	topologies []db.InsertDeploymentTopologyParams,
) ([]string, error) {
	existingSentinels, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FindSentinelsByEnvironmentIDRow, error) {
		return db.Query.FindSentinelsByEnvironmentID(runCtx, w.db.RO(), environment.ID)
	}, restate.WithName("find existing sentinels"))
	if err != nil {
		return nil, fault.Wrap(
			fmt.Errorf("failed to find existing sentinels: %w", err),
			fault.Public("Failed to read from database. Please try again."),
		)
	}

	existingSentinelsByRegion := make(map[string]db.Sentinel)
	for _, row := range existingSentinels {
		existingSentinelsByRegion[row.Region.ID] = row.Sentinel
	}

	desiredReplicas := sentinelReplicasForEnv(environment)

	// Insert a sentinel row (and outbox entry) for every region that doesn't
	// already have one, so krane can pick them up.
	var newSentinelIDs []string
	for _, topology := range topologies {
		_, ok := existingSentinelsByRegion[topology.RegionID]
		if ok {
			continue
		}

		sentinelID, err := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			id := uid.New(uid.SentinelPrefix)
			sentinelK8sName := uid.DNS1035()

			err := db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
				err := db.Query.InsertSentinel(txCtx, tx, db.InsertSentinelParams{
					ID:              id,
					WorkspaceID:     workspace.ID,
					EnvironmentID:   environment.ID,
					ProjectID:       project.ID,
					K8sAddress:      fmt.Sprintf("%s.%s.svc.cluster.local:%d", sentinelK8sName, sentinelNamespace, sentinelPort),
					K8sName:         sentinelK8sName,
					RegionID:        topology.RegionID,
					Image:           w.sentinelImage,
					DesiredReplicas: desiredReplicas,
					CpuMillicores:   250,
					MemoryMib:       256,
					CreatedAt:       time.Now().UnixMilli(),
				})
				if err != nil {
					if db.IsDuplicateKeyError(err) {
						return nil
					}
					return err
				}
				return db.Query.InsertDeploymentChange(txCtx, tx, db.InsertDeploymentChangeParams{
					ResourceType: db.DeploymentChangesResourceTypeSentinel,
					ResourceID:   id,
					RegionID:     topology.RegionID,
					CreatedAt:    time.Now().UnixMilli(),
				})
			})
			return id, err
		}, restate.WithName("ensure sentinel exists in db"))
		if err != nil {
			return nil, fault.Wrap(err, fault.Public("Traffic proxy could not be created for a region."))
		}

		newSentinelIDs = append(newSentinelIDs, sentinelID)
	}

	return newSentinelIDs, nil
}

// fanOutSentinelDeploys kicks off non-blocking RequestFuture calls to
// SentinelService.Deploy for each newly created sentinel. Returns the
// futures so the caller can drain them concurrently with other waits
// (e.g. [Workflow.waitForDeployments]).
//
// Each SentinelService.Deploy invocation is awakeable-based and blocks
// inside the sentinel VO until krane reports the desired image as running
// or until its own 10-minute deploy timeout fires.
func (w *Workflow) fanOutSentinelDeploys(
	ctx restate.ObjectContext,
	newSentinelIDs []string,
	desiredReplicas int32,
) []restate.ResponseFuture[*hydrav1.SentinelServiceDeployResponse] {
	if len(newSentinelIDs) == 0 {
		return nil
	}
	futures := make([]restate.ResponseFuture[*hydrav1.SentinelServiceDeployResponse], 0, len(newSentinelIDs))
	for _, id := range newSentinelIDs {
		fut := hydrav1.NewSentinelServiceClient(ctx, id).
			Deploy().
			RequestFuture(&hydrav1.SentinelServiceDeployRequest{
				Image:           w.sentinelImage,
				DesiredReplicas: desiredReplicas,
				CpuMillicores:   250,
				MemoryMib:       256,
			})
		futures = append(futures, fut)
	}
	return futures
}

// waitForSentinels blocks until every sentinel Deploy future resolves
// or one of them fails. Called after the pod readiness wait so that both
// sentinels and pods converge in parallel on krane's side; by the time we
// drain, most futures should already have resolved.
func (w *Workflow) waitForSentinels(
	newSentinelIDs []string,
	futures []restate.ResponseFuture[*hydrav1.SentinelServiceDeployResponse],
) error {
	for i, fut := range futures {
		resp, err := fut.Response()
		if err != nil {
			return fault.Wrap(err, fault.Public("Traffic proxy failed to start."))
		}
		if resp.GetStatus() != hydrav1.SentinelDeployStatus_SENTINEL_DEPLOY_STATUS_READY {
			return fault.New(fmt.Sprintf("sentinel %s deploy status: %s", newSentinelIDs[i], resp.GetStatus()),
				fault.Public("Traffic proxy failed to start."))
		}
	}
	return nil
}

// configureRouting sets up domain-based routing for a deployment. It generates
// domain names via [buildDomains] (per-commit, per-branch, and per-environment
// URLs), upserts a frontline route record for each domain, and then collects
// any existing sticky routes (environment-level, and live-level for non-rolled-back
// production) so they point to the new deployment.
//
// All collected route IDs are passed to the RoutingService in a single
// [hydrav1.AssignFrontlineRoutesRequest] so that the routing layer atomically
// switches traffic to this deployment's topologies.
func (w *Workflow) configureRouting(
	ctx restate.ObjectContext,
	workspace db.Workspace,
	project db.Project,
	app db.App,
	environment db.Environment,
	deployment db.Deployment,
) error {
	// Extract the fork owner from "owner/repo" for domain naming.
	forkOwner := ""
	if deployment.ForkRepositoryFullName.Valid {
		if parts := strings.SplitN(deployment.ForkRepositoryFullName.String, "/", 2); len(parts) == 2 && parts[0] != "" {
			forkOwner = parts[0]
		}
	}

	allDomains := buildDomains(
		workspace.Slug,
		project.Slug,
		app.Slug,
		environment.Slug,
		deployment.GitCommitSha.String,
		deployment.GitBranch.String,
		forkOwner,
		w.defaultDomain,
		// TODO: source type is hardcoded to CLI_UPLOAD regardless of actual source type
		ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
		deployment.ID,
	)

	existingRouteIDs := make([]string, 0)

	for _, domain := range allDomains {
		frontlineRouteID, getFrontlineRouteErr := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			return db.TxWithResultRetry(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (string, error) {
				found, err := db.Query.FindFrontlineRouteByFQDN(txCtx, tx, domain.domain)
				if err != nil {
					if db.IsNotFound(err) {
						err = db.Query.InsertFrontlineRoute(runCtx, tx, db.InsertFrontlineRouteParams{
							ID:                       uid.New(uid.FrontlineRoutePrefix),
							ProjectID:                sql.NullString{Valid: true, String: project.ID},
							AppID:                    sql.NullString{Valid: true, String: app.ID},
							DeploymentID:             sql.NullString{Valid: true, String: deployment.ID},
							EnvironmentID:            sql.NullString{Valid: true, String: deployment.EnvironmentID},
							FullyQualifiedDomainName: domain.domain,
							Sticky:                   domain.sticky,
							CreatedAt:                time.Now().UnixMilli(),
							UpdatedAt:                sql.NullInt64{Valid: false, Int64: 0},
						})
						return "", err

					}
					return "", err
				}
				return found.ID, nil
			})
		}, restate.WithName(fmt.Sprintf("inserting frontline route %s", domain.domain)))
		if getFrontlineRouteErr != nil {
			return fault.Wrap(getFrontlineRouteErr, fault.Public("Route records could not be created."))
		}
		if frontlineRouteID != "" {
			existingRouteIDs = append(existingRouteIDs, frontlineRouteID)
		}
	}

	// refresh app, cause it might have changed since we read it at the beginning of the workflow (e.g. another deployment promoted to live and updated current_deployment_id)
	app, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.App, error) {
		return db.Query.FindAppById(runCtx, w.db.RO(), app.ID)
	})
	if err != nil {
		return fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
	}

	routeIDs, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]string, error) {
		// using a transaction here to ensure we read a consistent set of sticky routes that won't change under us as we promote this deployment.
		// This is important to prevent a race
		return db.TxWithResult(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) ([]string, error) {
			// Fetch sticky routes for this environment
			stickyTypes := []db.FrontlineRoutesSticky{db.FrontlineRoutesStickyEnvironment}
			if !app.IsRolledBack {
				stickyTypes = append(stickyTypes, db.FrontlineRoutesStickyLive)
			}
			app, err := db.Query.FindAppById(txCtx, tx, app.ID)
			if err != nil {
				return nil, err
			}

			// if the app is rolled back, we should not consider live sticky routes for promotion, even for production, because the live deployment is not healthy and should not receive traffic
			if app.IsRolledBack {
				stickyTypes = []db.FrontlineRoutesSticky{db.FrontlineRoutesStickyEnvironment}
			}

			routes, err := db.Query.FindFrontlineRouteForPromotion(txCtx, tx, db.FindFrontlineRouteForPromotionParams{
				EnvironmentID: sql.NullString{Valid: true, String: deployment.EnvironmentID},
				Sticky:        stickyTypes,
			})
			if err != nil {
				return nil, err
			}

			routeIDs := make([]string, len(routes))
			for i, route := range routes {
				routeIDs[i] = route.ID
			}
			return routeIDs, nil
		})
	}, restate.WithName("finding sticky routes"))
	if err != nil {
		return fault.Wrap(
			fmt.Errorf("failed to find sticky routes: %w", err),
			fault.Public("Failed to read from database. Please try again."),
		)
	}

	// Routing VO is keyed by env_id — per-env serialization for both route
	// reassignment and live-deployment swaps.
	_, err = hydrav1.NewRoutingServiceClient(ctx, environment.ID).
		AssignFrontlineRoutes().Request(&hydrav1.AssignFrontlineRoutesRequest{
		DeploymentId:      deployment.ID,
		FrontlineRouteIds: routeIDs,
	})
	if err != nil {
		return fault.Wrap(
			fmt.Errorf("failed to assign domains: %w", err),
			fault.Public("Domain routing could not be updated."),
		)
	}

	return nil
}

// swapLiveDeployment delegates the live-deployment swap to RoutingService,
// which performs it atomically inside the env-keyed VO. The route reassignment
// happened earlier in [Workflow.assignFrontlineRoutes], so we pass an empty
// route list — this call only touches apps.current_deployment_id.
//
// This only applies to production environments that are not in a rolled-back
// state; for all other cases the method is a no-op and returns nil.
func (w *Workflow) swapLiveDeployment(
	ctx restate.ObjectContext,
	deployment db.Deployment,
	app db.App,
	environment db.Environment,
) error {
	if app.IsRolledBack || environment.Slug != "production" {
		return nil
	}

	swapResp, err := hydrav1.NewRoutingServiceClient(ctx, environment.ID).
		SwapLiveDeployment().Request(&hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:    deployment.ID,
		SetRollbackFlag: false,
	})
	if err != nil {
		return fault.Wrap(err, fault.Public("App live deployment could not be updated."))
	}

	if swapResp.GetPreviousDeploymentId() != "" {
		_, err = hydrav1.NewDeploymentServiceClient(ctx, swapResp.GetPreviousDeploymentId()).
			ScheduleDesiredStateChange().Request(
			&hydrav1.ScheduleDesiredStateChangeRequest{
				DelayMillis: (30 * time.Minute).Milliseconds(),
				State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STANDBY,
			},
			restate.WithIdempotencyKey(swapResp.GetPreviousDeploymentId()),
		)
		if err != nil {
			return fault.Wrap(err, fault.Public("Previous live deployment could not be scheduled for standby."))
		}
	}

	return nil
}

// waitForDeployments blocks until enough regions are healthy, or
// [regionReadyTimeout] elapses. A region is considered healthy when it has
// at least autoscaling_replicas_min running instances. The check tolerates
// one full regional outage: it requires (numRegions - 1) healthy regions,
// minimum 1.
//
// The wait is push-based via a Restate awakeable. The handler:
//  1. Stores {awakeable_id, deployment_id} in VO state under
//     [instancesReadyAwakeableKey]
//  2. Does an initial DB check in case instances are already healthy (e.g.
//     a redeploy against already-running pods, or a report that landed
//     between createTopologies and here) and self-resolves the awakeable
//     if so
//  3. Races the awakeable against a [regionReadyTimeout] timeout via
//     [restate.WaitFirst]
//
// The awakeable is resolved by [Workflow.NotifyInstancesReady], which is
// called from services/cluster's ReportDeploymentStatus RPC whenever krane
// reports an instance status change that pushes the deployment past the
// healthy-region threshold.
//
// State cleanup: the caller passes `compensation` so the state clear can
// be registered as a durable compensation (survives Restate cancellation)
// rather than relying on a Go defer.
func (w *Workflow) waitForDeployments(ctx restate.ObjectContext, compensation *compensation.Compensation, deploymentID string, topologies []db.InsertDeploymentTopologyParams) error {
	// Build per-region minimum replica requirements.
	regionMinReplicas := make(map[string]uint32, len(topologies))
	for _, topo := range topologies {
		regionMinReplicas[topo.RegionID] = topo.AutoscalingReplicasMin
	}
	requiredRegions := max(len(regionMinReplicas)-1, 1)

	logger.Info("waiting for deployments to be ready",
		"deployment_id", deploymentID,
		"total_regions", len(regionMinReplicas),
		"required_regions", requiredRegions,
	)

	// Create awakeable and stash it in VO state BEFORE doing the initial
	// health check. This prevents a race where an instance report lands
	// between our check and the state write, causing NotifyInstancesReady
	// to find no awakeable and return a no-op.
	awk := restate.Awakeable[restate.Void](ctx)
	restate.Set(ctx, instancesReadyAwakeableKey, awk.Id())

	// Clear state on failure so the VO doesn't keep a stale awakeable_id
	// around after the deployment terminates.
	compensation.AddCtx(func(ctx restate.ObjectContext) error {
		restate.Clear(ctx, instancesReadyAwakeableKey)
		return nil
	})

	// Initial check: if instances are already healthy, resolve immediately.
	alreadyHealthy, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		return w.checkInstancesHealthy(runCtx, deploymentID, regionMinReplicas, requiredRegions)
	}, restate.WithName("initial healthy-regions check"))
	if err != nil {
		return fault.Wrap(err, fault.Public("Failed to check deployment health."))
	}
	if alreadyHealthy {
		restate.ResolveAwakeable[restate.Void](ctx, awk.Id(), restate.Void{})
	}

	// Race the awakeable against a timeout.
	timeout := restate.After(ctx, regionReadyTimeout)
	winner, err := restate.WaitFirst(ctx, awk, timeout)
	if err != nil {
		return fmt.Errorf("wait for healthy regions or timeout: %w", err)
	}

	if winner == awk {
		// Drain the result to surface any rejection error.
		if _, err := awk.Result(); err != nil {
			return fmt.Errorf("awakeable result: %w", err)
		}
		// Clear eagerly on the happy path. The compensation registered
		// above still runs on any later error in the Deploy workflow.
		restate.Clear(ctx, instancesReadyAwakeableKey)
		logger.Info("deployments ready", "deployment_id", deploymentID)
		return nil
	}

	return fault.Wrap(
		restate.TerminalErrorf("not enough regions became healthy in %v, required %d of %d", regionReadyTimeout, requiredRegions, len(regionMinReplicas)),
		fault.Public("Not enough regions became healthy in time."),
	)
}

// checkInstancesHealthy returns true if the current running-instance counts
// satisfy the per-region minimum replica requirements for at least
// requiredRegions. It is the same logic used by ReportDeploymentStatus in
// services/cluster to decide whether to call NotifyInstancesReady.
func (w *Workflow) checkInstancesHealthy(
	ctx context.Context,
	deploymentID string,
	regionMinReplicas map[string]uint32,
	requiredRegions int,
) (bool, error) {
	instances, err := db.Query.FindInstancesByDeploymentId(ctx, w.db.RO(), deploymentID)
	if err != nil {
		return false, err
	}

	runningPerRegion := make(map[string]uint32)
	for _, instance := range instances {
		if instance.Status == db.InstancesStatusRunning {
			runningPerRegion[instance.RegionID]++
		}
	}

	healthyRegions := 0
	for regionID, minReplicas := range regionMinReplicas {
		if runningPerRegion[regionID] >= minReplicas {
			healthyRegions++
		}
	}

	logger.Info("checked instances",
		"deployment_id", deploymentID,
		"healthy_regions", healthyRegions,
		"required_regions", requiredRegions,
	)
	return healthyRegions >= requiredRegions, nil
}

// ghStatusReporter wraps a GitHubStatusServiceClient and silently skips all
// Restate calls when no GitHub repo connection exists, avoiding wasteful
// network round-trips for deployments without a connected repository.
type ghStatusReporter struct {
	client    hydrav1.GitHubStatusServiceClient
	connected bool
}

func (r *ghStatusReporter) ReportStatus(req *hydrav1.GitHubStatusReportRequest) {
	if !r.connected {
		return
	}

	r.client.ReportStatus().Send(req)
}

// initGitHubStatus looks up the repo connection and fires a GitHubStatusService.Init
// call. Returns a reporter so callers can send subsequent ReportStatus calls.
// If no GitHub repo is connected, the reporter silently discards all calls.
func (w *Workflow) initGitHubStatus(
	ctx restate.ObjectContext,
	deployment db.Deployment,
	project db.Project,
	app db.App,
	environment db.Environment,
	workspace db.Workspace,
) *ghStatusReporter {
	reporter := &ghStatusReporter{
		client:    hydrav1.NewGitHubStatusServiceClient(ctx, deployment.ID),
		connected: false,
	}

	repoConn, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.GithubRepoConnection, error) {
		found, findErr := db.Query.FindGithubRepoConnectionByAppId(runCtx, w.db.RO(), deployment.AppID)
		if findErr != nil {
			if db.IsNotFound(findErr) {
				// No connection — return zero value, not an error.
				// Returning an error here would cause Restate to retry forever.
				return db.GithubRepoConnection{}, nil //nolint:exhaustruct
			}
			return db.GithubRepoConnection{}, findErr //nolint:exhaustruct
		}
		return found, nil
	}, restate.WithName("find github repo connection"))
	if err != nil {
		logger.Warn("failed to look up github repo connection, skipping deployment status reporting",
			"app_id", deployment.AppID,
			"error", err,
		)

		return reporter
	}

	if repoConn.InstallationID == 0 {
		logger.Info("no github repo connection, skipping deployment status reporting",
			"app_id", deployment.AppID,
		)

		return reporter
	}

	reporter.connected = true

	envLabel := formatEnvironmentLabel(project.Slug, app.Slug, environment.Slug)
	prefix := formatDomainPrefix(project.Slug, app.Slug)
	envURL := fmt.Sprintf("https://%s-%s-%s.%s", prefix, environment.Slug, workspace.Slug, w.defaultDomain)
	logURL := fmt.Sprintf("%s/%s/projects/%s/deployments/%s", w.dashboardURL, workspace.Slug, project.ID, deployment.ID)

	var existingGHDeploymentID int64
	if deployment.GithubDeploymentID.Valid {
		existingGHDeploymentID = deployment.GithubDeploymentID.Int64
	}

	var prNumber int32
	if deployment.PrNumber.Valid {
		prNumber = int32(deployment.PrNumber.Int64)
	}

	reporter.client.Init().Send(&hydrav1.GitHubStatusInitRequest{
		InstallationId:             repoConn.InstallationID,
		Repo:                       repoConn.RepositoryFullName,
		CommitSha:                  deployment.GitCommitSha.String,
		Branch:                     deployment.GitBranch.String,
		EnvironmentLabel:           envLabel,
		EnvironmentUrl:             envURL,
		LogUrl:                     logURL,
		IsProduction:               environment.Slug == "production",
		ProjectSlug:                project.Slug,
		AppSlug:                    app.Slug,
		EnvSlug:                    environment.Slug,
		PrNumber:                   prNumber,
		ExistingGithubDeploymentId: existingGHDeploymentID,
	})

	return reporter
}

// formatEnvironmentLabel builds a human-readable label like "project - env"
// or "project/app - env" for non-default apps.
func formatEnvironmentLabel(projectSlug, appSlug, envSlug string) string {
	if appSlug != "default" {
		return projectSlug + "/" + appSlug + " - " + envSlug
	}
	return projectSlug + " - " + envSlug
}

// formatDomainPrefix builds the domain prefix like "project" or "project-app"
// for non-default apps.
func formatDomainPrefix(projectSlug, appSlug string) string {
	if appSlug != "default" {
		return projectSlug + "-" + appSlug
	}
	return projectSlug
}

// decryptEnvVars decrypts the encrypted environment variables blob via Vault
// and returns the plaintext key-value pairs. Returns nil if there are no env vars.
func (w *Workflow) decryptEnvVars(ctx context.Context, encrypted []byte, environmentID string) (map[string]string, error) {
	if len(encrypted) == 0 {
		return nil, nil
	}

	var secretsConfig ctrlv1.SecretsConfig
	if err := protojson.Unmarshal(encrypted, &secretsConfig); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidSecretsConfig, err)
	}

	bulkRes, err := w.vault.DecryptBulk(ctx, &vaultv1.DecryptBulkRequest{
		Keyring: environmentID,
		Items:   secretsConfig.GetSecrets(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to bulk decrypt env vars: %w", err)
	}

	return bulkRes.GetItems(), nil
}
