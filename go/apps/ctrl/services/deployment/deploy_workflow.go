package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metald/v1/metaldv1connect"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"google.golang.org/protobuf/proto"
)

// DeployWorkflow orchestrates the complete build and deployment process using Hydra
type DeployWorkflow struct {
	db           db.Database
	partitionDB  db.Database
	logger       logging.Logger
	metaldClient metaldv1connect.VmServiceClient
}

// NewDeployWorkflow creates a new deploy workflow instance
func NewDeployWorkflow(database db.Database, partitionDB db.Database,
	logger logging.Logger, metaldClient metaldv1connect.VmServiceClient,
) *DeployWorkflow {
	return &DeployWorkflow{
		db:           database,
		partitionDB:  partitionDB,
		logger:       logger,
		metaldClient: metaldClient,
	}
}

// Name returns the workflow name for registration
func (w *DeployWorkflow) Name() string {
	return "deployment"
}

// DeployRequest defines the input for the deploy workflow
type DeployRequest struct {
	WorkspaceID  string `json:"workspace_id"`
	ProjectID    string `json:"project_id"`
	KeyspaceID   string `json:"keyspace_id"`
	DeploymentID string `json:"deployment_id"`
	DockerImage  string `json:"docker_image"`
	Hostname     string `json:"hostname"`
}

// DeploymentResult holds the deployment outcome
type DeploymentResult struct {
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
}

// Run executes the complete build and deployment workflow
func (w *DeployWorkflow) Run(ctx hydra.WorkflowContext, req *DeployRequest) error {
	w.logger.Info("starting deployment workflow",
		"execution_id", ctx.ExecutionID(),
		"deployment_id", req.DeploymentID,
		"docker_image", req.DockerImage,
		"workspace_id", req.WorkspaceID,
		"project_id", req.ProjectID,
		"hostname", req.Hostname)

	// Step 2: Log deployment pending
	err := hydra.StepVoid(ctx, "log-deployment-pending", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			WorkspaceID:  req.WorkspaceID,
			ProjectID:    req.ProjectID,
			DeploymentID: req.DeploymentID,
			Status:       "pending",
			Message:      "Deployment queued and ready to start",
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log deployment pending", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 4: Update version status to building
	_, err = hydra.Step(ctx, "update-version-building", func(stepCtx context.Context) (*struct{}, error) {
		w.logger.Info("updating deployment status to building", "deployment_id", req.DeploymentID)
		updateErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        req.DeploymentID,
			Status:    db.DeploymentsStatusBuilding,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update version status to building: %w", updateErr)
		}
		w.logger.Info("deployment status updated to building", "deployment_id", req.DeploymentID)
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to initialize build", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	deployment, err := hydra.Step(ctx, "metald-create-deployment", func(stepCtx context.Context) (*metaldv1.CreateDeploymentResponse, error) {
		w.logger.Info("creating deployment", "deployment_id", req.DeploymentID, "docker_image", req.DockerImage, "workspace_id", req.WorkspaceID, "project_id", req.ProjectID)

		// Call metald CreateDeployment
		resp, err := w.metaldClient.CreateDeployment(stepCtx, connect.NewRequest(&metaldv1.CreateDeploymentRequest{
			Deployment: &metaldv1.DeploymentRequest{
				DeploymentId:  req.DeploymentID,
				Image:         req.DockerImage,
				VmCount:       1,
				Cpu:           1,
				MemorySizeMib: 1024,
			},
		}))
		if err != nil {
			w.logger.Error("metald CreateDeployment call failed", "error", err, "docker_image", req.DockerImage)
			return nil, fmt.Errorf("failed to create deployment: %w", err)
		}

		return resp.Msg, nil
	})
	if err != nil {
		w.logger.Error("Deployment  failed", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	w.logger.Info("Deployment created", "vm_ids", deployment.GetVmIds())

	// Step 12: Update version status to deploying
	_, err = hydra.Step(ctx, "update-version-deploying", func(stepCtx context.Context) (*struct{}, error) {
		w.logger.Info("starting deployment", "deployment_id", req.DeploymentID)

		deployingErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        req.DeploymentID,
			Status:    db.DeploymentsStatusDeploying,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if deployingErr != nil {
			return nil, fmt.Errorf("failed to update version status to deploying: %w", deployingErr)
		}
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to update version status to deploying", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	createdVMs, err := hydra.Step(ctx, "polling deployment prepare", func(stepCtx context.Context) ([]*metaldv1.GetDeploymentResponse_Vm, error) {

		instances := make(map[string]*metaldv1.GetDeploymentResponse_Vm)

		for i := range 300 {
			time.Sleep(time.Second)

			w.logger.Info("Polling deployment", "i", i)
			resp, err := w.metaldClient.GetDeployment(stepCtx, connect.NewRequest(&metaldv1.GetDeploymentRequest{
				DeploymentId: req.DeploymentID,
			}))
			if err != nil {
				w.logger.Error("metald GetDeployment call failed", "error", err, "deployment_id", req.DeploymentID)
				return nil, fmt.Errorf("failed to get deployment: %w", err)

			}

			allReady := true
			for _, instance := range resp.Msg.GetVms() {
				known, ok := instances[instance.Id]
				if !ok || known.State != instance.State {
					if err := partitiondb.Query.UpsertVM(stepCtx, w.partitionDB.RW(), partitiondb.UpsertVMParams{
						ID:            instance.Id,
						DeploymentID:  req.DeploymentID,
						Address:       sql.NullString{Valid: true, String: fmt.Sprintf("%s:%d", instance.Host, instance.Port)},
						CpuMillicores: 1000,                           // TODO derive from spec
						MemoryMb:      1024,                           // TODO derive from spec
						Status:        partitiondb.VmsStatusAllocated, // TODO
					}); err != nil {
						w.logger.Error("failed to upsert VM", "error", err, "vm_id", instance.Id)
						return nil, fmt.Errorf("failed to upsert VM %s: %w", instance.Id, err)
					}
					instances[instance.Id] = instance
					if instance.State != metaldv1.VmState_VM_STATE_RUNNING {
						allReady = false
						w.logger.Debug("VM not ready", "vm_id", instance.Id, "state", instance.State)
					}
				}
			}

			if allReady {
				return resp.Msg.GetVms(), nil
			}

		}
		return nil, fmt.Errorf("deployment never became ready")
	})
	if err != nil {
		w.logger.Error("Polling deployment prepare failed", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	err = hydra.StepVoid(ctx, "create-gateway-config", func(stepCtx context.Context) error {
		// Only create gateway config if hostname is provided
		if req.Hostname == "" {
			w.logger.Info("no hostname provided, skipping gateway configuration")
			return nil
		}

		w.logger.Info("creating gateway configuration", "hostname", req.Hostname, "deployment_id", req.DeploymentID)

		// Validate partition DB connection
		if w.partitionDB == nil {
			w.logger.Error("CRITICAL: partition database not initialized for gateway config")
			return fmt.Errorf("partition database not initialized for gateway config")
		}

		// Create VM protobuf objects for gateway config

		gatewayConfig := &partitionv1.GatewayConfig{
			Deployment: &partitionv1.Deployment{
				Id:        req.DeploymentID,
				IsEnabled: true,
			},
			Vms: make([]*partitionv1.VM, len(createdVMs)),
		}
		for i, vm := range createdVMs {
			gatewayConfig.Vms[i] = &partitionv1.VM{
				Id: vm.Id,
			}
		}

		// Only add AuthConfig if we have a KeyspaceID
		if req.KeyspaceID != "" {
			gatewayConfig.AuthConfig = &partitionv1.AuthConfig{

				KeyAuthId: req.KeyspaceID,
			}
		}

		// Marshal protobuf to bytes
		configBytes, err := proto.Marshal(gatewayConfig)
		if err != nil {
			w.logger.Error("failed to marshal gateway config", "error", err)
			return fmt.Errorf("failed to marshal gateway config: %w", err)
		}

		// Insert gateway config into partition database
		params := partitiondb.UpsertGatewayParams{
			Hostname: req.Hostname,
			Config:   configBytes,
		}

		if err := partitiondb.Query.UpsertGateway(stepCtx, w.partitionDB.RW(), params); err != nil {
			w.logger.Error("failed to upsert gateway config", "error", err, "hostname", req.Hostname)
			return fmt.Errorf("failed to upsert gateway config: %w", err)
		}
		w.logger.Info("gateway configuration created successfully", "hostname", req.Hostname)
		return nil
	})
	if err != nil {
		w.logger.Error("failed to create gateway configuration", "error", err, "hostname", req.Hostname)
		return err
	}

	// Step 19: Assign domains (create route entries)
	assignedHostnames, err := hydra.Step(ctx, "assign-domains", func(stepCtx context.Context) ([]string, error) {
		w.logger.Info("assigning domains to version", "deployment_id", req.DeploymentID)

		var hostnames []string

		// Generate primary hostname for this deployment
		// Use Git info for hostname generation
		gitInfo := git.GetInfo()
		branch := "main"               // Default branch
		identifier := req.DeploymentID // Use full version ID as identifier

		if gitInfo.IsRepo {
			if gitInfo.Branch != "" {
				branch = gitInfo.Branch
			}
			if gitInfo.CommitSHA != "" {
				identifier = gitInfo.CommitSHA
			}
		}

		// Generate hostnames: branch-identifier-workspace.unkey.app
		// Replace underscores with dashes for valid hostname format
		cleanIdentifier := strings.ReplaceAll(identifier, "_", "-")
		primaryHostname := fmt.Sprintf("%s-%s-%s.unkey.app", branch, cleanIdentifier, req.WorkspaceID)

		// Create domain entry for primary hostname
		domainID := uid.New("domain")
		insertErr := db.Query.InsertDomain(stepCtx, w.db.RW(), db.InsertDomainParams{
			ID:           domainID,
			WorkspaceID:  req.WorkspaceID,
			ProjectID:    sql.NullString{Valid: true, String: req.ProjectID},
			Domain:       primaryHostname,
			DeploymentID: sql.NullString{Valid: true, String: req.DeploymentID},
			CreatedAt:    time.Now().UnixMilli(),
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			Type:         db.DomainsTypeCustom,
		})
		if insertErr != nil {
			w.logger.Error("failed to create domain", "error", insertErr, "domain", primaryHostname, "deployment_id", req.DeploymentID)
			return nil, fmt.Errorf("failed to create route for hostname %s: %w", primaryHostname, insertErr)
		}

		hostnames = append(hostnames, primaryHostname)
		w.logger.Info("primary domain assigned successfully", "hostname", primaryHostname, "deployment_id", req.DeploymentID, "domain_id", domainID)

		return hostnames, nil
	})
	if err != nil {
		w.logger.Error("failed to assign domains", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 20: Log assigning domains
	err = hydra.StepVoid(ctx, "log-assigning-domains", func(stepCtx context.Context) error {
		var message string
		if len(assignedHostnames) > 0 {
			message = fmt.Sprintf("Assigned hostnames: %s", strings.Join(assignedHostnames, ", "))
		} else {
			message = "Domain assignment completed"
		}
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       db.DeploymentStepsStatusAssigningDomains,
			Message:      message,
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log assigning domains", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 21: Update deployment status to active
	_, err = hydra.Step(ctx, "update-deployment-ready", func(stepCtx context.Context) (*DeploymentResult, error) {
		completionTime := time.Now().UnixMilli()
		w.logger.Info("updating deployment status to ready", "deployment_id", req.DeploymentID, "completion_time", completionTime)
		activeErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        req.DeploymentID,
			Status:    db.DeploymentsStatusReady,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: completionTime},
		})
		if activeErr != nil {
			w.logger.Error("failed to update deployment status to active", "error", activeErr, "deployment_id", req.DeploymentID)
			return nil, fmt.Errorf("failed to update deployment status to active: %w", activeErr)
		}

		w.logger.Info("deployment complete", "deployment_id", req.DeploymentID, "status", "active")

		return &DeploymentResult{
			DeploymentID: req.DeploymentID,
			Status:       "active",
		}, nil
	})
	if err != nil {
		w.logger.Error("deployment failed", "error", err, "deployment_id", req.DeploymentID)
		return err
	}
	/*

		// Step 23: Scrape OpenAPI spec from container (using host port mapping)
		openapiSpec, err := hydra.Step(ctx, "scrape-openapi-spec", func(stepCtx context.Context) (string, error) {

			// Find the port mapping for container port 8080
			var hostPort int32
			for _, portMapping := range vmInfo.NetworkInfo.PortMappings {
				if portMapping.ContainerPort == 8080 {
					hostPort = portMapping.HostPort
					break
				}
			}

			if hostPort == 0 {
				w.logger.Warn("no host port mapping found for container port 8080", "deployment_id", req.DeploymentID)
				return "", nil
			}

			// Try multiple host addresses to reach the Docker host
			hostAddresses := []string{
				"host.docker.internal",    // Docker Desktop (Windows/Mac) and some Linux setups
				"gateway.docker.internal", // Docker gateway
				"172.17.0.1",              // Default Docker bridge gateway
				"172.18.0.1",              // Alternative Docker bridge
			}

			client := &http.Client{Timeout: 10 * time.Second}

			for _, hostAddr := range hostAddresses {
				openapiURL := fmt.Sprintf("http://%s:%d/openapi.yaml", hostAddr, hostPort)
				w.logger.Info("trying to scrape OpenAPI spec", "url", openapiURL, "host_port", hostPort, "deployment_id", req.DeploymentID)

				resp, err := client.Get(openapiURL)
				if err != nil {
					w.logger.Warn("OpenAPI scraping failed for host address", "error", err, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					w.logger.Warn("OpenAPI endpoint returned non-200 status", "status", resp.StatusCode, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
					continue
				}

				// Read the OpenAPI spec
				specBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					w.logger.Warn("failed to read OpenAPI spec response", "error", err, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
					continue
				}

				w.logger.Info("OpenAPI spec scraped successfully", "host_addr", hostAddr, "deployment_id", req.DeploymentID, "spec_size", len(specBytes))
				return string(specBytes), nil
			}

			return "", fmt.Errorf("failed to scrape OpenAPI spec from all host addresses: %v", hostAddresses)
		})
		if err != nil {
			w.logger.Error("failed to scrape OpenAPI spec", "error", err, "deployment_id", req.DeploymentID)
			return err
		}

		// Step 24: Update gateway config with OpenAPI spec
		err = hydra.StepVoid(ctx, "update-gateway-config-openapi", func(stepCtx context.Context) error {
			// Only update if we have both hostname and OpenAPI spec
			if req.Hostname == "" || openapiSpec == "" {
				w.logger.Info("skipping gateway config OpenAPI update",
					"has_hostname", req.Hostname != "",
					"has_openapi_spec", openapiSpec != "",
					"deployment_id", req.DeploymentID)
				return nil
			}

			w.logger.Info("updating gateway config with OpenAPI spec", "hostname", req.Hostname, "deployment_id", req.DeploymentID, "spec_size", len(openapiSpec))

			// Fetch existing gateway config
			existingConfig, err := partitiondb.Query.FindGatewayByHostname(stepCtx, w.partitionDB.RO(), req.Hostname)
			if err != nil {
				w.logger.Error("failed to fetch existing gateway config", "error", err, "hostname", req.Hostname)
				return fmt.Errorf("failed to fetch existing gateway config: %w", err)
			}

			// Unmarshal existing config
			var gatewayConfig partitionv1.GatewayConfig
			if err := proto.Unmarshal(existingConfig.Config, &gatewayConfig); err != nil {
				w.logger.Error("failed to unmarshal existing gateway config", "error", err, "hostname", req.Hostname)
				return fmt.Errorf("failed to unmarshal existing gateway config: %w", err)
			}

			// Add or update ValidationConfig with OpenAPI spec
			if gatewayConfig.ValidationConfig == nil {
				gatewayConfig.ValidationConfig = &partitionv1.ValidationConfig{}
			}
			gatewayConfig.ValidationConfig.OpenapiSpec = openapiSpec

			// Marshal updated config
			configBytes, err := proto.Marshal(&gatewayConfig)
			if err != nil {
				w.logger.Error("failed to marshal updated gateway config", "error", err)
				return fmt.Errorf("failed to marshal updated gateway config: %w", err)
			}

			// Update gateway config in partition database
			params := partitiondb.UpsertGatewayParams{
				Hostname: req.Hostname,
				Config:   configBytes,
			}

			if err := partitiondb.Query.UpsertGateway(stepCtx, w.partitionDB.RW(), params); err != nil {
				w.logger.Error("failed to update gateway config with OpenAPI spec", "error", err, "hostname", req.Hostname)
				return fmt.Errorf("failed to update gateway config with OpenAPI spec: %w", err)
			}

			w.logger.Info("gateway config updated with OpenAPI spec successfully", "hostname", req.Hostname, "deployment_id", req.DeploymentID)
			return nil
		})
		if err != nil {
			w.logger.Error("failed to update gateway config with OpenAPI spec", "error", err, "deployment_id", req.DeploymentID)
			// Don't fail the deployment for this
		}

		// Step 25: Store OpenAPI spec in database
		err = hydra.StepVoid(ctx, "store-openapi-spec", func(stepCtx context.Context) error {
			if openapiSpec == "" {
				w.logger.Info("no OpenAPI spec to store", "deployment_id", req.DeploymentID)
				return nil
			}

			// Store in database
			err := db.Query.UpdateDeploymentOpenapiSpec(stepCtx, w.db.RW(), db.UpdateDeploymentOpenapiSpecParams{
				ID:          req.DeploymentID,
				OpenapiSpec: sql.NullString{String: openapiSpec, Valid: true},
				UpdatedAt:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				w.logger.Warn("failed to store OpenAPI spec in database", "error", err, "deployment_id", req.DeploymentID)
				return nil // Don't fail the deployment
			}

			w.logger.Info("OpenAPI spec stored in database successfully", "deployment_id", req.DeploymentID, "spec_size", len(openapiSpec))
			return nil
		})
		if err != nil {
			w.logger.Error("failed to store OpenAPI spec", "error", err, "deployment_id", req.DeploymentID)
			return err
		}

	*/
	// Step 26: Log completed
	err = hydra.StepVoid(ctx, "log-completed", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "completed",
			Message:      "Deployment completed successfully",
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log completed", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	w.logger.Info("deployment workflow stage completed successfully", "deployment_id", req.DeploymentID)

	w.logger.Info("deployment workflow completed",
		"execution_id", ctx.ExecutionID(),
		"deployment_id", req.DeploymentID,
		"status", "succeeded",
		"workspace_id", req.WorkspaceID,
		"project_id", req.ProjectID,
		"docker_image", req.DockerImage,
		"hostname", req.Hostname)

	return nil
}
