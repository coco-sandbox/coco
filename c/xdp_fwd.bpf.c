// SPDX-License-Identifier: (GPL-2.0-only OR Apache-2.0)
// Copyright (C) 2026 The Coco Sandbox Authors

// xdp_fwd.bpf.c - AF_XDP fast path for overlay traffic
// Hook: XDP on overlay interface (cube-dev)
// Functions:
//   - Early reject of unsolicited inbound traffic
//   - DNAT rewrite of overlay IP to sandbox internal IP
//   - Fast path for known sessions

#include "maps.h"
#include "common.h"

/* XDP metadata passed between program stages */
struct xdp_meta {
    __u32 sandbox_id;
    __u32 action;  /* 0=pass, 1=drop, 2=redirect */
    __u32 new_daddr[4];
    __u16 new_dport;
};

/* Fast path for established sessions - skip policy check */
static __always_inline int fast_path_lookup(struct tuple5 *key, struct xdp_meta *meta) {
    /* Look up in ingress sessions for DNAT */
    struct dnat_entry *dnat = bpf_map_lookup_elem(&ingress_sessions, key);
    if (dnat) {
        meta->action = 2; /* redirect */
        meta->new_daddr[0] = dnat->internal_addr[0];
        meta->new_daddr[1] = dnat->internal_addr[1];
        meta->new_daddr[2] = dnat->internal_addr[2];
        meta->new_daddr[3] = dnat->internal_addr[3];
        meta->new_dport = dnat->internal_port;
        return 1;
    }
    return 0;
}

SEC("xdp")
int xdp_fwd(struct xdp_md *ctx) {
    void *data_end = (void *)(long)ctx->data_end;
    void *data = (void *)(long)ctx->data;

    /* Parse Ethernet */
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return XDP_PASS;

    /* IPv4 only */
    if (eth->h_proto != __builtin_bswap16(0x0800))
        return XDP_PASS;

    /* Parse IP header */
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return XDP_PASS;

    /* Build session key */
    struct tuple5 key = {};
    key.proto = ip->protocol;
    key.saddr[0] = (__u8)(ip->saddr & 0xFF);
    key.saddr[1] = (__u8)((ip->saddr >> 8) & 0xFF);
    key.saddr[2] = (__u8)((ip->saddr >> 16) & 0xFF);
    key.saddr[3] = (__u8)((ip->saddr >> 24) & 0xFF);
    key.daddr[0] = (__u8)(ip->daddr & 0xFF);
    key.daddr[1] = (__u8)((ip->daddr >> 8) & 0xFF);
    key.daddr[2] = (__u8)((ip->daddr >> 16) & 0xFF);
    key.daddr[3] = (__u8)((ip->daddr >> 24) & 0xFF);

    /* Get ports for TCP/UDP */
    if (ip->protocol == IPPROTO_TCP) {
        struct tcphdr *tcp = (void *)(ip + 1);
        if ((void *)(tcp + 1) > data_end)
            return XDP_PASS;
        key.sport = tcp->source;
        key.dport = tcp->dest;
    } else if (ip->protocol == IPPROTO_UDP) {
        struct udphdr *udp = (void *)(ip + 1);
        if ((void *)(udp + 1) > data_end)
            return XDP_PASS;
        key.sport = udp->source;
        key.dport = udp->dest;
    }

    /* Try fast path lookup */
    struct xdp_meta meta = {};
    if (fast_path_lookup(&key, &meta)) {
        if (meta.action == 2) {
            /* Redirect to sandbox - rewrite destination */
            __u32 new_daddr = ((__u32)meta.new_daddr[0]) |
                             ((__u32)meta.new_daddr[1] << 8) |
                             ((__u32)meta.new_daddr[2] << 16) |
                             ((__u32)meta.new_daddr[3] << 24);
            ip->daddr = new_daddr;

            if (ip->protocol == IPPROTO_TCP) {
                struct tcphdr *tcp = (void *)(ip + 1);
                tcp->dest = meta.new_dport;
            } else if (ip->protocol == IPPROTO_UDP) {
                struct udphdr *udp = (void *)(ip + 1);
                udp->dest = meta.new_dport;
            }

            /* Clear checksums for recalc */
            ip->check = 0;
            return XDP_PASS;
        } else if (meta.action == 1) {
            return XDP_DROP;
        }
    }

    /* No fast path match - do policy check */
    /* Check if destination is the gateway itself (allow) */
    /* For now, pass through to kernel networking */
    return XDP_PASS;
}

char LICENSE[] SEC("license") = "GPL";
