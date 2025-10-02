package deploy

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// createGatewayConfig creates a gateway configuration protobuf object.
//
// The config includes deployment ID, VM addresses for load balancing, and optional
// auth configuration. Gateway configs control how the edge gateways route traffic
// and authenticate requests for each deployment.
//
// ENCODING POLICY: Gateway configs are stored as JSON (using protojson.Marshal) for
// easier debugging and readability during development. This makes it simpler to inspect
// and modify configs directly in the database. Always use protojson.Marshal for writes
// and protojson.Unmarshal for reads.
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

// isLocalHostname checks if a hostname should be skipped from gateway config creation.
//
// Returns true for localhost and development domains (.local, .test TLDs) that should
// not get gateway configurations. Hostnames using the default domain (e.g., *.unkey.app)
// return false, as they represent production/staging environments and need gateway configs.
//
// This prevents unnecessary config creation during local development while ensuring
// production domains are properly configured.
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

// updateDeploymentStatus updates the status of a deployment in the database.
//
// This is a durable operation wrapped in restate.Run to ensure the status update
// is persisted even if the workflow is interrupted. Status updates are critical
// for tracking deployment progress and handling failures.
func (w *Workflow) updateDeploymentStatus(ctx restate.ObjectContext, deploymentID string, status db.DeploymentsStatus) error {

	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		updateErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        deploymentID,
			Status:    status,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			return restate.Void{}, fmt.Errorf("failed to update version status to building: %w", updateErr)
		}
		return restate.Void{}, nil
	}, restate.WithName(fmt.Sprintf("updating deployment status to %s", status)))
	return err

}
