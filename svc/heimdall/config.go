package heimdall

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/config"
)

type Config struct {
	Region        string `toml:"region" config:"required,nonempty"`
	Platform      string `toml:"platform" config:"required,nonempty"`
	ClickHouseURL string `toml:"clickhouse_url" config:"required,nonempty"`
	// CollectionInterval is the periodic cgroup-read cadence. Primary
	// signals are CRI TaskStart/TaskExit events and the pod informer;
	// the periodic tick is the backstop when both miss a transition.
	// 5s guarantees every 15s dashboard bucket gets ≥2 samples so CPU
	// deltas (max - min of the kernel counter) are always meaningful.
	// ~3 syscalls per pod per tick, negligible overhead.
	CollectionInterval time.Duration `toml:"collection_interval" config:"default=5s"`
	NodeName           string        `toml:"node_name" config:"required,nonempty"`
	// CRISocket is the containerd socket path used to capture container exit
	// events for ms-precise stop checkpoints. Empty disables CRI watching.
	CRISocket string `toml:"cri_socket" config:"default=/run/containerd/containerd.sock"`
	// KubeletRoot is the host path where kubelet's per-pod volume directories
	// live. Used to statfs ephemeral volume mounts for disk_used_bytes.
	// Empty disables used-bytes reporting.
	KubeletRoot string `toml:"kubelet_root" config:"default=/var/lib/kubelet"`
	// Collectors gates which metric kinds heimdall scrapes per tick. Empty
	// or omitted means all four enabled. Pass a subset to roll one out
	// at a time:
	//
	//   collectors = ["cpu", "memory"]   # disk + network disabled
	//
	// Disabled collectors skip their data source and write zero into the
	// schema's numeric columns. Notably, disabling "network" also skips
	// the eBPF program load + BPF map allocation + TCX attach workers,
	// so it's safe to leave off on nodes where eBPF isn't yet desired.
	// Per-checkpoint attributes record what was actually enabled, so
	// query-time consumers can distinguish "0 because disabled" from
	// "0 because measured to be zero".
	Collectors    []string             `toml:"collectors"`
	Observability config.Observability `toml:"observability"`
}

// allCollectors is the canonical set of collector names. Sorted in the
// order operators most often think about them (cgroup-cheap first, then
// disk, then the eBPF-heavy network).
var allCollectors = []string{"cpu", "memory", "disk", "network"}

func (c *Config) Validate() error {
	if len(c.Collectors) == 0 {
		c.Collectors = allCollectors
		return nil
	}
	known := map[string]bool{"cpu": true, "memory": true, "disk": true, "network": true}
	for _, name := range c.Collectors {
		if !known[name] {
			return fmt.Errorf(
				"collectors: unknown collector %q (must be one of %v)",
				name, allCollectors,
			)
		}
	}
	return nil
}
