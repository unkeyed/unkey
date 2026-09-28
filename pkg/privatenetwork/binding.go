package privatenetwork

import "strings"

// Zone is the private DNS zone that undns answers for bound callers.
const Zone = "unkey.internal"

// ReplicaHost resolves the caller deployment's ready replicas across regions.
const ReplicaHost = "unkey-replicas." + Zone

// HostVariable returns the environment variable that a caller deployment
// receives for the binding named name, and its hostname value. The hostname
// resolves to the target's ready Pod IPs; it carries no port or scheme because
// a binding grants every unicast TCP and UDP port.
func HostVariable(name string) (key, value string) {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_HOST", name + "." + Zone
}
