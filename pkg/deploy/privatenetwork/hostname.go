package privatenetwork

import (
	"crypto/sha256"
	"encoding/hex"
)

// Zone is the private DNS zone for connections and a deployment's own replicas.
const Zone = "unkey.internal"

// SelfName is the reserved connection alias that resolves a deployment's own
// ready replicas across regions.
const SelfName = "self"

// ConnectionServiceName returns the name of the Service that publishes the
// connection alias for one caller deployment. The name has a fixed length, so
// every legal alias fits in a Kubernetes Service name.
func ConnectionServiceName(callerDeploymentID, alias string) string {
	return "unkey-connection-" + connectionToken(callerDeploymentID, alias)
}

// RegionalConnectionServiceName returns the name of the Service that publishes
// the connection alias for one caller deployment in one region. region must be
// a DNS-1123 label of at most 22 characters, so the name fits in 63.
func RegionalConnectionServiceName(callerDeploymentID, alias, region string) string {
	return "unkey-r-" + connectionToken(callerDeploymentID, alias) + "-" + region
}

// ConnectionHost returns the absolute hostname that the caller deployment uses
// for alias. namespace is the Kubernetes namespace of the caller deployment and
// must be a DNS-1123 label. The hostname does not depend on DNS search domains.
func ConnectionHost(callerDeploymentID, alias, namespace string) string {
	return ConnectionServiceName(callerDeploymentID, alias) + "." + namespace + "." + Zone
}

func connectionToken(callerDeploymentID, alias string) string {
	sum := sha256.Sum256([]byte(callerDeploymentID + "/" + alias))
	return hex.EncodeToString(sum[:16])
}
