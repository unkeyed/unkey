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
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	// sentinelNamespace is the Kubernetes namespace where sentinel containers are deployed.
	sentinelNamespace = "sentinel"

	// sentinelPort is the port that sentinel containers listen on for traffic routing.
	sentinelPort = 8040
)

// Deploy executes a full deployment workflow for a new application version.
//
// This durable workflow orchestrates the complete deployment lifecycle: building
// Docker images (if source is provided), provisioning containers across regions,
// waiting for instances to become healthy, and configuring domain routing. The
// workflow is idempotent and can safely resume from any step after a crash.
//
// The deployment request must specify either a build context path (to build from
// source) or a pre-built Docker image. If BuildContextPath is set, the workflow
// triggers a Docker build through the build service before deployment. Otherwise,
// the provided DockerImage is deployed directly.
//
// The workflow creates deployment topologies for all configured regions, each with
// its own version number for independent scaling and rollback. Sentinel containers
// are automatically provisioned for environments that don't already have them,
// with production environments getting 3 replicas and others getting 1.
//
// Domain routing is configured through frontline routes, with sticky domains
// (branch and environment) automatically updating to point to the new deployment.
// For production deployments, the project's live deployment pointer is updated
// unless the project is in a rolled-back state.
//
// If any step fails, the deployment status is automatically set to failed via a
// deferred cleanup handler, ensuring the database reflects the true deployment state.
//
// Returns terminal errors for validation failures (missing image/context) and
// retryable errors for transient system failures.
func (w *Workflow) Deploy(ctx restate.WorkflowSharedContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	finishedSuccessfully := false

	deployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(stepCtx, w.db.RW(), req.GetDeploymentId())
	}, restate.WithName("finding deployment"))
	if err != nil {
		return nil, err
	}

	defer func() {
		if finishedSuccessfully {
			return
		}

		if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusFailed); err != nil {
			w.logger.Error("deployment failed but we can not set the status", "error", err.Error())
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
	project, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindProjectByIdRow, error) {
		return db.Query.FindProjectById(stepCtx, w.db.RW(), deployment.ProjectID)
	}, restate.WithName("finding project"))
	if err != nil {
		return nil, err
	}
	environment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindEnvironmentByIdRow, error) {
		return db.Query.FindEnvironmentById(stepCtx, w.db.RW(), deployment.EnvironmentID)
	}, restate.WithName("finding environment"))
	if err != nil {
		return nil, err
	}

	var dockerImage string

	if req.GetBuildContextPath() != "" {
		if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusBuilding); err != nil {
			return nil, err
		}

		s3DownloadURL, err := restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
			return w.buildStorage.GenerateDownloadURL(stepCtx, req.GetBuildContextPath(), 1*time.Hour)
		}, restate.WithName("generate s3 download url"))
		if err != nil {
			return nil, fmt.Errorf("failed to generate s3 download url: %w", err)
		}

		w.logger.Info("starting docker build",
			"deployment_id", deployment.ID,
			"build_context_path", req.GetBuildContextPath())

		build, err := hydrav1.NewBuildServiceClient(ctx).BuildDockerImage().Request(&hydrav1.BuildDockerImageRequest{
			S3Url:            s3DownloadURL,
			BuildContextPath: req.GetBuildContextPath(),
			DockerfilePath:   req.GetDockerfilePath(),
			ProjectId:        deployment.ProjectID,
			DeploymentId:     deployment.ID,
			WorkspaceId:      deployment.WorkspaceID,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to build docker image: %w", err)
		}
		dockerImage = build.GetImageName()

		err = restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
			return db.Query.UpdateDeploymentBuildID(stepCtx, w.db.RW(), db.UpdateDeploymentBuildIDParams{
				ID:        deployment.ID,
				BuildID:   sql.NullString{Valid: true, String: build.GetDepotBuildId()},
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update deployment build ID: %w", err)
		}

	} else if req.GetDockerImage() != "" {
		dockerImage = req.GetDockerImage()
		w.logger.Info("using prebuilt docker image",
			"deployment_id", deployment.ID,
			"image", dockerImage)
	} else {
		return nil, fmt.Errorf("either build_context_path or docker_image must be specified")
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
	w.logger.Info("waiting for deployments to be ready", "deployment_id", deployment.ID)

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

	w.logger.Info("deployments ready", "deployment_id", deployment.ID)

	allDomains := buildDomains(
		workspace.Slug,
		project.Slug,
		environment.Slug,
		deployment.GitCommitSha.String,
		deployment.GitBranch.String,
		w.defaultDomain,
		ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
	)

	existingRouteIDs := make([]string, 0)

	for _, domain := range allDomains {
		frontlineRouteID, getFrontlineRouteErr := restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
			return db.TxWithResultRetry(stepCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (string, error) {
				found, err := db.Query.FindFrontlineRouteByFQDN(txCtx, tx, domain.domain)
				if err != nil {
					if db.IsNotFound(err) {
						err = db.Query.InsertFrontlineRoute(stepCtx, tx, db.InsertFrontlineRouteParams{
							ID:                       uid.New("todo"),
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
		_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateProjectDeployments(stepCtx, w.db.RW(), db.UpdateProjectDeploymentsParams{
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

	w.logger.Info("deployment workflow completed",
		"deployment_id", deployment.ID,
		"status", "succeeded",
		"domains", len(allDomains),
	)

	finishedSuccessfully = true

	return &hydrav1.DeployResponse{}, nil
}
