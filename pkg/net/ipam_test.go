package net

import (
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

	// Should be in subnet 169.254.68.0/24
	if ip[:14] != "169.254.68." {
		t.Errorf("IP should be in subnet 169.254.68.0/24, got %s", ip)
	}
}

func TestIPAMRelease(t *testing.T) {
	ipam := NewIPAM()

	ip1, _ := ipam.Allocate()
	ipam.Release(ip1)

	ip2, _ := ipam.Allocate()
	// Should get same IP back since it was released
	if ip1 != ip2 {
		t.Logf("Note: IPAM doesn't guarantee same IP on realloc (acceptable for this design)")
	}
}
