package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	listenAddr := flag.String("listen", ":8080", "Listen address")
	backend := flag.String("backend", "", "Backend URL (can be specified multiple times)", )
	timeout := flag.Duration("timeout", 30*time.Second, "Request timeout")
	maxRetries := flag.Int("max-retries", 3, "Maximum number of retries")
	cacheEnabled := flag.Bool("cache", false, "Enable response caching")
	cacheTTL := flag.Duration("cache-ttl", 5*time.Minute, "Cache TTL")
	cacheMaxSize := flag.Int("cache-size", 100*1024*1024, "Maximum cache size in bytes")
	flag.Parse()

	log.Printf("coco-proxy starting (listen=%s)", *listenAddr)

	proxy := NewProxy(*timeout, *maxRetries)

	if *cacheEnabled {
		log.Printf("Response caching enabled (ttl=%v, max_size=%dMB)", *cacheTTL, *cacheMaxSize/(1024*1024))
	}

	for i, addr := range flag.Args() {
		name := fmt.Sprintf("backend-%d", i)
		if err := proxy.AddBackend(name, addr, 1); err != nil {
			log.Printf("Failed to add backend %s: %v", addr, err)
		}
	}

	http.HandleFunc("/", proxy.ServeHTTP)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down proxy")
	}()

	log.Printf("Proxy listening on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
}
