// SPDX-License-Identifier: GPL-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#include "common.h"
#include "maps.h"

enum stats_index {
    STATS_PACKETS_TOTAL = 0,
    STATS_BYTES_TOTAL = 1,
    STATS_PACKETS_DROPPED = 2,
    STATS_PACKETS_ALLOWED = 3,
    STATS_TCP_PACKETS = 4,
    STATS_UDP_PACKETS = 5,
    STATS_ICMP_PACKETS = 6,
    STATS_OTHER_PACKETS = 7,
};

static __always_inline void increment_stat(__u32 idx) {
    __u64 *val = bpf_map_lookup_elem(&coco_stats, &idx);
    if (val)
        (*val)++;
}

SEC("xdp")
int xdp_stats_prog(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    increment_stat(STATS_PACKETS_TOTAL);

    struct ethhdr *eth = data;
    if (eth + 1 > data_end) {
        increment_stat(STATS_PACKETS_DROPPED);
        return XDP_DROP;
    }

    __u16 proto = bpf_ntohs(eth->h_proto);
    if (proto == ETH_P_IP) {
        struct iphdr *ip = data + sizeof(*eth);
        if ((void *)(ip + 1) > data_end) {
            increment_stat(STATS_PACKETS_DROPPED);
            return XDP_DROP;
        }

        if (ip->protocol == IPPROTO_TCP)
            increment_stat(STATS_TCP_PACKETS);
        else if (ip->protocol == IPPROTO_UDP)
            increment_stat(STATS_UDP_PACKETS);
        else if (ip->protocol == IPPROTO_ICMP)
            increment_stat(STATS_ICMP_PACKETS);
        else
            increment_stat(STATS_OTHER_PACKETS);
    }

    increment_stat(STATS_PACKETS_ALLOWED);
    return XDP_PASS;
}

SEC("xdp")
int xdp_stats_ingress(struct xdp_md *ctx) {
    return xdp_stats_prog(ctx);
}

SEC("xdp")
int xdp_stats_egress(struct xdp_md *ctx) {
    return xdp_stats_prog(ctx);
}

char _license[] SEC("license") = "GPL";
