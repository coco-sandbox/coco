package net

import (
    "fmt"
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
    ip, err := parseIP(SubnetStart)
    if err != nil {
        panic("invalid SubnetStart: " + err.Error())
    }
    maxHosts := uint32(1<<8) - 2 // /24 = 256 - 2 reserved
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

func parseIP(s string) ([]byte, error) {
    parts := split(s, '.')
    if len(parts) != 4 {
        return nil, fmt.Errorf("invalid IP format: %s", s)
    }
    result := make([]byte, 4)
    for i, p := range parts {
        v, err := strconv.Atoi(p)
        if err != nil || v < 0 || v > 255 {
            return nil, fmt.Errorf("invalid octet at position %d: %s", i, p)
        }
        result[i] = byte(v)
    }
    return result, nil
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

// incrementIP adds offset to base IP
// For /24 subnet design, we only use last octet for host allocation
// This means all IPs are 169.254.68.x where x is the host number
func incrementIP(base []byte, offset uint32) []byte {
    host := uint32(base[3]) + offset
    if host > 254 {
        host = 254
    }
    return []byte{base[0], base[1], base[2], byte(host)}
}