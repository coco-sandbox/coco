// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package visor

import (
    "testing"
)

func TestClientBoot(t *testing.T) {
    client, err := Dial()
    if err != nil {
        t.Skipf("cocovisor not running: %v", err)
    }
    defer client.Close()

    resp, err := client.Boot(BootRequest{
        ID:       "test-sb",
        Rootfs:   "/var/lib/coco/images/test.rootfs",
        MemoryMB: 512,
        VCPUs:    2,
    })
    if err != nil {
        t.Fatalf("Boot failed: %v", err)
    }
    if resp.VsockCID == 0 {
        t.Error("VsockCID should not be 0")
    }
}

func TestClientStreamingExec(t *testing.T) {
    client, err := Dial()
    if err != nil {
        t.Skipf("cocovisor not running: %v", err)
    }
    defer client.Close()

    chunks := make([]ExecChunk, 0)
    err = client.Exec(ExecRequest{
        Cmd:  "echo",
        Args: []string{"hello"},
    }, func(chunk ExecChunk) error {
        chunks = append(chunks, chunk)
        return nil
    })
    if err != nil {
        t.Fatalf("Exec failed: %v", err)
    }
    if len(chunks) == 0 {
        t.Error("Expected at least one chunk")
    }
}