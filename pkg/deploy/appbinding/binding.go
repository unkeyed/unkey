package appbinding

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// Zone is the private DNS zone that undns answers for bound callers.
const Zone = "unkey.internal"

// HostVariable returns the environment variable that a caller deployment
// receives for the binding named name, and its hostname value. The hostname
// resolves to the target's ready Pod IPs; it carries no port or scheme because
// a binding grants every unicast TCP and UDP port.
func HostVariable(name string) (key, value string) {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_HOST", name + "." + Zone
}

// ReplicaHost returns the hostname that resolves a deployment's own ready
// replicas across regions: its app slug in [Zone]. It reports false when the
// slug is not a DNS label, in which case the deployment has no replica host.
func ReplicaHost(appSlug string) (string, bool) {
	if len(validation.IsDNS1123Label(appSlug)) != 0 {
		return "", false
	}
	return appSlug + "." + Zone, true
}
