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
	"strings"
)

// =============================================================================
// Config
// =============================================================================

const (
	apiBase  = "http://localhost:4747"
	red      = "\033[31m"
	green    = "\033[32m"
	yellow   = "\033[33m"
	blue     = "\033[34m"
	cyan     = "\033[36m"
	reset    = "\033[0m"
	bold     = "\033[1m"
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
	case "health":
		handleHealth()
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
	case "destroy":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl sandbox destroy <id>")
			os.Exit(1)
		}
		destroySandbox(os.Args[3])
	case "get":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl sandbox get <id>")
			os.Exit(1)
		}
		getSandbox(os.Args[3])
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
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("%s✗ Parse error:%s %v\n", red, reset, err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusCreated {
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
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("%s✗ Parse error:%s %v\n", red, reset, err)
		os.Exit(1)
	}

	items := result["items"].([]any)
	fmt.Printf("%s%sSandboxes (%d)%s\n", bold, cyan, len(items), reset)
	fmt.Println(strings.Repeat("─", 60))

	for _, item := range items {
		sb := item.(map[string]any)
		fmt.Printf("%s%-20s%s %s%-12s%s %s%s%s\n",
			bold, sb["id"].(string), reset,
			yellow, sb["state"].(string), reset,
			blue, sb["template"].(string), reset)
	}
}

func destroySandbox(id string) {
	req, _ := http.NewRequest("DELETE", apiBase+"/v1/sandboxes/"+id, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("%s✗ Parse error:%s %v\n", red, reset, err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, result["error"])
		os.Exit(1)
	}

	fmt.Printf("%s✓ Destroyed sandbox%s %s\n", green, reset, id)
}

func getSandbox(id string) {
	resp, err := http.Get(apiBase + "/v1/sandboxes/" + id)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("%s✗ Parse error:%s %v\n", red, reset, err)
		os.Exit(1)
	}

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("%s✗ Sandbox not found:%s %s\n", red, reset, id)
		os.Exit(1)
	}

	sb := result["sandbox"].(map[string]any)
	fmt.Printf("%s%sSandbox Details%s\n", bold, cyan, reset)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("ID:        %s\n", sb["id"])
	fmt.Printf("Name:      %s\n", sb["name"])
	fmt.Printf("State:     %s\n", sb["state"])
	fmt.Printf("Template:  %s\n", sb["template"])
	fmt.Printf("Host Node: %s\n", sb["host_node"])
}

// =============================================================================
// Exec Command
// =============================================================================

func handleExec() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: cococtl exec <sandbox-id> <command> [args...]")
		os.Exit(1)
	}

	id := os.Args[2]
	cmd := os.Args[3:]
	if len(os.Args) > 4 {
		cmd = os.Args[3:]
	}

	body, _ := json.Marshal(map[string]any{
		"cmd": cmd,
	})

	resp, err := http.Post(apiBase+"/v1/sandboxes/"+id+"/exec", "application/json", strings.NewReader(string(body)))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("%s✗ Sandbox not found:%s %s\n", red, reset, id)
		os.Exit(1)
	}

	data, _ := io.ReadAll(resp.Body)
	fmt.Print(string(data))
}

// =============================================================================
// Health Check
// =============================================================================

func handleHealth() {
	resp, err := http.Get(apiBase + "/health")
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("%s✗ Parse error:%s %v\n", red, reset, err)
		os.Exit(1)
	}

	healthy := result["healthy"].(bool)
	if healthy {
		fmt.Printf("%s✓ Healthy%s\n", green, reset)
	} else {
		fmt.Printf("%s✗ Unhealthy%s\n", red, reset)
	}
	fmt.Printf("Version:   %s\n", result["version"])
	fmt.Printf("Uptime:    %.1fs\n", result["uptime_seconds"].(float64))
	fmt.Printf("Node ID:   %s\n", result["node_id"])
	fmt.Printf("Sandboxes: %d\n", int(result["sandboxes"].(float64)))
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
		if len(os.Args) > 4 {
			path = os.Args[4]
		}
		fsTree(id, path, 3)
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
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("%s✗ Parse error:%s %v\n", red, reset, err)
		os.Exit(1)
	}

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("%s✗ Sandbox not found%s\n", red, reset)
		os.Exit(1)
	}

	items := result["items"].([]any)
	fmt.Printf("%s%s%s/\n", bold, cyan, result["path"])
	fmt.Println(strings.Repeat("─", 40))

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
	resp, err := http.Get(apiBase + "/v1/sandboxes/" + id + "/fs/tree?path=" + q + "&depth=" + fmt.Sprintf("%d", depth))
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("%s✗ Parse error:%s %v\n", red, reset, err)
		os.Exit(1)
	}

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("%s✗ Sandbox not found%s\n", red, reset)
		os.Exit(1)
	}

	tree := result["tree"].(map[string]any)
	printTree(tree, 0)
}

func printTree(node map[string]any, indent int) {
	prefix := strings.Repeat("  ", indent)
	icon := "📄"
	if node["type"] == "dir" {
		icon = "📁"
	}
	fmt.Printf("%s%s%s/\n", prefix, icon, node["name"])

	if children, ok := node["children"].([]any); ok {
		for _, child := range children {
			printTree(child.(map[string]any), indent+1)
		}
	}
}

func fsCat(id, path string) {
	q := url.QueryEscape(path)
	resp, err := http.Get(apiBase + "/v1/sandboxes/" + id + "/fs/cat?path=" + q)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
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

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("%s✗ Sandbox not found%s\n", red, reset)
		os.Exit(1)
	}

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

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("%s✗ Sandbox not found%s\n", red, reset)
		os.Exit(1)
	}

	fmt.Printf("%s✓ Created directory%s %s\n", green, reset, path)
}

func fsRm(id, path string, recursive bool) {
	q := url.QueryEscape(path)
	rec := "false"
	if recursive {
		rec = "true"
	}
	req, _ := http.NewRequest("DELETE", apiBase+"/v1/sandboxes/"+id+"/fs/rm?path="+q+"&recursive="+rec, nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("%s✗ Sandbox not found%s\n", red, reset)
		os.Exit(1)
	}

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
  cococtl sandbox destroy <id>
  cococtl sandbox get <id>

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

Examples:
  cococtl sandbox create myapp alpine
  cococtl sandbox list
  cococtl fs ls sb_abc123 /
  cococtl exec sb_abc123 ls -la

`, bold, cyan, reset)
}

func printSandboxUsage() {
	fmt.Println(`Usage: cococtl sandbox <create|list|destroy|get> [args]`)
	fmt.Println(`  create <name> [template]  Create a new sandbox`)
	fmt.Println(`  list                       List all sandboxes`)
	fmt.Println(`  destroy <id>               Destroy a sandbox`)
	fmt.Println(`  get <id>                   Get sandbox details`)
}

func printFsUsage() {
	fmt.Println(`Usage: cococtl fs <ls|tree|cat|write|mkdir|rm> <id> [args]`)
	fmt.Println(`  ls <id> [path]           List directory`)
	fmt.Println(`  tree <id> [path] [depth] Show directory tree`)
	fmt.Println(`  cat <id> <path>          Read file`)
	fmt.Println(`  write <id> <path> <content> Write file`)
	fmt.Println(`  mkdir <id> <path>        Create directory`)
	fmt.Println(`  rm <id> <path> [-r]      Remove file/directory`)
}