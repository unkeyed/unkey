package privatenetwork

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

// TargetDeploymentAnnotation records the deployment that a connection Service
// resolves to. Readers must reject it unless the Service named by the
// ExternalName carries the same deployment label.
const TargetDeploymentAnnotation = "private-network.unkey.com/target-deployment-id"

// RegionAnnotation records the region of a regional connection Service. Its
// ExternalName is the region-prefixed [ServiceHost] of the shared Service,
// which resolves the shared Service's endpoints whose hostname is the region.
const RegionAnnotation = "private-network.unkey.com/region"

// ServiceName returns the name of the shared headless Service that publishes
// the ready replicas of one deployment port. The name never changes target.
func ServiceName(deploymentID string, port int32) string {
	sum := sha256.Sum256([]byte(deploymentID + "/" + strconv.Itoa(int(port))))
	return "unkey-private-" + hex.EncodeToString(sum[:16])
}

// ServiceHost returns the in-cluster DNS name of a Service. A connection
// Service's ExternalName is the ServiceHost of its shared target Service.
func ServiceHost(name, namespace string) string {
	return name + "." + namespace + ".svc.cluster.local"
}
