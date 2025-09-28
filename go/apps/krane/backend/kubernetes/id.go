package kubernetes

import "strings"

// safeIDForK8s converts deployment IDs to Kubernetes-safe resource names.
//
// Replaces underscores with hyphens and converts to lowercase to comply
// with DNS-1123 label requirements.
func safeIDForK8s(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "_", "-"))
}
