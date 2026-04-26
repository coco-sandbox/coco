// SPDX-License-Identifier: GPL-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#include "common.h"
#include "maps.h"
#include "utils.h"

static __always_inline struct nat_entry *lookup_snat(__u32 internal_ip, __u16 internal_port) {
    struct flow_key key = {
        .src_ip = internal_ip,
        .dst_ip = 0,
        .src_port = internal_port,
        .dst_port = 0,
        .protocol = IPPROTO_TCP,
    };
    return bpf_map_lookup_elem(&coco_nat, &key);
}

static __always_inline struct nat_entry *lookup_dnat(__u32 external_ip, __u16 external_port) {
    struct flow_key key = {
        .src_ip = 0,
        .dst_ip = external_ip,
        .src_port = 0,
        .dst_port = external_port,
        .protocol = IPPROTO_TCP,
    };
    return bpf_map_lookup_elem(&coco_nat, &key);
}

SEC("xdp")
int xdp_nat_prog(struct xdp_md *ctx) {
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

    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = (void *)ip + ip->ihl * 4;
        if ((void *)(tcp + 1) > data_end)
            return XDP_DROP;

        struct nat_entry *nat = lookup_dnat(ip->daddr, tcp->dest);
        if (nat && nat->active) {
            ip->daddr = nat->internal_ip;
            tcp->dest = nat->internal_port;
            swap_src_dst_ip(ip);
            swap_ports(tcp);
            swap_src_dst_mac(data);
        }
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = (void *)ip + ip->ihl * 4;
        if ((void *)(udp + 1) > data_end)
            return XDP_DROP;

        struct nat_entry *nat = lookup_dnat(ip->daddr, udp->dest);
        if (nat && nat->active) {
            ip->daddr = nat->internal_ip;
            udp->dest = nat->internal_port;
            swap_src_dst_ip(ip);
            swap_src_dst_mac(data);
        }
    }

    ip->check = 0;
    ip->check = ip_fast_csum(ip, ip->ihl);

    return XDP_PASS;
}

SEC("xdp")
int xdp_nat_ingress(struct xdp_md *ctx) {
    return xdp_nat_prog(ctx);
}

char _license[] SEC("license") = "GPL";
