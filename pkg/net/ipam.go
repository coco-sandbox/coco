package net

import (
    "fmt"
    "net"
    "strconv"
    "sync"
)

const (
    SubnetStart = "169.254.68.0"
    SubnetMask  = 24
)

type IPAM struct {
    mu        sync.Mutex
    allocated map[string]bool
    baseIP    []byte
    lastIP    uint32
    maxHosts  uint32
}

func NewIPAM() *IPAM {
    ip := parseIP(SubnetStart)
    maxHosts := (1 << (24 - SubnetMask)) - 2 // Reserve .0 and .255
    return &IPAM{
        allocated: make(map[string]bool),
        baseIP:    ip,
        lastIP:    0,
        maxHosts:  maxHosts,
    }
}

func (ipam *IPAM) Allocate() (string, error) {
    ipam.mu.Lock()
    defer ipam.mu.Unlock()

    for i := uint32(0); i < ipam.maxHosts; i++ {
        candidate := (ipam.lastIP % ipam.maxHosts) + 1
        ip := incrementIP(ipam.baseIP, candidate)
        ipStr := fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])

        if !ipam.allocated[ipStr] {
            ipam.allocated[ipStr] = true
            ipam.lastIP = candidate
            return ipStr + "/24", nil
        }
    }

    return "", fmt.Errorf("no available IPs")
}

func (ipam *IPAM) Release(ip string) {
    ipam.mu.Lock()
    defer ipam.mu.Unlock()

    // Strip CIDR suffix if present
    if i := indexByte(ip, '/'); i >= 0 {
        ip = ip[:i]
    }
    delete(ipam.allocated, ip)
}

func indexByte(s string, c byte) int {
    for i := 0; i < len(s); i++ {
        if s[i] == c {
            return i
        }
    }
    return -1
}

func parseIP(s string) []byte {
    var a, b, c, d byte
    parts := make([]int, 4)
    for i, p := range split(s, '.') {
        parts[i], _ = strconv.Atoi(p)
    }
    return []byte{byte(parts[0]), byte(parts[1]), byte(parts[2]), byte(parts[3])}
}

func split(s string, c byte) []string {
    var result []string
    start := 0
    for i := 0; i <= len(s); i++ {
        if i == len(s) || s[i] == c {
            result = append(result, s[start:i])
            start = i + 1
        }
    }
    return result
}

func incrementIP(base []byte, offset uint32) []byte {
    val := uint32(base[0])<<24 | uint32(base[1])<<16 | uint32(base[2])<<8 | uint32(base[3])
    val += offset
    return []byte{
        byte(val >> 24),
        byte(val >> 16),
        byte(val >> 8),
        byte(val),
    }
}