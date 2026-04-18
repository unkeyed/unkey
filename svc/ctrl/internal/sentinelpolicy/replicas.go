// Package sentinelpolicy holds cross-cutting policy rules for sentinel
// provisioning that are shared between the deploy worker (which creates
// sentinels) and the ctrl sentinel service (which edits them).
package sentinelpolicy

// MinReplicasForEnv returns the minimum replica count a sentinel must run
// with for a given environment slug. Production gets 3 for HA; everything
// else (preview, staging, etc.) gets 1. Callers may scale above this floor
// but not below.
func MinReplicasForEnv(envSlug string) int32 {
	if envSlug == "production" {
		return 3
	}
	return 1
}
