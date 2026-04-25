// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	apiBase = "http://localhost:4747"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "sandbox":
		if len(os.Args) < 3 {
			printSandboxUsage()
			os.Exit(1)
		}
		subcmd := os.Args[2]
		switch subcmd {
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
		default:
			printSandboxUsage()
		}
	case "exec":
		if len(os.Args) < 4 {
			fmt.Println("Usage: cococtl exec <sandbox-id> <command>")
			os.Exit(1)
		}
		execSandbox(os.Args[3], os.Args[4:])
	case "health":
		healthCheck()
	default:
		printUsage()
	}
}

func createSandbox(name, template string) {
	reqBody := fmt.Sprintf(`{"name":"%s","template":"%s"}`, name, template)
	resp, err := http.Post(apiBase+"/v1/sandboxes", "application/json", strings.NewReader(reqBody))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func listSandboxes() {
	resp, err := http.Get(apiBase + "/v1/sandboxes")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func destroySandbox(id string) {
	req, _ := http.NewRequest("DELETE", apiBase+"/v1/sandboxes/"+id, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func execSandbox(id string, cmd []string) {
	reqBody, _ := json.Marshal(map[string]any{
		"cmd": cmd,
	})
	resp, err := http.Post(apiBase+"/v1/sandboxes/"+id+"/exec", "application/json", strings.NewReader(string(reqBody)))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func healthCheck() {
	resp, err := http.Get(apiBase + "/health")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func printUsage() {
	fmt.Println("Coco CLI - Command line interface for Coco Sandbox")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  cococtl <command> [args]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  sandbox create [name] [template]  Create a new sandbox")
	fmt.Println("  sandbox list                      List all sandboxes")
	fmt.Println("  sandbox destroy <id>              Destroy a sandbox")
	fmt.Println("  exec <sandbox-id> <command>      Execute command in sandbox")
	fmt.Println("  health                            Check API health")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  cococtl sandbox create myapp alpine")
	fmt.Println("  cococtl sandbox list")
	fmt.Println("  cococtl exec sb_12345678 ls -la")
}

func printSandboxUsage() {
	fmt.Println("Usage: cococtl sandbox <create|list|destroy> [args]")
}