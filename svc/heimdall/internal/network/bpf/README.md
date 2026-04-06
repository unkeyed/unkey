# heimdall network eBPF

Two `cgroup_skb` programs attached per customer pod cgroup. Every packet
going in or out runs through them, the destination IP is classified as
public or private, and one of four byte counters in a per-cgroup BPF map
entry gets atomically incremented. Heimdall reads those counters on its
5-second tick and writes them to ClickHouse. Billing later computes
`max(counter) - min(counter)` over any window, same shape as CPU.

## What each section does

| `SEC()` | What it does |
|---|---|
| `cgroup_skb/egress` | Packets leaving processes in this cgroup. Increments `egress_public` or `egress_private`. |
| `cgroup_skb/ingress` | Packets arriving at processes in this cgroup. Increments `ingress_public` or `ingress_private`. |
| `license` | BPF license declaration, required for the verifier to accept the program. |

## How to regenerate

```
make generate-bpf
```

Requires `clang` with BPF target support. On macOS: `brew install llvm`
(the Apple clang at `/usr/bin/clang` does not include the BPF target).
Brew installs llvm keg-only so the Makefile prepends
`/opt/homebrew/opt/llvm/bin` to `PATH` before invoking `bpf2go`. On Linux:
`apt install clang` or equivalent.

The generated `bpf_bpfel.go` and `bpf_bpfel.o` in the parent directory are
committed to the repo so Go developers without a BPF toolchain don't need
clang just to build heimdall.

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

- Kernel 5.8 or newer (for `CAP_BPF` split from `CAP_SYS_ADMIN`)
- Container has `CAP_BPF` and `CAP_NET_ADMIN` (see the DaemonSet manifest)
- Process calls `rlimit.RemoveMemlock()` at startup (cilium/ebpf handles this)

## Verifying attach in a live cluster

```
kubectl debug node/<node> -it --image=quay.io/cilium/cilium-bpftool -- \
  bpftool cgroup tree /sys/fs/cgroup/kubepods.slice/.../kubepods-pod<uid>.slice
```

Expect both `count_egress` and `count_ingress` attached, plus whatever
Cilium has at the root cgroup. If only ours is present, we displaced
Cilium and `BPF_F_ALLOW_MULTI` attach semantics need a closer look.
