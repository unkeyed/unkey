//go:build !linux

package collector

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type cgroupReader struct {
	root string
}

type cpuReading struct {
	usageUsec int64
	readAt    time.Time
}

type podResources struct {
	cpuMillicores int32
	memoryBytes   int64
}

// readPodResources is a no-op on non-Linux platforms.
func (r *cgroupReader) readPodResources(_ types.UID, _ corev1.PodQOSClass, _ *cpuReading) (podResources, cpuReading, error) {
	return podResources{cpuMillicores: 0, memoryBytes: 0}, cpuReading{usageUsec: 0, readAt: time.Now()}, nil
}
