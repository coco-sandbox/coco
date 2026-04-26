#ifndef _COCO_MAPS_H
#define _COCO_MAPS_H

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

struct flow_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
};

struct flow_value {
    __u64 packets;
    __u64 bytes;
    __u32 vm_id;
    __u64 last_seen;
};

struct vm_info {
    __u32 vm_id;
    __u8  mac[6];
    __u32 ip_addr;
    __u8  active;
};

struct nat_entry {
    __u32 internal_ip;
    __u32 external_ip;
    __u16 internal_port;
    __u16 external_port;
    __u8  protocol;
    __u8  active;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct flow_key);
    __type(value, struct flow_value);
    __uint(max_entries, 4096);
} coco_flows SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);
    __type(value, struct vm_info);
    __uint(max_entries, 256);
} coco_vms SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct flow_key);
    __type(value, struct nat_entry);
    __uint(max_entries, 1024);
} coco_nat SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __type(key, __u32);
    __type(value, __u64);
    __uint(max_entries, 64);
} coco_stats SEC(".maps");

#endif
