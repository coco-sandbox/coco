// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

#ifndef COMMON_H
#define COMMON_H

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>

struct tuple_5 {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8 proto;
};

struct nat_session {
    __u32 orig_ip;
    __u32 nat_ip;
    __u16 orig_port;
    __u16 nat_port;
};

#endif