//go:build linux

package network

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// SO_NETNS_COOKIE is the socket option (SOL_SOCKET) that returns the
// kernel's per-netns cookie for the netns the socket was created in.
// Added in Linux 5.12. x/sys/unix exposes the constant from 6.4+; hardcode
// the value so we don't need the newer import.
const soNetnsCookie = 71

// attachPodEth0 does the full per-pod attach in one locked-thread window:
//
//  1. Opens the host + pod netns fds.
//  2. setns into the pod netns.
//  3. Finds pod-side eth0 (first veth with ParentIndex > 0).
//  4. Reads the pod netns cookie via SO_NETNS_COOKIE — we use the cookie
//     as the POD_KEY rodata constant for this program pair, so the key
//     is reproducible from outside the BPF program (getsockopt on any
//     socket in the pod netns returns the same value) and unique
//     per-pod.
//  5. Clones the BPF spec, sets POD_KEY = cookie, loads into the kernel.
//     The shared pinned counter map is reused via pinOpts.
//  6. Attaches both programs via TCX, head-anchored.
//  7. setns back to the host netns.
//
// Why per-pod load: the alternative — a single shared program keyed by
// bpf_get_netns_cookie(skb) — breaks on ingress because skb->sk is NULL
// when Cilium's bpf_redirect_peer delivers packets into the pod netns.
// The helper falls back to init_net's cookie, so every pod's ingress
// conflates into one map slot. Baking POD_KEY into .rodata per-pod
// sidesteps the helper entirely. See docs/engineering/infra/metering/heimdall.mdx.
//
// Runs on a locked OS thread; if setns back to the host netns fails we
// deliberately leak the thread so the runtime never hands it to another
// goroutine still pinned in the pod netns.
func attachPodEth0(
	netnsPath string,
	spec *ebpf.CollectionSpec,
	pinOpts ebpf.MapOptions,
) (coll *ebpf.Collection, egress, ingress link.Link, cookie uint64, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origNs, err := os.Open("/proc/self/ns/net")
	if err != nil {
		logger.Debug("attachPodEth0: open host netns failed", "error", err.Error())
		return nil, nil, nil, 0, fmt.Errorf("%w: open host netns: %w", ErrNetnsOpen, err)
	}
	defer func() { _ = origNs.Close() }()

	podNs, err := os.Open(netnsPath)
	if err != nil {
		logger.Debug("attachPodEth0: open pod netns failed", "netns_path", netnsPath, "error", err.Error())
		return nil, nil, nil, 0, fmt.Errorf("%w: open pod netns %s: %w", ErrNetnsOpen, netnsPath, err)
	}
	defer func() { _ = podNs.Close() }()

	if setnsErr := unix.Setns(int(podNs.Fd()), unix.CLONE_NEWNET); setnsErr != nil {
		logger.Debug("attachPodEth0: setns into pod netns failed", "netns_path", netnsPath, "errno", setnsErr.Error())
		return nil, nil, nil, 0, fmt.Errorf("%w: setns into pod: %w", ErrNetnsOpen, setnsErr)
	}

	// Everything from here runs inside the pod netns. On any exit path we
	// MUST try to setns back before the deferred UnlockOSThread hands the
	// thread back to the Go runtime. The re-setns happens at the bottom
	// on the happy path; failure paths explicitly re-setns first.

	eth0, findErr := findPodEth0()
	if findErr != nil {
		_ = restoreNetns(origNs)
		return nil, nil, nil, 0, findErr
	}

	cookie, cookieErr := readNetnsCookie()
	if cookieErr != nil {
		_ = restoreNetns(origNs)
		logger.Debug("attachPodEth0: read netns cookie failed", "netns_path", netnsPath, "error", cookieErr.Error())
		return nil, nil, nil, 0, fmt.Errorf("%w: read netns cookie: %w", ErrNetnsOpen, cookieErr)
	}

	// Clone the spec so each pod gets its own POD_KEY rewrite. The shared
	// pinned map (pod_counters, pinned under LIBBPF_PIN_BY_NAME) is reused
	// across all clones — pinOpts.PinPath makes the loader look up the
	// existing pin instead of creating a new map.
	perPodSpec, copyErr := spec.Copy(), error(nil)
	if perPodSpec == nil {
		_ = restoreNetns(origNs)
		return nil, nil, nil, 0, fmt.Errorf("%w: CollectionSpec.Copy returned nil", ErrTCXAttach)
	}
	if podKey, ok := perPodSpec.Variables["POD_KEY"]; ok {
		if setErr := podKey.Set(cookie); setErr != nil {
			_ = restoreNetns(origNs)
			logger.Debug("attachPodEth0: set POD_KEY failed", "netns_path", netnsPath, "cookie", cookie, "error", setErr.Error())
			return nil, nil, nil, 0, fmt.Errorf("%w: set POD_KEY=%d: %w", ErrTCXAttach, cookie, setErr)
		}
	} else {
		_ = restoreNetns(origNs)
		return nil, nil, nil, 0, fmt.Errorf("%w: POD_KEY variable missing from spec (bpf2go regen needed?)", ErrTCXAttach)
	}

	coll, copyErr = ebpf.NewCollectionWithOptions(perPodSpec, ebpf.CollectionOptions{ //nolint:exhaustruct // Programs optional
		Maps: pinOpts,
	})
	if copyErr != nil {
		_ = restoreNetns(origNs)
		logger.Debug("attachPodEth0: NewCollection failed", "netns_path", netnsPath, "cookie", cookie, "error", copyErr.Error())
		return nil, nil, nil, 0, fmt.Errorf("%w: load per-pod programs: %w", ErrTCXAttach, copyErr)
	}

	countEgress := coll.Programs["count_egress"]
	countIngress := coll.Programs["count_ingress"]
	if countEgress == nil || countIngress == nil {
		coll.Close()
		_ = restoreNetns(origNs)
		return nil, nil, nil, 0, fmt.Errorf("%w: program missing after load (have %v)", ErrTCXAttach, collectionProgramNames(coll))
	}

	// count_egress at tc-egress of eth0: packets leaving the pod.
	// count_ingress at tc-ingress of eth0: packets arriving at the pod.
	// Head anchor so we run ahead of any other TCX program that might
	// install here later; our handlers return TCX_NEXT so they don't
	// terminate the chain.
	egress, attachErr := link.AttachTCX(link.TCXOptions{ //nolint:exhaustruct // Flags/ExpectedRevision optional
		Interface: int(eth0),
		Program:   countEgress,
		Attach:    ebpf.AttachTCXEgress,
		Anchor:    link.Head(),
	})
	if attachErr != nil {
		coll.Close()
		_ = restoreNetns(origNs)
		logger.Debug("attachPodEth0: TCX egress attach failed", "netns_path", netnsPath, "ifindex", eth0, "error", attachErr.Error())
		return nil, nil, nil, 0, fmt.Errorf("attach tcx egress on pod eth0 (pod egress): %w: %w", ErrTCXAttach, attachErr)
	}

	ingress, attachErr = link.AttachTCX(link.TCXOptions{ //nolint:exhaustruct
		Interface: int(eth0),
		Program:   countIngress,
		Attach:    ebpf.AttachTCXIngress,
		Anchor:    link.Head(),
	})
	if attachErr != nil {
		_ = egress.Close()
		coll.Close()
		_ = restoreNetns(origNs)
		logger.Debug("attachPodEth0: TCX ingress attach failed", "netns_path", netnsPath, "ifindex", eth0, "error", attachErr.Error())
		return nil, nil, nil, 0, fmt.Errorf("attach tcx ingress on pod eth0 (pod ingress): %w: %w", ErrTCXAttach, attachErr)
	}

	if setnsBackErr := unix.Setns(int(origNs.Fd()), unix.CLONE_NEWNET); setnsBackErr != nil {
		// Leak the thread (see function-level comment). Clean up the
		// links + collection we just created so the caller doesn't see a
		// half-finished state.
		_ = egress.Close()
		_ = ingress.Close()
		coll.Close()
		runtime.LockOSThread()
		logger.Debug("attachPodEth0: setns back to host failed — leaking OS thread", "netns_path", netnsPath, "errno", setnsBackErr.Error())
		return nil, nil, nil, 0, fmt.Errorf("%w: setns back to host: %w", ErrNetnsOpen, setnsBackErr)
	}

	return coll, egress, ingress, cookie, nil
}

// collectionProgramNames returns the program names in a Collection for
// error messages. Only called on the sad path.
func collectionProgramNames(c *ebpf.Collection) []string {
	names := make([]string, 0, len(c.Programs))
	for name := range c.Programs {
		names = append(names, name)
	}
	return names
}

// restoreNetns switches back to the original netns. Used on error paths
// where we've already setns'd into the pod netns and need to get out.
// Distinct from the happy-path setns-back so the error wrap doesn't
// hide the *original* failure the caller cares about.
func restoreNetns(origNs *os.File) error {
	if err := unix.Setns(int(origNs.Fd()), unix.CLONE_NEWNET); err != nil {
		runtime.LockOSThread()
		logger.Debug("restoreNetns: setns back to host failed — leaking OS thread", "errno", err.Error())
		return err
	}
	return nil
}

// findPodEth0 is called from inside the pod netns. Returns the ifindex of
// the first veth link it finds — by CNI convention this is eth0, and the
// CNI only ever puts one veth pair in each pod netns, so "first" is
// unambiguous.
func findPodEth0() (uint32, error) {
	links, err := netlink.LinkList()
	if err != nil {
		logger.Debug("findPodEth0: netlink link list failed", "error", err.Error())
		return 0, fmt.Errorf("%w: netlink link list: %w", ErrVethLookup, err)
	}

	seen := make([]string, 0, len(links))
	for _, lk := range links {
		attrs := lk.Attrs()
		if attrs == nil {
			continue
		}
		seen = append(seen, fmt.Sprintf("%s(%s)", attrs.Name, lk.Type()))

		if lk.Type() != "veth" {
			continue
		}

		// CNI-created pod veths have ParentIndex pointing at the
		// host-side peer. Skip anything that doesn't (guards against
		// a future exotic netns layout).
		if attrs.ParentIndex == 0 {
			continue
		}

		return uint32(attrs.Index), nil
	}

	logger.Debug("findPodEth0: no veth in pod netns", "links_seen", seen)
	return 0, fmt.Errorf("%w: no veth found in pod netns", ErrVethLookup)
}

// readNetnsCookie returns the kernel's per-netns 64-bit cookie for the
// netns the calling thread is currently in. Matches the value BPF
// programs see via bpf_get_netns_cookie. Requires kernel >= 5.12 for
// the socket option; pre-5.12 kernels return ENOPROTOOPT and we error.
func readNetnsCookie() (uint64, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return 0, fmt.Errorf("socket in pod netns: %w", err)
	}
	defer func() { _ = unix.Close(fd) }()

	// Syscalls(SOL_SOCKET, SO_NETNS_COOKIE) returns a uint64 — 8 bytes.
	cookie, err := unix.GetsockoptUint64(fd, unix.SOL_SOCKET, soNetnsCookie)
	if err != nil {
		return 0, fmt.Errorf("getsockopt SO_NETNS_COOKIE: %w", err)
	}
	if cookie == 0 {
		// A zero cookie is what the kernel returns for the init netns
		// pre-5.12, which means pod cookies would collide with the
		// host's. Refuse to attach — better to see "no data" than
		// silently cross-attribute.
		return 0, errors.New("kernel returned zero netns cookie (kernel too old? needs >= 5.12)")
	}
	return cookie, nil
}
