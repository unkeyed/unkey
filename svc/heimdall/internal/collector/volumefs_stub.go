//go:build !linux

package collector

import "k8s.io/apimachinery/pkg/types"

// readEphemeralUsedBytes is a no-op on non-Linux platforms (heimdall only
// runs on Linux in production; this stub keeps tests building on macOS).
func readEphemeralUsedBytes(_ string, _ types.UID) int64 {
	return 0
}
