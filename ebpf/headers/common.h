#ifndef _COCO_COMMON_H
#define _COCO_COMMON_H

#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

#define ETH_P_IP  0x0800
#define ETH_P_IPV6 0x86DD
#define ETH_P_ARP 0x0806

#define IPPROTO_ICMP   1
#define IPPROTO_TCP    6
#define IPPROTO_UDP    17

#define COCO_MAX_VM 256
#define COCO_MAX_FLOWS 4096

struct eth_hdr {
    __be16 h_proto;
    __u8 h_source[6];
    __u8 h_dest[6];
};

struct iphdr {
    __u8  ihl:4,
          version:4;
    __u8  tos;
    __be16 tot_len;
    __be16 id;
    __be16 frag_off;
    __u8  ttl;
    __u8  protocol;
    __sum16 check;
    __be32 saddr;
    __be32 daddr;
};

struct tcphdr {
    __be16 source;
    __be16 dest;
    __be32 seq;
    __be32 ack_seq;
    __u8  res1:4,
          doff:4;
    __u8  flags;
    __be16 window;
    __sum16 check;
    __be16 urg_ptr;
};

struct udphdr {
    __be16 source;
    __be16 dest;
    __be16 len;
    __sum16 check;
};

#endif
