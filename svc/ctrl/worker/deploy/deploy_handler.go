package deploy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
)

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
	compensation := NewCompensation()

	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, compensation.Execute(ctx))
		}
	}()

	logger.Info("deployment workflow started", "req", fmt.Sprintf("%+v", req))

	compensation.Add("mark deployment as failed", func(runCtx restate.RunContext) error {
		return db.Query.UpdateDeploymentStatus(runCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
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

	// --- Deploy ---
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepDeploying, deployment, func(stepCtx restate.ObjectContext) error {

		topologies, err := w.createTopologies(stepCtx, compensation, workspace, deployment)
		if err != nil {
			return fault.Wrap(err, fault.Public("Regional deployment targets could not be prepared."))
		}

		if err = w.ensureSentinels(stepCtx, workspace, project, environment, topologies); err != nil {
			return fault.Wrap(err, fault.Public("Sentinels could not be started."))
		}

		if err := w.ensureCiliumNetworkPolicy(stepCtx, workspace, project, environment, topologies, deployment); err != nil {
			return fault.Wrap(err, fault.Public("Applying network policies failed."))
		}

		if err = w.waitForDeployments(stepCtx, deployment.ID, topologies); err != nil {
			return fault.Wrap(err, fault.Public("Instances did not become healthy in time."))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// --- Network ---
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepNetwork, deployment, func(stepCtx restate.ObjectContext) error {

		return w.configureRouting(stepCtx, workspace, project, app, environment, deployment)
	})
	if err != nil {
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
		return nil, err
	}
	logger.Info("deployment workflow completed",
		"deployment_id", deployment.ID,
		"status", "succeeded",
	)

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
				if w.allowUnauthenticatedDeployments {
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
					fault.Public("Selected Git branch could not be resolved."),
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
		}

		build, err := w.buildDockerImageFromGit(ctx, gitBuildParams{
			InstallationID: source.Git.GetInstallationId(),
			Repository:     source.Git.GetRepository(),
			CommitSHA:      commitSHA,
			ContextPath:    source.Git.GetContextPath(),
			DockerfilePath: source.Git.GetDockerfilePath(),
			ProjectID:      deployment.ProjectID,
			AppID:          deployment.AppID,
			DeploymentID:   deployment.ID,
			WorkspaceID:    deployment.WorkspaceID,
		})
		if err != nil {
			return fault.Wrap(
				fmt.Errorf("failed to build docker image from git: %w", err),
				fault.Public("Build failed. Please check the build logs for details."),
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

// createTopologies determines the target regions and replica counts, obtains a
// monotonic version for each region from VersioningService, and bulk-inserts the
// deployment topology records.
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
	compensation *Compensation,
	workspace db.Workspace,
	deployment db.Deployment,
) ([]db.InsertDeploymentTopologyParams, error) {
	// Read region config from runtime settings to determine per-region replica counts.
	// If regionConfig is empty, deploy to all available regions with 1 replica each (default).
	// If regionConfig has entries, only deploy to those regions with the specified counts.
	regionConfig := map[string]int{}
	runtimeSettings, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindAppRuntimeSettingsByAppAndEnvRow, error) {
		return db.Query.FindAppRuntimeSettingsByAppAndEnv(runCtx, w.db.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
		})
	}, restate.WithName("find runtime settings for region config"))
	if err != nil {
		return nil, fault.Wrap(
			fmt.Errorf("failed to find runtime settings for environment %s: %w", deployment.EnvironmentID, err),
			fault.Public("Failed to read from database. Please try again."),
		)
	}
	if len(runtimeSettings.AppRuntimeSetting.RegionConfig) > 0 {
		for region, count := range runtimeSettings.AppRuntimeSetting.RegionConfig {
			regionConfig[region] = count
		}
	}

	if len(regionConfig) == 0 {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("no regions configured for app %s in environment %s", deployment.AppID, deployment.EnvironmentID), 400),
			fault.Public("No regions configured. Please configure at least one region before deploying."),
		)
	}

	regions := make([]string, 0, len(regionConfig))
	for r := range regionConfig {
		regions = append(regions, r)
	}

	topologies := make([]db.InsertDeploymentTopologyParams, 0, len(regions))

	for _, region := range regions {
		versionResp, err := hydrav1.NewVersioningServiceClient(ctx, region).NextVersion().Request(&hydrav1.NextVersionRequest{})
		if err != nil {
			return nil, fault.Wrap(
				fmt.Errorf("failed to get next version: %w", err),
				fault.Public("Failed to generate new version."),
			)
		}

		replicas := int32(1)
		if count, ok := regionConfig[region]; ok {
			replicas = int32(count)
		}

		topologies = append(topologies, db.InsertDeploymentTopologyParams{
			WorkspaceID:     workspace.ID,
			DeploymentID:    deployment.ID,
			Region:          region,
			DesiredReplicas: replicas,
			DesiredStatus:   db.DeploymentTopologyDesiredStatusStarting,
			Version:         versionResp.GetVersion(),
			CreatedAt:       time.Now().UnixMilli(),
		})
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			return db.BulkQuery.InsertDeploymentTopologies(txCtx, tx, topologies)
		})
	}, restate.WithName("insert deployment topologies"))
	if err != nil {
		return nil, fault.Wrap(
			fmt.Errorf("failed to insert deployment topologies: %w", err),
			fault.Public("Deployment targets could not be saved."),
		)
	}

	// In case anything goes wrong, delete the inserted topologies.
	// Deleting by version keeps retries safe: a newer retry creates a higher
	// version and will not be removed by this compensation.
	for _, topo := range topologies {
		compensation.Add(
			fmt.Sprintf("delete deployment topology %s/%s/%d", topo.DeploymentID, topo.Region, topo.Version),
			func(runCtx restate.RunContext) error {
				return db.Query.DeleteDeploymentTopologyByDeploymentRegionVersion(runCtx, w.db.RW(), db.DeleteDeploymentTopologyByDeploymentRegionVersionParams{
					DeploymentID: topo.DeploymentID,
					Region:       topo.Region,
					Version:      topo.Version,
				})
			},
		)
	}

	return topologies, nil
}

// ensureSentinels creates sentinel instances in any region that doesn't already
// have one for this environment. Sentinels are the reverse-proxy layer that
// routes frontline traffic to deployment containers, so every region serving
// traffic needs one.
//
// Production environments get 3 sentinel replicas for availability; all others
// get 1. The insert relies on a unique index on (environment_id, region) to be
// idempotent — if a concurrent workflow already created the sentinel, the
// duplicate key error is silently ignored.
func (w *Workflow) ensureSentinels(
	ctx restate.ObjectContext,
	workspace db.Workspace,
	project db.Project,
	environment db.Environment,
	topologies []db.InsertDeploymentTopologyParams,
) error {
	existingSentinels, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Sentinel, error) {
		return db.Query.FindSentinelsByEnvironmentID(runCtx, w.db.RO(), environment.ID)
	}, restate.WithName("find existing sentinels"))
	if err != nil {
		return fault.Wrap(
			fmt.Errorf("failed to find existing sentinels: %w", err),
			fault.Public("Failed to read from database. Please try again."),
		)
	}

	existingSentinelsByRegion := make(map[string]db.Sentinel)
	for _, sentinel := range existingSentinels {
		existingSentinelsByRegion[sentinel.Region] = sentinel
	}

	for _, topology := range topologies {
		_, ok := existingSentinelsByRegion[topology.Region]
		if !ok {

			desiredReplicas := int32(1)
			if environment.Slug == "production" {
				desiredReplicas = 3
			}

			sentinelVersion, err := hydrav1.NewVersioningServiceClient(ctx, topology.Region).NextVersion().Request(&hydrav1.NextVersionRequest{})
			if err != nil {
				return fault.Wrap(
					fmt.Errorf("failed to get next version for sentinel: %w", err),
					fault.Public("Traffic proxies could not be versioned in one region."),
				)
			}

			err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
				sentinelID := uid.New(uid.SentinelPrefix)
				sentinelK8sName := uid.DNS1035()

				return db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
					// we rely on the unique index of environmentID + region here to create or noop
					err := db.Query.InsertSentinel(txCtx, tx, db.InsertSentinelParams{
						ID:                sentinelID,
						WorkspaceID:       workspace.ID,
						EnvironmentID:     environment.ID,
						ProjectID:         project.ID,
						K8sAddress:        fmt.Sprintf("%s.%s.svc.cluster.local:%d", sentinelK8sName, sentinelNamespace, sentinelPort),
						K8sName:           sentinelK8sName,
						Region:            topology.Region,
						Image:             w.sentinelImage,
						Health:            db.SentinelsHealthUnknown,
						DesiredReplicas:   desiredReplicas,
						AvailableReplicas: 0,
						CpuMillicores:     256,
						MemoryMib:         256,
						Version:           sentinelVersion.GetVersion(),
						CreatedAt:         time.Now().UnixMilli(),
					})
					if err != nil {
						if db.IsDuplicateKeyError(err) {
							return nil
						}
						return err
					}
					return nil
				})
			}, restate.WithName("ensure sentinel exists in db"))
			if err != nil {
				return fault.Wrap(err, fault.Public("Traffic proxy could not be created for a region."))
			}

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
	allDomains := buildDomains(
		workspace.Slug,
		project.Slug,
		app.Slug,
		environment.Slug,
		deployment.GitCommitSha.String,
		deployment.GitBranch.String,
		w.defaultDomain,
		// TODO: source type is hardcoded to CLI_UPLOAD regardless of actual source type
		ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
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
							ProjectID:                project.ID,
							AppID:                    app.ID,
							DeploymentID:             deployment.ID,
							EnvironmentID:            deployment.EnvironmentID,
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
				EnvironmentID: deployment.EnvironmentID,
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

	// Key routing service by app ID for per-app serialization
	_, err = hydrav1.NewRoutingServiceClient(ctx, app.ID).
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

// swapLiveDeployment atomically updates the apps's live deployment pointer to
// this deployment and schedules the previous live deployment for standby after
// 30 minutes via [hydrav1.DeploymentServiceClient.ScheduleDesiredStateChange].
//
// The read-then-update happens inside a single transaction to prevent a race where
// two concurrent deploys both capture the same previous deployment ID and one of
// them never gets scheduled for standby.
//
// This only applies to production environments that are not in a rolled-back state;
// for all other cases the method is a no-op and returns nil.
func (w *Workflow) swapLiveDeployment(
	ctx restate.ObjectContext,
	deployment db.Deployment,
	app db.App,
	environment db.Environment,
) error {
	if app.IsRolledBack || environment.Slug != "production" {
		return nil
	}

	// Atomically read the current deployment and swap it to the new one.
	// This prevents a race where two concurrent deploys both capture the same
	// currentDeploymentID and one of them never gets scheduled for standby.
	previousDeploymentID, err := restate.Run(ctx, func(runCtx restate.RunContext) (sql.NullString, error) {
		return db.TxWithResult(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (sql.NullString, error) {
			currentApp, findErr := db.Query.FindAppById(txCtx, tx, deployment.AppID)
			if findErr != nil {
				return sql.NullString{}, findErr
			}

			updateErr := db.Query.UpdateAppDeployments(txCtx, tx, db.UpdateAppDeploymentsParams{
				IsRolledBack:        false,
				AppID:               deployment.AppID,
				CurrentDeploymentID: sql.NullString{Valid: true, String: deployment.ID},
				UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if updateErr != nil {
				return sql.NullString{}, updateErr
			}

			return currentApp.CurrentDeploymentID, nil
		})
	}, restate.WithName("swapping app live deployment"))
	if err != nil {
		return fault.Wrap(err, fault.Public("App live deployment could not be updated."))
	}

	if previousDeploymentID.Valid {
		_, err = hydrav1.NewDeploymentServiceClient(ctx, previousDeploymentID.String).
			ScheduleDesiredStateChange().Request(
			&hydrav1.ScheduleDesiredStateChangeRequest{
				DelayMillis: (30 * time.Minute).Milliseconds(),
				State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STANDBY,
			},
			restate.WithIdempotencyKey(previousDeploymentID.String),
		)
		if err != nil {
			return fault.Wrap(err, fault.Public("Previous live deployment could not be scheduled for standby."))
		}
	}

	return nil
}

// waitForDeployments polls instance status across all regions until enough
// regions have all their desired replicas running, or [regionReadyTimeout]
// elapses. Each region is polled concurrently via restate.RunAsync.
//
// The method requires min(2, len(topologies)) healthy regions to succeed.
// This threshold ensures at least two regions are serving traffic before the
// workflow proceeds to routing, while still allowing single-region deployments
// to pass. Regions that time out or error are skipped rather than failing the
// entire deployment, so a degraded region does not block progress.
func (w *Workflow) waitForDeployments(ctx restate.ObjectContext, deploymentID string, topologies []db.InsertDeploymentTopologyParams) error {
	logger.Info("waiting for deployments to be ready", "deployment_id", deploymentID)

	deadline, err := restate.Run(ctx, func(_ restate.RunContext) (time.Time, error) {
		return time.Now().Add(regionReadyTimeout), nil
	}, restate.WithName("calculate deadline"))
	if err != nil {
		return fault.Wrap(err, fault.Public("Deployment readiness checks could not start."))
	}

	readygates := make([]restate.Future, len(topologies))
	for i, region := range topologies {
		promise := restate.RunAsync(ctx, func(runCtx restate.RunContext) (bool, error) {
			for time.Now().Before(deadline) {
				time.Sleep(time.Second)

				instances, err := db.Query.FindInstancesByDeploymentIdAndRegion(runCtx, w.db.RO(), db.FindInstancesByDeploymentIdAndRegionParams{
					Deploymentid: deploymentID,
					Region:       region.Region,
				})
				if err != nil {
					return false, err
				}
				logger.Info("checking instances for region", "deployment_id", deploymentID, "region", region.Region, "instances_found", len(instances))
				if len(instances) < int(region.DesiredReplicas) {
					logger.Info("not all instances are up yet", "deployment_id", deploymentID, "region", region.Region, "instances_found", len(instances), "desired_replicas", region.DesiredReplicas)
					continue
				}
				allRunning := true
				for _, instance := range instances {
					if instance.Status != db.InstancesStatusRunning {
						logger.Info("instance not running yet", "deployment_id", deploymentID, "region", instance.Region, "instance_id", instance.ID, "status", instance.Status)
						allRunning = false
						break
					}
				}
				if allRunning {
					return true, nil
				}
			}
			return false, nil
		}, restate.WithName(fmt.Sprintf("wait for %d instances in %s", region.DesiredReplicas, region.Region)))
		readygates[i] = promise

	}
	requiredHealthyRegions := min(2, len(topologies))
	healthyRegions := 0

	for fut, err := range restate.Wait(ctx, readygates...) {
		if err != nil {
			continue
		}
		raf, ok := fut.(restate.RunAsyncFuture[bool])
		if !ok {
			return fault.Wrap(
				fmt.Errorf("unexpected future type: %T", fut),
				fault.Public("Deployment readiness checks returned an unexpected response."),
			)
		}
		ready, err := raf.Result()
		if err != nil {
			continue
		}
		if ready {
			healthyRegions++
		}

		if healthyRegions >= requiredHealthyRegions {
			break
		}
	}
	if healthyRegions < requiredHealthyRegions {
		return fault.Wrap(
			fmt.Errorf("only %d healthy regions, required at least %d", healthyRegions, requiredHealthyRegions),
			fault.Public("Not enough regions became healthy."),
		)
	}

	logger.Info("deployments ready", "deployment_id", deploymentID)
	return nil
}
