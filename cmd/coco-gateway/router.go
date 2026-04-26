package main

import (
	"net/http"
	"strings"

	"github.com/coco-sandbox/coco/pkg/api/handlers"
)

func registerRoutes(mux *http.ServeMux, gw *GatewayServer) {
	sbHandler := handlers.NewSandboxHandler(gw)

	mux.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			sbHandler.HandleCreate(w, r)
		case http.MethodGet:
			sbHandler.HandleList(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/sandboxes/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1/sandboxes/")
		parts := strings.SplitN(path, "/", 2)
		id := parts[0]
		if id == "" {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				sbHandler.HandleGet(w, r, id)
			case http.MethodDelete:
				sbHandler.HandleDelete(w, r, id)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		action := parts[1]
		switch action {
		case "pause":
			sbHandler.HandlePause(w, r, id)
		case "resume":
			sbHandler.HandleResume(w, r, id)
		case "fork":
			sbHandler.HandleFork(w, r, id)
		case "hibernate":
			sbHandler.HandleHibernate(w, r, id)
		case "resume-hibernate":
			sbHandler.HandleResumeHibernate(w, r, id)
		default:
			http.NotFound(w, r)
		}
	})
}
