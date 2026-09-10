package github

import "fmt"

// DeployAuthorizationStatusContext returns a stable GitHub commit status
// context for one app and environment. This isolates each target's approval
// while allowing a newer deployment of that target to replace an older status.
func DeployAuthorizationStatusContext(appID, environmentID string) string {
	return fmt.Sprintf("Unkey Deploy Authorization / %s / %s", appID, environmentID)
}
