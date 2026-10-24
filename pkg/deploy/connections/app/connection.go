package app

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"k8s.io/apimachinery/pkg/util/validation"
)

// IsValidName accepts canonical connection names, excluding the reserved unkey
// prefix and [privatenetwork.SelfName].
func IsValidName(name string) bool {
	return len(validation.IsDNS1035Label(name)) == 0 && !strings.HasPrefix(name, "unkey") && name != privatenetwork.SelfName
}

// HostVariable returns the environment variable that the caller deployment
// receives for the connection named name, and its hostname value. namespace is
// the caller deployment's Kubernetes namespace. The hostname resolves to the
// target's ready Pod IPs; it carries no port or scheme because a connection
// grants every unicast TCP and UDP port.
func HostVariable(callerDeploymentID, name, namespace string) (key, value string) {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_HOST", privatenetwork.ConnectionHost(callerDeploymentID, name, namespace)
}
