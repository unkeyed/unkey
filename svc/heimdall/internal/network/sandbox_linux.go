//go:build linux

package network

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Label names on containerd containers, stable UAPI.
//   - criPodUIDLabel is kubelet's pod-identity label (same one cri_linux.go
//     uses to map Task events back to pods).
//   - criKindLabel / criKindSandbox identify the pause/sandbox container
//     (which carries the network namespace reference in its OCI spec).
//     Containerd's CRI plugin sets this on every sandbox; the kubelet-level
//     `io.kubernetes.container.name=POD` label is only visible through the
//     CRI API, not on containerd's own Container list.
const (
	criPodUIDLabel = "io.kubernetes.pod.uid"
	criKindLabel   = "io.cri-containerd.kind"
	criKindSandbox = "sandbox"
)

// sandboxNetnsPath finds the containerd sandbox container for the given pod
// UID and returns its network-namespace path (e.g. /var/run/netns/cni-<uuid>).
// Works for runc and gVisor pods alike: both get an OCI network namespace
// entry pointing at the CNI-created netns on disk. This is the only reliable
// way to find a gVisor pod's CNI netns - the pod's cgroup.procs PIDs live in
// gVisor's internal sandbox netns, not the CNI netns where the veth lives.
func (r *linuxReader) sandboxNetnsPath(ctx context.Context, podUID string) (string, error) {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")

	// containerd filter syntax quotes label keys with the full dotted name.
	// Matching on both labels narrows the result to the one pause/sandbox
	// container belonging to this pod, across all runtime types.
	filter := fmt.Sprintf(
		`labels."%s"==%s,labels."%s"==%s`,
		criPodUIDLabel, podUID,
		criKindLabel, criKindSandbox,
	)

	containers, err := r.listSandboxes(ctx, filter)
	if err != nil {
		logger.Debug("sandbox netns: containerd list failed", "pod_uid", podUID, "error", err.Error())
		return "", fmt.Errorf("containerd list: %w", err)
	}

	if len(containers) == 0 {
		logger.Debug("sandbox netns: no containerd sandbox container for pod", "pod_uid", podUID)
		return "", fmt.Errorf("%w: pod %s", ErrSandboxNotFound, podUID)
	}

	// Multiple sandboxes can match one pod UID: after a gVisor memory-limit
	// kill kubelet recreates the sandbox under the same pod, and containerd
	// keeps the stopped sandbox's container record (OCI spec included) until
	// RemovePodSandbox, which kubelet GC runs about a minute later. The
	// stopped record still names its netns path even though StopPodSandbox
	// already removed the file, and containerd's list order is database
	// scan order, not creation time. So a path is only a candidate if its
	// netns still exists. If every record's netns is gone, the new sandbox
	// has not been created yet; report ENOENT so the worker files it as
	// benign churn and the next tick retries.
	staleNetns := ""
	for _, c := range containers {
		spec, err := c.Spec(ctx)
		if err != nil {
			logger.Debug("sandbox netns: failed to load OCI spec",
				"pod_uid", podUID, "sandbox_id", c.ID(), "error", err.Error())
			continue
		}
		if spec.Linux == nil {
			logger.Debug("sandbox netns: OCI spec has no linux section",
				"pod_uid", podUID, "sandbox_id", c.ID())
			continue
		}
		path := networkNamespacePath(spec)
		if path == "" {
			logger.Debug("sandbox netns: OCI spec.linux.namespaces has no network entry with a path",
				"pod_uid", podUID, "sandbox_id", c.ID(), "namespaces", spec.Linux.Namespaces)
			continue
		}
		if r.netnsGone(path) {
			logger.Debug("sandbox netns: skipping stopped sandbox, netns removed",
				"pod_uid", podUID, "sandbox_id", c.ID(), "netns_path", path)
			staleNetns = path
			continue
		}
		return path, nil
	}

	if staleNetns != "" {
		return "", fmt.Errorf("no sandbox with a live netns (last stopped: %s): %w", staleNetns, os.ErrNotExist)
	}

	return "", errors.New("sandbox spec has no network namespace path")
}

// networkNamespacePath returns linux.namespaces[type=network].path from an
// OCI spec, or "" when the spec has none (host-network sandboxes).
func networkNamespacePath(spec *oci.Spec) string {
	for _, ns := range spec.Linux.Namespaces {
		if ns.Type == "network" && ns.Path != "" {
			return ns.Path
		}
	}
	return ""
}
