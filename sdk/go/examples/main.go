// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	coco "github.com/coco-sandbox/coco/sdk/go"
)

func main() {
	// Create client with custom options
	client, err := coco.NewClient(
		coco.WithAddress("localhost:4747"),
		coco.WithAPIKey("my-api-key"),
		coco.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Check health
	health, err := client.Health(ctx)
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("Health: healthy=%v, version=%s\n", health.Healthy, health.Version)

	// Check ready
	ready, err := client.Ready(ctx)
	if err != nil {
		log.Fatalf("Ready check failed: %v", err)
	}
	fmt.Printf("Ready: ready=%v, checks=%v\n", ready.Ready, ready.Checks)

	// Create a sandbox
	fmt.Println("\n--- Creating sandbox ---")
	createResp, err := client.CreateSandbox(ctx, &coco.SandboxConfig{
		Name:     "example-sandbox",
		Template: "alpine",
		MemoryMB: 512,
		VCPUs:    2,
		Labels:   map[string]string{"env": "development"},
	})
	if err != nil {
		log.Fatalf("CreateSandbox failed: %v", err)
	}
	fmt.Printf("Created: id=%s, name=%s, state=%s\n", createResp.ID, createResp.Name, createResp.State)
	sandboxID := createResp.ID

	// Get sandbox
	fmt.Println("\n--- Getting sandbox ---")
	sb, err := client.GetSandbox(ctx, sandboxID)
	if err != nil {
		log.Fatalf("GetSandbox failed: %v", err)
	}
	fmt.Printf("Sandbox: id=%s, state=%s, vsock_cid=%d, pid=%d\n", sb.ID, sb.State, sb.VsockCID, sb.PID)

	// List sandboxes
	fmt.Println("\n--- Listing sandboxes ---")
	listResp, err := client.ListSandboxes(ctx,
		coco.WithOffset(0),
		coco.WithLimit(10),
	)
	if err != nil {
		log.Fatalf("ListSandboxes failed: %v", err)
	}
	fmt.Printf("Total sandboxes: %d\n", listResp.TotalCount)
	for _, s := range listResp.Items {
		fmt.Printf("  - %s (%s)\n", s.Name, s.State)
	}

	// Execute command
	fmt.Println("\n--- Executing command ---")
	stdout, stderr, exitCode, err := client.ExecSync(ctx, sandboxID, &coco.ExecRequest{
		Cmd:  "echo",
		Args: []string{"hello", "world"},
	})
	if err != nil {
		log.Fatalf("ExecSync failed: %v", err)
	}
	fmt.Printf("stdout: %s", stdout)
	fmt.Printf("stderr: %s", stderr)
	fmt.Printf("exit code: %d\n", exitCode)

	// Execute with streaming
	fmt.Println("\n--- Executing with streaming ---")
	err = client.Exec(ctx, sandboxID, &coco.ExecRequest{
		Cmd: "ls",
		Args: []string{"-la"},
	}, func(chunk *coco.ExecChunk) error {
		switch chunk.Type {
		case "stdout":
			fmt.Print(chunk.Data)
		case "stderr":
			fmt.Fprintf(stderr, chunk.Data)
		case "exit":
			fmt.Printf("\nexit code: %d\n", chunk.ExitCode)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Exec streaming failed: %v", err)
	}

	// Create checkpoint
	fmt.Println("\n--- Creating checkpoint ---")
	cp, err := client.CreateCheckpoint(ctx, sandboxID, "before-test", "Before running test")
	if err != nil {
		log.Fatalf("CreateCheckpoint failed: %v", err)
	}
	fmt.Printf("Checkpoint: id=%s, name=%s\n", cp.ID, cp.Name)

	// List checkpoints
	fmt.Println("\n--- Listing checkpoints ---")
	checkpoints, err := client.ListCheckpoints(ctx, sandboxID)
	if err != nil {
		log.Fatalf("ListCheckpoints failed: %v", err)
	}
	for _, c := range checkpoints {
		fmt.Printf("  - %s (%s)\n", c.Name, c.CreatedAt)
	}

	// Fork sandbox
	fmt.Println("\n--- Forking sandbox ---")
	forkResp, err := client.ForkSandbox(ctx, sandboxID, &coco.ForkRequest{
		Name: "forked-sandbox",
	})
	if err != nil {
		log.Fatalf("ForkSandbox failed: %v", err)
	}
	fmt.Printf("Forked: id=%s, name=%s, state=%s\n", forkResp.ID, forkResp.Name, forkResp.State)

	// Hibernate sandbox
	fmt.Println("\n--- Hibernating sandbox ---")
	hibResp, err := client.HibernateSandbox(ctx, sandboxID)
	if err != nil {
		log.Fatalf("HibernateSandbox failed: %v", err)
	}
	fmt.Printf("Hibernated: state=%s, duration_ms=%d\n", hibResp.State, hibResp.HibernationDurationMs)

	// Resume sandbox
	fmt.Println("\n--- Resuming sandbox ---")
	_, err = client.ResumeSandbox(ctx, sandboxID)
	if err != nil {
		log.Fatalf("ResumeSandbox failed: %v", err)
	}
	fmt.Println("Resumed")

	// Get metrics
	fmt.Println("\n--- Fetching metrics ---")
	metrics, err := client.Metrics(ctx)
	if err != nil {
		log.Fatalf("Metrics failed: %v", err)
	}
	// Print first few lines
	lines := 0
	for _, line := range metrics {
		fmt.Print(string(line))
		lines++
		if lines > 10 {
			fmt.Println("...")
			break
		}
	}

	// Destroy sandbox
	fmt.Println("\n--- Destroying sandbox ---")
	destroyResp, err := client.DestroySandbox(ctx, sandboxID)
	if err != nil {
		log.Fatalf("DestroySandbox failed: %v", err)
	}
	fmt.Printf("Destroyed: success=%v\n", destroyResp.Success)

	fmt.Println("\nAll examples completed successfully!")
}
