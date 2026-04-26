// SPDX-License-Identifier: (GPL-2.0-only OR Apache-2.0)
// Copyright (C) 2026 The Coco Sandbox Authors
//
// tc_shaper.bpf.c - Traffic Shaper (spec/03 §3.3)
//
// TC egress program implementing per-sandbox token-bucket rate limiting
// with priority-weighted refill, satisfying spec/03 §3.3 (Traffic Shaper)
// and §4.4 (Rate Limiting). Tokens refill at `rate_bps * priority_weight`
// bytes per second; packets exceeding the bucket are dropped.
//
// Map: coco_shaper_buckets keyed by sandbox source IPv4 address.
// Userspace (coco-net) is responsible for populating buckets when a
// sandbox is created and clearing them when it is destroyed; numeric
// limits are deployment-defined per spec/10.

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/pkt_cls.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#define NSEC_PER_SEC 1000000000ULL

/* Per-sandbox shaper state. priority_weight is multiplied with rate_bps to
 * derive the effective refill rate; values >1 give the sandbox preferential
 * service during contention, values <1 deprioritize it. */
struct shaper_bucket {
    __u64 tokens;            /* current token balance, in bytes */
    __u64 last_refill_ns;    /* monotonic timestamp of the last refill */
    __u64 rate_bps;          /* sustained refill rate (bytes per second) */
    __u64 burst_bytes;       /* bucket capacity (max tokens) */
    __u32 priority_weight;   /* refill multiplier, scaled by 1024 (1024 = 1.0) */
    __u32 _pad;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);                /* sandbox source IPv4 (network byte order) */
    __type(value, struct shaper_bucket);
    __uint(max_entries, 4096);
} coco_shaper_buckets SEC(".maps");

/* Refill the bucket for the elapsed interval since last_refill_ns. The
 * refill is rate_bps * priority_weight / 1024, capped at burst_bytes. */
static __always_inline void refill(struct shaper_bucket *b, __u64 now_ns) {
    __u64 last = b->last_refill_ns;
    if (last == 0 || now_ns <= last) {
        b->last_refill_ns = now_ns;
        return;
    }
    __u64 elapsed = now_ns - last;
    /* Avoid overflow: rate * weight first scaled down, then * elapsed. */
    __u64 weighted_rate = (b->rate_bps * (__u64)b->priority_weight) >> 10;
    __u64 add = (weighted_rate * elapsed) / NSEC_PER_SEC;
    __u64 newtokens = b->tokens + add;
    if (newtokens > b->burst_bytes)
        newtokens = b->burst_bytes;
    b->tokens = newtokens;
    b->last_refill_ns = now_ns;
}

SEC("tc")
int tc_shaper(struct __sk_buff *skb) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;

    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return TC_ACT_OK;

    __u32 src = ip->saddr;
    struct shaper_bucket *b = bpf_map_lookup_elem(&coco_shaper_buckets, &src);
    if (!b)
        /* No shaper bucket installed for this sandbox: pass through.
         * Userspace populates buckets only for shaped sandboxes. */
        return TC_ACT_OK;

    __u64 now = bpf_ktime_get_ns();
    refill(b, now);

    __u32 pkt_len = skb->len;
    if (b->tokens < pkt_len)
        return TC_ACT_SHOT;

    b->tokens -= pkt_len;
    return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "GPL";
