package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metald/v1/metaldv1connect"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
)

// DeployWorkflow orchestrates the complete build and deployment process using Hydra
type DeployWorkflow struct {
	db                db.Database
	partitionDB       db.Database
	logger            logging.Logger
	deploymentBackend DeploymentBackend
	defaultDomain     string
}

type DeployWorkflowConfig struct {
	Logger          logging.Logger
	DB              db.Database
	PartitionDB     db.Database
	MetalD          metaldv1connect.VmServiceClient
	MetaldBackend   string
	DefaultDomain   string
	IsRunningDocker bool
}

// NewDeployWorkflow creates a new deploy workflow instance
func NewDeployWorkflow(cfg DeployWorkflowConfig) *DeployWorkflow {
	// Create the appropriate deployment backend
	deploymentBackend, err := NewDeploymentBackend(cfg.MetalD, cfg.MetaldBackend, cfg.Logger, cfg.IsRunningDocker)
	if err != nil {
		// Log error but continue - workflow will fail when trying to use the backend
		cfg.Logger.Error("failed to initialize deployment backend",
			"error", err,
			"fallback", cfg.MetaldBackend)
	}

	return &DeployWorkflow{
		db:                cfg.DB,
		partitionDB:       cfg.PartitionDB,
		logger:            cfg.Logger,
		deploymentBackend: deploymentBackend,
		defaultDomain:     cfg.DefaultDomain,
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
	)
	// Log deployment pending
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
		return err
	}

	// Update version status to building
	_, err = hydra.Step(ctx, "update-version-building", func(stepCtx context.Context) (*struct{}, error) {
		updateErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        req.DeploymentID,
			Status:    db.DeploymentsStatusBuilding,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update version status to building: %w", updateErr)
		}
		return &struct{}{}, nil
	})
	if err != nil {
		return err
	}

	deployment, err := hydra.Step(ctx, "create-deployment", func(stepCtx context.Context) (*metaldv1.CreateDeploymentResponse, error) {
		if w.deploymentBackend == nil {
			return nil, fmt.Errorf("deployment backend not initialized")
		}

		// Create deployment request
		deploymentReq := &metaldv1.CreateDeploymentRequest{
			Deployment: &metaldv1.DeploymentRequest{
				DeploymentId:  req.DeploymentID,
				Image:         req.DockerImage,
				VmCount:       1,
				Cpu:           1,
				MemorySizeMib: 1024,
			},
		}

		resp, err := w.deploymentBackend.CreateDeployment(stepCtx, deploymentReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create deployment for image %s: %w", req.DockerImage, err)
		}

		return resp, nil
	})
	if err != nil {
		return err
	}

	w.logger.Info("deployment created", "deployment_id", req.DeploymentID, "vm_count", len(deployment.GetVmIds()))

	// Update version status to deploying
	_, err = hydra.Step(ctx, "update-version-deploying", func(stepCtx context.Context) (*struct{}, error) {
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
		return err
	}

	createdVMs, err := hydra.Step(ctx, "polling deployment prepare", func(stepCtx context.Context) ([]*metaldv1.GetDeploymentResponse_Vm, error) {
		instances := make(map[string]*metaldv1.GetDeploymentResponse_Vm)

		for i := range 300 {
			time.Sleep(time.Second)
			if i%10 == 0 { // Log every 10 seconds instead of every second
				w.logger.Info("polling deployment status", "deployment_id", req.DeploymentID, "iteration", i)
			}

			if w.deploymentBackend == nil {
				return nil, fmt.Errorf("deployment backend not initialized")
			}

			vms, err := w.deploymentBackend.GetDeployment(stepCtx, req.DeploymentID)
			if err != nil {
				return nil, fmt.Errorf("failed to get deployment %s: %w", req.DeploymentID, err)
			}

			allReady := true
			for _, instance := range vms {
				known, ok := instances[instance.Id]
				if !ok || known.State != instance.State {
					upsertParams := partitiondb.UpsertVMParams{
						ID:            instance.Id,
						DeploymentID:  req.DeploymentID,
						Address:       sql.NullString{Valid: true, String: fmt.Sprintf("%s:%d", instance.Host, instance.Port)},
						CpuMillicores: 1000,                         // TODO derive from spec
						MemoryMb:      1024,                         // TODO derive from spec
						Status:        partitiondb.VmsStatusRunning, // TODO
					}

					w.logger.Info("upserting VM to database",
						"vm_id", instance.Id,
						"deployment_id", req.DeploymentID,
						"address", fmt.Sprintf("%s:%d", instance.Host, instance.Port),
						"status", "running")

					if err := partitiondb.Query.UpsertVM(stepCtx, w.partitionDB.RW(), upsertParams); err != nil {
						return nil, fmt.Errorf("failed to upsert VM %s: %w", instance.Id, err)
					}

					w.logger.Info("successfully upserted VM to database", "vm_id", instance.Id)

					instances[instance.Id] = instance
				}

				w.logger.Debug("checking VM readiness", "vm_id", instance.Id, "state", instance.State.String())
				if instance.State != metaldv1.VmState_VM_STATE_RUNNING {
					allReady = false
					w.logger.Debug("vm not ready", "vm_id", instance.Id, "state", instance.State.String())
				}
			}

			if allReady {
				w.logger.Info("all VMs ready, deployment complete",
					"deployment_id", req.DeploymentID,
					"vm_count", len(vms),
					"vms", func() []string {
						var ids []string
						for _, vm := range vms {
							ids = append(ids, vm.Id)
						}
						return ids
					}())

				return vms, nil
			}

			w.logger.Debug("deployment not ready yet, continuing to poll",
				"deployment_id", req.DeploymentID,
				"iteration", i,
				"all_ready", allReady)
		}

		return nil, fmt.Errorf("deployment never became ready")
	})
	if err != nil {
		return err
	}

	allDomains, err := hydra.Step(ctx, "generate-all-domains", func(stepCtx context.Context) ([]string, error) {
		var domains []string

		// Generate auto-generated hostname for this deployment
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

		// Generate primary hostname: branch-identifier-workspace.domain
		cleanIdentifier := strings.ToLower(strings.ReplaceAll(identifier, "_", "-"))
		cleanBranch := strings.ToLower(strings.ReplaceAll(branch, "/", "-"))
		cleanWorkspaceID := strings.ToLower(req.WorkspaceID)
		autoGeneratedHostname := fmt.Sprintf("%s-%s-%s.%s", cleanBranch, cleanIdentifier, cleanWorkspaceID, w.defaultDomain)
		domains = append(domains, autoGeneratedHostname)

		w.logger.Info("generated all domains",
			"deployment_id", req.DeploymentID,
			"total_domains", len(domains),
			"domains", domains,
		)

		return domains, nil
	})
	if err != nil {
		return err
	}

	// Create database entries for all domains
	err = hydra.StepVoid(ctx, "create-domain-entries", func(stepCtx context.Context) error {
		// Prepare bulk insert parameters
		domainParams := make([]db.InsertDomainParams, 0, len(allDomains))
		currentTime := time.Now().UnixMilli()

		for _, domain := range allDomains {
			domainID := uid.New("domain")
			domainParams = append(domainParams, db.InsertDomainParams{
				ID:           domainID,
				WorkspaceID:  req.WorkspaceID,
				ProjectID:    sql.NullString{Valid: true, String: req.ProjectID},
				Domain:       domain,
				DeploymentID: sql.NullString{Valid: true, String: req.DeploymentID},
				CreatedAt:    currentTime,
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: currentTime},
				Type:         db.DomainsTypeCustom,
			})
		}

		// Perform bulk insert
		if err := db.BulkQuery.InsertDomains(stepCtx, w.db.RW(), domainParams); err != nil {
			return fmt.Errorf("failed to create %d domain entries for deployment %s: %w", len(allDomains), req.DeploymentID, err)
		}

		w.logger.Info("domain entries created in bulk", "deployment_id", req.DeploymentID, "domain_count", len(allDomains))

		return nil
	})
	if err != nil {
		return err
	}

	// Create gateway configs for all domains in bulk (except local ones)
	err = hydra.StepVoid(ctx, "create-gateway-configs-bulk", func(stepCtx context.Context) error {
		// Prepare gateway configs for all non-local domains
		var gatewayParams []partitiondb.UpsertGatewayParams
		var skippedDomains []string

		for _, domain := range allDomains {
			if isLocalHostname(domain, w.defaultDomain) {
				skippedDomains = append(skippedDomains, domain)
				continue
			}

			// Create gateway config for this domain
			gatewayConfig, err := w.createGatewayConfig(req.DeploymentID, req.KeyspaceID, createdVMs)
			if err != nil {
				w.logger.Error("failed to create gateway config for domain",
					"domain", domain,
					"error", err,
					"deployment_id", req.DeploymentID)
				// Continue with other domains rather than failing the entire deployment
				continue
			}

			// Marshal protobuf to bytes
			configBytes, err := protojson.Marshal(gatewayConfig)
			if err != nil {
				w.logger.Error("failed to marshal gateway config", "error", err, "domain", domain)
				continue
			}

			gatewayParams = append(gatewayParams, partitiondb.UpsertGatewayParams{
				WorkspaceID: req.WorkspaceID,
				Hostname:    domain,
				Config:      configBytes,
			})
		}

		// Perform bulk upsert for all gateway configs
		if len(gatewayParams) > 0 {
			if err := partitiondb.BulkQuery.UpsertGateway(stepCtx, w.partitionDB.RW(), gatewayParams); err != nil {
				return fmt.Errorf("failed to upsert %d gateway configs for deployment %s: %w", len(gatewayParams), req.DeploymentID, err)
			}
		}

		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       db.DeploymentStepsStatusAssigningDomains,
			Message:      fmt.Sprintf("Created %d gateway configs for %d domains (skipped %d local domains)", len(gatewayParams), len(allDomains), len(skippedDomains)),
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		return err
	}

	// Update deployment status to ready
	_, err = hydra.Step(ctx, "update-deployment-ready", func(stepCtx context.Context) (*DeploymentResult, error) {
		w.logger.Info("updating deployment status to ready", "deployment_id", req.DeploymentID)
		completionTime := time.Now().UnixMilli()
		activeErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        req.DeploymentID,
			Status:    db.DeploymentsStatusReady,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: completionTime},
		})
		if activeErr != nil {
			return nil, fmt.Errorf("failed to update deployment %s status to ready: %w", req.DeploymentID, activeErr)
		}
		w.logger.Info("deployment status updated to ready", "deployment_id", req.DeploymentID)

		// TODO: This section will be removed in the future in favor of "Promote to Production"
		err = db.Query.UpdateProjectActiveDeploymentId(stepCtx, w.db.RW(), db.UpdateProjectActiveDeploymentIdParams{
			ID:                 req.ProjectID,
			ActiveDeploymentID: sql.NullString{Valid: true, String: req.DeploymentID},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update project %s active deployment ID to %s: %w", req.ProjectID, req.DeploymentID, err)
		}

		return &DeploymentResult{
			DeploymentID: req.DeploymentID,
			Status:       "ready",
		}, nil
	})
	if err != nil {
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
					w.logger.Warn("openapi scraping failed for host address", "error", err, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					w.logger.Warn("openapi endpoint returned non-200 status", "status", resp.StatusCode, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
					continue
				}

				// Read the OpenAPI spec
				specBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					w.logger.Warn("failed to read OpenAPI spec response", "error", err, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
					continue
				}

				w.logger.Info("openapi spec scraped successfully", "host_addr", hostAddr, "deployment_id", req.DeploymentID, "spec_size", len(specBytes))
				return string(specBytes), nil
			}

			return "", fmt.Errorf("failed to scrape OpenAPI spec from all host addresses: %v", hostAddresses)
		})
		if err != nil {
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
				return fmt.Errorf("failed to fetch existing gateway config for %s: %w", req.Hostname, err)
			}

			// Unmarshal existing config
			// IMPORTANT: Gateway configs are stored as JSON in the database for compatibility with the gateway service
			var gatewayConfig partitionv1.GatewayConfig
			if err := protojson.Unmarshal(existingConfig.Config, &gatewayConfig); err != nil {
				return fmt.Errorf("failed to unmarshal existing gateway config for %s: %w", req.Hostname, err)
			}

			// Add or update ValidationConfig with OpenAPI spec
			if gatewayConfig.ValidationConfig == nil {
				gatewayConfig.ValidationConfig = &partitionv1.ValidationConfig{}
			}
			gatewayConfig.ValidationConfig.OpenapiSpec = openapiSpec

			// Marshal updated config
			// Gateway configs must be stored as JSON for compatibility with the gateway service
			configBytes, err := protojson.Marshal(&gatewayConfig)
			if err != nil {
				return fmt.Errorf("failed to marshal updated gateway config: %w", err)
			}

			// Update gateway config in partition database
			params := partitiondb.UpsertGatewayParams{
				Hostname: req.Hostname,
				Config:   configBytes,
			}

			if err := partitiondb.Query.UpsertGateway(stepCtx, w.partitionDB.RW(), params); err != nil {
				return fmt.Errorf("failed to update gateway config with OpenAPI spec for %s: %w", req.Hostname, err)
			}

			w.logger.Info("gateway config updated with OpenAPI spec successfully", "hostname", req.Hostname, "deployment_id", req.DeploymentID)
			return nil
		})
		if err != nil {
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

			w.logger.Info("openapi spec stored in database successfully", "deployment_id", req.DeploymentID, "spec_size", len(openapiSpec))
			return nil
		})
		if err != nil {
			return err
		}

	*/
	// Log deployment completed
	err = hydra.StepVoid(ctx, "log-completed", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "completed",
			Message:      "Deployment completed successfully",
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		return err
	}

	w.logger.Info("deployment workflow completed",
		"deployment_id", req.DeploymentID,
		"status", "succeeded",
		"domains", len(allDomains))

	return nil
}

// createGatewayConfig creates a gateway configuration protobuf object
//
// ENCODING POLICY FOR GATEWAY CONFIGS:
// Gateway configs are stored as JSON (using protojson.Marshal) for easier debugging
// and readability during development/demo. This makes it simpler to inspect and
// modify configs directly in the database.
// IMPORTANT: Always use protojson.Marshal for writes and protojson.Unmarshal for reads.
func (w *DeployWorkflow) createGatewayConfig(deploymentID, keyspaceID string, vms []*metaldv1.GetDeploymentResponse_Vm) (*partitionv1.GatewayConfig, error) {
	// Create VM protobuf objects for gateway config
	gatewayConfig := &partitionv1.GatewayConfig{
		Deployment: &partitionv1.Deployment{
			Id:        deploymentID,
			IsEnabled: true,
		},
		Vms: make([]*partitionv1.VM, len(vms)),
	}

	for i, vm := range vms {
		gatewayConfig.Vms[i] = &partitionv1.VM{
			Id: vm.Id,
		}
	}

	// Only add AuthConfig if we have a KeyspaceID
	if keyspaceID != "" {
		gatewayConfig.AuthConfig = &partitionv1.AuthConfig{
			KeyAuthId: keyspaceID,
		}
	}

	return gatewayConfig, nil
}

// isLocalHostname checks if a hostname should be skipped from gateway config creation
// Returns true for localhost/development domains that shouldn't get gateway configs
func isLocalHostname(hostname, defaultDomain string) bool {
	// Lowercase for case-insensitive comparison
	hostname = strings.ToLower(hostname)
	defaultDomain = strings.ToLower(defaultDomain)

	// Exact matches for common local hosts - these should be skipped
	if hostname == "localhost" || hostname == "127.0.0.1" {
		return true
	}

	// If hostname uses the default domain, it should NOT be skipped (return false)
	// This allows gateway configs to be created for the default domain
	if strings.HasSuffix(hostname, "."+defaultDomain) || hostname == defaultDomain {
		return false
	}

	// Check for local-only TLD suffixes - these should be skipped
	// Note: .dev is a real TLD owned by Google, so it's excluded
	localSuffixes := []string{
		".local",
		".test",
	}

	for _, suffix := range localSuffixes {
		if strings.HasSuffix(hostname, suffix) {
			return true
		}
	}

	return false
}
