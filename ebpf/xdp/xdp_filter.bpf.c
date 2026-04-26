// SPDX-License-Identifier: GPL-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#include "common.h"
#include "maps.h"
#include "utils.h"

static __always_inline int is_vm_allowed(__u32 ip) {
    struct vm_info *vm = bpf_map_lookup_elem(&coco_vms, &ip);
    return vm && vm->active;
}

SEC("xdp")
int xdp_filter_prog(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if (eth + 1 > data_end)
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
