// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Config
// =============================================================================

const (
	apiBase = "http://localhost:4747"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	cyan    = "\033[36m"
	reset   = "\033[0m"
	bold    = "\033[1m"
)

// =============================================================================
// Main
// =============================================================================

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	cmd := os.Args[1]

	switch cmd {
	case "sandbox":
		handleSandbox()
	case "fs":
		handleFs()
	case "exec":
		handleExec()
	case "checkpoint":
		handleCheckpoint()
	case "health":
		handleHealth()
	case "metrics":
		handleMetrics()
	default:
		printUsage()
	}
}

// =============================================================================
// Sandbox Commands
// =============================================================================

func handleSandbox() {
	if len(os.Args) < 3 {
		printSandboxUsage()
		os.Exit(1)
	}

	sub := os.Args[2]
	switch sub {
	case "create":
		name := "default"
		template := "alpine"
		if len(os.Args) > 3 {
			name = os.Args[3]
		}
		if len(os.Args) > 4 {
			template = os.Args[4]
		}
		createSandbox(name, template)

	case "list":
		listSandboxes()

	case "get":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl sandbox get <id>")
			os.Exit(1)
		}
		getSandbox(os.Args[3])

	case "destroy":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl sandbox destroy <id>")
			os.Exit(1)
		}
		destroySandbox(os.Args[3])

	case "fork":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl sandbox fork <id> [name]")
			os.Exit(1)
		}
		name := ""
		if len(os.Args) > 4 {
			name = os.Args[4]
		}
		forkSandbox(os.Args[3], name)

	case "hibernate":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl sandbox hibernate <id>")
			os.Exit(1)
		}
		hibernateSandbox(os.Args[3])

	case "resume":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl sandbox resume <id>")
			os.Exit(1)
		}
		resumeSandbox(os.Args[3])

	case "pause":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl sandbox pause <id>")
			os.Exit(1)
		}
		pauseSandbox(os.Args[3])

	default:
		printSandboxUsage()
	}
}

func createSandbox(name, template string) {
	body := fmt.Sprintf(`{"name":"%s","template":"%s"}`, name, template)
	resp, err := http.Post(apiBase+"/v1/sandboxes", "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 201 {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, result["error"])
		os.Exit(1)
	}

	sb := result["sandbox"].(map[string]any)
	fmt.Printf("%s✓ Created sandbox%s %s\n", green, reset, sb["id"])
	fmt.Printf("  Name:     %s\n", sb["name"])
	fmt.Printf("  State:    %s\n", sb["state"])
	fmt.Printf("  Template: %s\n", sb["template"])
}

func listSandboxes() {
	resp, err := http.Get(apiBase + "/v1/sandboxes")
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	items := result["items"].([]any)
	fmt.Printf("%s%sSandboxes (%d)%s\n", bold, cyan, len(items), reset)
	fmt.Println(strings.Repeat("─", 60))

	for _, item := range items {
		sb := item.(map[string]any)
		fmt.Printf("%s%-20s%s %s%-12s%s\n",
			bold, sb["id"].(string), reset,
			yellow, sb["state"].(string), reset)
	}
}

func getSandbox(id string) {
	resp, err := http.Get(apiBase + "/v1/sandboxes/" + id)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		fmt.Printf("%s✗ Sandbox not found:%s %s\n", red, reset, id)
		os.Exit(1)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	sb := result["sandbox"].(map[string]any)

	fmt.Printf("%s%sSandbox Details%s\n", bold, cyan, reset)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("ID:        %s\n", sb["id"])
	fmt.Printf("Name:      %s\n", sb["name"])
	fmt.Printf("State:     %s\n", sb["state"])
	fmt.Printf("Template:  %s\n", sb["template"])
	if cid, ok := sb["vsock_cid"]; ok {
		fmt.Printf("Vsock CID: %v\n", cid)
	}
}

func destroySandbox(id string) {
	req, _ := http.NewRequest("DELETE", apiBase+"/v1/sandboxes/"+id, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 200 {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, result["error"])
		os.Exit(1)
	}

	fmt.Printf("%s✓ Destroyed sandbox%s %s\n", green, reset, id)
}

func forkSandbox(id, name string) {
	body := `{}`
	if name != "" {
		body = fmt.Sprintf(`{"name":"%s"}`, name)
	}

	resp, err := http.Post(apiBase+"/v1/sandboxes/"+id+"/fork", "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 201 {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, result["error"])
		os.Exit(1)
	}

	fmt.Printf("%s✓ Forked sandbox%s %s → %s\n", green, reset, id, result["id"])
	fmt.Printf("  Parent:  %s\n", result["parent_id"])
	fmt.Printf("  Fork ms: %v\n", result["fork_duration_ms"])
}

func hibernateSandbox(id string) {
	resp, err := http.Post(apiBase+"/v1/sandboxes/"+id+"/hibernate", "application/json", strings.NewReader("{}"))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 200 {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, result["error"])
		os.Exit(1)
	}

	fmt.Printf("%s✓ Hibernated sandbox%s %s\n", green, reset, id)
	fmt.Printf("  Size bytes: %v\n", result["size_bytes"])
	fmt.Printf("  Duration ms: %v\n", result["hibernate_duration_ms"])
}

func resumeSandbox(id string) {
	resp, err := http.Post(apiBase+"/v1/sandboxes/"+id+"/resume", "application/json", strings.NewReader("{}"))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 200 {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, result["error"])
		os.Exit(1)
	}

	fmt.Printf("%s✓ Resumed sandbox%s %s\n", green, reset, id)
}

func pauseSandbox(id string) {
	req, _ := http.NewRequest("POST", apiBase+"/v1/sandboxes/"+id+"/pause", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 200 {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, result["error"])
		os.Exit(1)
	}

	fmt.Printf("%s✓ Paused sandbox%s %s\n", green, reset, id)
}

// =============================================================================
// Exec
// =============================================================================

func handleExec() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cococtl exec <sandbox-id> <command> [args...]")
		os.Exit(1)
	}

	id := os.Args[2]
	cmd := os.Args[3:]

	body, _ := json.Marshal(map[string]any{"cmd": cmd})
	resp, err := http.Post(apiBase+"/v1/sandboxes/"+id+"/exec", "application/json", strings.NewReader(string(body)))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		fmt.Printf("%s✗ Sandbox not found:%s %s\n", red, reset, id)
		os.Exit(1)
	}

	data, _ := io.ReadAll(resp.Body)
	fmt.Print(string(data))
}

// =============================================================================
// Health & Metrics
// =============================================================================

func handleHealth() {
	resp, err := http.Get(apiBase + "/health")
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	healthy := result["healthy"].(bool)
	if healthy {
		fmt.Printf("%s✓ Healthy%s\n", green, reset)
	} else {
		fmt.Printf("%s✗ Unhealthy%s\n", red, reset)
	}
	fmt.Printf("Version: %s\n", result["version"])
}

func handleMetrics() {
	resp, err := http.Get(apiBase + "/metrics")
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	fmt.Print(string(data))
}

// =============================================================================
// Checkpoint Commands
// =============================================================================

func handleCheckpoint() {
	if len(os.Args) < 3 {
		printCheckpointUsage()
		os.Exit(1)
	}

	sub := os.Args[2]
	switch sub {
	case "create":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl checkpoint create <sandbox-id> [name]")
			os.Exit(1)
		}
		id := os.Args[3]
		name := fmt.Sprintf("ckpt-%d", time.Now().Unix())
		if len(os.Args) > 4 {
			name = os.Args[4]
		}
		createCheckpoint(id, name)

	case "list":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl checkpoint list <sandbox-id>")
			os.Exit(1)
		}
		listCheckpoints(os.Args[3])

	case "delete":
		if len(os.Args) < 5 {
			fmt.Println("Usage: cococtl checkpoint delete <sandbox-id> <checkpoint-id>")
			os.Exit(1)
		}
		deleteCheckpoint(os.Args[3], os.Args[4])

	default:
		printCheckpointUsage()
	}
}

func createCheckpoint(id, name string) {
	body := fmt.Sprintf(`{"name":"%s"}`, name)
	resp, err := http.Post(apiBase+"/v1/sandboxes/"+id+"/checkpoint", "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 201 {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, result["error"])
		os.Exit(1)
	}

	fmt.Printf("%s✓ Created checkpoint%s %s\n", green, reset, result["checkpoint_id"])
	fmt.Printf("  Name: %s\n", result["name"])
}

func listCheckpoints(id string) {
	resp, err := http.Get(apiBase + "/v1/sandboxes/" + id + "/checkpoints")
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	checkpoints := result["checkpoints"].([]any)
	fmt.Printf("%s%sCheckpoints (%d)%s\n", bold, cyan, len(checkpoints), reset)
	for _, c := range checkpoints {
		ck := c.(map[string]any)
		fmt.Printf("  %s %s\n", ck["id"], ck["name"])
	}
}

func deleteCheckpoint(sandboxID, checkpointID string) {
	req, _ := http.NewRequest("DELETE", apiBase+"/v1/sandboxes/"+sandboxID+"/checkpoints/"+checkpointID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("%s✓ Deleted checkpoint%s %s\n", green, reset, checkpointID)
}

// =============================================================================
// Undo / Redo
// =============================================================================

func handleUndoRedo(cmd string) {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: cococtl %s <sandbox-id> [checkpoint-id]\n", cmd)
		os.Exit(1)
	}

	id := os.Args[2]
	ckptID := ""
	if len(os.Args) > 3 {
		ckptID = os.Args[3]
	}

	body := `{}`
	if ckptID != "" {
		body = fmt.Sprintf(`{"checkpoint_id":"%s"}`, ckptID)
	}

	resp, err := http.Post(apiBase+"/v1/sandboxes/"+id+"/"+cmd, "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode != 200 {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, result["error"])
		os.Exit(1)
	}

	fmt.Printf("%s✓ %s sandbox%s %s\n", green, strings.ToUpper(cmd[:4]), reset, id)
	if ckptID != "" {
		fmt.Printf("  Checkpoint: %s\n", ckptID)
	}
}

// =============================================================================
// File Operations
// =============================================================================

func handleFs() {
	if len(os.Args) < 4 {
		printFsUsage()
		os.Exit(1)
	}

	sub := os.Args[2]
	id := os.Args[3]

	switch sub {
	case "ls":
		path := "/"
		if len(os.Args) > 4 {
			path = os.Args[4]
		}
		fsLs(id, path)

	case "tree":
		path := "/"
		depth := 3
		if len(os.Args) > 4 {
			path = os.Args[4]
		}
		if len(os.Args) > 5 {
			depth, _ = strconv.Atoi(os.Args[5])
		}
		fsTree(id, path, depth)

	case "cat":
		if len(os.Args) < 5 {
			fmt.Println("Usage: cococtl fs cat <id> <path>")
			os.Exit(1)
		}
		fsCat(id, os.Args[4])

	case "write":
		if len(os.Args) < 6 {
			fmt.Println("Usage: cococtl fs write <id> <path> <content>")
			os.Exit(1)
		}
		fsWrite(id, os.Args[4], os.Args[5])

	case "mkdir":
		if len(os.Args) < 5 {
			fmt.Println("Usage: cococtl fs mkdir <id> <path>")
			os.Exit(1)
		}
		fsMkdir(id, os.Args[4])

	case "rm":
		if len(os.Args) < 5 {
			fmt.Println("Usage: cococtl fs rm <id> <path> [-r]")
			os.Exit(1)
		}
		path := os.Args[4]
		recursive := len(os.Args) > 5 && os.Args[5] == "-r"
		fsRm(id, path, recursive)

	default:
		printFsUsage()
	}
}

func fsLs(id, path string) {
	q := url.QueryEscape(path)
	resp, err := http.Get(apiBase + "/v1/sandboxes/" + id + "/fs/ls?path=" + q)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode == 404 {
		fmt.Printf("%s✗ Sandbox not found%s\n", red, reset)
		os.Exit(1)
	}

	items := result["items"].([]any)
	fmt.Printf("%s%s/%s\n", bold, cyan, result["path"])
	for _, item := range items {
		e := item.(map[string]any)
		icon := "📄"
		if e["type"] == "dir" {
			icon = "📁"
		}
		fmt.Printf("%s %-20s %10d bytes\n", icon, e["name"], int64(e["size"].(float64)))
	}
}

func fsTree(id, path string, depth int) {
	q := url.QueryEscape(path)
	resp, err := http.Get(apiBase + "/v1/sandboxes/" + id + "/fs/tree?path=" + q + "&depth=" + strconv.Itoa(depth))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode == 404 {
		fmt.Printf("%s✗ Sandbox not found%s\n", red, reset)
		os.Exit(1)
	}

	// Simple tree output
	fmt.Printf("%s/\n", result["path"])
}

func fsCat(id, path string) {
	q := url.QueryEscape(path)
	resp, err := http.Get(apiBase + "/v1/sandboxes/" + id + "/fs/cat?path=" + q)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		fmt.Printf("%s✗ Sandbox not found%s\n", red, reset)
		os.Exit(1)
	}

	data, _ := io.ReadAll(resp.Body)
	fmt.Print(string(data))
}

func fsWrite(id, path, content string) {
	body := fmt.Sprintf(`{"path":"%s"}`, content)
	req, _ := http.NewRequest("PUT", apiBase+"/v1/sandboxes/"+id+"/fs/write?path="+url.QueryEscape(path), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("%s✓ Wrote to%s %s\n", green, reset, path)
}

func fsMkdir(id, path string) {
	body := fmt.Sprintf(`{"path":"%s"}`, path)
	resp, err := http.Post(apiBase+"/v1/sandboxes/"+id+"/fs/mkdir", "application/json", strings.NewReader(body))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("%s✓ Created directory%s %s\n", green, reset, path)
}

func fsRm(id, path string, recursive bool) {
	q := url.QueryEscape(path)
	rec := "false"
	if recursive {
		rec = "true"
	}
	req, _ := http.NewRequest("DELETE", apiBase+"/v1/sandboxes/"+id+"/fs/rm?path="+q+"&recursive="+rec, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("%s✓ Removed%s %s\n", green, reset, path)
}

// =============================================================================
// Usage
// =============================================================================

func printUsage() {
	fmt.Printf(`%s%sCoco Sandbox CLI%s

Usage: cococtl <command> [args]

Sandbox Management:
  cococtl sandbox create <name> [template]
  cococtl sandbox list
  cococtl sandbox get <id>
  cococtl sandbox destroy <id>
  cococtl sandbox fork <id> [name]
  cococtl sandbox hibernate <id>
  cococtl sandbox resume <id>
  cococtl sandbox pause <id>

Checkpoint:
  cococtl checkpoint create <id> [name]
  cococtl checkpoint list <id>
  cococtl checkpoint delete <id> <checkpoint-id>
  cococtl undo <id> [checkpoint-id]
  cococtl redo <id> [checkpoint-id]

File Operations:
  cococtl fs ls <id> [path]
  cococtl fs tree <id> [path] [depth]
  cococtl fs cat <id> <path>
  cococtl fs write <id> <path> <content>
  cococtl fs mkdir <id> <path>
  cococtl fs rm <id> <path> [-r]

Other:
  cococtl exec <id> <cmd> [args...]
  cococtl health
  cococtl metrics

`, bold, cyan, reset)
}

func printSandboxUsage() {
	fmt.Println(`Usage: cococtl sandbox <create|list|get|destroy|fork|hibernate|resume|pause> [args]`)
}

func printCheckpointUsage() {
	fmt.Println(`Usage: cococtl checkpoint <create|list|delete> [args]`)
	fmt.Println(`       cococtl undo <id> [checkpoint-id]`)
	fmt.Println(`       cococtl redo <id> [checkpoint-id]`)
}

func printFsUsage() {
	fmt.Println(`Usage: cococtl fs <ls|tree|cat|write|mkdir|rm> <id> [args]`)
}