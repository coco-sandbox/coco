#ifndef _COCO_XDP_COMMON_H
#define _COCO_XDP_COMMON_H

#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>

#define XDP_PASS 1
#define XDP_DROP 2
#define XDP_TX 3

#ifndef IPPROTO_ICMP
#define IPPROTO_ICMP   1
#endif
#ifndef IPPROTO_TCP
#define IPPROTO_TCP    6
#endif
#ifndef IPPROTO_UDP
#define IPPROTO_UDP    17
#endif

static __always_inline __u16 ip_fast_csum(const struct iphdr *iph, __u32 ihl) {
    __u32 sum = 0;
    __u16 *ptr = (__u16 *)iph;
    for (__u32 i = 0; i < ihl; i++)
        sum += *ptr++;
    return ~(__u16)(sum + (sum >> 16));
}

#endif
