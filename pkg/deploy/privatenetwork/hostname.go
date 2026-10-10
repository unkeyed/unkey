package privatenetwork

import "k8s.io/apimachinery/pkg/util/validation"

// Zone is the private DNS zone for connections and a deployment's own replicas.
const Zone = "unkey.internal"

// ReplicaHost returns the hostname that resolves a deployment's own ready
// replicas across regions: its app slug in [Zone]. It reports false when the
// slug is not a DNS label, in which case the deployment has no replica host.
func ReplicaHost(appSlug string) (string, bool) {
	if len(validation.IsDNS1123Label(appSlug)) != 0 {
		return "", false
	}
	return appSlug + "." + Zone, true
}
