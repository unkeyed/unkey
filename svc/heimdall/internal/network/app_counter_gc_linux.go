//go:build linux

package network

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/cilium/ebpf"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/unkeyed/unkey/pkg/fault"
	"golang.org/x/sys/unix"
)

func pruneAppCounters(ctx context.Context, cd *containerd.Client, counters *ebpf.Map) error {
	return pruneSandboxAppCounters(ctx, counters, cd.Containers, sandboxNetnsCookie)
}

func pruneSandboxAppCounters(
	ctx context.Context,
	counters *ebpf.Map,
	listSandboxes func(context.Context, ...string) ([]containerd.Container, error),
	readCookie func(string) (uint64, bool, error),
) error {
	var candidates []uint64
	var cookie, bytes uint64
	iterator := counters.Iterate()
	for iterator.Next(&cookie, &bytes) {
		candidates = append(candidates, cookie)
	}
	if err := iterator.Err(); err != nil {
		return fault.Wrap(err, fault.Internal("inventory app counters"))
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	sandboxes, err := listSandboxes(ctx, fmt.Sprintf(`labels."%s"==%s`, criKindLabel, criKindSandbox))
	if err != nil {
		return fault.Wrap(err, fault.Internal("list sandboxes for app counter cleanup"))
	}
	live, err := liveSandboxCookies(ctx, sandboxes, readCookie)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	for _, cookie := range candidates {
		if _, ok := live[cookie]; ok {
			continue
		}
		if err := counters.Delete(cookie); err != nil && !errors.Is(err, ebpf.ErrKeyNotExist) {
			return fault.Wrap(err, fault.Internal(fmt.Sprintf("delete orphan app counter %d", cookie)))
		}
	}
	return nil
}

func liveSandboxCookies(ctx context.Context, sandboxes []containerd.Container, readCookie func(string) (uint64, bool, error)) (map[uint64]struct{}, error) {
	live := make(map[uint64]struct{}, len(sandboxes))
	for _, sandbox := range sandboxes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		spec, err := sandbox.Spec(ctx)
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal(fmt.Sprintf("read sandbox %s spec for app counter cleanup", sandbox.ID())))
		}
		if spec == nil || spec.Linux == nil {
			return nil, fmt.Errorf("sandbox %s has no linux spec for app counter cleanup", sandbox.ID())
		}
		for _, namespace := range spec.Linux.Namespaces {
			if namespace.Type == "network" && namespace.Path == "" {
				return nil, fmt.Errorf("sandbox %s has an empty network namespace path for app counter cleanup", sandbox.ID())
			}
		}
		path := networkNamespacePath(spec)
		if path == "" {
			continue
		}
		cookie, exists, err := readCookie(path)
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal(fmt.Sprintf("read sandbox %s netns cookie", sandbox.ID())))
		}
		if !exists {
			continue
		}
		if cookie == 0 {
			return nil, fmt.Errorf("sandbox %s has a zero netns cookie", sandbox.ID())
		}
		live[cookie] = struct{}{}
	}
	return live, nil
}

func sandboxNetnsCookie(path string) (cookie uint64, exists bool, err error) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		var original, target *os.File
		original, err = os.Open("/proc/thread-self/ns/net")
		if err != nil {
			return
		}
		defer func() { err = errors.Join(err, original.Close()) }()
		target, err = os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			err = nil
			return
		}
		if err != nil {
			return
		}
		defer func() { err = errors.Join(err, target.Close()) }()
		if err = unix.Setns(int(target.Fd()), unix.CLONE_NEWNET); err != nil {
			return
		}

		cookie, err = readNetnsCookie()
		// A failed restore leaves the thread locked so the Go runtime retires
		// it when this goroutine exits rather than reusing the pod namespace.
		err = errors.Join(err, restoreNetns(original))
		exists = err == nil
	}()
	<-done
	return cookie, exists, err
}
