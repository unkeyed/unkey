package deploy

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	workflowsv1 "github.com/unkeyed/unkey/go/gen/proto/workflows/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
)

const hardcodedNamespace = "unkey" // TODO change to workspace scope

// DeployWorkflow orchestrates the complete build and deployment process using Hydra
type Workflow struct {
	workflowsv1.UnimplementedDeployWorkflowsServer
	db            db.Database
	partitionDB   db.Database
	logger        logging.Logger
	krane         kranev1connect.DeploymentServiceClient
	defaultDomain string
}

var _ workflowsv1.DeployWorkflowsServer = (*Workflow)(nil)

type Config struct {
	Logger        logging.Logger
	DB            db.Database
	PartitionDB   db.Database
	Krane         kranev1connect.DeploymentServiceClient
	DefaultDomain string
}

// New creates a new deploy workflow instance
func New(cfg Config) *Workflow {
	return &Workflow{
		db:            cfg.DB,
		partitionDB:   cfg.PartitionDB,
		logger:        cfg.Logger,
		krane:         cfg.Krane,
		defaultDomain: cfg.DefaultDomain,
	}
}

// invocation := restateIngress.WorkflowSend[deploymentworkflow.DeployRequest](s.restate, "DeployWorkflow", deploymentID, "Run").Send(ctx, deployReq)

func (w *Workflow) Deploy(ctx restate.WorkflowSharedContext, req *workflowsv1.DeployRequest) (*workflowsv1.DeployResponse, error) {

	deployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindDeploymentByIdRow, error) {
		return db.Query.FindDeploymentById(stepCtx, w.db.RW(), req.DeploymentId)
	}, restate.WithName("finding deployment"))
	if err != nil {
		return nil, err
	}

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

	// Update version status to building
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		updateErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        deployment.ID,
			Status:    db.DeploymentsStatusBuilding,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			return restate.Void{}, fmt.Errorf("failed to update version status to building: %w", updateErr)
		}
		return restate.Void{}, nil
	}, restate.WithName("updating deployment status to building"))
	if err != nil {
		return nil, err
	}

	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		// Create deployment request

		_, err := w.krane.CreateDeployment(stepCtx, connect.NewRequest(&kranev1.CreateDeploymentRequest{

			Deployment: &kranev1.DeploymentRequest{
				Namespace:     hardcodedNamespace,
				DeploymentId:  deployment.ID,
				Image:         req.DockerImage,
				Replicas:      1,
				CpuMillicores: 512,
				MemorySizeMib: 512,
			},
		}))
		if err != nil {
			return restate.Void{}, fmt.Errorf("krane CreateDeployment failed for image %s: %w", req.DockerImage, err)
		}

		return restate.Void{}, nil
	}, restate.WithName("creating deployment in krane"))
	if err != nil {
		return nil, err
	}

	w.logger.Info("deployment created", "deployment_id", deployment.ID)

	// Update version status to deploying
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		deployingErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        deployment.ID,
			Status:    db.DeploymentsStatusDeploying,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if deployingErr != nil {
			return restate.Void{}, fmt.Errorf("failed to update version status to deploying: %w", deployingErr)
		}
		return restate.Void{}, nil
	}, restate.WithName("updating deployment status to deploying"))
	if err != nil {
		return nil, err
	}
	createdInstances, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]*kranev1.Instance, error) {
		// prevent updating the db unnecessarily

		for i := range 300 {
			time.Sleep(time.Second)
			if i%10 == 0 { // Log every 10 seconds instead of every second
				w.logger.Info("polling deployment status", "deployment_id", deployment.ID, "iteration", i)
			}

			resp, err := w.krane.GetDeployment(stepCtx, connect.NewRequest(&kranev1.GetDeploymentRequest{
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
				if instance.Status != kranev1.DeploymentStatus_DEPLOYMENT_STATUS_RUNNING {
					allReady = false
				}

				var status partitiondb.VmsStatus
				switch instance.Status {
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
					ID:            instance.Id,
					DeploymentID:  deployment.ID,
					Address:       sql.NullString{Valid: true, String: instance.Address},
					CpuMillicores: 1000,   // TODO derive from spec
					MemoryMb:      1024,   // TODO derive from spec
					Status:        status, // TODO
				}

				w.logger.Info("upserting VM to database",
					"vm_id", instance.Id,
					"deployment_id", deployment.ID,
					"address", instance.Address,
					"status", "running")

				if err := partitiondb.Query.UpsertVM(stepCtx, w.partitionDB.RW(), upsertParams); err != nil {
					return nil, fmt.Errorf("failed to upsert VM %s: %w", instance.Id, err)
				}

				w.logger.Info("successfully upserted VM to database", "vm_id", instance.Id)

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

			resp, err := http.DefaultClient.Get(openapiURL)
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
			specBytes, err := io.ReadAll(resp.Body)
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

	// Create database entries for all domains

	// Track domains that actually need to be changed in the dataplane
	changedDomains := []string{}

	for _, domain := range allDomains {

		_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {

			now := time.Now().UnixMilli()

			// This is more verbose than we initially thought
			// A simple ON DUPLICATE UPDATE was insufficient, because it could leak domains into other workspaces
			// because workspace slugs can change over time.
			// And we also need more control over updating rolled back domains
			return restate.Void{}, db.Tx(stepCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {

				existing, err := db.Query.FindDomainByDomain(txCtx, tx, domain.domain)
				if err != nil {
					if !db.IsNotFound(err) {
						return fmt.Errorf("failed to find domain entry for deployment %s: %w", deployment.ID, err)

					}

					// Domain does not exist, create it
					insertError := db.Query.InsertDomain(txCtx, tx, db.InsertDomainParams{
						ID:            uid.New("domain"),
						WorkspaceID:   deployment.WorkspaceID,
						ProjectID:     sql.NullString{Valid: true, String: deployment.ProjectID},
						EnvironmentID: sql.NullString{Valid: true, String: deployment.EnvironmentID},
						Domain:        domain.domain,
						Sticky:        domain.sticky,
						DeploymentID:  sql.NullString{Valid: true, String: deployment.ID},
						CreatedAt:     now,
						Type:          db.DomainsTypeWildcard,
					})
					if insertError != nil {
						return fmt.Errorf("failed to create domain entry for deployment %s: %w", deployment.ID, err)
					}
					changedDomains = append(changedDomains, domain.domain)
					return nil
				}

				if project.IsRolledBack {
					w.logger.Info("Skipping domain cause project is rolled back",
						"domain_id", existing.ID,
						"domain", existing.Domain,
					)
					return nil
				}
				updateErr := db.Query.ReassignDomain(txCtx, tx, db.ReassignDomainParams{
					ID:                existing.ID,
					TargetWorkspaceID: workspace.ID,
					DeploymentID:      sql.NullString{Valid: true, String: deployment.ID},
				})

				if updateErr != nil {
					return fmt.Errorf("failed to update domain entry for deployment %s: %w", deployment.ID, updateErr)
				}
				changedDomains = append(changedDomains, existing.Domain)

				return nil

			})
		}, restate.WithName("upserting domain"))
	}

	if err != nil {
		return nil, err
	}

	// Create gateway configs for all domains in bulk (except local ones)
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		// Prepare gateway configs for all non-local domains
		var gatewayParams []partitiondb.UpsertGatewayParams
		var skippedDomains []string
		for _, domain := range changedDomains {
			if isLocalHostname(domain, w.defaultDomain) {
				skippedDomains = append(skippedDomains, domain)
				continue
			}

			// Create VM protobuf objects for gateway config
			gatewayConfig := &partitionv1.GatewayConfig{
				Deployment: &partitionv1.Deployment{
					Id:        deployment.ID,
					IsEnabled: true,
				},
				Vms: make([]*partitionv1.VM, len(createdInstances)),
			}

			for i, vm := range createdInstances {
				gatewayConfig.Vms[i] = &partitionv1.VM{
					Id: vm.Id,
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

			// Marshal protobuf to bytes
			configBytes, err := protojson.Marshal(gatewayConfig)
			if err != nil {
				w.logger.Error("failed to marshal gateway config", "error", err, "domain", domain)
				continue
			}

			gatewayParams = append(gatewayParams, partitiondb.UpsertGatewayParams{
				WorkspaceID:  deployment.WorkspaceID,
				DeploymentID: deployment.ID,
				Hostname:     domain,
				Config:       configBytes,
			})
		}
		// Perform bulk upsert for all gateway configs
		if len(gatewayParams) > 0 {
			if err := partitiondb.BulkQuery.UpsertGateway(stepCtx, w.partitionDB.RW(), gatewayParams); err != nil {
				return restate.Void{}, fmt.Errorf("failed to upsert %d gateway configs for deployment %s: %w", len(gatewayParams), deployment.ID, err)
			}
		}

		return restate.Void{}, db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: deployment.ID,
			Status:       db.DeploymentStepsStatusAssigningDomains,
			Message:      fmt.Sprintf("Created %d gateway configs for %d domains (skipped %d local domains)", len(gatewayParams), len(allDomains), len(skippedDomains)),
			CreatedAt:    time.Now().UnixMilli(),
		})
	}, restate.WithName("creating gateway configs"))
	if err != nil {
		return nil, err
	}

	// Update deployment status to ready
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        deployment.ID,
			Status:    db.DeploymentsStatusReady,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("updating deployment status to ready"))
	if err != nil {
		return nil, err
	}

	if !project.IsRolledBack {
		// only update this if the deployment is not rolled back
		_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			return restate.Void{}, db.Query.UpdateProjectDeployments(stepCtx, w.db.RW(), db.UpdateProjectDeploymentsParams{
				ID:               deployment.ProjectID,
				LiveDeploymentID: sql.NullString{Valid: true, String: deployment.ID},
				UpdatedAt:        sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
		}, restate.WithName("updating project live deployment"))
		if err != nil {
			return nil, err
		}
	}

	/*

		// Step 23: Scrape OpenAPI spec from container (using host port mapping)
		openapiSpec, err := restate.Run(ctx, "scrape-openapi-spec", func(stepCtx restate.RunContext) (string, error) {

			// Find the port mapping for container port 8080
			var hostPort int32
			for _, portMapping := range vmInfo.NetworkInfo.PortMappings {
				if portMapping.ContainerPort == 8080 {
					hostPort = portMapping.HostPort
					break
				}
			}

			if hostPort == 0 {
				w.logger.Warn("no host port mapping found for container port 8080", "deployment_id", deployment.ID)
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
				w.logger.Info("trying to scrape OpenAPI spec", "url", openapiURL, "host_port", hostPort, "deployment_id", deployment.ID)

				resp, err := client.Get(openapiURL)
				if err != nil {
					w.logger.Warn("openapi scraping failed for host address", "error", err, "host_addr", hostAddr, "deployment_id", deployment.ID)
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					w.logger.Warn("openapi endpoint returned non-200 status", "status", resp.StatusCode, "host_addr", hostAddr, "deployment_id", deployment.ID)
					continue
				}

				// Read the OpenAPI spec
				specBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					w.logger.Warn("failed to read OpenAPI spec response", "error", err, "host_addr", hostAddr, "deployment_id", deployment.ID)
					continue
				}

				w.logger.Info("openapi spec scraped successfully", "host_addr", hostAddr, "deployment_id", deployment.ID, "spec_size", len(specBytes))
				return string(specBytes), nil
			}

			return "", fmt.Errorf("failed to scrape OpenAPI spec from all host addresses: %v", hostAddresses)
		})
		if err != nil {
			return err
		}

		// Step 24: Update gateway config with OpenAPI spec
		err = restate.Run(ctx, "update-gateway-config-openapi", func(stepCtx restate.RunContext) error {
			// Only update if we have both hostname and OpenAPI spec
			if req.Hostname == "" || openapiSpec == "" {
				w.logger.Info("skipping gateway config OpenAPI update",
					"has_hostname", req.Hostname != "",
					"has_openapi_spec", openapiSpec != "",
					"deployment_id", deployment.ID)
				return nil
			}

			w.logger.Info("updating gateway config with OpenAPI spec", "hostname", req.Hostname, "deployment_id", deployment.ID, "spec_size", len(openapiSpec))

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

			w.logger.Info("gateway config updated with OpenAPI spec successfully", "hostname", req.Hostname, "deployment_id", deployment.ID)
			return nil
		})
		if err != nil {
			// Don't fail the deployment for this
		}

		// Step 25: Store OpenAPI spec in database
		err = restate.Run(ctx, "store-openapi-spec", func(stepCtx restate.RunContext) error {
			if openapiSpec == "" {
				w.logger.Info("no OpenAPI spec to store", "deployment_id", deployment.ID)
				return nil
			}

			// Store in database
			err := db.Query.UpdateDeploymentOpenapiSpec(stepCtx, w.db.RW(), db.UpdateDeploymentOpenapiSpecParams{
				ID:          deployment.ID,
				OpenapiSpec: sql.NullString{String: openapiSpec, Valid: true},
				UpdatedAt:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				w.logger.Warn("failed to store OpenAPI spec in database", "error", err, "deployment_id", deployment.ID)
				return nil // Don't fail the deployment
			}

			w.logger.Info("openapi spec stored in database successfully", "deployment_id", deployment.ID, "spec_size", len(openapiSpec))
			return nil
		})
		if err != nil {
			return err
		}

	*/
	// Log deployment completed
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
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

	return &workflowsv1.DeployResponse{}, nil
}

// createGatewayConfig creates a gateway configuration protobuf object
//
// ENCODING POLICY FOR GATEWAY CONFIGS:
// Gateway configs are stored as JSON (using protojson.Marshal) for easier debugging
// and readability during development/demo. This makes it simpler to inspect and
// modify configs directly in the database.
// IMPORTANT: Always use protojson.Marshal for writes and protojson.Unmarshal for reads.
func createGatewayConfig(deploymentID, keyspaceID string, instances []*kranev1.Instance) (*partitionv1.GatewayConfig, error) {
	// Create VM protobuf objects for gateway config
	gatewayConfig := &partitionv1.GatewayConfig{
		Deployment: &partitionv1.Deployment{
			Id:        deploymentID,
			IsEnabled: true,
		},
		Vms: make([]*partitionv1.VM, len(instances)),
	}

	for i, vm := range instances {
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
