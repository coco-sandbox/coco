package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	cfg := parseFlags()

	log.Printf("coco-proxy starting (listen=%s, upstream=%s)", cfg.ListenAddr, cfg.UpstreamAddr)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleProxy(cfg.UpstreamAddr))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Proxy listening on %s", cfg.ListenAddr)
		if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down")
}

type Config struct {
	ListenAddr   string
	UpstreamAddr string
	CacheEnabled bool
}

func parseFlags() *Config {
	listen := flag.String("listen", ":8080", "Listen address")
	upstream := flag.String("upstream", "http://localhost:9090", "Upstream server")
	cache := flag.Bool("cache", true, "Enable caching")

	flag.Parse()

	return &Config{
		ListenAddr:   *listen,
		UpstreamAddr: *upstream,
		CacheEnabled: *cache,
	}
}

func handleProxy(upstream string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
