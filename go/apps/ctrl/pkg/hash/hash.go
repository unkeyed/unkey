package hash

import (
	"crypto/sha256"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/db"
)

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
func Deployment(deployment db.FindDeploymentTopologyByIDAndRegionRow) string {
	hash := fmt.Sprintf("%x", sha256.Sum256(fmt.Appendf(nil, "%v", []any{
		deployment.ID,
		deployment.Replicas,
		deployment.Image,
		deployment.Region,
		deployment.CpuMillicores,
		deployment.MemoryMib,
		deployment.DesiredState,
	})))

	return hash
}
