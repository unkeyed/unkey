package deploy

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"google.golang.org/protobuf/proto"
)

// Deploy orchestrates the complete deployment of a new Docker image.
//
// This durable workflow performs the following steps:
// 1. Load deployment, workspace, project, and environment data
// 2. Create deployment in Krane (container orchestration)
// 3. Poll for all instances to become ready
// 4. Register VMs in partition database
// 5. Scrape OpenAPI spec from running instances (if available)
// 6. Assign domains and create gateway configs via routing service
// 7. Update deployment status to ready
// 8. Update project's live deployment pointer (if production and not rolled back)
//
// Each step is wrapped in restate.Run for durability. If the workflow is interrupted,
// it resumes from the last completed step. A deferred error handler ensures that
// failed deployments are properly marked in the database even if the workflow crashes.
//
// The workflow uses a 5-minute polling loop to wait for instances to become ready,
// checking Krane deployment status every second and logging progress every 10 seconds.
func (w *Workflow) Deploy(ctx restate.ObjectContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	finishedSuccessfully := false

	deployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindDeploymentByIdRow, error) {
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

	workspace, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Workspace, error) {
		return db.Query.FindWorkspaceByID(stepCtx, w.db.RW(), deployment.WorkspaceID)
	}, restate.WithName("finding workspace"))
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

	// Log deployment pending
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		err = db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			WorkspaceID:  deployment.WorkspaceID,
			ProjectID:    deployment.ProjectID,
			DeploymentID: deployment.ID,
			Status:       "pending",
			Message:      "Deployment queued and ready to start",
			CreatedAt:    time.Now().UnixMilli(),
		})
		return restate.Void{}, err
	}, restate.WithName("logging deployment pending"))
	if err != nil {
		return nil, err
	}

	var dockerImage string

	if req.GetBuildContextPath() != "" {
		// Build Docker image from uploaded context
		if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusBuilding); err != nil {
			return nil, err
		}

		_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
				WorkspaceID:  deployment.WorkspaceID,
				ProjectID:    deployment.ProjectID,
				DeploymentID: deployment.ID,
				Status:       "pending",
				Message:      "Building Docker image from source",
				CreatedAt:    time.Now().UnixMilli(),
			})
		}, restate.WithName("logging build start"))
		if err != nil {
			return nil, err
		}

		dockerImage, err = restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
			w.logger.Info("starting docker build",
				"deployment_id", deployment.ID,
				"build_context_path", req.GetBuildContextPath())

			buildReq := connect.NewRequest(&ctrlv1.CreateBuildRequest{
				UnkeyProjectId:   deployment.ProjectID,
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

		_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
				WorkspaceID:  deployment.WorkspaceID,
				ProjectID:    deployment.ProjectID,
				DeploymentID: deployment.ID,
				Status:       "pending",
				Message:      fmt.Sprintf("Docker image built successfully: %s", dockerImage),
				CreatedAt:    time.Now().UnixMilli(),
			})
		}, restate.WithName("logging build complete"))
		if err != nil {
			return nil, err
		}

	} else if req.GetDockerImage() != "" {
		dockerImage = req.GetDockerImage()
		w.logger.Info("using prebuilt docker image",
			"deployment_id", deployment.ID,
			"image", dockerImage)
	} else {
		return nil, fmt.Errorf("either build_context_path or docker_image must be specified")
	}

	// Update version status to deploying
	if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusDeploying); err != nil {
		return nil, err
	}

	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		// Create deployment request

		_, err = w.krane.CreateDeployment(stepCtx, connect.NewRequest(&kranev1.CreateDeploymentRequest{
			Deployment: &kranev1.DeploymentRequest{
				Namespace:     hardcodedNamespace,
				DeploymentId:  deployment.ID,
				Image:         req.GetDockerImage(),
				Replicas:      1,
				CpuMillicores: 512,
				MemorySizeMib: 512,
			},
		}))
		if err != nil {
			return restate.Void{}, fmt.Errorf("krane CreateDeployment failed for image %s: %w", req.GetDockerImage(), err)
		}

		return restate.Void{}, nil
	}, restate.WithName("creating deployment in krane"))
	if err != nil {
		return nil, err
	}

	w.logger.Info("deployment created", "deployment_id", deployment.ID)

	createdInstances, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]*kranev1.Instance, error) {
		// prevent updating the db unnecessarily

		for i := range 300 {
			time.Sleep(time.Second)
			if i%10 == 0 { // Log every 10 seconds instead of every second
				w.logger.Info("polling deployment status", "deployment_id", deployment.ID, "iteration", i)
			}

			var resp *connect.Response[kranev1.GetDeploymentResponse]
			resp, err = w.krane.GetDeployment(stepCtx, connect.NewRequest(&kranev1.GetDeploymentRequest{
				Namespace:    hardcodedNamespace,
				DeploymentId: deployment.ID,
			}))
			if err != nil {
				return nil, fmt.Errorf("krane GetDeployment failed for deployment %s: %w", deployment.ID, err)
			}

			w.logger.Info("deployment status",
				"deployment_id", deployment.ID,
				"status", resp.Msg,
			)

			allReady := true
			for _, instance := range resp.Msg.GetInstances() {
				if instance.GetStatus() != kranev1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING {
					allReady = false
				}

				var status partitiondb.VmsStatus
				switch instance.GetStatus() {
				case kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING:
					status = partitiondb.VmsStatusProvisioning
				case kranev1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING:
					status = partitiondb.VmsStatusRunning

				case kranev1.DeploymentStatus_DEPLOYMENT_STATUS_TERMINATING:
					status = partitiondb.VmsStatusStopping
				case kranev1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED:
					status = partitiondb.VmsStatusAllocated
				}

				upsertParams := partitiondb.UpsertVMParams{
					ID:           instance.GetId(),
					DeploymentID: deployment.ID,
					Address:      sql.NullString{Valid: true, String: instance.GetAddress()},
					// nolint: godox
					// TODO: Make sure configurable later
					CpuMillicores: 1000,
					MemoryMb:      1024,
					Status:        status,
				}

				w.logger.Info("upserting VM to database",
					"vm_id", instance.GetId(),
					"deployment_id", deployment.ID,
					"address", instance.GetAddress(),
					"status", status)
				if err = partitiondb.Query.UpsertVM(stepCtx, w.partitionDB.RW(), upsertParams); err != nil {
					return nil, fmt.Errorf("failed to upsert VM %s: %w", instance.GetId(), err)
				}

				w.logger.Info("successfully upserted VM to database", "vm_id", instance.GetId())

			}

			if allReady {
				return resp.Msg.GetInstances(), nil
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
			openapiURL := fmt.Sprintf("http://%s/openapi.yaml", instance.GetAddress())
			w.logger.Info("trying to scrape OpenAPI spec", "url", openapiURL, "host_port", instance.GetAddress(), "deployment_id", deployment.ID)

			var resp *http.Response
			resp, err = http.DefaultClient.Get(openapiURL)
			if err != nil {
				w.logger.Warn("openapi scraping failed for host address", "error", err, "host_addr", instance.GetAddress(), "deployment_id", deployment.ID)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				w.logger.Warn("openapi endpoint returned non-200 status", "status", resp.StatusCode, "host_addr", instance.GetAddress(), "deployment_id", deployment.ID)
				continue
			}

			// Read the OpenAPI spec
			var specBytes []byte
			specBytes, err = io.ReadAll(resp.Body)
			if err != nil {
				w.logger.Warn("failed to read OpenAPI spec response", "error", err, "host_addr", instance.GetAddress(), "deployment_id", deployment.ID)
				continue
			}

			w.logger.Info("openapi spec scraped successfully", "host_addr", instance.GetAddress(), "deployment_id", deployment.ID, "spec_size", len(specBytes))
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
				UpdatedAt:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				OpenapiSpec: sql.NullString{Valid: true, String: openapiSpec},
			})
		}, restate.WithName("update deployment openapi spec"))
	}

	allDomains := buildDomains(
		workspace.Slug,
		project.Slug,
		environment.Slug,
		deployment.GitCommitSha.String,
		deployment.GitBranch.String,
		w.defaultDomain,
		ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD, // hardcoded for now cause I really need to move on
	)

	// Create VM protobuf objects for gateway config
	gatewayConfig := &partitionv1.GatewayConfig{
		Project:          nil,
		AuthConfig:       nil,
		ValidationConfig: nil,
		Deployment: &partitionv1.Deployment{
			Id:        deployment.ID,
			IsEnabled: true,
		},
		Vms: make([]*partitionv1.VM, len(createdInstances)),
	}

	for i, vm := range createdInstances {
		gatewayConfig.Vms[i] = &partitionv1.VM{
			Id: vm.GetId(),
		}
	}

	// Only add AuthConfig if we have a KeyspaceID
	if req.GetKeyAuthId() != "" {
		gatewayConfig.AuthConfig = &partitionv1.AuthConfig{
			KeyAuthId: req.GetKeyAuthId(),
		}
	}

	if openapiSpec != "" {
		gatewayConfig.ValidationConfig = &partitionv1.ValidationConfig{
			OpenapiSpec: openapiSpec,
		}
	}

	// Build domain assignment requests
	domainRequests := make([]*hydrav1.DomainToAssign, 0, len(allDomains))
	for _, domain := range allDomains {
		sticky := hydrav1.DomainSticky_DOMAIN_STICKY_UNSPECIFIED
		if domain.sticky.Valid {
			switch domain.sticky.DomainsSticky {
			case db.DomainsStickyBranch:
				sticky = hydrav1.DomainSticky_DOMAIN_STICKY_BRANCH
			case db.DomainsStickyEnvironment:
				sticky = hydrav1.DomainSticky_DOMAIN_STICKY_ENVIRONMENT
			case db.DomainsStickyLive:
				sticky = hydrav1.DomainSticky_DOMAIN_STICKY_LIVE
			}
		}
		domainRequests = append(domainRequests, &hydrav1.DomainToAssign{
			Name:   domain.domain,
			Sticky: sticky,
		})
	}

	// Call RoutingService to assign domains atomically
	_, err = hydrav1.NewRoutingServiceClient(ctx, project.ID).
		AssignDomains().Request(&hydrav1.AssignDomainsRequest{
		WorkspaceId:   workspace.ID,
		ProjectId:     project.ID,
		EnvironmentId: environment.ID,
		DeploymentId:  deployment.ID,
		Domains:       domainRequests,
		GatewayConfig: gatewayConfig,
		IsRolledBack:  project.IsRolledBack,
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
		"domains", len(allDomains))

	finishedSuccessfully = true

	return &hydrav1.DeployResponse{}, nil
}
