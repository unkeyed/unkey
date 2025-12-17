package deploy

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"google.golang.org/protobuf/proto"
)

func (w *Workflow) Deploy(ctx restate.ObjectContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	finishedSuccessfully := false

	deployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(stepCtx, w.db.RW(), req.GetDeploymentId())
	}, restate.WithName("finding deployment"))
	if err != nil {
		return nil, err
	}

	// If anything goes wrong, we need to update the deployment status to failed
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
				ws.K8sNamespace.String = uid.Nano(8)
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
		// Build Docker image from uploaded context
		if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusBuilding); err != nil {
			return nil, err
		}

		dockerImage, err = restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
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
				return "", fmt.Errorf("build failed: %w", err)
			}

			imageName := buildResp.Msg.GetImageName()
			w.logger.Info("docker build completed",
				"deployment_id", deployment.ID,
				"image_name", imageName)

			return imageName, nil
		}, restate.WithName("building docker image"))
		if err != nil {
			return nil, fmt.Errorf("failed to build docker image: %w", err)
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

	// later we read this from project config
	regions := []string{"aws:us-east-1"}

	topologies := make([]db.InsertDeploymentTopologyParams, len(regions))
	for i, region := range regions {
		topologies[i] = db.InsertDeploymentTopologyParams{
			WorkspaceID:  workspace.ID,
			DeploymentID: deployment.ID,
			Region:       region,
			Replicas:     1,
			Status:       db.DeploymentTopologyStatusStarting,
			CreatedAt:    time.Now().UnixMilli(),
		}
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return db.BulkQuery.InsertDeploymentTopologies(runCtx, w.db.RW(), topologies)
	}, restate.WithName("insert-deployment-topologies"))
	if err != nil {
		return nil, err
	}

	if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusDeploying); err != nil {
		return nil, err
	}

	for _, region := range regions {
		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return w.cluster.EmitDeploymentState(runCtx, map[string]string{"region": region, "shard": "default"}, &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Apply{
					Apply: &ctrlv1.ApplyDeployment{
						Namespace:     workspace.K8sNamespace.String,
						K8SCrdName:    deployment.K8sCrdName,
						WorkspaceId:   workspace.ID,
						ProjectId:     project.ID,
						EnvironmentId: environment.ID,
						DeploymentId:  deployment.ID,
						Image:         dockerImage,
						Replicas:      1,
						CpuMillicores: 256,
						MemoryMib:     256,
					},
				},
			})
		}, restate.WithName(fmt.Sprintf("apply deployment in %s", region)))
		if err != nil {
			return nil, err
		}
	}

	w.logger.Info("deployment created", "deployment_id", deployment.ID)

	createdInstances, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.Instance, error) {

		for i := range 300 {
			time.Sleep(time.Second)
			if i%10 == 0 { // Log every 10 seconds instead of every second
				w.logger.Info("polling deployment status", "deployment_id", deployment.ID, "iteration", i)
			}

			instances, err := db.Query.FindInstancesByDeploymentId(stepCtx, w.db.RO(), deployment.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to find instances for deployment %s: %w", deployment.ID, err)
			}
			if len(instances) == 0 {
				continue
			}

			allReady := true
			for _, instance := range instances {
				if instance.Status != db.InstancesStatusRunning {
					allReady = false
					break
				}
			}

			if allReady {
				return instances, nil
			}
			// next loop
		}

		return nil, fmt.Errorf("deployment never became ready")
	}, restate.WithName("polling deployment status"))
	if err != nil {
		return nil, err
	}

	openapiSpec, err := restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
		for _, instance := range createdInstances {
			openapiURL := fmt.Sprintf("http://%s/openapi.yaml", instance.Address)
			w.logger.Info("trying to scrape OpenAPI spec", "url", openapiURL, "host_port", instance.Address, "deployment_id", deployment.ID)

			var resp *http.Response
			resp, err = http.DefaultClient.Get(openapiURL)
			if err != nil {
				w.logger.Warn("openapi scraping failed for host address", "error", err, "host_addr", instance.Address, "deployment_id", deployment.ID)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				w.logger.Warn("openapi endpoint returned non-200 status", "status", resp.StatusCode, "host_addr", instance.Address, "deployment_id", deployment.ID)
				continue
			}

			// Read the OpenAPI spec
			var specBytes []byte
			specBytes, err = io.ReadAll(resp.Body)
			if err != nil {
				w.logger.Warn("failed to read OpenAPI spec response", "error", err, "host_addr", instance.Address, "deployment_id", deployment.ID)
				continue
			}

			w.logger.Info("openapi spec scraped successfully", "host_addr", instance.Address, "deployment_id", deployment.ID, "spec_size", len(specBytes))
			return base64.StdEncoding.EncodeToString(specBytes), nil
		}
		// not an error really, just no OpenAPI spec found
		return "", nil
	}, restate.WithName("scrape openapi spec"))
	if err != nil {
		return nil, err
	}

	if openapiSpec != "" {
		_, err = restate.Run(ctx, func(innerCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateDeploymentOpenapiSpec(innerCtx, w.db.RW(), db.UpdateDeploymentOpenapiSpecParams{
				ID:          deployment.ID,
				OpenapiSpec: sql.NullString{Valid: true, String: openapiSpec},
				UpdatedAt:   sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			})
		}, restate.WithName("update deployment openapi spec"))
	}

	allHostnames := buildDomains(
		workspace.Slug,
		project.Slug,
		environment.Slug,
		deployment.GitCommitSha.String,
		deployment.GitBranch.String,
		w.defaultDomain,
		ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD, // hardcoded for now cause I really need to move on
	)

	// Build domain assignment requests

	existingRouteIDs := make([]string, 0)

	for _, hostname := range allHostnames {
		frontlineRouteID, getFrontlineRouteErr := restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
			return db.TxWithResult(stepCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (string, error) {
				found, err := db.Query.FindFrontlineRouteByHostname(txCtx, tx, hostname.domain)
				if err != nil {
					if db.IsNotFound(err) {
						err = db.Query.InsertFrontlineRoute(stepCtx, tx, db.InsertFrontlineRouteParams{
							ID:            uid.New("todo"),
							ProjectID:     project.ID,
							DeploymentID:  deployment.ID,
							EnvironmentID: deployment.EnvironmentID,
							Hostname:      hostname.domain,
							Sticky:        hostname.sticky,
							CreatedAt:     time.Now().UnixMilli(),
							UpdatedAt:     sql.NullInt64{Valid: false, Int64: 0},
						})
						// return empty string cause this frontline is already updated since we just created it
						return "", err

					}
					return "", err
				}
				return found.ID, nil

			})
		})
		if getFrontlineRouteErr != nil {
			return nil, getFrontlineRouteErr
		}
		if frontlineRouteID != "" {
			existingRouteIDs = append(existingRouteIDs, frontlineRouteID)
		}
	}

	// Call RoutingService to assign domains atomically
	_, err = hydrav1.NewRoutingServiceClient(ctx, project.ID).
		AssignFrontlineRoutes().Request(&hydrav1.AssignFrontlineRoutesRequest{
		DeploymentId:      deployment.ID,
		FrontlineRouteIds: existingRouteIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assign domains: %w", err)
	}

	// Update deployment status to ready
	if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusReady); err != nil {
		return nil, err
	}

	if !project.IsRolledBack && environment.Slug == "production" {
		// only update this if the deployment is not rolled back
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

	// Log deployment completed
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			ProjectID:    deployment.ProjectID,
			WorkspaceID:  deployment.WorkspaceID,
			DeploymentID: deployment.ID,
			Status:       "completed",
			Message:      "Deployment completed successfully",
			CreatedAt:    time.Now().UnixMilli(),
		})
	}, restate.WithName("logging deployment completed"))
	if err != nil {
		return nil, err
	}

	w.logger.Info("deployment workflow completed",
		"deployment_id", deployment.ID,
		"status", "succeeded",
		"hostnames", len(allHostnames),
	)

	finishedSuccessfully = true

	return &hydrav1.DeployResponse{}, nil
}
