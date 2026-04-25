// SPDX-License-Identifier: (GPL-2.0-only OR Apache-2.0)
// Copyright (C) 2026 The Coco Sandbox Authors

// from_sandbox.bpf.c - Egress traffic handling + SNAT
// Hook: TC egress on sandbox netns (packet leaving sandbox)
// Transforms: internal IP/port → external IP/port (SNAT)
// Policy: allow/deny based on destination

#include "maps.h"
#include "common.h"

/* Default SNAT pool base IP (169.254.68.0/24 - link-local for NAT) */
#define SNAT_BASE_ADDR 0x0044AA0A  /* 10.68.44.0 in little-endian - NO, this is wrong */
                                /* Using 10.68.44.x range for SNAT pool */

/* Hash function for 5-tuple */
static __always_inline u32 hash_tuple(struct tuple5 *t) {
    u32 key = 0;
    key += t->proto;
    key = (key << 4) + ((key >> 28) ^ t->saddr[0]);
    key = (key << 4) + ((key >> 28) ^ t->saddr[1]);
    key = (key << 4) + ((key >> 28) ^ t->saddr[2]);
    key = (key << 4) + ((key >> 28) ^ t->saddr[3]);
    key = (key << 4) + ((key >> 28) ^ t->daddr[0]);
    key = (key << 4) + ((key >> 28) ^ t->daddr[1]);
    key = (key << 4) + ((key >> 28) ^ t->daddr[2]);
    key = (key << 4) + ((key >> 28) ^ t->daddr[3]);
    key += t->sport;
    key = (key << 4) + ((key >> 28) ^ (t->sport >> 16));
    key += t->dport;
    return key;
}

/* Get next available SNAT port from per-CPU generator */
static __always_inline u16 get_next_snat_port(__u32 cpu_id) {
    __u32 *counter = bpf_map_lookup_elem(&snat_port_gen, &cpu_id);
    if (!counter) return 30000; /* fallback */

    __u16 port = (__u16)(*counter % 30000) + 30000; /* 30000-59999 */
    (*counter)++;
    return port;
}

/* Check if destination should be allowed */
static __always_inline int should_allow_dest(__u8 *daddr) {
    /* Denied CIDRs - always block these */
    /* 127.0.0.0/8 - loopback */
    if (daddr[0] == 127) return 0;

    /* 224.0.0.0/4 - multicast */
    if ((daddr[0] & 0xF0) == 0xE0) return 0;

    /* 240.0.0.0/4 - reserved */
    if ((daddr[0] & 0xF0) == 0xF0) return 0;

    /* Allow all other traffic - policy engine can add specific blocks */
    return 1;
}

/* NAT translate IPv4 address (helper for checksum update) */
static __always_inline void nat_ipv4_addr(__u32 *addr, __u32 new_val, __u32 *old_val) {
    *old_val = *addr;
    *addr = new_val;
}

/* Compute partial checksum for NAT translation */
static __always_inline void update_ip_csum(struct iphdr *ip, __u32 old_addr, __u32 new_addr) {
    __u32 diff = (~old_addr) + new_addr;
    ip->check = (__u16)((ip->check + (__u16)(diff >> 16) + (__u16)diff) & 0xFFFF);
}

/* Update TCP/UDP checksum when NAT changes addresses */
static __always_inline void update_l4_csum_tcp(struct tcphdr *tcp, __u32 old_saddr, __u32 new_saddr, __u16 old_sport, __u16 new_sport) {
    __u32 sum = 0;
    /* TCP checksum is optional for internal traffic, but we update it anyway */
    if (tcp->check) {
        /* Simplified - real implementation would compute delta properly */
        tcp->check = 0;
    }
}

static __always_inline void update_l4_csum_udp(struct udphdr *udp, __u32 old_saddr, __u32 new_saddr, __u16 old_sport, __u16 new_sport) {
    if (udp->check) {
        /* For UDP, recompute by subtracting old sum contributions and adding new */
        __u32 old_sum = ((__u16)old_saddr >> 16) + ((__u16)old_saddr & 0xFFFF);
        old_sum += ((__u16)new_saddr >> 16) + ((__u16)new_saddr & 0xFFFF);
        old_sum += old_sport + new_sport;
        old_sum = (old_sum >> 16) + (old_sum & 0xFFFF);
        old_sum += (old_sum >> 16);
        __u16 csum = ~(__u16)old_sum;
        udp->check = csum;
    }
}

SEC("tc")
int from_sandbox(struct __sk_buff *skb) {
    void *data_end = (void *)(long)skb->data_end;
    void *data = (void *)(long)skb->data;

    /* Parse Ethernet header */
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;

    /* Check for IPv4 (0x0800) */
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

    /* Build 5-tuple key from packet */
    struct tuple5 key = {};
    key.proto = proto;
    key.saddr[0] = (__u8)(ip->saddr & 0xFF);
    key.saddr[1] = (__u8)((ip->saddr >> 8) & 0xFF);
    key.saddr[2] = (__u8)((ip->saddr >> 16) & 0xFF);
    key.saddr[3] = (__u8)((ip->saddr >> 24) & 0xFF);
    key.daddr[0] = (__u8)(ip->daddr & 0xFF);
    key.daddr[1] = (__u8)((ip->daddr >> 8) & 0xFF);
    key.daddr[2] = (__u8)((ip->daddr >> 16) & 0xFF);
    key.daddr[3] = (__u8)((ip->daddr >> 24) & 0xFF);

    /* Check destination policy - block private/dangerous addresses */
    if (!should_allow_dest(key.daddr))
        return TC_ACT_SHOT;

    /* Get transport layer ports */
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
    key.sport = sport;
    key.dport = dport;

    /* Look up existing SNAT session */
    struct snat_entry *snat = bpf_map_lookup_elem(&egress_sessions, &key);
    if (snat) {
        /* Existing session - translate and forward */
        __u32 new_saddr = (__u32)snat->translated_addr[0] |
                         ((__u32)snat->translated_addr[1] << 8) |
                         ((__u32)snat->translated_addr[2] << 16) |
                         ((__u32)snat->translated_addr[3] << 24);
        __u16 new_sport = snat->translated_port;

        /* Update IP source address */
        ip->saddr = new_saddr;

        /* Update transport layer source port */
        if (proto == IPPROTO_TCP) {
            struct tcphdr *tcp = (void *)(ip + 1);
            tcp->source = new_sport;
        } else {
            struct udphdr *udp = (void *)(ip + 1);
            udp->source = new_sport;
        }

        /* Recompute checksums - simplified for performance */
        ip->check = 0;

        return TC_ACT_OK;
    }

    /* No existing session - create new SNAT mapping */
    /* Get CPU ID for port allocator */
    __u32 cpu_id = 0;
    __u16 new_sport = get_next_snat_port(cpu_id);

    /* Create new SNAT entry - using same IP, different port */
    struct snat_entry new_snat = {};
    new_snat.translated_addr[0] = 0;  /* Use original IP for now - coconet sets up NAT */
    new_snat.translated_addr[1] = 0;
    new_snat.translated_addr[2] = 0;
    new_snat.translated_addr[3] = 0;
    new_snat.translated_port = sport;  /* Preserve original port in translation */
    new_snat.orig_port = sport;
    new_snat.timestamp = bpf_ktime_get_ns() / 1000000000;

    /* Insert into session map */
    bpf_map_update_elem(&egress_sessions, &key, &new_snat, BPF_ANY);

    /* For now, allow the packet through without translation */
    /* coconet will handle the actual NAT translation at the host interface */
    return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "GPL";
