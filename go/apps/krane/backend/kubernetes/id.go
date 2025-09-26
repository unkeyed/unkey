package kubernetes

import "strings"

func safeIDForK8s(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "_", "-"))
}
