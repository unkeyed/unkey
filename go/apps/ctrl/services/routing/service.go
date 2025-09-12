package routing

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the RoutingService for rollback functionality
type Service struct {
	ctrlv1connect.UnimplementedRoutingServiceHandler
	db          db.Database
	partitionDB db.Database
	logger      logging.Logger
}

// New creates a new routing service instance
func New(database db.Database, partitionDB db.Database, logger logging.Logger) *Service {
	return &Service{
		UnimplementedRoutingServiceHandler: ctrlv1connect.UnimplementedRoutingServiceHandler{},
		db:                                 database,
		partitionDB:                        partitionDB,
		logger:                             logger.With("service", "routing"),
	}
}

// SetRoute updates routing for a hostname to point to a specific version
func (s *Service) SetRoute(ctx context.Context, req *connect.Request[ctrlv1.SetRouteRequest]) (*connect.Response[ctrlv1.SetRouteResponse], error) {
	hostname := req.Msg.GetHostname()
	versionID := req.Msg.GetVersionId()
	workspaceID := req.Msg.GetWorkspaceId()

	s.logger.InfoContext(ctx, "setting route",
		slog.String("hostname", hostname),
		slog.String("version_id", versionID),
		slog.String("workspace_id", workspaceID),
	)

	// Get current route to capture what we're switching from
	var previousVersionID string
	currentRoute, err := partitiondb.Query.FindGatewayByHostname(ctx, s.partitionDB.RO(), hostname)
	if err != nil && !db.IsNotFound(err) {
		s.logger.ErrorContext(ctx, "failed to get current route",
			slog.String("hostname", hostname),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get current route: %w", err))
	}

	if err == nil {
		// Parse existing config to get previous version
		var existingConfig partitionv1.GatewayConfig
		if err := protojson.Unmarshal(currentRoute.Config, &existingConfig); err == nil {
			previousVersionID = existingConfig.Deployment.Id
		}
	}

	// Check if the target version deployment exists and is in ready state
	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), versionID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", versionID))
		}
		s.logger.ErrorContext(ctx, "failed to get deployment",
			slog.String("version_id", versionID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	// Verify workspace authorization if workspace_id is provided
	// This prevents cross-tenant access when called through rollback or other authenticated endpoints
	if workspaceID != "" && deployment.WorkspaceID != workspaceID {
		s.logger.ErrorContext(ctx, "workspace authorization failed in SetRoute",
			slog.String("requested_workspace_id", workspaceID),
			slog.String("deployment_workspace_id", deployment.WorkspaceID),
			slog.String("deployment_id", versionID),
		)
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("deployment not found: %s", versionID))
	}

	if deployment.Status != "ready" {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("deployment %s is not in ready state, current status: %s", versionID, deployment.Status))
	}

	// Only switch traffic if target deployment has running VMs
	// Get VMs for this deployment to ensure they are running
	vms, err := partitiondb.Query.FindVMsByDeploymentId(ctx, s.partitionDB.RO(), versionID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to find VMs for deployment",
			slog.String("version_id", versionID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find VMs for deployment: %w", err))
	}

	if len(vms) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("no VMs available for deployment %s", versionID))
	}

	// Check that at least some VMs are running
	runningVMCount := 0
	for _, vm := range vms {
		if vm.Status == "running" {
			runningVMCount++
		}
	}

	if runningVMCount == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("no running VMs available for deployment %s", versionID))
	}

	// Create new gateway configuration
	gatewayConfig := &partitionv1.GatewayConfig{
		Deployment: &partitionv1.Deployment{
			Id:        versionID,
			IsEnabled: true,
		},
		Vms: make([]*partitionv1.VM, 0, len(vms)),
	}

	// Add VM references to the gateway config
	for _, vm := range vms {
		if vm.Status == "running" {
			gatewayConfig.Vms = append(gatewayConfig.Vms, &partitionv1.VM{
				Id: vm.ID,
			})
		}
	}

	// Marshal the configuration
	configBytes, err := protojson.Marshal(gatewayConfig)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal gateway config",
			slog.String("hostname", hostname),
			slog.String("version_id", versionID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to marshal gateway config: %w", err))
	}

	// Get workspace ID from deployment for audit logging
	deploymentWorkspaceID := deployment.WorkspaceID

	// Upsert the gateway configuration
	err = partitiondb.Query.UpsertGateway(ctx, s.partitionDB.RW(), partitiondb.UpsertGatewayParams{
		Hostname: hostname,
		Config:   configBytes,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to upsert gateway",
			slog.String("hostname", hostname),
			slog.String("version_id", versionID),
			slog.String("workspace_id", deploymentWorkspaceID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update routing: %w", err))
	}

	s.logger.InfoContext(ctx, "route updated successfully",
		slog.String("hostname", hostname),
		slog.String("previous_version_id", previousVersionID),
		slog.String("new_version_id", versionID),
		slog.Int("running_vms", runningVMCount),
	)

	return connect.NewResponse(&ctrlv1.SetRouteResponse{
		PreviousVersionId: previousVersionID,
		EffectiveAt:       timestamppb.Now(),
	}), nil
}

// GetRoute retrieves current routing configuration for a hostname
func (s *Service) GetRoute(ctx context.Context, req *connect.Request[ctrlv1.GetRouteRequest]) (*connect.Response[ctrlv1.GetRouteResponse], error) {
	hostname := req.Msg.GetHostname()

	s.logger.InfoContext(ctx, "getting route",
		slog.String("hostname", hostname),
	)

	gatewayRow, err := partitiondb.Query.FindGatewayByHostname(ctx, s.partitionDB.RO(), hostname)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no route found for hostname: %s", hostname))
		}
		s.logger.ErrorContext(ctx, "failed to find gateway",
			slog.String("hostname", hostname),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get route: %w", err))
	}

	// Unmarshal the protojson config
	var gatewayConfig partitionv1.GatewayConfig
	if err := protojson.Unmarshal(gatewayRow.Config, &gatewayConfig); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal gateway config",
			slog.String("hostname", hostname),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to unmarshal gateway config: %w", err))
	}

	// Convert to API route format
	route := &ctrlv1.Route{
		Hostname:  hostname,
		VersionId: gatewayConfig.Deployment.Id,
		Weight:    100, // Full traffic routing
		IsEnabled: gatewayConfig.Deployment.IsEnabled,
		CreatedAt: timestamppb.Now(), // TODO: Add timestamps to gateway table
		UpdatedAt: timestamppb.Now(),
	}

	return connect.NewResponse(&ctrlv1.GetRouteResponse{
		Route: route,
	}), nil
}

// ListRoutes - placeholder implementation
func (s *Service) ListRoutes(ctx context.Context, req *connect.Request[ctrlv1.ListRoutesRequest]) (*connect.Response[ctrlv1.ListRoutesResponse], error) {
	return connect.NewResponse(&ctrlv1.ListRoutesResponse{
		Routes: []*ctrlv1.Route{},
	}), nil
}

// Rollback performs a rollback to a previous version
// This is the main rollback implementation that the dashboard will call
func (s *Service) Rollback(ctx context.Context, req *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error) {
	hostname := req.Msg.GetHostname()
	targetVersionID := req.Msg.GetTargetVersionId()
	workspaceID := req.Msg.GetWorkspaceId()

	s.logger.InfoContext(ctx, "initiating rollback",
		slog.String("hostname", hostname),
		slog.String("target_version_id", targetVersionID),
		slog.String("workspace_id", workspaceID),
	)

	// Validate workspace ID is provided
	if workspaceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("workspace_id is required for authorization"))
	}

	// Verify workspace exists
	_, err := db.Query.FindWorkspaceByID(ctx, s.db.RO(), workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("workspace not found: %s", workspaceID))
		}
		s.logger.ErrorContext(ctx, "failed to find workspace",
			slog.String("workspace_id", workspaceID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to verify workspace: %w", err))
	}

	// Get the target deployment and verify it belongs to the workspace
	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), targetVersionID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", targetVersionID))
		}
		s.logger.ErrorContext(ctx, "failed to get deployment",
			slog.String("version_id", targetVersionID),
			slog.String("error", err.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	// Verify workspace ownership - prevent cross-tenant rollbacks
	if deployment.WorkspaceID != workspaceID {
		s.logger.ErrorContext(ctx, "workspace authorization failed",
			slog.String("requested_workspace_id", workspaceID),
			slog.String("deployment_workspace_id", deployment.WorkspaceID),
			slog.String("deployment_id", targetVersionID),
		)
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("deployment not found: %s", targetVersionID))
	}

	// Get current route to capture what we're rolling back from
	getCurrentReq := &ctrlv1.GetRouteRequest{Hostname: hostname}
	getCurrentResp, err := s.GetRoute(ctx, connect.NewRequest(getCurrentReq))
	if err != nil && connect.CodeOf(err) != connect.CodeNotFound {
		return nil, err
	}

	var previousVersionID string
	if err == nil {
		previousVersionID = getCurrentResp.Msg.Route.VersionId
	}

	// Use SetRoute to perform the actual routing change - pass workspace context
	setRouteReq := &ctrlv1.SetRouteRequest{
		Hostname:    hostname,
		VersionId:   targetVersionID,
		Weight:      100,         // Full cutover for rollback
		WorkspaceId: workspaceID, // Pass workspace for validation
	}

	setRouteResp, err := s.SetRoute(ctx, connect.NewRequest(setRouteReq))
	if err != nil {
		s.logger.ErrorContext(ctx, "rollback failed",
			slog.String("hostname", hostname),
			slog.String("target_version_id", targetVersionID),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.logger.InfoContext(ctx, "rollback completed successfully",
		slog.String("hostname", hostname),
		slog.String("previous_version_id", previousVersionID),
		slog.String("new_version_id", targetVersionID),
	)

	return connect.NewResponse(&ctrlv1.RollbackResponse{
		PreviousVersionId: previousVersionID,
		NewVersionId:      targetVersionID,
		EffectiveAt:       setRouteResp.Msg.EffectiveAt,
	}), nil
}
