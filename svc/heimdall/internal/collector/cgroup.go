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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type cgroupReader struct {
	root   string       // e.g. "/sys/fs/cgroup"
	driver CgroupDriver // systemd or cgroupfs; decides path shape
}

// cgroupReading is the raw counter + gauge snapshot read from the cgroup.
// No derived values — billing math happens at query time.
type cgroupReading struct {
	// cpuUsageUsec is cgroup v2 cpu.stat:usage_usec — a monotonic microsecond
	// counter of CPU time consumed by all processes in this cgroup. Does NOT
	// include throttled time (that's tracked separately), which aligns with
	// "bill only for work actually done."
	cpuUsageUsec int64

	// memoryBytes is the "working set" size — memory.current minus
	// inactive_file pages. Matches kubelet's
	// container_memory_working_set_bytes and k8s OOM decisions. Using
	// memory.current directly would overcharge customers for reclaimable
	// kernel page cache they cannot shed.
	//
	// Zero means "memory unknown" (the cgroup went away between our CPU read
	// and our memory read, a race that happens near teardown). Callers still
	// emit the checkpoint — losing the final CPU counter is worse than an
	// undercharged memory interval.
	memoryBytes int64
}

// read returns the current CPU counter and working-set memory for a pod's
// cgroup. Returns os.ErrNotExist if cpu.stat is missing (cgroup fully
// torn down). If cpu.stat succeeds but memory files fail (the narrow race
// window of partial teardown), returns the CPU reading with memoryBytes=0.
func (r *cgroupReader) read(uid types.UID, qos corev1.PodQOSClass) (cgroupReading, error) {
	dir := r.cgroupPath(uid, qos)

	usageUsec, err := parseCPUStat(filepath.Join(dir, "cpu.stat"))
	if err != nil {
		// cgroup fully gone — caller already handles os.ErrNotExist as a
		// normal "container vanished" outcome.
		return cgroupReading{}, fmt.Errorf("reading cpu.stat: %w", err)
	}

	// CPU is the most valuable data — the final reading caps max(cpu_usage_usec)
	// for billing. Don't sacrifice it if memory reads race with teardown.
	current, err := parseMemoryCurrent(filepath.Join(dir, "memory.current"))
	if err != nil {
		return cgroupReading{cpuUsageUsec: usageUsec, memoryBytes: 0}, nil
	}

	inactiveFile, err := parseMemoryStatInactiveFile(filepath.Join(dir, "memory.stat"))
	if err != nil {
		return cgroupReading{cpuUsageUsec: usageUsec, memoryBytes: 0}, nil
	}

	// Kernel updates memory.current and memory.stat non-atomically — on skew
	// where inactive_file reflects a slightly older larger value, subtraction
	// could go negative. Clamp to 0.
	var workingSet int64
	if current > inactiveFile {
		workingSet = current - inactiveFile
	}

	return cgroupReading{cpuUsageUsec: usageUsec, memoryBytes: workingSet}, nil
}

// cgroupPath returns the pod-level cgroup directory for the given pod,
// formatted for the active cgroup driver.
//
// systemd: kubepods.slice/kubepods-<qos>.slice/kubepods-<qos>-pod<UID_>.slice
//   - UID dashes replaced with underscores
//   - Guaranteed: kubepods.slice/kubepods-pod<UID_>.slice (no qos subdir)
//
// cgroupfs: kubepods/<qos>/pod<UID->/
//   - UID keeps dashes
//   - Guaranteed: kubepods/pod<UID->/ (no qos subdir)
func (r *cgroupReader) cgroupPath(uid types.UID, qos corev1.PodQOSClass) string {
	if r.driver == CgroupDriverCgroupfs {
		return r.cgroupfsPath(uid, qos)
	}

	return r.systemdPath(uid, qos)
}

func (r *cgroupReader) systemdPath(uid types.UID, qos corev1.PodQOSClass) string {
	uidStr := strings.ReplaceAll(string(uid), "-", "_")

	switch qos {
	case corev1.PodQOSGuaranteed:
		return filepath.Join(r.root, "kubepods.slice", fmt.Sprintf("kubepods-pod%s.slice", uidStr))
	case corev1.PodQOSBurstable:
		return filepath.Join(r.root, "kubepods.slice", "kubepods-burstable.slice", fmt.Sprintf("kubepods-burstable-pod%s.slice", uidStr))
	case corev1.PodQOSBestEffort:
		return filepath.Join(r.root, "kubepods.slice", "kubepods-besteffort.slice", fmt.Sprintf("kubepods-besteffort-pod%s.slice", uidStr))
	default:
		// Krane pods have requests but no limits — kubelet classifies as burstable.
		return filepath.Join(r.root, "kubepods.slice", "kubepods-burstable.slice", fmt.Sprintf("kubepods-burstable-pod%s.slice", uidStr))
	}
}

func (r *cgroupReader) cgroupfsPath(uid types.UID, qos corev1.PodQOSClass) string {
	uidStr := string(uid) // dashes preserved

	switch qos {
	case corev1.PodQOSGuaranteed:
		return filepath.Join(r.root, "kubepods", fmt.Sprintf("pod%s", uidStr))
	case corev1.PodQOSBurstable:
		return filepath.Join(r.root, "kubepods", "burstable", fmt.Sprintf("pod%s", uidStr))
	case corev1.PodQOSBestEffort:
		return filepath.Join(r.root, "kubepods", "besteffort", fmt.Sprintf("pod%s", uidStr))
	default:
		return filepath.Join(r.root, "kubepods", "burstable", fmt.Sprintf("pod%s", uidStr))
	}
}

// parseCPUStat reads cpu.stat and extracts usage_usec (monotonic microseconds).
func parseCPUStat(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		rest, ok := strings.CutPrefix(scanner.Text(), "usage_usec ")
		if !ok {
			continue
		}

		return strconv.ParseInt(rest, 10, 64)
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

// parseMemoryStatInactiveFile reads the inactive_file line from memory.stat.
// This is page cache the kernel can reclaim under memory pressure and should
// not be billed to the customer. Returns 0 if the line is missing (treat as
// no reclaimable cache rather than failing).
func parseMemoryStatInactiveFile(path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		rest, ok := strings.CutPrefix(scanner.Text(), "inactive_file ")
		if !ok {
			continue
		}

		return strconv.ParseInt(rest, 10, 64)
	}

	return 0, nil
}
