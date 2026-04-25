// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// =============================================================================
// FS Operations (read from host filesystem — placeholder for real sandbox fs)
// =============================================================================

func handleFS(w http.ResponseWriter, r *http.Request, id string, action string) {
	state.mu.RLock()
	_, ok := state.sandboxes[id]
	state.mu.RUnlock()

	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Sandbox %s not found", id))
		return
	}

	basePath := "/var/lib/coco/sandboxes/" + id

	switch action {
	case "ls":
		handleFSLs(w, r, basePath)
	case "tree":
		handleFSTree(w, r, basePath)
	case "cat":
		handleFSCat(w, r, basePath)
	case "write":
		if r.Method == http.MethodPut {
			handleFSWrite(w, r, basePath)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "PUT required for write")
		}
	case "mkdir":
		if r.Method == http.MethodPost {
			handleFSMkdir(w, r, basePath)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "POST required for mkdir")
		}
	case "rm":
		if r.Method == http.MethodDelete {
			handleFSRm(w, r, basePath)
		} else {
			writeError(w, http.StatusMethodNotAllowed, "DELETE required for rm")
		}
	default:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Unknown fs action: %s", action))
	}
}

func handleFSLs(w http.ResponseWriter, r *http.Request, basePath string) {
	queryPath := r.URL.Query().Get("path")
	if queryPath == "" {
		queryPath = "/"
	}
	fullPath := filepath.Join(basePath, queryPath)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Path not found: %s", queryPath))
		return
	}

	type Entry struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Size    int64  `json:"size"`
		ModTime int64  `json:"mtime"`
	}
	var items []Entry
	for _, e := range entries {
		info, _ := e.Info()
		items = append(items, Entry{
			Name: e.Name(),
			Type: "file",
			Size: info.Size(),
			ModTime: info.ModTime().Unix(),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items, "path": queryPath})
}

func handleFSTree(w http.ResponseWriter, r *http.Request, basePath string) {
	queryPath := r.URL.Query().Get("path")
	if queryPath == "" {
		queryPath = "/"
	}
	fullPath := filepath.Join(basePath, queryPath)

	// Simple recursive tree — limited depth
	type TreeNode struct {
		Name  string      `json:"name"`
		Type  string      `json:"type"`
		Children []TreeNode `json:"children,omitempty"`
	}

	var buildTree func(path string, depth int) []TreeNode
	buildTree = func(path string, depth int) []TreeNode {
		if depth > 3 {
			return nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		var nodes []TreeNode
		for _, e := range entries {
			info, _ := e.Info()
			node := TreeNode{Name: e.Name(), Type: "file"}
			if e.IsDir() {
				node.Type = "dir"
				node.Children = buildTree(filepath.Join(path, e.Name()), depth+1)
			}
			_ = info
			nodes = append(nodes, node)
		}
		return nodes
	}

	tree := buildTree(fullPath, 0)
	writeJSON(w, http.StatusOK, map[string]any{"tree": tree, "path": queryPath})
}

func handleFSCat(w http.ResponseWriter, r *http.Request, basePath string) {
	queryPath := r.URL.Query().Get("path")
	if queryPath == "" {
		writeError(w, http.StatusBadRequest, "path query parameter required")
		return
	}
	fullPath := filepath.Join(basePath, queryPath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("File not found: %s", queryPath))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":  queryPath,
		"size":  len(data),
		"content": string(data),
	})
}

func handleFSWrite(w http.ResponseWriter, r *http.Request, basePath string) {
	queryPath := r.URL.Query().Get("path")
	if queryPath == "" {
		writeError(w, http.StatusBadRequest, "path query parameter required")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Read body failed: %v", err))
		return
	}

	fullPath := filepath.Join(basePath, queryPath)
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	os.WriteFile(fullPath, body, 0644)

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": queryPath, "size": len(body)})
}

func handleFSMkdir(w http.ResponseWriter, r *http.Request, basePath string) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "path required")
		return
	}

	fullPath := filepath.Join(basePath, req.Path)
	os.MkdirAll(fullPath, 0755)

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": req.Path})
}

func handleFSRm(w http.ResponseWriter, r *http.Request, basePath string) {
	queryPath := r.URL.Query().Get("path")
	if queryPath == "" {
		writeError(w, http.StatusBadRequest, "path query parameter required")
		return
	}

	fullPath := filepath.Join(basePath, queryPath)
	err := os.RemoveAll(fullPath)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Remove failed: %s", queryPath))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": queryPath})
}