/* SPDX-License-Identifier: (GPL-2.0-only OR Apache-2.0) */
/* Copyright (C) 2026 The Coco Sandbox Authors */

/* Common map definitions shared across all eBPF programs.
 * These maps are created once by coconet and reused by
 * from_sandbox.bpf.c and from_world.bpf.c. */

#ifndef COCO_MAPS_H
#define COCO_MAPS_H

#include <linux/bpf.h>

/* Maximum entries per map (must be power of 2 for hash) */
#define EGRESS_SESSIONS_MAX 65536
#define INGRESS_SESSIONS_MAX 65536
#define POLICY_RULES_MAX 1024
#define SNAT_PORT_GEN_MAX 256
#define XSK_MAP_MAX 16
#define SESSION_TIMEOUT_SEC 300

/* Egress sessions: 5-tuple → SNAT entry
 * Used by from_sandbox.bpf.c to translate sandbox internal IP/port
 * to host IP/port for outbound traffic. */
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(key_size, sizeof(struct tuple5));
    __uint(value_size, sizeof(struct snat_entry));
    __uint(max_entries, EGRESS_SESSIONS_MAX);
    __uint(map_flags, 0); /* no flags */
} egress_sessions SEC(".maps");

/* Ingress sessions: reverse 5-tuple → DNAT entry
 * Used by from_world.bpf.c to translate host IP/port
 * back to sandbox internal IP/port for inbound traffic. */
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(key_size, sizeof(struct tuple5));
    __uint(value_size, sizeof(struct dnat_entry));
    __uint(max_entries, INGRESS_SESSIONS_MAX);
    __uint(map_flags, 0);
} ingress_sessions SEC(".maps");

/* SNAT port allocator: per-CPU counter for port selection
 * Prevents port conflicts when multiple sandboxes SNAT
 * to the same host IP. */
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(key_size, sizeof(__u32));
    __uint(value_size, sizeof(__u32)); /* current port counter */
    __uint(max_entries, SNAT_PORT_GEN_MAX);
} snat_port_gen SEC(".maps");

/* AF_XDP socket map: queue index → socket descriptor
 * Used by xdp_fwd.bpf.c for zero-copy packet processing. */
struct {
    __uint(type, BPF_MAP_TYPE_XSKMAP);
    __uint(key_size, sizeof(__u32)); /* queue index */
    __uint(value_size, sizeof(void *)); /* AF_XDP socket pointer */
    __uint(max_entries, XSK_MAP_MAX);
} xsks SEC(".maps");

/* Policy rules: CIDR prefix → action
 * Used by both egress and ingress for allow/deny decisions. */
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(key_size, sizeof(struct cidr_key));
    __uint(value_size, sizeof(__u32)); /* action: 0=allow, 1=deny */
    __uint(max_entries, POLICY_RULES_MAX);
} policy_rules SEC(".maps");

/* Bloom filter for fast negative matches
 * LPM for positive matches. */
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(key_size, sizeof(__u32));
    __uint(value_size, sizeof(__u64)); /* bloom filter bits */
    __uint(max_entries, 32); /* 2048-bit filter */
} bloom_filter SEC(".maps");

/* LPM trie for longest-prefix match routing/policy */
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __uint(key_size, sizeof(struct lpm_key)); /* variable length */
    __uint(value_size, sizeof(__u32)); /* action or route */
    __uint(max_entries, POLICY_RULES_MAX);
    __uint(map_flags, BPF_F_NO_PREALLOC);
} lpm_trie SEC(".maps");

/* Session timeout tracking: 5-tuple → timestamp
 * Used to expire stale sessions. */
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(key_size, sizeof(struct tuple5));
    __uint(value_size, sizeof(__u32)); /* last activity timestamp */
    __uint(max_entries, EGRESS_SESSIONS_MAX);
    __uint(map_flags, BPF_F_NO_COMMON_LRU);
} session_timeout SEC(".maps");

/* Per-sandbox statistics */
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(key_size, sizeof(__u32)); /* sandbox ID */
    __uint(value_size, sizeof(struct sb_stats));
    __uint(max_entries, 1024);
} sandbox_stats SEC(".maps");

struct sb_stats {
    __u64 rx_bytes;
    __u64 tx_bytes;
    __u64 rx_packets;
    __u64 tx_packets;
    __u64 drops;
    __u64 last_update;
};

/* CIDR key for policy lookup */
struct cidr_key {
    __u32 addr;    /* IP address */
    __u32 prefix;  /* Prefix length (0-32) */
};

/* LPM key for longest-prefix match */
struct lpm_key {
    __u32 prefix_len; /* prefix length in bits */
    __u32 addr;       /* IP address */
};

/* XDP metadata passed between program stages */
struct xdp_meta {
    __u32 sandbox_id;
    __u32 action;
    __u32 new_saddr[4];
    __u32 new_daddr[4];
    __u16 new_sport;
    __u16 new_dport;
};

#endif /* COCO_MAPS_H */