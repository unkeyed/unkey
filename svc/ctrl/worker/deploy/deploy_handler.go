package deploy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/restate/compensation"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/encoding/protojson"
)

// errInvalidSecretsConfig is returned when the encrypted environment variables
// blob cannot be parsed. This is a permanent error — the data is malformed and
// retrying will not help.
var errInvalidSecretsConfig = errors.New("invalid secrets config")

const (
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

	// runMaxAttempts bounds per-Run retries. When a Run exceeds this count it
	// returns a TerminalError into Go, which unwinds the handler and fires the
	// compensation stack. Unbounded Runs chew through the service-level
	// invocation retry policy, which Restate tears down without re-entering the
	// handler — the deferred compensation.Execute never runs in that path.
	//
	// 5 attempts with Restate's default exponential backoff covers transient
	// blips (DB reconnect, GitHub 5xx, network hiccup) while failing fast on
	// persistent misconfiguration.
	runMaxAttempts uint = 5

	// buildImageRetryCeiling is an additional wall-clock cap on the outer
	// build Run. A single build attempt can legitimately take many minutes,
	// so attempts-based bounding alone is too coarse here — we also want a
	// hard total-time ceiling. Whichever bound fires first wins.
	buildImageRetryCeiling = 30 * time.Minute
)

// Deploy executes a full deployment workflow for a new application version.
//
// This is a Restate durable workflow, meaning it is idempotent and can safely
// resume from any step after a crash. The workflow orchestrates five phases:
//
//  1. [Workflow.Build]: resolve or build the container image, called in the
//     "builds" scope so Restate caps concurrent builds per workspace
//  2. [Workflow.createTopologies]: provision deployment topologies across regions
//  3. [Workflow.waitForDeployments]: block until enough regions are healthy
//  4. [Workflow.configureRouting]: assign domain routes to the deployment
//  5. [Workflow.swapLiveDeployment]: promote to live (production only)
//
// Network policies are not provisioned by the control plane any more —
// krane installs a per-deployment CiliumNetworkPolicy when it applies the
// ReplicaSet, so frontline ingress is allowed without a separate workflow
// step.
//
// Each phase is wrapped in deployment step tracking so the UI can show progress.
// A compensation stack (executed in reverse on failure) cleans up partial state
// such as inserted topologies and updates the deployment status to failed.
//
// Returns terminal errors for validation failures and retryable errors for
// transient system failures.
func (w *Workflow) Deploy(ctx restate.WorkflowContext, req *hydrav1.DeployRequest) (_ *hydrav1.DeployResponse, retErr error) {
	err := assert.All(
		assert.NotEmpty(req.GetDeploymentId(), "deployment_id is required"),
	)
	if err != nil {
		return nil, fault.Wrap(
			restate.ToTerminalError(err),
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
		// A plain update would overwrite cancelled, superseded, or ready
		return w.db.UpdateDeploymentStatusIfActive(runCtx, db.UpdateDeploymentStatusIfActiveParams{
			ID:                  req.GetDeploymentId(),
			Status:              mysqltype.DeploymentsStatusFailed,
			UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ProgressingStatuses: mysqltype.ProgressingDeploymentStatuses,
		})
	})

	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindDeploymentForDeployRow, error) {
		found, err := w.db.FindDeploymentForDeploy(runCtx, req.GetDeploymentId())
		if db.IsNotFound(err) {
			return found, restate.ToTerminalError(err)
		}
		return found, err
	}, restate.WithName("finding deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
	}

	// A cancel that lands before invocation_id is on the row cannot reach
	// Restate, so it only writes the status. This check is what stops the
	// invocation in that case. Returning nil keeps Restate from counting it
	// as failed
	if deployment.Status.IsTerminal() {
		logger.Info("deployment is already terminal, not deploying",
			"deployment_id", deployment.ID,
			"status", deployment.Status,
		)
		return &hydrav1.DeployResponse{}, nil
	}

	if deployment.GitBranch.Valid {
		skipped, skipErr := w.skipIfSuperseded(ctx, deployment)
		if skipErr != nil {
			return nil, skipErr
		}
		if skipped {
			return &hydrav1.DeployResponse{}, nil
		}
	}

	// Request, not Send: a Send would detach the Build and leave it running
	// with nothing waiting on it. Cancelling this invocation removes a Build
	// that is still queued. A Build that is already running holds the
	// workspace's build slot until its image build returns, because that
	// build is one restate.Run with no call for the cancel to land on
	_, err = hydrav1.NewDeployWorkflowClient(ctx, deployment.ID, restate.WithScope(restateadmin.BuildConcurrencyScope)).
		Build().
		Request(req, restate.WithLimitKey(deployment.WorkspaceID))
	if err != nil {
		// A killed Build leaves its step open. This ends whatever is still
		// open; a step that already ended keeps its own reason
		endErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return w.db.EndActiveDeploymentStepsForDeployments(runCtx, db.EndActiveDeploymentStepsForDeploymentsParams{
				EndedAt:       sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				Error:         sql.NullString{Valid: true, String: "The build did not complete."},
				DeploymentIds: []string{deployment.ID},
			})
		}, restate.WithName("end open build steps"), restate.WithMaxRetryAttempts(runMaxAttempts))
		return nil, fault.Wrap(errors.Join(err, endErr), fault.Public("The build did not complete."))
	}

	// Build returns without an error on a deployment that was cancelled or
	// superseded while it waited, and a cancel can land during the build
	deployment, err = restate.Run(ctx, func(runCtx restate.RunContext) (db.FindDeploymentForDeployRow, error) {
		found, err := w.db.FindDeploymentForDeploy(runCtx, deployment.ID)
		if db.IsNotFound(err) {
			return found, restate.ToTerminalError(err)
		}
		return found, err
	}, restate.WithName("loading deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
	}
	if deployment.Status.IsTerminal() {
		logger.Info("deployment became terminal during build, not deploying",
			"deployment_id", deployment.ID,
			"status", deployment.Status,
		)
		return &hydrav1.DeployResponse{}, nil
	}

	ghStatus := w.initGitHubStatus(ctx, deployment)

	ghStatus.ReportStatus(&hydrav1.GitHubStatusReportRequest{
		State:       hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_IN_PROGRESS,
		Description: "Deploying to regions...",
	})

	// --- Deploy ---
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepDeploying, deployment.ID, func() error {
		topologies, err := w.createTopologies(ctx, compensation, deployment)
		if err != nil {
			return fault.Wrap(err, fault.Public("Regional deployment targets could not be prepared."))
		}

		if err = w.waitForDeployments(ctx, deployment.ID, topologies); err != nil {
			return fault.Wrap(err, fault.Public("Instances did not become healthy in time."))
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
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepNetwork, deployment.ID, func() error {
		return w.configureRouting(ctx, deployment)
	})
	if err != nil {
		ghStatus.ReportStatus(&hydrav1.GitHubStatusReportRequest{
			State:       hydrav1.GitHubDeploymentState_GITHUB_DEPLOYMENT_STATE_FAILURE,
			Description: "Routing configuration failed",
		})
		return nil, err
	}

	// --- Finalize ---
	err = w.DeploymentStep(ctx, db.DeploymentStepsStepFinalizing, deployment.ID, func() error {
		// A cancel can land in the database while this step runs and this
		// handler only learns of it at its next Restate call. A plain update
		// would then overwrite cancelled with ready while the compensations
		// stop the pods
		err = restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
			return w.db.UpdateDeploymentStatusIfActive(stepCtx, db.UpdateDeploymentStatusIfActiveParams{
				ID:                  deployment.ID,
				Status:              mysqltype.DeploymentsStatusReady,
				UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				ProgressingStatuses: mysqltype.ProgressingDeploymentStatuses,
			})
		}, restate.WithName("updating deployment status to ready"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return fault.Wrap(err, fault.Public("Deployment completed but final status could not be saved."))
		}

		if deployment.EnvironmentKind.IsProduction() {
			if err = w.swapLiveDeployment(ctx, deployment); err != nil {
				return fault.Wrap(err, fault.Public("Deployment is ready but could not be promoted to live."))
			}
		} else if deployment.EnvironmentKind.IsPreview() {
			if err = w.spinDownPreviousDeployments(ctx, deployment); err != nil {
				// This isn't a real issue, our cron job will eventually spin the preview deployments down anyways
				logger.Error("unable to spin down previous preview deployments", "error", err)
			}
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

	logger.Info(
		"deployment workflow completed",
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

	return &hydrav1.DeployResponse{}, nil
}

// createTopologies saves the deployment's regions and replica counts from its
// runtime settings. Missing region settings fail the deployment.
// If a later step fails, cleanup stops these topologies but keeps the rows.
func (w *Workflow) createTopologies(
	ctx restate.ObjectContext,
	compensation *compensation.Compensation,
	deployment db.FindDeploymentForDeployRow,
) ([]db.InsertDeploymentTopologyParams, error) {
	// Read regional settings to determine per-region replica counts.
	// If no regional settings exist, fail with a terminal error.
	regionalSettings, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FindAppRegionalSettingsByAppAndEnvRow, error) {
		return w.db.FindAppRegionalSettingsByAppAndEnv(runCtx, db.FindAppRegionalSettingsByAppAndEnvParams{
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
		})
	}, restate.WithName("find regional settings"), restate.WithMaxRetryAttempts(runMaxAttempts))
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
			logger.Warn(
				"skipping non-schedulable region",
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
			restate.ToTerminalError(fmt.Errorf("no schedulable regions configured for app %s in environment %s", deployment.AppID, deployment.EnvironmentID), restate.WithErrorCode(400)),
			fault.Public(deployfail.MsgNoSchedulableRegions),
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

		//nolint: exhaustruct
		topologies = append(topologies, db.InsertDeploymentTopologyParams{
			WorkspaceID:                deployment.WorkspaceID,
			DeploymentID:               deployment.ID,
			RegionID:                   rs.RegionID,
			AutoscalingReplicasMin:     autoscalingMin,
			AutoscalingReplicasMax:     autoscalingMax,
			AutoscalingThresholdCpu:    rs.AutoscalingThresholdCpu,
			AutoscalingThresholdMemory: rs.AutoscalingThresholdMemory,
			DesiredStatus:              db.DeploymentTopologyDesiredStatusRunning,
		})
	}

	reservation, err := restate.Run(ctx, func(runCtx restate.RunContext) (topologyReservation, error) {
		return w.reserveTopologies(runCtx, reserveTopologiesRequest{Deployment: deployment, Topologies: topologies})
	}, restate.WithName("reserve deployment topologies"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fault.Wrap(
			fmt.Errorf("failed to reserve deployment topologies: %w", err),
			fault.Public("Deployment targets could not be saved."),
		)
	}
	if reservation.Message != "" {
		return nil, fault.Wrap(
			restate.ToTerminalError(errors.New(reservation.Detail)),
			fault.Public(reservation.Message),
		)
	}

	// Keep stopped rows for debugging instead of deleting them.
	for _, topo := range topologies {
		compensation.Add(
			fmt.Sprintf("stop deployment topology %s/%s", topo.DeploymentID, topo.RegionID),
			func(runCtx restate.RunContext) error {
				return w.db.UpdateDeploymentTopologyDesiredStatus(runCtx, db.UpdateDeploymentTopologyDesiredStatusParams{
					DeploymentID:  topo.DeploymentID,
					RegionID:      topo.RegionID,
					DesiredStatus: db.DeploymentTopologyDesiredStatusStopped,
					UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				})
			},
		)
	}

	return topologies, nil
}

type reserveTopologiesRequest struct {
	Deployment db.FindDeploymentForDeployRow
	Topologies []db.InsertDeploymentTopologyParams
}

// A zero topologyReservation means the topologies were inserted. Otherwise
// Message is the public quota message and Detail the consumed and limit values
type topologyReservation struct {
	Message string
	Detail  string
}

// reserveTopologies checks the workspace quota and inserts the topologies in
// one transaction. Two deployments of one workspace can otherwise both read
// the same sum and both pass. The lock on the limits row serialises them, and
// the sum leaves out this deployment's own rows so a re-run of the Run does
// not count itself and refuse a deployment that fits
func (w *Workflow) reserveTopologies(ctx context.Context, req reserveTopologiesRequest) (topologyReservation, error) {
	return db.TxWithResultRetry(ctx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (topologyReservation, error) {
		queries := db.NewQueries(tx)
		limits, err := queries.LockLimitsByWorkspaceID(txCtx, req.Deployment.WorkspaceID)
		if err != nil {
			return topologyReservation{}, err
		}
		allocated, err := queries.SumAllocatedResourcesByWorkspaceID(txCtx, db.SumAllocatedResourcesByWorkspaceIDParams{
			WorkspaceID:         req.Deployment.WorkspaceID,
			ExcludeDeploymentID: req.Deployment.ID,
		})
		if err != nil {
			return topologyReservation{}, err
		}
		for _, topo := range req.Topologies {
			replicas := int64(topo.AutoscalingReplicasMax)
			allocated.TotalCpuMillicores += int64(req.Deployment.CpuMillicores) * replicas
			allocated.TotalMemoryMib += int64(req.Deployment.MemoryMib) * replicas
			allocated.TotalStorageMib += int64(req.Deployment.StorageMib) * replicas
		}

		cpuMillicoresMax := int64(limits.CpuCoresMax) * 1_000
		switch {
		case allocated.TotalCpuMillicores > cpuMillicoresMax:
			return topologyReservation{
				Message: deployfail.MsgCPUQuotaExceeded,
				Detail:  fmt.Sprintf("CPU limit exceeded: consumed %d, limit %d", allocated.TotalCpuMillicores, cpuMillicoresMax),
			}, nil
		case allocated.TotalMemoryMib > int64(limits.MemoryMibMax):
			return topologyReservation{
				Message: deployfail.MsgMemoryQuotaExceeded,
				Detail:  fmt.Sprintf("Memory limit exceeded: consumed %d, limit %d", allocated.TotalMemoryMib, limits.MemoryMibMax),
			}, nil
		case allocated.TotalStorageMib > int64(limits.StorageMibMax):
			return topologyReservation{
				Message: deployfail.MsgStorageQuotaExceeded,
				Detail:  fmt.Sprintf("Storage limit exceeded: consumed %d, limit %d", allocated.TotalStorageMib, limits.StorageMibMax),
			}, nil
		}

		now := time.Now().UnixMilli()
		topologies := slices.Clone(req.Topologies)
		for i := range topologies {
			topologies[i].CreatedAt = now
		}
		return topologyReservation{}, db.NewBulkQueries(tx).InsertDeploymentTopologies(txCtx, topologies)
	})
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
	deployment db.FindDeploymentForDeployRow,
) error {
	// Extract the fork owner from "owner/repo" for domain naming.
	forkOwner := ""
	if deployment.ForkRepositoryFullName.Valid {
		if parts := strings.SplitN(deployment.ForkRepositoryFullName.String, "/", 2); len(parts) == 2 && parts[0] != "" {
			forkOwner = parts[0]
		}
	}

	// CLI deploys reuse the same git SHA across iterative `unkey deploy`
	// runs, so the per-commit domain needs a random suffix to avoid
	// successive deploys overwriting each other. Webhook/dashboard/api
	// deploys always advance the SHA so they don't need it. The trigger
	// column on the deployment row is the authoritative signal.
	uniquifyCommitDomain := deployment.Trigger == db.DeploymentsTriggerCli

	allDomains := buildDomains(
		deployment.WorkspaceSlug,
		deployment.ProjectSlug,
		deployment.AppSlug,
		deployment.EnvironmentSlug,
		deployment.GitCommitSha.String,
		deployment.GitBranch.String,
		forkOwner,
		w.defaultDomain,
		deployment.EnvironmentKind.IsProduction(),
		uniquifyCommitDomain,
		deployment.ID,
	)

	existingRouteIDs := make([]string, 0)

	for _, domain := range allDomains {
		frontlineRouteID, getFrontlineRouteErr := restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
			return db.TxWithResultRetry(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (string, error) {
				found, err := db.NewQueries(tx).FindFrontlineRouteByFQDN(txCtx, domain.domain)
				if err != nil {
					if db.IsNotFound(err) {
						err = db.NewQueries(tx).InsertFrontlineRoute(runCtx, db.InsertFrontlineRouteParams{
							ID:                       uid.New(uid.FrontlineRoutePrefix),
							ProjectID:                deployment.ProjectID,
							AppID:                    deployment.AppID,
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
		}, restate.WithName(fmt.Sprintf("inserting frontline route %s", domain.domain)), restate.WithMaxRetryAttempts(runMaxAttempts))
		if getFrontlineRouteErr != nil {
			return fault.Wrap(getFrontlineRouteErr, fault.Public("Route records could not be created."))
		}
		if frontlineRouteID != "" {
			existingRouteIDs = append(existingRouteIDs, frontlineRouteID)
		}
	}

	// refresh app, cause it might have changed since we read it at the beginning of the workflow (e.g. another deployment promoted to live and updated current_deployment_id)
	app, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.App, error) {
		return w.db.FindAppById(runCtx, deployment.AppID)
	}, restate.WithName("refresh app before promotion"), restate.WithMaxRetryAttempts(runMaxAttempts))
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
			app, err := db.NewQueries(tx).FindAppById(txCtx, app.ID)
			if err != nil {
				return nil, err
			}

			// if the app is rolled back, we should not consider live sticky routes for promotion, even for production, because the live deployment is not healthy and should not receive traffic
			if app.IsRolledBack {
				stickyTypes = []db.FrontlineRoutesSticky{db.FrontlineRoutesStickyEnvironment}
			}

			routes, err := db.NewQueries(tx).FindFrontlineRoutesByEnvironmentAndSticky(txCtx, db.FindFrontlineRoutesByEnvironmentAndStickyParams{
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
	}, restate.WithName("finding sticky routes"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return fault.Wrap(
			fmt.Errorf("failed to find sticky routes: %w", err),
			fault.Public("Failed to read from database. Please try again."),
		)
	}

	// Routing VO is keyed by env_id — per-env serialization for both route
	// reassignment and live-deployment swaps.
	_, err = hydrav1.NewRoutingServiceClient(ctx, deployment.EnvironmentID).
		AssignFrontlineRoutes().Request(&hydrav1.AssignFrontlineRoutesRequest{
		DeploymentId:      deployment.ID,
		FrontlineRouteIds: append(routeIDs, existingRouteIDs...),
	})
	if err != nil {
		return fault.Wrap(
			fmt.Errorf("failed to assign domains: %w", err),
			fault.Public("Domain routing could not be updated."),
		)
	}

	return nil
}

func (w *Workflow) spinDownPreviousDeployments(
	ctx restate.ObjectContext,
	deployment db.FindDeploymentForDeployRow,
) error {
	previousDeploymentIDs, err := restate.Run(ctx, func(ctx restate.RunContext) ([]string, error) {
		return w.db.ListRunningDeploymentsByBranch(ctx, db.ListRunningDeploymentsByBranchParams{
			GitBranch:       deployment.GitBranch,
			WorkspaceID:     deployment.WorkspaceID,
			ProjectID:       deployment.ProjectID,
			AppID:           deployment.AppID,
			EnvironmentID:   deployment.EnvironmentID,
			NotDeploymentID: deployment.ID,
		})
	})
	if err != nil {
		return err
	}
	for _, previousDeploymentID := range previousDeploymentIDs {
		_, err := hydrav1.NewDeploymentServiceClient(ctx, previousDeploymentID).
			ScheduleDesiredStateChange().Request(
			&hydrav1.ScheduleDesiredStateChangeRequest{
				DelayMillis: time.Minute.Milliseconds(), // give frontline a graceperiod to clear their caches
				State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
				Overwrite:   false, // do not overwrite a previously scheduled transition
			},
			restate.WithIdempotencyKey(previousDeploymentID),
		)
		if err != nil {
			return fault.Wrap(err, fault.Public("Previous live deployment could not be scheduled for standby."))
		}
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
	deployment db.FindDeploymentForDeployRow,
) error {
	if deployment.AppIsRolledBack || !deployment.EnvironmentKind.IsProduction() {
		return nil
	}

	swapResp, err := hydrav1.NewRoutingServiceClient(ctx, deployment.EnvironmentID).
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
				State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
				Overwrite:   true,
			},
			restate.WithIdempotencyKey(swapResp.GetPreviousDeploymentId()),
		)
		if err != nil {
			return fault.Wrap(err, fault.Public("Previous live deployment could not be scheduled for standby."))
		}
	}

	return nil
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
	deployment db.FindDeploymentForDeployRow,
) *ghStatusReporter {
	reporter := &ghStatusReporter{
		client:    hydrav1.NewGitHubStatusServiceClient(ctx, deployment.ID),
		connected: false,
	}

	repoConn, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.GithubRepoConnection, error) {
		found, findErr := w.db.FindGithubRepoConnectionByAppId(runCtx, deployment.AppID)
		if findErr != nil {
			if db.IsNotFound(findErr) {
				// No connection — return zero value, not an error.
				// Returning an error here would cause Restate to retry forever.
				return db.GithubRepoConnection{}, nil //nolint:exhaustruct
			}
			return db.GithubRepoConnection{}, findErr //nolint:exhaustruct
		}
		return found, nil
	}, restate.WithName("find github repo connection"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		logger.Warn(
			"failed to look up github repo connection, skipping deployment status reporting",
			"app_id", deployment.AppID,
			"error", err,
		)

		return reporter
	}

	if repoConn.InstallationID == 0 {
		logger.Info(
			"no github repo connection, skipping deployment status reporting",
			"app_id", deployment.AppID,
		)

		return reporter
	}

	reporter.connected = true

	envLabel := formatEnvironmentLabel(deployment.ProjectSlug, deployment.AppSlug, deployment.EnvironmentSlug)
	prefix := formatDomainPrefix(deployment.ProjectSlug, deployment.AppSlug)
	envURL := fmt.Sprintf("https://%s-%s-%s.%s", prefix, deployment.EnvironmentSlug, deployment.WorkspaceSlug, w.defaultDomain)
	logURL := fmt.Sprintf("%s/%s/projects/%s/deployments/%s", w.dashboardURL, deployment.WorkspaceSlug, deployment.ProjectID, deployment.ID)

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
		IsProduction:               deployment.EnvironmentKind.IsProduction(),
		ProjectSlug:                deployment.ProjectSlug,
		AppSlug:                    deployment.AppSlug,
		EnvSlug:                    deployment.EnvironmentSlug,
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
