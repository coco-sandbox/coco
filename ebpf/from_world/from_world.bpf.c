// SPDX-License-Identifier: (GPL-2.0-only OR Apache-2.0)
// Copyright (C) 2026 The Coco Sandbox Authors

// from_world.bpf.c - Ingress traffic handling + DNAT
// Hook: TC ingress on host interface
// Transforms: external IP/port → internal sandbox IP/port (DNAT)
// Looks up reverse session from ingress_sessions map

#include "maps.h"
#include "common.h"

/* Reverse lookup key - swap dest and source from original connection */
static __always_inline void build_reverse_key(struct tuple5 *rev, struct tuple5 *orig) {
    rev->proto = orig->proto;
    /* Swap addresses */
    rev->saddr[0] = orig->daddr[0];
    rev->saddr[1] = orig->daddr[1];
    rev->saddr[2] = orig->daddr[2];
    rev->saddr[3] = orig->daddr[3];
    rev->daddr[0] = orig->saddr[0];
    rev->daddr[1] = orig->saddr[1];
    rev->daddr[2] = orig->saddr[2];
    rev->daddr[3] = orig->saddr[3];
    /* Swap ports */
    rev->sport = orig->dport;
    rev->dport = orig->sport;
}

SEC("tc")
int from_world(struct __sk_buff *skb) {
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;

    /* Parse Ethernet header */
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;

    /* Check for IPv4 */
    if (eth->h_proto != __builtin_bswap16(0x0800))
        return TC_ACT_OK;

    /* Parse IPv4 header */
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return TC_ACT_OK;

    /* Only handle TCP/UDP */
    __u8 proto = ip->protocol;
    if (proto != IPPROTO_TCP && proto != IPPROTO_UDP)
        return TC_ACT_OK;

    /* Build forward 5-tuple key from packet (destination = host, source = external) */
    struct tuple5 forward_key = {};
    forward_key.proto = proto;
    forward_key.saddr[0] = (__u8)(ip->saddr & 0xFF);
    forward_key.saddr[1] = (__u8)((ip->saddr >> 8) & 0xFF);
    forward_key.saddr[2] = (__u8)((ip->saddr >> 16) & 0xFF);
    forward_key.saddr[3] = (__u8)((ip->saddr >> 24) & 0xFF);
    forward_key.daddr[0] = (__u8)(ip->daddr & 0xFF);
    forward_key.daddr[1] = (__u8)((ip->daddr >> 8) & 0xFF);
    forward_key.daddr[2] = (__u8)((ip->daddr >> 16) & 0xFF);
    forward_key.daddr[3] = (__u8)((ip->daddr >> 24) & 0xFF);

    __u16 sport = 0, dport = 0;
    if (proto == IPPROTO_TCP) {
        struct tcphdr *tcp = (void *)(ip + 1);
        if ((void *)(tcp + 1) > data_end)
            return TC_ACT_OK;
        sport = tcp->source;
        dport = tcp->dest;
    } else if (proto == IPPROTO_UDP) {
        struct udphdr *udp = (void *)(ip + 1);
        if ((void *)(udp + 1) > data_end)
            return TC_ACT_OK;
        sport = udp->source;
        dport = udp->dest;
    }
    forward_key.sport = sport;
    forward_key.dport = dport;

    /* Look up DNAT entry using reverse 5-tuple (host:port → sandbox) */
    struct dnat_entry *dnat = bpf_map_lookup_elem(&ingress_sessions, &forward_key);
    if (!dnat) {
        /* No session found - could be new connection or unmatched */
        /* For now, let it through - coconet will handle routing */
        return TC_ACT_OK;
    }

    /* Found a translation entry - DNAT to internal sandbox IP */
    __u32 internal_addr = (__u32)dnat->internal_addr[0] |
                        ((__u32)dnat->internal_addr[1] << 8) |
                        ((__u32)dnat->internal_addr[2] << 16) |
                        ((__u32)dnat->internal_addr[3] << 24);
    __u16 internal_port = dnat->internal_port;

    /* Update destination IP in packet */
    ip->daddr = internal_addr;

    /* Update destination port in transport header */
    if (proto == IPPROTO_TCP) {
        struct tcphdr *tcp = (void *)(ip + 1);
        tcp->dest = internal_port;
    } else {
        struct udphdr *udp = (void *)(ip + 1);
        udp->dest = internal_port;
    }

    /* Recompute IP checksum after DNAT */
    __u16 *csum = (__u16 *)&(ip->check);
    *csum = 0;
    *csum = ip_fast_csum(ip, ip->ihl);

    /* Get ifindex for this sandbox from the ingress session */
    /* Note: bpf_sk_fullsock not available in tc context, use skb->ifindex */

    return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "GPL";
