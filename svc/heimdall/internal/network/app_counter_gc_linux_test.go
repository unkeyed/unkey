//go:build linux

package network

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/cilium/ebpf"
	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestPruneSandboxAppCountersPreservesLiveValues(t *testing.T) {
	for _, liveCount := range []int{0, 1, 3} {
		t.Run(strconv.Itoa(liveCount)+" live", func(t *testing.T) {
			counters := loadTestObjects(t).AppEgress
			initial := map[uint64]uint64{1: 0, 2: 999, 1 << 40: 1 << 50}
			for cookie, bytes := range initial {
				require.NoError(t, counters.Put(cookie, bytes))
			}
			cookies := []uint64{1, 2, 1 << 40}[:liveCount]
			var sandboxes []containerd.Container
			for range cookies {
				sandboxes = append(sandboxes, fakeSandbox{netns: "/live"})
			}
			sandboxes = append(sandboxes,
				fakeSandbox{netns: "/gone"},
				appCounterSpecSandbox{spec: &oci.Spec{Linux: &specs.Linux{}}},
			)
			list := func(ctx context.Context, filters ...string) ([]containerd.Container, error) {
				namespace, ok := namespaces.Namespace(ctx)
				require.True(t, ok)
				require.Equal(t, "k8s.io", namespace)
				require.Equal(t, []string{`labels."io.cri-containerd.kind"==sandbox`}, filters)
				return sandboxes, nil
			}
			index := 0
			readCookie := func(path string) (uint64, bool, error) {
				if path == "/gone" {
					return 0, false, nil
				}
				cookie := cookies[index]
				index++
				return cookie, true, nil
			}

			require.NoError(t, pruneSandboxAppCounters(t.Context(), counters, list, readCookie))
			want := make(map[uint64]uint64)
			for _, cookie := range cookies {
				want[cookie] = initial[cookie]
			}
			got, err := appCounterValues(counters)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestPruneSandboxAppCountersDoesNotDeleteOnInventoryFailure(t *testing.T) {
	inventoryErr := errors.New("inventory unavailable")
	for _, stage := range []string{"list", "spec", "empty namespace", "cookie", "cancellation"} {
		t.Run(stage, func(t *testing.T) {
			counters := loadTestObjects(t).AppEgress
			initial := map[uint64]uint64{1: 101, 2: 202, 3: 303}
			for cookie, bytes := range initial {
				require.NoError(t, counters.Put(cookie, bytes))
			}
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			list := func(context.Context, ...string) ([]containerd.Container, error) {
				if stage == "list" {
					return nil, inventoryErr
				}
				sandboxes := []containerd.Container{fakeSandbox{netns: "/live"}}
				if stage == "spec" {
					sandboxes = append(sandboxes, appCounterSpecSandbox{err: inventoryErr})
				} else if stage == "empty namespace" {
					sandboxes = append(sandboxes, fakeSandbox{})
				} else {
					sandboxes = append(sandboxes, fakeSandbox{netns: "/uncertain"})
				}
				return sandboxes, nil
			}
			readCookie := func(path string) (uint64, bool, error) {
				if path == "/live" {
					return 1, true, nil
				}
				if stage == "cancellation" {
					cancel()
					return 2, true, nil
				}
				return 0, false, inventoryErr
			}

			err := pruneSandboxAppCounters(ctx, counters, list, readCookie)
			if stage == "cancellation" {
				require.ErrorIs(t, err, context.Canceled)
			} else if stage == "empty namespace" {
				require.Error(t, err)
			} else {
				require.ErrorIs(t, err, inventoryErr)
			}
			got, err := appCounterValues(counters)
			require.NoError(t, err)
			require.Equal(t, initial, got)
		})
	}
}

func TestPruneSandboxAppCountersDoesNotDeleteCountersCreatedDuringInventory(t *testing.T) {
	counters := loadTestObjects(t).AppEgress
	require.NoError(t, counters.Put(uint64(1), uint64(123)))
	list := func(context.Context, ...string) ([]containerd.Container, error) {
		require.NoError(t, counters.Put(uint64(2), uint64(456)))
		return nil, nil
	}

	require.NoError(t, pruneSandboxAppCounters(t.Context(), counters, list, sandboxNetnsCookie))
	got, err := appCounterValues(counters)
	require.NoError(t, err)
	require.Equal(t, map[uint64]uint64{2: 456}, got)
}

func TestLiveSandboxCookiesPreservesSharedNamespaces(t *testing.T) {
	live, err := liveSandboxCookies(t.Context(), []containerd.Container{
		fakeSandbox{netns: "/live"},
		fakeSandbox{netns: "/gone"},
		fakeSandbox{netns: "/live"},
	}, func(path string) (uint64, bool, error) {
		if path == "/gone" {
			return 0, false, nil
		}
		return 1 << 40, true, nil
	})
	require.NoError(t, err)
	require.Equal(t, map[uint64]struct{}{1 << 40: {}}, live)
}

func TestLiveSandboxCookiesExcludesHostNetworkSandboxes(t *testing.T) {
	for _, test := range []struct {
		name       string
		namespaces []specs.LinuxNamespace
	}{
		{name: "no namespaces"},
		{name: "other namespaces", namespaces: []specs.LinuxNamespace{{Type: specs.PIDNamespace, Path: "/pid"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			reads := 0
			live, err := liveSandboxCookies(t.Context(), []containerd.Container{
				appCounterSpecSandbox{spec: &oci.Spec{Linux: &specs.Linux{Namespaces: test.namespaces}}},
				fakeSandbox{netns: "/live"},
			}, func(path string) (uint64, bool, error) {
				require.Equal(t, "/live", path)
				reads++
				return 1, true, nil
			})
			require.NoError(t, err)
			require.Equal(t, map[uint64]struct{}{1: {}}, live)
			require.Equal(t, 1, reads)
		})
	}
}

func TestLiveSandboxCookiesRejectsIncompleteInventory(t *testing.T) {
	for _, test := range []struct {
		name    string
		sandbox containerd.Container
	}{
		{name: "spec read failure", sandbox: appCounterSpecSandbox{err: os.ErrNotExist}},
		{name: "nil spec", sandbox: appCounterSpecSandbox{}},
		{name: "missing linux", sandbox: appCounterSpecSandbox{spec: &oci.Spec{}}},
		{name: "empty network namespace path", sandbox: fakeSandbox{}},
		{name: "empty duplicate network namespace path", sandbox: appCounterSpecSandbox{spec: &oci.Spec{Linux: &specs.Linux{
			Namespaces: []specs.LinuxNamespace{
				{Type: specs.NetworkNamespace, Path: "/live"},
				{Type: specs.NetworkNamespace, Path: ""},
			},
		}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			reads := 0
			readCookie := func(string) (uint64, bool, error) {
				reads++
				return 1, true, nil
			}
			live, err := liveSandboxCookies(t.Context(), []containerd.Container{
				fakeSandbox{netns: "/live"}, test.sandbox,
			}, readCookie)
			require.Error(t, err)
			require.Nil(t, live)
			require.Equal(t, 1, reads)
		})
	}
}

func TestLiveSandboxCookiesRejectsCookieErrors(t *testing.T) {
	for _, cookieErr := range []error{unix.EPERM, unix.ENOENT, nil} {
		live, err := liveSandboxCookies(t.Context(), []containerd.Container{fakeSandbox{netns: "/live"}},
			func(string) (uint64, bool, error) { return 0, true, cookieErr })
		require.Error(t, err)
		require.Nil(t, live)
		if cookieErr != nil {
			require.ErrorIs(t, err, cookieErr)
		}
	}
}

func TestSandboxNetnsCookieSkipsOnlyMissingNamespace(t *testing.T) {
	cookie, exists, err := sandboxNetnsCookie(filepath.Join(t.TempDir(), "gone"))
	require.NoError(t, err)
	require.False(t, exists)
	require.Zero(t, cookie)

	path := filepath.Join(t.TempDir(), "not-a-namespace")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	cookie, exists, err = sandboxNetnsCookie(path)
	require.Error(t, err)
	require.False(t, exists)
	require.Zero(t, cookie)
}

type appCounterSpecSandbox struct {
	fakeSandbox
	spec *oci.Spec
	err  error
}

func (sandbox appCounterSpecSandbox) Spec(context.Context) (*oci.Spec, error) {
	return sandbox.spec, sandbox.err
}

func appCounterValues(counters *ebpf.Map) (map[uint64]uint64, error) {
	values := make(map[uint64]uint64)
	var cookie, bytes uint64
	iterator := counters.Iterate()
	for iterator.Next(&cookie, &bytes) {
		values[cookie] = bytes
	}
	return values, iterator.Err()
}
