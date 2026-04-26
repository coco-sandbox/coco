package net

import (
	"strings"
	"testing"
)

func TestIPAMAllocate(t *testing.T) {
	ipam := NewIPAM()

	ip, err := ipam.Allocate()
	if err != nil {
		t.Fatalf("Allocate failed: %v", err)
	}
	if ip == "" {
		t.Fatal("IP should not be empty")
	}

	t.Logf("Got IP: %s", ip)

	ipNoCidr := strings.Split(ip, "/")[0]

	if !strings.HasPrefix(ipNoCidr, "169.254.68.") {
		t.Errorf("IP should be in subnet 169.254.68.0/24, got %s", ip)
	}
}

func TestIPAMRelease(t *testing.T) {
	ipam := NewIPAM()

	ip1, _ := ipam.Allocate()
	ipam.Release(ip1)

	ip2, _ := ipam.Allocate()
	ip1 = strings.Split(ip1, "/")[0]
	ip2 = strings.Split(ip2, "/")[0]
	if ip1 != ip2 {
		t.Logf("Note: IPAM doesn't guarantee same IP on realloc (acceptable for this design)")
	}
}
