// SPDX-License-Identifier: (GPL-2.0-only OR Apache-2.0)
// Copyright (C) 2026 The Coco Sandbox Authors

// from_world.bpf.c - Ingress traffic handling + DNAT
// Hook: TC ingress on host interface

#include "maps.h"
#include "common.h"

SEC("tc")
int from_world(struct __sk_buff *skb) {
    // Parse incoming packet
    // Lookup reverse session in ingress map
    // Rewrite to sandbox internal IP/port
    // Redirect to sandbox netns
    return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "GPL";