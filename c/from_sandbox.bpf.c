// SPDX-License-Identifier: (GPL-2.0-only OR Apache-2.0)
// Copyright (C) 2026 The Coco Sandbox Authors

// from_sandbox.bpf.c - Egress traffic handling + SNAT
// Hook: TC ingress/egress on sandbox netns

#include "maps.h"
#include "common.h"

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(key_size, sizeof(struct tuple5));
    __uint(value_size, sizeof(struct snat_entry));
    __uint(max_entries, 64 * 1024);
} egress_sessions SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(key_size, sizeof(__u32));
    __uint(value_size, sizeof(__u32));
    __uint(max_entries, 1);
} snat_port_gen SEC(".maps");

SEC("tc")
int from_sandbox(struct __sk_buff *skb) {
    // Parse Ethernet → IP → TCP/UDP
    // Lookup/create SNAT session
    // Rewrite source IP/port + checksums
    // Redirect to host interface
    return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "GPL";