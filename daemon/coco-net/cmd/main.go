package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/coco-sandbox/coco/daemon/coco-net/conntrack"
	"github.com/coco-sandbox/coco/daemon/coco-net/ebpf"
	"github.com/coco-sandbox/coco/daemon/coco-net/ipam"
	"github.com/coco-sandbox/coco/daemon/coco-net/netns"
	"github.com/coco-sandbox/coco/daemon/coco-net/policy"
	"github.com/coco-sandbox/coco/daemon/coco-net/rate"
)

type Config struct {
	ListenAddr    string
	Subnet        string
	MetricsAddr   string
	EBPFEnabled   bool
	RateLimitRPS  float64
	RateLimitBurst int
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	cfg := parseFlags()

	log.Printf("coco-net starting (listen=%s, subnet=%s)", cfg.ListenAddr, cfg.Subnet)

	ipam, err := ipam.New(cfg.Subnet)
	if err != nil {
		log.Fatalf("Failed to create IPAM: %v", err)
	}

	ct := conntrack.New()
	netnsMgr := netns.NewManager()
	policyEng := policy.NewEngine()
	rateLimiter := rate.NewLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	var ebpfLoader *ebpf.Loader
	if cfg.EBPFEnabled {
		ebpfLoader = ebpf.NewLoader()
		log.Println("eBPF support enabled")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ip/allocate", handleIPAllocate(ipam))
	mux.HandleFunc("/ip/release", handleIPRelease(ipam))
	mux.HandleFunc("/policy/add", handlePolicyAdd(policyEng))
	mux.HandleFunc("/health", handleHealth)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("HTTP server listening on %s", cfg.ListenAddr)
		if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down")
}

func parseFlags() *Config {
	listenAddr := flag.String("listen", ":8081", "Listen address")
	subnet := flag.String("subnet", "10.0.0.0/24", "Network subnet")
	metricsAddr := flag.String("metrics", ":9091", "Metrics address")
	ebpfEnabled := flag.Bool("ebpf", false, "Enable eBPF support")
	rateLimitRPS := flag.Float64("rate-limit-rps", 1000, "Rate limit RPS")
	rateLimitBurst := flag.Int("rate-limit-burst", 2000, "Rate limit burst")

	flag.Parse()

	return &Config{
		ListenAddr:    *listenAddr,
		Subnet:        *subnet,
		MetricsAddr:   *metricsAddr,
		EBPFEnabled:   *ebpfEnabled,
		RateLimitRPS:  *rateLimitRPS,
		RateLimitBurst: *rateLimitBurst,
	}
}

func handleIPAllocate(ipam *ipam.IPAM) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sandboxID := r.URL.Query().Get("sandbox_id")
		if sandboxID == "" {
			http.Error(w, "sandbox_id required", http.StatusBadRequest)
			return
		}

		ip, err := ipam.Allocate(sandboxID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, `{"ip": "%s"}`, ip.String())
	}
}

func handleIPRelease(ipam *ipam.IPAM) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sandboxID := r.URL.Query().Get("sandbox_id")
		if sandboxID == "" {
			http.Error(w, "sandbox_id required", http.StatusBadRequest)
			return
		}

		if err := ipam.Release(sandboxID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, `{"status": "released"}`)
	}
}

func handlePolicyAdd(policyEng *policy.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sandboxID := r.URL.Query().Get("sandbox_id")
		if sandboxID == "" {
			http.Error(w, "sandbox_id required", http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, `{"status": "policy added"}`)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"status": "healthy"}`)
}
