#ifndef _COCO_UTILS_H
#define _COCO_UTILS_H

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>

static inline int parse_eth_protocol(struct xdp_md *ctx, __u8 *protocol) {
    void *data = (void *)(long)ctx->data;
    void *data_end = (void *)(long)ctx->data_end;

    struct ethhdr *eth = data;
    if (eth + 1 > data_end)
        return -1;

    *protocol = eth->h_proto;
    return 0;
}

static inline int parse_ip_header(struct iphdr **ip_hdr, void *data, void *data_end) {
    struct iphdr *ip = data + sizeof(struct ethhdr);
    if (ip + 1 > data_end)
        return -1;

    *ip_hdr = ip;
    return 0;
}

static inline __u32 get_vm_id_from_ip(__u32 ip) {
    return ip & 0xFF;
}

static inline void swap_src_dst_mac(void *data) {
    struct ethhdr *eth = data;
    __u8 tmp[6];

    __builtin_memcpy(tmp, eth->h_source, 6);
    __builtin_memcpy(eth->h_source, eth->h_dest, 6);
    __builtin_memcpy(eth->h_dest, tmp, 6);
}

static inline void swap_src_dst_ip(struct iphdr *ip) {
    __u32 tmp = ip->saddr;
    ip->saddr = ip->daddr;
    ip->daddr = tmp;
}

static inline void swap_ports(void *transport) {
    struct tcphdr *tcp = transport;
    __u16 tmp = tcp->source;
    tcp->source = tcp->dest;
    tcp->dest = tmp;
}

#endif
