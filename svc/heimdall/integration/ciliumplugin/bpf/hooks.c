#include "../../../internal/network/bpf/network_helpers.h"

struct observation {
    __u64 before;
    __u64 after;
    __u64 pass;
    __u64 drop;
    __u64 redirect;
    __u64 other;
};

struct {
    __uint(type, 2);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct observation);
} observations SEC(".maps");

SEC("freplace") int before(struct __sk_buff *ctx) {
    __u32 key = 0;
    struct observation *value = bpf_map_lookup_elem(&observations, &key);
    if (value)
        __sync_fetch_and_add(&value->before, 1);
    return -1;
}

SEC("freplace") int after(struct __sk_buff *ctx, int action) {
    __u32 key = 0;
    struct observation *value = bpf_map_lookup_elem(&observations, &key);
    if (!value)
        return -1;
    __sync_fetch_and_add(&value->after, 1);
    if (action == 0)
        __sync_fetch_and_add(&value->pass, 1);
    else if (action == 2)
        __sync_fetch_and_add(&value->drop, 1);
    else if (action == 7)
        __sync_fetch_and_add(&value->redirect, 1);
    else
        __sync_fetch_and_add(&value->other, 1);
    return -1;
}

char LICENSE[] SEC("license") = "GPL";
