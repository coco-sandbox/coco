package ipam

import (
	"fmt"
	"net"
	"sync"
)

type Pool struct {
	mu       sync.RWMutex
	subnet   *net.IPNet
	reserved map[string]net.IP
	available []net.IP
}

func NewPool(subnet *net.IPNet) *Pool {
	p := &Pool{
		subnet:   subnet,
		reserved: make(map[string]net.IP),
		available: make([]net.IP, 0),
	}

	p.initAvailable()

	return p
}

func (p *Pool) initAvailable() {
	ip := p.subnet.IP.To4()
	if ip == nil {
		ip = p.subnet.IP.To16()
	}

	first := p.firstUsable()
	last := p.lastUsable()

	for n := first; n <= last; n++ {
		ip := net.IP{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
		p.available = append(p.available, ip)
	}
}

func (p *Pool) firstUsable() uint32 {
	ip := p.subnet.IP.To4()
	mask := p.subnet.Mask

	network := ipToUint(ip) & ipToUint(net.IP(mask).To4())

	return network + 1
}

func (p *Pool) lastUsable() uint32 {
	ip := p.subnet.IP.To4()
	mask := p.subnet.Mask

	network := ipToUint(ip) & ipToUint(net.IP(mask).To4())
	broadcast := network | ^ipToUint(net.IP(mask).To4())

	return broadcast - 1
}

func (p *Pool) Acquire() (net.IP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.available) == 0 {
		return nil, fmt.Errorf("no available IPs")
	}

	ip := p.available[len(p.available)-1]
	p.available = p.available[:len(p.available)-1]

	return ip, nil
}

func (p *Pool) Release(ip net.IP) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, reserved := range p.reserved {
		if reserved.Equal(ip) {
			return fmt.Errorf("IP is reserved")
		}
	}

	p.available = append(p.available, ip)

	return nil
}

func (p *Pool) Reserve(ip net.IP) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, available := range p.available {
		if available.Equal(ip) {
			p.available = removeIP(p.available, ip)
			return nil
		}
	}

	for _, reserved := range p.reserved {
		if reserved.Equal(ip) {
			return fmt.Errorf("IP already reserved")
		}
	}

	return fmt.Errorf("IP not in pool")
}

func (p *Pool) Available() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.available)
}

func (p *Pool) Reserved() map[string]net.IP {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]net.IP)
	for k, v := range p.reserved {
		result[k] = v
	}

	return result
}

func ipToUint(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func removeIP(slice []net.IP, ip net.IP) []net.IP {
	for i, v := range slice {
		if v.Equal(ip) {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
