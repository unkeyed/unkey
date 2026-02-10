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
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	// sentinelNamespace isolates sentinel resources from tenant namespaces to
	// simplify RBAC and keep routing infrastructure separate from workloads.
	sentinelNamespace = "sentinel"

	// sentinelPort is the port exposed by sentinel services for frontline traffic
	// and must match the container port and service configuration.
	sentinelPort = 8040
)

// Deploy executes a full deployment workflow for a new application version.
//
// This durable workflow orchestrates the complete deployment lifecycle: building
// Docker images (if a GitSource is provided via Depot), provisioning containers
// across regions, waiting for instances to become healthy, and configuring domain
// routing. The workflow is idempotent and can safely resume from any step after
// a crash.
//
// The deployment request specifies a source as a oneof: either a GitSource (which
// triggers a Docker build through Depot) or a DockerImage (which is deployed
// directly).
//
// The workflow creates deployment topologies for all configured regions, each with
// a version obtained from VersioningService and 1 desired replica. Sentinel
// containers are automatically provisioned for environments that don't already
// have them, with production sentinels getting 3 replicas and others getting 1.
//
// Domain routing is configured through frontline routes. Sticky routes
// (environment, and live for non-rolled-back production) are reassigned to the
// new deployment. For production deployments, the project's live deployment
// pointer is updated unless the project is in a rolled-back state. After a
// successful deploy, the previous live deployment is scheduled for standby after
// 30 minutes via DeploymentService.ScheduleDesiredStateChange.
//
// If any step fails, the deployment status is automatically set to failed via a
// deferred cleanup handler, ensuring the database reflects the true deployment state.
//
// Returns terminal errors for validation failures and retryable errors for
// transient system failures.
func (w *Workflow) Deploy(ctx restate.WorkflowSharedContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	finishedSuccessfully := false

	err := assert.All(
		assert.NotEmpty(req.GetDeploymentId(), "deployment_id is required"),
	)
	if err != nil {
		return nil, restate.TerminalError(err)
	}

	logger.Info("deployment workflow started", "req", fmt.Sprintf("%+v", req))

	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(runCtx, w.db.RW(), req.GetDeploymentId())
	}, restate.WithName("finding deployment"))
	if err != nil {
		return nil, err
	}

	defer func() {
		if finishedSuccessfully {
			return
		}

		if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusFailed); err != nil {
			logger.Error("deployment failed but we can not set the status", "error", err.Error())
		}
	}()

	workspace, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Workspace, error) {

		var ws db.Workspace
		err := db.TxRetry(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {

			found, err := db.Query.FindWorkspaceByID(txCtx, tx, deployment.WorkspaceID)
			if err != nil {
				if db.IsNotFound(err) {
					return restate.TerminalError(errors.New("workspace not found"))
				}
				return err
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
		return nil, err
	}
	project, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindProjectByIdRow, error) {
		return db.Query.FindProjectById(runCtx, w.db.RW(), deployment.ProjectID)
	}, restate.WithName("finding project"))
	if err != nil {
		return nil, err
	}

	// storing for later to spin this one down
	previousLiveDeploymentID := project.LiveDeploymentID

	environment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindEnvironmentByIdRow, error) {
		return db.Query.FindEnvironmentById(runCtx, w.db.RW(), deployment.EnvironmentID)
	}, restate.WithName("finding environment"))
	if err != nil {
		return nil, err
	}

	dockerImage := ""

	switch source := req.GetSource().(type) {
	case *hydrav1.DeployRequest_DockerImage:
		dockerImage = source.DockerImage.GetImage()
	case *hydrav1.DeployRequest_Git:
		build, err := w.buildDockerImageFromGit(ctx, gitBuildParams{
			InstallationID: source.Git.GetInstallationId(),
			Repository:     source.Git.GetRepository(),
			CommitSHA:      source.Git.GetCommitSha(),
			ContextPath:    source.Git.GetContextPath(),
			DockerfilePath: source.Git.GetDockerfilePath(),
			ProjectID:      deployment.ProjectID,
			DeploymentID:   deployment.ID,
			WorkspaceID:    deployment.WorkspaceID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to build docker image from git: %w", err)
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
			return nil, fmt.Errorf("failed to update deployment build ID: %w", err)
		}

	default:
		return nil, restate.TerminalError(fmt.Errorf("unknown source type: %T", source))
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Query.UpdateDeploymentImage(runCtx, w.db.RW(), db.UpdateDeploymentImageParams{
			ID:        deployment.ID,
			Image:     sql.NullString{Valid: true, String: dockerImage},
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("update deployment image"))
	if err != nil {
		return nil, err
	}

	if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusDeploying); err != nil {
		return nil, err
	}

	topologies := make([]db.InsertDeploymentTopologyParams, len(w.availableRegions))

	for i, region := range w.availableRegions {
		versionResp, err := hydrav1.NewVersioningServiceClient(ctx, region).NextVersion().Request(&hydrav1.NextVersionRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to get next version: %w", err)
		}

		topologies[i] = db.InsertDeploymentTopologyParams{
			WorkspaceID:     workspace.ID,
			DeploymentID:    deployment.ID,
			Region:          region,
			DesiredReplicas: 1,
			DesiredStatus:   db.DeploymentTopologyDesiredStatusStarting,
			Version:         versionResp.GetVersion(),
			CreatedAt:       time.Now().UnixMilli(),
		}
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			return db.BulkQuery.InsertDeploymentTopologies(txCtx, tx, topologies)
		})
	}, restate.WithName("insert deployment topologies"))
	if err != nil {
		return nil, fmt.Errorf("failed to insert deployment topologies: %w", err)
	}

	// Ensure sentinels exist in each region for this deployment

	existingSentinels, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Sentinel, error) {
		return db.Query.FindSentinelsByEnvironmentID(runCtx, w.db.RO(), environment.ID)
	}, restate.WithName("find existing sentinels"))

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
				return nil, fmt.Errorf("failed to get next version for sentinel: %w", err)
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
				return nil, err
			}

		}

	}

	if err := w.ensureCiliumNetworkPolicy(ctx, workspace, project, environment, topologies); err != nil {
		return nil, err
	}
	logger.Info("waiting for deployments to be ready", "deployment_id", deployment.ID)

	readygates := make([]restate.Future, len(topologies))
	for i, region := range topologies {
		promise := restate.RunAsync(ctx, func(runCtx restate.RunContext) (bool, error) {

			for {
				time.Sleep(time.Second)

				instances, err := db.Query.FindInstancesByDeploymentIdAndRegion(runCtx, w.db.RO(), db.FindInstancesByDeploymentIdAndRegionParams{
					Deploymentid: deployment.ID,
					Region:       region.Region,
				})
				if err != nil {
					return false, err
				}
				if len(instances) < int(region.DesiredReplicas) {
					continue
				}
				allRunning := true
				for _, instance := range instances {
					if instance.Status != db.InstancesStatusRunning {
						allRunning = false
						break
					}
				}
				if allRunning {
					return true, nil
				}

			}

		}, restate.WithName(fmt.Sprintf("wait for instances in %s", region.Region)))
		readygates[i] = promise
	}

	for _, err := range restate.Wait(ctx, readygates...) {
		if err != nil {
			return nil, err
		}
	}

	logger.Info("deployments ready", "deployment_id", deployment.ID)

	allDomains := buildDomains(
		workspace.Slug,
		project.Slug,
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
			return nil, getFrontlineRouteErr
		}
		if frontlineRouteID != "" {
			existingRouteIDs = append(existingRouteIDs, frontlineRouteID)
		}
	}

	// Fetch sticky routes for this environment
	stickyTypes := []db.FrontlineRoutesSticky{db.FrontlineRoutesStickyEnvironment}
	if !project.IsRolledBack && environment.Slug == "production" {
		// Only reassign live routes when not rolled back - rollbacks keep live routes on the previous deployment
		stickyTypes = append(stickyTypes, db.FrontlineRoutesStickyLive)
	}

	stickyRoutes, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindFrontlineRouteForPromotionRow, error) {
		return db.Query.FindFrontlineRouteForPromotion(stepCtx, w.db.RO(), db.FindFrontlineRouteForPromotionParams{
			EnvironmentID: deployment.EnvironmentID,
			Sticky:        stickyTypes,
		})
	}, restate.WithName("finding sticky routes"))
	if err != nil {
		return nil, fmt.Errorf("failed to find sticky routes: %w", err)
	}

	for _, route := range stickyRoutes {
		existingRouteIDs = append(existingRouteIDs, route.ID)
	}

	_, err = hydrav1.NewRoutingServiceClient(ctx, project.ID).
		AssignFrontlineRoutes().Request(&hydrav1.AssignFrontlineRoutesRequest{
		DeploymentId:      deployment.ID,
		FrontlineRouteIds: existingRouteIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assign domains: %w", err)
	}

	if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusReady); err != nil {
		return nil, err
	}

	if !project.IsRolledBack && environment.Slug == "production" {
		_, err = restate.Run(ctx, func(runCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateProjectDeployments(runCtx, w.db.RW(), db.UpdateProjectDeploymentsParams{
				IsRolledBack:     false,
				ID:               deployment.ProjectID,
				LiveDeploymentID: sql.NullString{Valid: true, String: deployment.ID},
				UpdatedAt:        sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName("updating project live deployment"))
		if err != nil {
			return nil, err
		}
	}

	if previousLiveDeploymentID.Valid {
		hydrav1.NewDeploymentServiceClient(ctx, previousLiveDeploymentID.String).
			ScheduleDesiredStateChange().Send(
			&hydrav1.ScheduleDesiredStateChangeRequest{
				After: time.Now().Add(30 * time.Minute).UnixMilli(),
				State: hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STANDBY,
			},
			restate.WithIdempotencyKey(deployment.ID),
		)
	}

	logger.Info("deployment workflow completed",
		"deployment_id", deployment.ID,
		"status", "succeeded",
		"domains", len(allDomains),
	)

	finishedSuccessfully = true

	return &hydrav1.DeployResponse{}, nil
}
