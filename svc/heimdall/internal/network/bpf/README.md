# heimdall network eBPF

Two `tc` programs attached via TCX on the pod-side eth0 inside each pod's
netns. Every packet going in or out runs through them, the destination IP
is classified as public or private, and one of four byte counters in a
shared BPF map keyed by pod netns cookie gets atomically incremented.
Heimdall reads those counters on its 5-second tick and writes them to
ClickHouse. Billing later computes `max(counter) - min(counter)` over any
window, same shape as CPU.

We attach inside the pod netns rather than at the pod cgroup because
`cgroup_skb` is useless under gVisor (runsc traps customer syscalls in
userspace and never hits the host socket layer) and the host-side veth
is bypassed by Cilium's `bpf_redirect_peer`. See `../doc.go` and
`docs/engineering/infra/metering/heimdall.mdx` for the full design write-up
including the `TC_ACT_UNSPEC` / `TCX_NEXT` trap that keeps DNS working
for gVisor pods.

## What each section does

| `SEC()` | What it does |
|---|---|
| `tc` (`count_egress`) | Attached as TCX-egress on pod-side eth0. Increments `egress_public` or `egress_private` keyed by destination IP. |
| `tc` (`count_ingress`) | Attached as TCX-ingress on pod-side eth0. Increments `ingress_public` or `ingress_private` keyed by source IP. |
| `license` | BPF license declaration, required for the verifier to accept the program. |

## How to regenerate

```bash
make generate-bpf
```

Runs `bpf2go` inside a pinned `linux/amd64` Docker image (defined in
`Dockerfile.gen` next to this README). The image installs a fixed
clang-18, llvm-18, and Go from go.mod, so the generated `bpf_bpfel.o`
is bytewise-identical across macOS dev machines, Linux dev machines,
and the CI drift-check runner. Without this pinning, host clang
version drift (Apple/brew clang 22 vs Ubuntu apt clang 18) silently
shifts the .o bytes and the drift check fails on perfectly correct
C source.

Requires Docker. The first run pulls Ubuntu 24.04 + the toolchain
(~10s on a warm cache). Subsequent runs reuse the local image and
take a few seconds.

The generated `bpf_bpfel.go` and `bpf_bpfel.o` in the parent directory
are committed to the repo so Go developers without Docker don't need
to regenerate just to build heimdall — they only run `make generate-bpf`
when they actually edit the C source.

## Header strategy (no vmlinux.h, no libbpf)

`network_helpers.h` is a hand-written ~120-line header covering only the
types and macros this program references: three kernel structs
(`__sk_buff`, `iphdr`, `ipv6hdr`), three BPF helpers, a few map-type
constants, and the `SEC()` macro. This follows the same pattern upstream
cilium/ebpf uses in its examples directory (see their `common.h`) and what
the kernel's own `perf` tool does for its BPF skels.

Adding a new BPF helper: append one line to `network_helpers.h` of the form

```c
static <return-type> (*bpf_new_helper)(<args>) = (void *)<id>;
```

where `<id>` is from `enum bpf_func_id` in the kernel's
`include/uapi/linux/bpf.h`. Helper IDs are frozen kernel UAPI; they never
change.

## Why no CO-RE

CO-RE (compile-once run-everywhere) relocates struct field offsets at
program load time so a program compiled against one kernel's struct
layout still works on another. It exists because internal kernel structs
(`task_struct`, `sock`, `skb_shared_info`, ...) can shift fields between
releases.

We only touch UAPI structs — `__sk_buff`, `iphdr`, `ipv6hdr` — whose
offsets are part of the kernel's ABI contract with userspace and have not
moved in 20+ years. No CO-RE needed. The day this program reaches into an
internal struct, we adopt CO-RE then.

## Runtime requirements

- Kernel 6.6 or newer (for TCX attach support; CAP_BPF was split from
  CAP_SYS_ADMIN earlier in 5.8 but TCX is the binding constraint)
- Container has `CAP_BPF` and `CAP_NET_ADMIN` (see the DaemonSet manifest)
- Process calls `rlimit.RemoveMemlock()` at startup (cilium/ebpf handles this)

## Verifying attach in a live cluster

TCX programs are visible per-interface inside each pod's netns. From the
node:

```bash
kubectl debug node/<node> -it --image=quay.io/cilium/cilium-bpftool -- \
  nsenter --net=/proc/<pause-pid>/ns/net bpftool net show
```

Expect `count_egress` and `count_ingress` listed under `tcx/egress` and
`tcx/ingress` for `eth0` respectively. If they're absent, check
`heimdall_network_attach_failures_total{reason=...}` for the failure
class.

The shared counter map is pinned at the path returned by `network.PinDir()`
(currently `/sys/fs/bpf/heimdall/v2`); inspect it with:

```bash
bpftool map dump pinned /sys/fs/bpf/heimdall/v2/pod_counters
```
