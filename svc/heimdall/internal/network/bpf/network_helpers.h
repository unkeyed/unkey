// SPDX-License-Identifier: Apache-2.0
//
// Self-contained header for network.bpf.c. Replaces ~4.5 MB of vendored
// kernel + libbpf headers with the minimal subset our program actually
// references. If the program ever needs more BPF helpers, more kernel
// structs, or more map types, extend this file.
//
// Why hand-written instead of vmlinux.h + libbpf:
//   - vmlinux.h is a 4.3 MB BTF dump; we use 3 structs from it.
//   - libbpf's bpf_helper_defs.h declares ~200 helpers; we use 3.
//   - clang on macOS has no kernel headers, so including <linux/...>
//     directly doesn't work. A vendored slim header runs everywhere.

#ifndef __NETWORK_HELPERS_H__
#define __NETWORK_HELPERS_H__

// ─── primitive types ──────────────────────────────────────────────────
// bpf2go's clang invocation has no <linux/types.h>. Provide them here.

typedef unsigned char        __u8;
typedef unsigned short       __u16;
typedef unsigned int         __u32;
typedef unsigned long long   __u64;
typedef __u16                __be16;
typedef __u32                __be32;

// ─── BPF program section macro ────────────────────────────────────────
// Tells the loader which kernel attach point this program belongs to.

#define SEC(name) __attribute__((section(name), used))

// ─── BPF map definition macros (libbpf-style) ─────────────────────────
// `__uint(type, x)` and `__type(key, x)` declare a struct field whose
// *type's array dimension* encodes the value, and the BTF emitter reads
// that back to populate the map's metadata. Standard libbpf idiom.

#define __uint(name, val) int (*name)[val]
#define __type(name, val) typeof(val) *name

// ─── BPF map type constants ───────────────────────────────────────────
// LRU hash is the only map type we use. Per-CPU variants are not needed
// because byte counts are large enough that __sync_fetch_and_add
// contention is negligible at our packet rates.

#define BPF_MAP_TYPE_LRU_HASH 9

// ─── BPF map flags ────────────────────────────────────────────────────

#define BPF_NOEXIST 1  // create only; fail if key exists

// ─── libbpf-style map pinning ────────────────────────────────────────
// When a map is declared with `__uint(pinning, LIBBPF_PIN_BY_NAME)`,
// the loader attaches it to a path in bpffs named after the map. If the
// path already exists (from a previous process), the loader reuses the
// existing map rather than creating a new one. That's how we preserve
// per-pod byte counters across heimdall pod restarts.

#define LIBBPF_PIN_NONE 0
#define LIBBPF_PIN_BY_NAME 1

// ─── tc action codes ──────────────────────────────────────────────────
// Return values from SEC("tc") programs under TCX. In a multi-program
// chain the return code decides what happens after our program runs:
//   TC_ACT_UNSPEC (-1) = TCX_NEXT - non-terminating, hand off to the
//                        next program in the chain. This is what an
//                        *observer* must return; returning anything
//                        else short-circuits later programs, including
//                        Cilium's cil_from_container, which is where
//                        gVisor pods get their ClusterIP→pod-IP
//                        translation (gVisor bypasses Cilium's
//                        socket-LB, so breaking that breaks DNS).
//   TC_ACT_OK      (0) = TCX_PASS - accept AND terminate the chain.
//                        Do not use in an observer.

#define TC_ACT_UNSPEC (-1)
#define TC_ACT_OK 0

// ─── BPF helper function numbers + declarations ───────────────────────
// Helper IDs are stable kernel UAPI; signatures match `enum bpf_func_id`
// and `bpf_helper_defs.h`. Adding a new helper = one entry below.

static void *(*bpf_map_lookup_elem)(void *map, const void *key) = (void *)1;
static long (*bpf_map_update_elem)(void *map, const void *key, const void *value, __u64 flags) = (void *)2;

// ─── Endian conversion ────────────────────────────────────────────────
// IP headers come in network byte order (big-endian). BPF runs in CPU
// byte order. On all our targets (x86_64, aarch64) that's little-endian,
// so we byte-swap. __builtin_bswap32 is a clang intrinsic, no headers
// needed. bpf_htonl/htons are identical to the ntoh variants on LE but
// named after their C idiom - use bpf_htons(ETH_P_IP) for constant-order
// comparisons against fields like skb->protocol that are already big-endian.

#define bpf_ntohl(x) __builtin_bswap32(x)
#define bpf_ntohs(x) __builtin_bswap16(x)
#define bpf_htonl(x) __builtin_bswap32(x)
#define bpf_htons(x) __builtin_bswap16(x)

// ─── Ethernet protocol constants ─────────────────────────────────────
// EtherType values carried in skb->protocol (already network byte order).

#define ETH_P_IP   0x0800
#define ETH_P_IPV6 0x86DD
#define ETH_HLEN   14

// ─── Compiler hint ────────────────────────────────────────────────────

#ifndef __always_inline
#define __always_inline inline __attribute__((always_inline))
#endif

// ─── Kernel struct subsets ────────────────────────────────────────────
// Only fields our classifier reads. Layouts must match the kernel ABI
// for the L3 access at offset 0 of skb->data to land on the right bytes.
// These structs are stable kernel UAPI, ABI-frozen across versions.

// __sk_buff is the BPF program's context for cgroup_skb. Hundreds of
// fields exist; we only touch `data`, `data_end`, and `len`. Using `__u32`
// for the pointer fields matches the kernel's userspace-safe layout.
struct __sk_buff {
    __u32 len;
    __u32 pkt_type;
    __u32 mark;
    __u32 queue_mapping;
    __u32 protocol;
    __u32 vlan_present;
    __u32 vlan_tci;
    __u32 vlan_proto;
    __u32 priority;
    __u32 ingress_ifindex;
    __u32 ifindex;
    __u32 tc_index;
    __u32 cb[5];
    __u32 hash;
    __u32 tc_classid;
    __u32 data;
    __u32 data_end;
};

// IPv4 header. Bit-fields layout (ihl, version) is little-endian
// dependent in the kernel header; our targets are LE, so this matches.
struct iphdr {
    __u8  ihl     : 4;
    __u8  version : 4;
    __u8  tos;
    __be16 tot_len;
    __be16 id;
    __be16 frag_off;
    __u8  ttl;
    __u8  protocol;
    __u16 check;
    __be32 saddr;
    __be32 daddr;
};

// IPv6 address. Union mirrors the kernel's so users can read individual
// bytes via in6_u.u6_addr8 (what the classifier does).
struct in6_addr {
    union {
        __u8  u6_addr8[16];
        __u16 u6_addr16[8];
        __u32 u6_addr32[4];
    } in6_u;
};

// IPv6 header. The first byte is `version:4 + priority:4`; we read the
// version nibble out of skb->data directly via the leading byte trick
// rather than parsing this struct, so the bit-field layout doesn't
// matter for our use.
struct ipv6hdr {
    __u8  priority_version;
    __u8  flow_lbl[3];
    __be16 payload_len;
    __u8  nexthdr;
    __u8  hop_limit;
    struct in6_addr saddr;
    struct in6_addr daddr;
};

#endif // __NETWORK_HELPERS_H__
