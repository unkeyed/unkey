package hash

import (
	"crypto/sha256"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/db"
)

// Sentinel creates a deterministic hash for sentinel configuration.
//
// This function hashes all relevant sentinel fields including ID,
// image, replica count, CPU allocation, and memory
// configuration. The hash can be used to detect configuration
// changes and uniquely identify sentinel resources.
//
// Returns a hex-encoded SHA-256 hash of the sentinel configuration.
func Sentinel(sentinel db.Sentinel) string {
	hash := fmt.Sprintf("%x", sha256.Sum256(fmt.Appendf(nil, "%v", []any{
		sentinel.ID,
		sentinel.Image,
		sentinel.DesiredReplicas,
		sentinel.CpuMillicores,
		sentinel.MemoryMib,
	})))

	return hash
}

// Deployment creates a deterministic hash for deployment configuration.
//
// This function hashes all relevant deployment fields including ID,
// replica count, image, region, resources, and desired state.
// The hash can be used to detect configuration changes and
// uniquely identify deployment resources.
//
// Returns a hex-encoded SHA-256 hash of the deployment configuration.
func Deployment(deployment db.FindDeploymentTopologyByIDAndRegionRow) string {
	hash := fmt.Sprintf("%x", sha256.Sum256(fmt.Appendf(nil, "%v", []any{
		deployment.ID,
		deployment.DesiredReplicas,
		deployment.Image,
		deployment.Region,
		deployment.CpuMillicores,
		deployment.MemoryMib,
		deployment.DesiredState,
	})))

	return hash
}
