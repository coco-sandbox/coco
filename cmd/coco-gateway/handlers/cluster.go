package handlers

import (
	"encoding/json"
	"net/http"
)

type ClusterHandler struct{}

func NewClusterHandler() *ClusterHandler {
	return &ClusterHandler{}
}

func (h *ClusterHandler) Info(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"name":       "coco-cluster",
		"nodes":       0,
		"leader":      "unknown",
		"version":     "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ClusterHandler) Nodes(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"items": []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ClusterHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	resp := map[string]interface{}{
		"id":      id,
		"address": "unknown",
		"status":  "unknown",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
