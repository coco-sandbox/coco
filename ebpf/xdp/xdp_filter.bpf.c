// SPDX-License-Identifier: GPL-2.0
// Copyright (C) 2026 The Coco Sandbox Authors
//
// xdp_filter.bpf.c - XDP Filter with Rate Limiting (spec/03 §3.1 + §4.4)
//
// Default-deny policy with token-bucket rate limiting per sandbox.
// Rate limits are deployment-defined (spec/10).

#include <linux/bpf.h>
#include <linux/icmp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#include "common.h"
#include "maps.h"
#include "utils.h"

static __always_inline int is_vm_allowed(__u32 ip) {
    struct vm_info *vm = bpf_map_lookup_elem(&coco_vms, &ip);
    return vm && vm->active;
}

static __always_inline void refill_rate_bucket(struct rate_limit_bucket *b, __u64 now_ns) {
    __u64 last = b->last_refill_ns;
    if (last == 0 || now_ns <= last) {
        b->last_refill_ns = now_ns;
        return;
    }

    __u64 elapsed = now_ns - last;
    __u64 add = (b->rate_pps * elapsed) / NSEC_PER_SEC;
    __u64 new_tokens = b->tokens + add;
    if (new_tokens > b->burst_packets)
        new_tokens = b->burst_packets;
    b->tokens = new_tokens;
    b->last_refill_ns = now_ns;
}

static __always_inline int check_rate_limit(__u32 ip) {
    struct rate_limit_bucket *b = bpf_map_lookup_elem(&coco_rate_limits, &ip);
    if (!b)
        return 0;

    __u64 now = bpf_ktime_get_ns();
    refill_rate_bucket(b, now);

    if (b->tokens < 1)
        return -1;

    b->tokens -= 1;
    return 0;
}

SEC("xdp")
int xdp_filter_prog(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_DROP;

    __u16 proto = bpf_ntohs(eth->h_proto);

    if (proto == ETH_P_IP) {
        struct iphdr *ip = data + sizeof(*eth);
        if ((void *)(ip + 1) > data_end)
            return XDP_DROP;

        if (!is_vm_allowed(ip->daddr))
            return XDP_DROP;

        if (!is_vm_allowed(ip->saddr))
            return XDP_DROP;

        if (check_rate_limit(ip->saddr) < 0)
            return XDP_DROP;

        if (ip->protocol == IPPROTO_ICMP) {
            struct icmphdr *icmp = (void *)ip + ip->ihl * 4;
            if ((void *)(icmp + 1) > data_end)
                return XDP_DROP;

            if (icmp->type != ICMP_ECHOREPLY && icmp->type != ICMP_ECHO)
                return XDP_DROP;
        }
    }

    __u32 idx = 0;
    __u64 *counter = bpf_map_lookup_elem(&coco_stats, &idx);
    if (counter)
        (*counter)++;

    return XDP_PASS;
}

SEC("xdp")
int xdp_filter_ingress(struct xdp_md *ctx) {
    return xdp_filter_prog(ctx);
}

SEC("xdp")
int xdp_filter_egress(struct xdp_md *ctx) {
    return xdp_filter_prog(ctx);
}

char _license[] SEC("license") = "GPL";
