//go:build linux

package collector

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type cgroupReader struct {
	root string // e.g. "/sys/fs/cgroup"
}

type cpuReading struct {
	usageUsec int64
	readAt    time.Time
}

type podResources struct {
	cpuMillicores int32
	memoryBytes   int64
}

// readPodResources reads CPU and memory stats from the pod's cgroup directory.
// If prev is nil, only the raw reading is returned (no CPU rate can be computed).
// Returns the current cpuReading for use as next tick's prev.
func (r *cgroupReader) readPodResources(uid types.UID, qos corev1.PodQOSClass, prev *cpuReading) (podResources, cpuReading, error) {
	dir := r.cgroupPath(uid, qos)

	usageUsec, err := parseCPUStat(filepath.Join(dir, "cpu.stat"))
	if err != nil {
		return podResources{}, cpuReading{}, fmt.Errorf("reading cpu.stat: %w", err)
	}

	memBytes, err := parseMemoryCurrent(filepath.Join(dir, "memory.current"))
	if err != nil {
		return podResources{}, cpuReading{}, fmt.Errorf("reading memory.current: %w", err)
	}

	now := time.Now()
	reading := cpuReading{usageUsec: usageUsec, readAt: now}

	res := podResources{cpuMillicores: 0, memoryBytes: memBytes}

	if prev != nil {
		elapsed := now.Sub(prev.readAt)
		if elapsed > 0 {
			deltaUsec := usageUsec - prev.usageUsec
			if deltaUsec >= 0 {
				// Convert: (deltaUsec / elapsedUsec) * 1000 = millicores
				elapsedUsec := elapsed.Microseconds()
				res.cpuMillicores = int32(deltaUsec * 1000 / elapsedUsec)
			}
			// Negative delta = counter reset (container restart). Skip CPU, keep memory.
		}
	}

	return res, reading, nil
}

func (r *cgroupReader) cgroupPath(uid types.UID, qos corev1.PodQOSClass) string {
	uidStr := strings.ReplaceAll(string(uid), "-", "_")
	switch qos {
	case corev1.PodQOSGuaranteed:
		return filepath.Join(r.root, "kubepods.slice", fmt.Sprintf("kubepods-pod%s.slice", uidStr))
	case corev1.PodQOSBurstable:
		return filepath.Join(r.root, "kubepods.slice", "kubepods-burstable.slice", fmt.Sprintf("kubepods-burstable-pod%s.slice", uidStr))
	case corev1.PodQOSBestEffort:
		return filepath.Join(r.root, "kubepods.slice", "kubepods-besteffort.slice", fmt.Sprintf("kubepods-besteffort-pod%s.slice", uidStr))
	default:
		// Fallback to burstable (most common for krane pods with requests but no limits)
		return filepath.Join(r.root, "kubepods.slice", "kubepods-burstable.slice", fmt.Sprintf("kubepods-burstable-pod%s.slice", uidStr))
	}
}

// parseCPUStat reads cpu.stat and extracts usage_usec.
func parseCPUStat(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "usage_usec ") {
			val, err := strconv.ParseInt(strings.TrimPrefix(line, "usage_usec "), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parsing usage_usec: %w", err)
			}
			return val, nil
		}
	}
	return 0, fmt.Errorf("usage_usec not found in %s", path)
}

// parseMemoryCurrent reads memory.current (single integer, bytes).
func parseMemoryCurrent(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
}
