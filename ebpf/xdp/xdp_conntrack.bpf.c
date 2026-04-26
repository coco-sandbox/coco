// SPDX-License-Identifier: GPL-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#include "common.h"
#include "maps.h"
#include "utils.h"

#define CT_STATE_NEW 0
#define CT_STATE_ESTABLISHED 1

static __always_inline int update_flow(struct flow_key *key, struct xdp_md *ctx) {
    struct flow_value *flow = bpf_map_lookup_elem(&coco_flows, key);
    if (!flow) {
        struct flow_value new_flow = {
            .packets = 1,
            .bytes = ctx->data_end - ctx->data,
            .vm_id = get_vm_id_from_ip(key->dst_ip),
            .last_seen = bpf_ktime_get_ns(),
        };
        bpf_map_update_elem(&coco_flows, key, &new_flow, BPF_NOEXIST);
    } else {
        flow->packets++;
        flow->bytes += ctx->data_end - ctx->data;
        flow->last_seen = bpf_ktime_get_ns();
    }
    return 0;
}

SEC("xdp")
int xdp_conntrack_prog(struct xdp_md *ctx) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if (eth + 1 > data_end)
        return XDP_DROP;

    __u16 proto = bpf_ntohs(eth->h_proto);
    if (proto != ETH_P_IP)
        return XDP_PASS;

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

    update_flow(&key, ctx);

    return XDP_PASS;
}

SEC("xdp")
int xdp_conntrack_ingress(struct xdp_md *ctx) {
    return xdp_conntrack_prog(ctx);
}

SEC("xdp")
int xdp_conntrack_egress(struct xdp_md *ctx) {
    return xdp_conntrack_prog(ctx);
}

char _license[] SEC("license") = "GPL";
