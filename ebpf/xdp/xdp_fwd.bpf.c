// SPDX-License-Identifier: GPL-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>
#include <netinet/in.h>

#include "common.h"
#include "maps.h"
#include "utils.h"

SEC("xdp")
int xdp_fwd_prog(struct xdp_md *ctx) {
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

        struct flow_key key = {
            .src_ip = ip->saddr,
            .dst_ip = ip->daddr,
            .protocol = ip->protocol,
        };

        if (ip->protocol == IPPROTO_TCP) {
            struct tcphdr *tcp = (void *)ip + ip->ihl * 4;
            if ((void *)(tcp + 1) > data_end)
                return XDP_DROP;
            key.src_port = tcp->source;
            key.dst_port = tcp->dest;
        } else if (ip->protocol == IPPROTO_UDP) {
            struct udphdr *udp = (void *)ip + ip->ihl * 4;
            if ((void *)(udp + 1) > data_end)
                return XDP_DROP;
            key.src_port = udp->source;
            key.dst_port = udp->dest;
        }

        struct flow_value *flow = bpf_map_lookup_elem(&coco_flows, &key);
        if (flow) {
            flow->packets++;
            flow->bytes += data_end - data;
            flow->last_seen = bpf_ktime_get_ns();
        }
    }

    return XDP_PASS;
}

SEC("xdp")
int xdp_fwd_ingress(struct xdp_md *ctx) {
    return xdp_fwd_prog(ctx);
}

SEC("xdp")
int xdp_fwd_egress(struct xdp_md *ctx) {
    return xdp_fwd_prog(ctx);
}

char _license[] SEC("license") = "GPL";
