package net

import (
    "testing"
)

func TestTAPManagerCreate(t *testing.T) {
    mgr := NewTAPManager()

    tap, err := mgr.Create("vnet1")
    if err != nil {
        t.Fatalf("Create TAP failed: %v", err)
    }
    if tap.Name != "vnet1" {
        t.Errorf("Expected name vnet1, got %s", tap.Name)
    }

    // Cleanup
    mgr.Destroy("vnet1")
}

func TestTAPManagerIPAssignment(t *testing.T) {
    mgr := NewTAPManager()

    tap, _ := mgr.Create("vnet2")
    err := mgr.SetIP("vnet2", "169.254.68.10", 24)
    if err != nil {
        t.Fatalf("SetIP failed: %v", err)
    }
    if tap.IP.String() != "169.254.68.10" {
        t.Errorf("Expected IP 169.254.68.10, got %s", tap.IP.String())
    }

    mgr.Destroy("vnet2")
}