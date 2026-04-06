//go:build !linux

package collector

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type cgroupReader struct {
	root   string
	driver CgroupDriver
}

type cgroupReading struct {
	cpuUsageUsec int64
	memoryBytes  int64
}

// read is a no-op on non-Linux platforms.
func (r *cgroupReader) read(_ types.UID, _ corev1.PodQOSClass) (cgroupReading, error) {
	return cgroupReading{cpuUsageUsec: 0, memoryBytes: 0}, nil
}

// cgroupPath returns empty on non-Linux. The network reader skips attach on
// empty paths, so host-network pods and macOS dev all flow through the same
// no-op branch.
func (r *cgroupReader) cgroupPath(_ types.UID, _ corev1.PodQOSClass) string {
	return ""
}
