//go:build linux

package collector

import (
	"fmt"
	"os"
	"path/filepath"
)

// CgroupDriver identifies kubelet's cgroup path convention.
type CgroupDriver int

const (
	CgroupDriverUnknown CgroupDriver = iota
	// CgroupDriverSystemd uses kubepods.slice/kubepods-<qos>.slice/kubepods-<qos>-pod<UID-with-underscores>.slice.
	// Default on EKS AL2023 and most managed K8s.
	CgroupDriverSystemd
	// CgroupDriverCgroupfs uses kubepods/<qos>/pod<UID-with-dashes>/.
	// Default when kubelet runs inside a Docker container (minikube-docker,
	// kind, etc.) where systemd isn't the PID 1.
	CgroupDriverCgroupfs
)

func (d CgroupDriver) String() string {
	switch d {
	case CgroupDriverSystemd:
		return "systemd"
	case CgroupDriverCgroupfs:
		return "cgroupfs"
	case CgroupDriverUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// Preflight verifies this node is billable before the collector starts and
// returns the detected cgroup driver. Heimdall silently undercharges if we
// miss checkpoints, so we'd rather refuse to start than run on a node we
// can't measure correctly.
//
// Preconditions:
//   - cgroup v2 unified hierarchy is mounted at cgroupRoot
//   - kubelet uses either the systemd or cgroupfs cgroup driver
func Preflight(cgroupRoot string) (CgroupDriver, error) {
	// cgroup v2 is identified by cgroup.controllers at the root. v1 hybrid
	// mounts use per-controller subdirs without this file.
	controllers := filepath.Join(cgroupRoot, "cgroup.controllers")
	if _, err := os.Stat(controllers); err != nil {
		return CgroupDriverUnknown, fmt.Errorf("cgroup v2 not detected at %s (missing cgroup.controllers): %w", cgroupRoot, err)
	}

	systemd := filepath.Join(cgroupRoot, "kubepods.slice")
	cgroupfs := filepath.Join(cgroupRoot, "kubepods")

	if _, err := os.Stat(systemd); err == nil {
		return CgroupDriverSystemd, nil
	}

	if _, err := os.Stat(cgroupfs); err == nil {
		return CgroupDriverCgroupfs, nil
	}

	return CgroupDriverUnknown, fmt.Errorf(
		"neither %s nor %s exists — is kubelet running and scheduling pods?",
		systemd, cgroupfs,
	)
}
