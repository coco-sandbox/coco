/* SPDX-License-Identifier: (GPL-2.0-only OR Apache-2.0) */
/* Copyright (C) 2026 The Coco Sandbox Authors */

#ifndef COCO_COMMON_H
#define COCO_COMMON_H

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/icmp.h>
#include <stddef.h>

/* Tuple for 5-tuple session lookup */
struct tuple5 {
    __u8  proto;
    __u8  _pad0[3];
    __u8  saddr[4];   /* Source address (IPv4) */
    __u8  daddr[4];   /* Destination address */
    __u16 sport;      /* Source port (network order) */
    __u16 dport;     /* Destination port (network order) */
};

/* SNAT entry stored in map */
struct snat_entry {
    __u8  translated_addr[4]; /* NAT'd source address */
    __u16 translated_port;      /* NAT'd source port */
    __u16 orig_port;            /* Original source port */
    __u32 timestamp;            /* For session timeout */
};

/* DNAT entry for ingress translation */
struct dnat_entry {
    __u8  internal_addr[4];  /* Sandbox internal IP */
    __u16 internal_port;     /* Sandbox internal port */
    __u16 orig_port;         /* Original destination port */
    __u32 timestamp;
};

/* Metadata passed via BPF context */
struct metadata {
    __u32 sandbox_id;
    __u32 action;   /* 0=allow, 1=deny, 2=log */
    __u64 timestamp;
};

/* Maximum MTU we support */
#define COCO_MAX_MTU 4096

/* Session timeout in seconds */
#define SESSION_TIMEOUT 300

/* TCP state tracking */
enum tcp_state {
    TCP_CLOSED      = 0,
    TCP_SYN_SENT    = 1,
    TCP_SYN_ACKED   = 2,
    TCP_ESTABLISHED  = 3,
    TCP_FIN_WAIT    = 4,
    TCP_CLOSING     = 5,
};

/* Helper: check if IP is private */
static __always_inline int is_private_ip(__u8 *addr)
{
    /* 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 */
    if (addr[0] == 10) return 1;
    if (addr[0] == 172 && addr[1] >= 16 && addr[1] <= 31) return 1;
    if (addr[0] == 192 && addr[1] == 168) return 1;
    return 0;
}

/* Helper: parse IPv4 header safely */
static __always_inline struct iphdr *parse_ip_hdr(void *data, void *data_end)
{
    struct ethhdr *eth = data;
    struct iphdr *ip;

    if ((void *)(eth + 1) > data_end)
        return NULL;

    if (eth->h_proto != __builtin_bswap16(ETH_P_IP))
        return NULL;

    ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return NULL;

    return ip;
}

/* Helper: parse TCP header safely */
static __always_inline struct tcphdr *parse_tcp_hdr(void *ip_data, void *data_end)
{
    struct tcphdr *tcp = ip_data;
    if ((void *)(tcp + 1) > data_end)
        return NULL;
    return tcp;
}

/* Helper: parse UDP header safely */
static __always_inline struct udphdr *parse_udp_hdr(void *ip_data, void *data_end)
{
    struct udphdr *udp = ip_data;
    if ((void *)(udp + 1) > data_end)
        return NULL;
    return udp;
}

/* Helper: compute IP checksum */
static __always_inline void ip_checksum(struct iphdr *ip)
{
    __u32 sum = 0;
    __u16 *p = (__u16 *)ip;
    int i;

    ip->check = 0;

    for (i = 0; i < (int)(ip->ihl) * 2; i++) {
        sum += __builtin_bswap16(p[i]);
    }

    while (sum >> 16) {
        sum = (sum & 0xFFFF) + (sum >> 16);
    }

    ip->check = ~__builtin_bswap16(sum);
}

/* Helper: compute TCP/UDP pseudo-header checksum */
static __always_inline __u16 l4_checksum(void *l3_hdr, void *l4_hdr, __u32 len)
{
    __u32 sum = 0;
    __u16 *p = (__u16 *)l4_hdr;
    int i, hdr_len = 20; /* minimum TCP/UDP header */

    /* Add pseudo-header */
    struct {
        __u32 src;
        __u32 dst;
        __u8  zero;
        __u8  proto;
        __u16 length;
    } __attribute__((packed)) pseudo;

    pseudo.src = ((struct iphdr *)l3_hdr)->saddr;
    pseudo.dst = ((struct iphdr *)l3_hdr)->daddr;
    pseudo.zero = 0;
    pseudo.proto = ((struct iphdr *)l3_hdr)->protocol;
    pseudo.length = __builtin_bswap16(len);

    p = (__u16 *)&pseudo;
    for (i = 0; i < 6; i++) {
        sum += __builtin_bswap16(p[i]);
    }

    /* Add L4 header and payload */
    p = (__u16 *)l4_hdr;
    for (i = 0; i < (int)(hdr_len / 2); i++) {
        sum += __builtin_bswap16(p[i]);
    }

    while (sum >> 16) {
        sum = (sum & 0xFFFF) + (sum >> 16);
    }

    return ~__builtin_bswap16(sum);
}

#endif /* COCO_COMMON_H */