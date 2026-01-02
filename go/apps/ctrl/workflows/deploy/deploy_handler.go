package deploy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"google.golang.org/protobuf/proto"
)

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
		err := db.Tx(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {

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
	var buildID *string

	if req.GetBuildContextPath() != "" {
		if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusBuilding); err != nil {
			return nil, err
		}

		result, err := restate.Run(ctx, func(stepCtx restate.RunContext) (*ctrlv1.CreateBuildResponse, error) {
			w.logger.Info("starting docker build",
				"deployment_id", deployment.ID,
				"build_context_path", req.GetBuildContextPath())

			buildReq := connect.NewRequest(&ctrlv1.CreateBuildRequest{
				UnkeyProjectId:   deployment.ProjectID,
				WorkspaceId:      deployment.WorkspaceID,
				DeploymentId:     deployment.ID,
				BuildContextPath: req.GetBuildContextPath(),
				DockerfilePath:   proto.String(req.GetDockerfilePath()),
			})

			var buildResp *connect.Response[ctrlv1.CreateBuildResponse]
			buildResp, err = w.buildClient.CreateBuild(stepCtx, buildReq)
			if err != nil {
				return &ctrlv1.CreateBuildResponse{}, fmt.Errorf("build failed: %w", err)
			}

			w.logger.Info("docker build completed", "deployment_id", deployment.ID, "image_name", buildResp.Msg.GetImageName())

			return buildResp.Msg, nil
		}, restate.WithName("building docker image"))
		if err != nil {
			return nil, fmt.Errorf("failed to build docker image: %w", err)
		}
		dockerImage = result.GetImageName()
		buildID = ptr.P(result.GetBuildId())

		err = restate.RunVoid(ctx, func(stepCtx restate.RunContext) error {
			return db.Query.UpdateDeploymentBuildID(stepCtx, w.db.RW(), db.UpdateDeploymentBuildIDParams{
				ID:        deployment.ID,
				BuildID:   sql.NullString{Valid: true, String: result.GetBuildId()},
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
		topologies[i] = db.InsertDeploymentTopologyParams{
			WorkspaceID:     workspace.ID,
			DeploymentID:    deployment.ID,
			Region:          region,
			DesiredReplicas: 1,
			DesiredStatus:   db.DeploymentTopologyDesiredStatusStarting,
			CreatedAt:       time.Now().UnixMilli(),
		}
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.BulkQuery.InsertDeploymentTopologies(runCtx, w.db.RW(), topologies)
	}, restate.WithName("insert deployment topologies"))

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

			err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {

				sentinelID := uid.New(uid.SentinelPrefix)
				sentinelK8sName := uid.DNS1035()

				// we rely on the unique indess of environmendID + region here to create or noop
				err = db.Query.InsertSentinel(runCtx, w.db.RW(), db.InsertSentinelParams{
					ID:                sentinelID,
					WorkspaceID:       workspace.ID,
					EnvironmentID:     environment.ID,
					ProjectID:         project.ID,
					K8sAddress:        fmt.Sprintf("%s.%s.svc.cluster.local", sentinelK8sName, workspace.K8sNamespace.String),
					K8sName:           sentinelK8sName,
					Region:            topology.Region,
					Image:             w.sentinelImage,
					Health:            db.SentinelsHealthUnknown,
					DesiredReplicas:   desiredReplicas,
					AvailableReplicas: 0,
					CpuMillicores:     256,
					MemoryMib:         256,
					CreatedAt:         time.Now().UnixMilli(),
				})
				if err != nil && !db.IsDuplicateKeyError(err) {
					return err
				}
				return nil

			}, restate.WithName("ensure sentinel exists in db"))
			if err != nil {
				return nil, err
			}

		}

		sentinels, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.Sentinel, error) {
			return db.Query.FindSentinelsByEnvironmentID(runCtx, w.db.RO(), environment.ID)
		}, restate.WithName("find all sentinels"))
		if err != nil {
			return nil, err
		}

		for _, sentinel := range sentinels {
			err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
				return w.cluster.EmitState(runCtx, sentinel.Region,
					&ctrlv1.State{
						AcknowledgeId: nil,
						Kind: &ctrlv1.State_Sentinel{
							Sentinel: &ctrlv1.SentinelState{
								State: &ctrlv1.SentinelState_Apply{
									Apply: &ctrlv1.ApplySentinel{
										K8SNamespace:  workspace.K8sNamespace.String,
										K8SName:       sentinel.K8sName,
										WorkspaceId:   sentinel.WorkspaceID,
										ProjectId:     sentinel.ProjectID,
										EnvironmentId: sentinel.EnvironmentID,
										SentinelId:    sentinel.ID,
										Image:         w.sentinelImage,
										Replicas:      sentinel.DesiredReplicas,
										CpuMillicores: int64(sentinel.CpuMillicores),
										MemoryMib:     int64(sentinel.MemoryMib),
									},
								},
							},
						},
					})

			}, restate.WithName(fmt.Sprintf("emit sentinel apply for %s in %s", sentinel.ID, sentinel.Region)))
			if err != nil {
				return nil, err
			}
		}
	}

	for _, region := range topologies {
		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return w.cluster.EmitState(runCtx, region.Region,
				&ctrlv1.State{
					AcknowledgeId: nil,
					Kind: &ctrlv1.State_Deployment{
						Deployment: &ctrlv1.DeploymentState{
							State: &ctrlv1.DeploymentState_Apply{
								Apply: &ctrlv1.ApplyDeployment{
									K8SNamespace:                  workspace.K8sNamespace.String,
									K8SName:                       deployment.K8sName,
									WorkspaceId:                   workspace.ID,
									ProjectId:                     deployment.ProjectID,
									EnvironmentId:                 deployment.EnvironmentID,
									DeploymentId:                  deployment.ID,
									Image:                         dockerImage,
									Replicas:                      region.DesiredReplicas,
									CpuMillicores:                 int64(deployment.CpuMillicores),
									MemoryMib:                     int64(deployment.MemoryMib),
									BuildId:                       buildID,
									EncryptedEnvironmentVariables: deployment.EncryptedEnvironmentVariables,
									ReadinessId:                   ptr.P(deployment.ID),
								},
							},
						},
					},
				},
			)

		}, restate.WithName(fmt.Sprintf("emit deployment apply %s in %s", deployment.ID, region.Region)))
		if err != nil {
			return nil, err
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
			return db.TxWithResult(stepCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (string, error) {
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
