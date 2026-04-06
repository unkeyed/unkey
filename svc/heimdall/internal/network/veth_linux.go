//go:build linux

package network

import (
	"fmt"
	"os"
	"runtime"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// hostVethIfindex opens netnsPath, enters it, uses netlink to find the
// first veth device (eth0 in CNI-created netns), and returns its peer
// ifindex - i.e. the host-side veth ifindex that tc attaches to.
// Netlink returns peer ifindex values from the peer's namespace perspective
// (the host), which is exactly what TCX needs.
//
// The goroutine is locked to its OS thread for the setns dance; if the
// switch back to the host netns fails we deliberately leak the thread so
// the runtime never puts it back into the pool in a broken state.
func hostVethIfindex(netnsPath string) (uint32, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origNs, err := os.Open("/proc/self/ns/net")
	if err != nil {
		return 0, fmt.Errorf("%w: open host netns: %w", ErrNetnsOpen, err)
	}
	defer func() { _ = origNs.Close() }()

	podNs, err := os.Open(netnsPath)
	if err != nil {
		return 0, fmt.Errorf("%w: open pod netns %s: %w", ErrNetnsOpen, netnsPath, err)
	}
	defer func() { _ = podNs.Close() }()

	if err := unix.Setns(int(podNs.Fd()), unix.CLONE_NEWNET); err != nil {
		return 0, fmt.Errorf("%w: setns into pod: %w", ErrNetnsOpen, err)
	}

	ifindex, findErr := findVethPeerIfindex()

	if err := unix.Setns(int(origNs.Fd()), unix.CLONE_NEWNET); err != nil {
		// Leak the thread: runtime.UnlockOSThread would return it to the
		// pool still pinned to the pod netns, poisoning future goroutines.
		runtime.LockOSThread()
		return 0, fmt.Errorf("%w: setns back to host: %w", ErrNetnsOpen, err)
	}

	return ifindex, findErr
}

// findVethPeerIfindex is called from inside a netns. It returns the peer
// ifindex of the first veth device it finds. CNI-created pod netns always
// have exactly one veth (eth0), so "first" is unambiguous.
func findVethPeerIfindex() (uint32, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return 0, fmt.Errorf("%w: netlink link list: %w", ErrVethLookup, err)
	}

	for _, lk := range links {
		attrs := lk.Attrs()
		if attrs == nil {
			continue
		}

		if lk.Type() != "veth" {
			continue
		}

		if attrs.ParentIndex == 0 {
			continue
		}

		return uint32(attrs.ParentIndex), nil
	}

	return 0, fmt.Errorf("%w: no veth found in pod netns", ErrVethLookup)
}
