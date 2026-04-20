package heimdall

import (
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
	KubeletRoot   string               `toml:"kubelet_root" config:"default=/var/lib/kubelet"`
	Observability config.Observability `toml:"observability"`
}

func (c *Config) Validate() error {
	return nil
}
