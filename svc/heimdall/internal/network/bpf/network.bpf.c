// SPDX-License-Identifier: Apache-2.0
//
// Per-pod network byte counter for heimdall billing.
//
// Two tc (sched_cls) programs attach to the POD-SIDE eth0 inside each
// pod's netns:
//   * count_egress  - runs at tc-egress-of-eth0  (packets leaving the pod).
//                     Classify iph->daddr, accumulate into egress_* slots.
//   * count_ingress - runs at tc-ingress-of-eth0 (packets entering the pod).
//                     Classify iph->saddr, accumulate into ingress_* slots.
//
// Why pod-side eth0 instead of the host-side veth: on EKS the CNI is
// Cilium in native / BPF-host-routing mode. Cilium's bpf_redirect_peer()
// punts packets directly into the target pod netns from cil_from_container
// on the host NIC, so the per-pod host-side veth (lxc<hash> / eni<hash>)
// never sees data packets at all — a TCX hook there reads zero bytes.
// Verified in Cilium source (bpf/lib/local_delivery.h + bpf_lxc.c) and
// in production (host-side veth RX/TX counters sit at ~700 bytes while
// the pod is actively moving MBs). Pod-side eth0 is the one place every
// packet must cross, independent of CNI routing choices and of whether
// the workload is runc or gVisor (runsc writes to eth0 via AF_PACKET).
//
// Map keying: pod-side eth0 is ifindex 3 in every CNI netns, so we can
// not key by skb->ifindex — collisions across pods. Instead we key by
// bpf_get_netns_cookie(skb), a per-netns 64-bit identifier the kernel
// maintains for this purpose. Userspace reads the same cookie via the
// SO_NETNS_COOKIE socket option after setns'ing into the pod netns.
//
// We never drop packets and always return TC_ACT_UNSPEC (TCX_NEXT) so
// the rest of the TCX chain runs - in particular any Cilium hook the
// peer-redirect path may eventually install on the pod-side device.
// Returning TC_ACT_OK instead would terminate the chain.
// Errors fail open (silently undercount) per the never-overcharge invariant.

#include "network_helpers.h"

char LICENSE[] SEC("license") = "Apache-2.0";

// POD_KEY is baked into each program's .rodata at load time by the Go
// loader (one fresh program pair per pod, each with its own constant). We
// use it as the map key instead of bpf_get_netns_cookie(skb) because that
// helper returns init_net's cookie at tc-ingress on pod-side eth0:
// ingress skbs delivered via Cilium's bpf_redirect_peer (BPF host-routing
// on EKS) land in the pod netns with skb->sk == NULL, and the kernel
// helper falls back to init_net. Every pod's ingress traffic then
// conflates into one map slot keyed by init_net cookie. Baking a
// per-program constant sidesteps the helper entirely, so egress and
// ingress write to the same slot, and each pod gets its own slot.
// See network_linux.go attachSync + specForPod for the load path.
//
// `volatile` keeps the compiler from folding the zero initialiser into
// constant loads; `const` + the default-zero init place the symbol in
// .rodata where cilium/ebpf's CollectionSpec.Variables can rewrite it
// before LoadAndAssign.
volatile const __u64 POD_KEY = 0;

// Per-pod byte counters. Keyed by POD_KEY (see above) so one pinned map
// serves every pod with no collisions. LRU so entries for dead pods age
// out without explicit cleanup — the slot is reclaimed once no program
// writes to it.
//
// 16384 entries: ~40x the default kubelet --max-pods (110 on EKS, 250 on
// larger instance types), which gives plenty of headroom for cluster-wide
// pod churn without LRU eviction kicking in. Eviction is a real
// correctness issue because it silently resets a per-pod counter without
// a corresponding container_uid change, breaking the monotonicity
// invariant billing math depends on — the huge margin is cheap insurance.
// Memory cost: 16384 × 32 bytes = 512 KiB of kernel memory per node.
struct counters {
    __u64 egress_public;
    __u64 egress_private;
    __u64 ingress_public;
    __u64 ingress_private;
};

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __type(key, __u64);
    __type(value, struct counters);
    __uint(max_entries, 16384);
    // Pin the map in bpffs so it survives heimdall pod restarts. Without
    // pinning, restarting heimdall reloads the BPF program and gets a
    // fresh zero-filled map, so per-pod byte counters appear to reset
    // across the restart. Billing queries that rely on counter monotonicity
    // would silently undercount whatever traffic crossed the gap. The Go
    // loader must pass a matching PinPath so the existing pinned map is
    // reused rather than replaced.
    __uint(pinning, LIBBPF_PIN_BY_NAME);
} pod_counters SEC(".maps");

// is_v4_private returns 1 if addr is in RFC1918, CGNAT, link-local,
// loopback, multicast, "this network", or limited-broadcast space — i.e.
// anything that is not billable public-internet egress.
// `addr` is __be32 straight out of the IP header; reading it byte-by-byte
// gives a.b.c.d directly on any endianness. All cluster CIDRs sit under
// 10.0.0.0/8.
static __always_inline int is_v4_private(__be32 addr) {
    const __u8 *b = (const __u8 *)&addr;
    if (b[0] == 0)                                 return 1;  // 0.0.0.0/8 "this network" (RFC 1122)
    if (b[0] == 10)                                return 1;  // 10.0.0.0/8
    if (b[0] == 127)                               return 1;  // loopback
    if (b[0] == 172 && (b[1] & 0xF0) == 0x10)      return 1;  // 172.16.0.0/12
    if (b[0] == 192 && b[1] == 168)                return 1;  // 192.168.0.0/16
    if (b[0] == 169 && b[1] == 254)                return 1;  // link-local
    if (b[0] == 100 && (b[1] & 0xC0) == 0x40)      return 1;  // 100.64.0.0/10 CGNAT
    if ((b[0] & 0xF0) == 0xE0)                     return 1;  // 224.0.0.0/4 multicast
    if (b[0] == 255 && b[1] == 255 && b[2] == 255 && b[3] == 255)
                                                   return 1;  // 255.255.255.255 limited broadcast
    return 0;
}

// is_v6_private returns 1 if `addr` is in any of:
//   ::1/128 (loopback), fe80::/10 (link-local), fc00::/7 (unique local),
//   ff00::/8 (multicast - not billable internet egress).
static __always_inline int is_v6_private(const struct in6_addr *addr) {
    __u8 b0 = addr->in6_u.u6_addr8[0];
    __u8 b1 = addr->in6_u.u6_addr8[1];

    if (b0 == 0xFF)                         return 1;  // ff00::/8
    if (b0 == 0xFE && (b1 & 0xC0) == 0x80)  return 1;  // fe80::/10
    if ((b0 & 0xFE) == 0xFC)                return 1;  // fc00::/7

    // ::1/128 - four u32 words, first three zero, last is htonl(1).
    if (addr->in6_u.u6_addr32[0] == 0 &&
        addr->in6_u.u6_addr32[1] == 0 &&
        addr->in6_u.u6_addr32[2] == 0 &&
        addr->in6_u.u6_addr32[3] == bpf_htonl(1)) return 1;

    return 0;
}

// lookup_or_init returns the counters entry for `key`, creating a
// zero-initialised one on first sight. The explicit fast-path lookup
// keeps the steady-state cost at one hash probe per packet.
static __always_inline struct counters *lookup_or_init(__u64 key) {
    struct counters *c = bpf_map_lookup_elem(&pod_counters, &key);
    if (c) return c;

    struct counters zero = {0, 0, 0, 0};
    bpf_map_update_elem(&pod_counters, &key, &zero, BPF_NOEXIST);
    return bpf_map_lookup_elem(&pod_counters, &key);
}

// account adds skb->len to the right per-pod counter slot. `is_pod_egress`
// selects which end of the flow to classify and which counter pair to hit.
static __always_inline void account(struct __sk_buff *skb, int is_pod_egress) {
    void *data     = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // tc on veth sees L2-framed skbs. Skip the Ethernet header and use
    // skb->protocol (already big-endian) to dispatch without a bswap.
    data += ETH_HLEN;
    if (data > data_end) return;

    int private;
    if (skb->protocol == bpf_htons(ETH_P_IP)) {
        struct iphdr *iph = data;
        if ((void *)(iph + 1) > data_end) return;
        private = is_v4_private(is_pod_egress ? iph->daddr : iph->saddr);
    } else if (skb->protocol == bpf_htons(ETH_P_IPV6)) {
        struct ipv6hdr *ip6h = data;
        if ((void *)(ip6h + 1) > data_end) return;
        private = is_v6_private(is_pod_egress ? &ip6h->daddr : &ip6h->saddr);
    } else {
        return;
    }

    // POD_KEY is baked into this program's .rodata at load time (see the
    // comment on the symbol declaration at the top of the file). Reading
    // a rodata constant is a single memory load — faster than the helper
    // call the previous version used, and crucially: same value for both
    // egress and ingress (the helper disagreed across directions).
    struct counters *c = lookup_or_init(POD_KEY);
    if (!c) return;

    __u64 *slot;
    if (is_pod_egress) slot = private ? &c->egress_private  : &c->egress_public;
    else               slot = private ? &c->ingress_private : &c->ingress_public;
    __sync_fetch_and_add(slot, (__u64)skb->len);
}

// count_egress is attached to tc-egress of pod-side eth0. Packets leaving
// that hook originate from the pod and are heading out - i.e. the pod's
// *egress*. Classify by destination IP.
SEC("tc")
int count_egress(struct __sk_buff *skb) {
    account(skb, 1);
    return TC_ACT_UNSPEC;
}

// count_ingress is attached to tc-ingress of pod-side eth0. Packets arriving
// on that hook are about to be delivered to the pod - i.e. the pod's
// *ingress*. Classify by source IP.
SEC("tc")
int count_ingress(struct __sk_buff *skb) {
    account(skb, 0);
    return TC_ACT_UNSPEC;
}
