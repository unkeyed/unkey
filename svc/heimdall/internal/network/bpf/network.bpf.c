// SPDX-License-Identifier: Apache-2.0
//
// Per-pod network byte counter for heimdall billing.
//
// Two tc (sched_cls) programs attach to the host-side veth of each pod:
//   * count_egress  - runs at tc-ingress-of-veth (packets leaving the pod).
//                     Classify iph->daddr, accumulate into egress_* slots.
//   * count_ingress - runs at tc-egress-of-veth (packets entering the pod).
//                     Classify iph->saddr, accumulate into ingress_* slots.
//
// Why tc on veth instead of cgroup_skb on the pod cgroup: customer pods run
// under gVisor (runsc), which is a userspace kernel. Syscalls issued by the
// sandboxed app are trapped by runsc and never reach the host kernel, so the
// host's cgroup_skb hook only sees runsc's aggregated control-plane traffic.
// The host-side veth (Cilium's lxc<hash>) does carry real L3 packets for
// both gVisor and runc runtimes, so hooking there gives correct per-pod
// attribution uniformly.
//
// We never drop packets and always return TC_ACT_UNSPEC (TCX_NEXT) so the
// rest of the TCX chain runs - in particular Cilium's cil_from_container,
// which is where gVisor pods get their ClusterIP→pod-IP translation.
// Returning TC_ACT_OK instead would terminate the chain and break DNS.
// Errors fail open (silently undercount) per the never-overcharge invariant.

#include "network_helpers.h"

char LICENSE[] SEC("license") = "Apache-2.0";

// Per-veth byte counters. Keyed by the host-side veth's ifindex (what
// `skb->ifindex` reports for a tc program attached to that interface).
// LRU so interfaces that disappear without an explicit detach age out
// without cleanup work in userspace.
//
// 16384 entries: ~40x the default kubelet --max-pods (110 on EKS, 250 on
// larger instance types), which gives plenty of headroom for cluster-wide
// pod churn without LRU eviction kicking in. Eviction is a real
// correctness issue because it silently resets a per-veth counter without
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
    __type(key, __u32);
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

// lookup_or_init returns the counters entry for ifindex, creating a
// zero-initialised one on first sight. The explicit fast-path lookup
// keeps the steady-state cost at one hash probe per packet.
static __always_inline struct counters *lookup_or_init(__u32 ifindex) {
    struct counters *c = bpf_map_lookup_elem(&pod_counters, &ifindex);
    if (c) return c;

    struct counters zero = {0, 0, 0, 0};
    bpf_map_update_elem(&pod_counters, &ifindex, &zero, BPF_NOEXIST);
    return bpf_map_lookup_elem(&pod_counters, &ifindex);
}

// account adds skb->len to the right per-veth counter slot. `is_pod_egress`
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

    struct counters *c = lookup_or_init(skb->ifindex);
    if (!c) return;

    __u64 *slot;
    if (is_pod_egress) slot = private ? &c->egress_private  : &c->egress_public;
    else               slot = private ? &c->ingress_private : &c->ingress_public;
    __sync_fetch_and_add(slot, (__u64)skb->len);
}

// count_egress is attached to tc-ingress of the host-side veth. Packets
// arriving on that hook originate from the pod and are heading to the rest
// of the network - i.e. the pod's *egress*.
SEC("tc")
int count_egress(struct __sk_buff *skb) {
    account(skb, 1);
    return TC_ACT_UNSPEC;
}

// count_ingress is attached to tc-egress of the host-side veth. Packets
// leaving that hook are about to be delivered to the pod - i.e. the pod's
// *ingress*.
SEC("tc")
int count_ingress(struct __sk_buff *skb) {
    account(skb, 0);
    return TC_ACT_UNSPEC;
}
