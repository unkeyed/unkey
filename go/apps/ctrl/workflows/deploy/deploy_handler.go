package deploy

import (
	"context"
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
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
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

	var dockerImage string
	var buildID string

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
		dockerImage = result.ImageName
		buildID = result.BuildId

	} else if req.GetDockerImage() != "" {
		dockerImage = req.GetDockerImage()
		w.logger.Info("using prebuilt docker image",
			"deployment_id", deployment.ID,
			"image", dockerImage)
	} else {
		return nil, fmt.Errorf("either build_context_path or docker_image must be specified")
	}

	if err = w.updateDeploymentStatus(ctx, deployment.ID, db.DeploymentsStatusDeploying); err != nil {
		return nil, err
	}

	var encryptedSecretsBlob []byte

	if len(deployment.SecretsConfig) > 0 && w.vault != nil {
		var secretsConfig ctrlv1.SecretsConfig
		if err = protojson.Unmarshal(deployment.SecretsConfig, &secretsConfig); err != nil {
			return nil, fmt.Errorf("invalid secrets config: %w", err)
		}

		encrypted, encryptErr := w.vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: deployment.EnvironmentID,
			Data:    string(deployment.SecretsConfig),
		})
		if encryptErr != nil {
			return nil, fmt.Errorf("failed to encrypt secrets blob: %w", encryptErr)
		}
		encryptedSecretsBlob = []byte(encrypted.GetEncrypted())
	}

	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		req := &kranev1.DeploymentRequest{
			Namespace:            hardcodedNamespace,
			DeploymentId:         deployment.ID,
			Image:                dockerImage,
			Replicas:             1,
			CpuMillicores:        512,
			MemorySizeMib:        512,
			EnvironmentSlug:      environment.Slug,
			EncryptedSecretsBlob: encryptedSecretsBlob,
			EnvironmentId:        deployment.EnvironmentID,
			BuildId:              buildID,
		}
		_, err = w.krane.CreateDeployment(stepCtx, connect.NewRequest(&kranev1.CreateDeploymentRequest{
			Deployment: req,
		}))
		if err != nil {
			return restate.Void{}, fmt.Errorf("krane CreateDeployment failed for image %s: %w", dockerImage, err)
		}

		return restate.Void{}, nil
	}, restate.WithName("creating deployment in krane"))
	if err != nil {
		return nil, err
	}

	w.logger.Info("deployment created", "deployment_id", deployment.ID)

	createdInstances, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]*kranev1.Instance, error) {
		storedInstances := map[string]db.InstancesStatus{}

		for i := range 300 {
			time.Sleep(time.Second)
			if i%10 == 0 {
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

				var status db.InstancesStatus
				switch instance.GetStatus() {
				case kranev1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING:
					status = db.InstancesStatusProvisioning
				case kranev1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING:
					status = db.InstancesStatusRunning

				case kranev1.DeploymentStatus_DEPLOYMENT_STATUS_TERMINATING:
					status = db.InstancesStatusStopping
				case kranev1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED:
					status = db.InstancesStatusAllocated
				}

				w.logger.Info("upserting instance to database",
					"instance_id", instance.GetId(),
					"deployment_id", deployment.ID,
					"address", instance.GetAddress(),
					"status", status)

				previousStatus, ok := storedInstances[instance.GetId()]
				if !ok || previousStatus != status {
					if err = db.Query.UpsertInstance(stepCtx, w.db.RW(), db.UpsertInstanceParams{
						ID:            instance.GetId(),
						DeploymentID:  deployment.ID,
						WorkspaceID:   deployment.WorkspaceID,
						ProjectID:     deployment.ProjectID,
						Region:        "TODO",
						Address:       instance.GetAddress(),
						CpuMillicores: 1000,
						MemoryMb:      1024,
						Status:        status,
					}); err != nil {
						return nil, fmt.Errorf("failed to upsert instance %s: %w", instance.GetId(), err)
					}
					w.logger.Info("successfully upserted instance to database", "instance_id", instance.GetId())
					storedInstances[instance.GetId()] = status

				}

				if allReady {
					return resp.Msg.GetInstances(), nil
				}
			}
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

			var specBytes []byte
			specBytes, err = io.ReadAll(resp.Body)
			if err != nil {
				w.logger.Warn("failed to read OpenAPI spec response", "error", err, "host_addr", instance.GetAddress(), "deployment_id", deployment.ID)
				continue
			}

			w.logger.Info("openapi spec scraped successfully", "host_addr", instance.GetAddress(), "deployment_id", deployment.ID, "spec_size", len(specBytes))
			return base64.StdEncoding.EncodeToString(specBytes), nil
		}
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

	allHostnames := buildDomains(
		workspace.Slug,
		project.Slug,
		environment.Slug,
		deployment.GitCommitSha.String,
		deployment.GitBranch.String,
		w.defaultDomain,
		ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
	)

	existingRouteIDs := make([]string, 0)

	for _, hostname := range allHostnames {
		ingressRouteID, getIngressRouteErr := restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
			return db.TxWithResult(stepCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) (string, error) {
				found, err := db.Query.FindIngressRouteByHostname(txCtx, tx, hostname.domain)
				if err != nil {
					if db.IsNotFound(err) {
						err = db.Query.InsertIngressRoute(stepCtx, tx, db.InsertIngressRouteParams{
							ID:            uid.New("todo"),
							ProjectID:     project.ID,
							DeploymentID:  deployment.ID,
							EnvironmentID: deployment.EnvironmentID,
							Hostname:      hostname.domain,
							Sticky:        hostname.sticky,
							CreatedAt:     time.Now().UnixMilli(),
							UpdatedAt:     sql.NullInt64{Valid: false, Int64: 0},
						})
						return "", err

					}
					return "", err
				}
				return found.ID, nil

			})
		})
		if getIngressRouteErr != nil {
			return nil, getIngressRouteErr
		}
		if ingressRouteID != "" {
			existingRouteIDs = append(existingRouteIDs, ingressRouteID)
		}
	}

	_, err = hydrav1.NewRoutingServiceClient(ctx, project.ID).
		AssignIngressRoutes().Request(&hydrav1.AssignIngressRoutesRequest{
		DeploymentId:    deployment.ID,
		IngressRouteIds: existingRouteIDs,
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
