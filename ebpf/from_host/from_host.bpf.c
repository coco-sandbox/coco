// SPDX-License-Identifier: GPL-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>
#include <linux/if_ether.h>
#include <linux/ip.h>

#define ETH_P_COCO 0xFFFF

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);
    __type(value, __u32);
    __uint(max_entries, 256);
} vm_routes SEC(".maps");

SEC("xdp")
int from_host_prog(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if (eth + 1 > data_end)
        return XDP_DROP;

    __u16 proto = bpf_ntohs(eth->h_proto);

    if (proto == ETH_P_COCO) {
        return XDP_PASS;
    }

    if (proto == ETH_P_IP) {
        struct iphdr *ip = data + sizeof(*eth);
        if ((void *)(ip + 1) > data_end)
            return XDP_DROP;

        __u32 *route = bpf_map_lookup_elem(&vm_routes, &ip->daddr);
        if (route) {
            return XDP_TX;
        }
    }

    return XDP_PASS;
}

char _license[] SEC("license") = "GPL";
