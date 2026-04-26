package visor

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound       = errors.New("visor not found")
	ErrPoolFull       = errors.New("pool is full")
	ErrAlreadyExists  = errors.New("visor already exists")
	ErrNotRunning     = errors.New("visor not running")
	ErrAlreadyRunning = errors.New("visor already running")
)

type State int

const (
	StateCreated State = iota
	StateStarting
	StateRunning
	StateStopping
	StateStopped
	StateFailed
)

func (s State) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type Visor struct {
	ID      string
	Config  *Config
	State   State
	proc    *Process
	created time.Time
	started time.Time
}

type Config struct {
	ID        string
	Image     string
	Kernel    string
	Rootfs    string
	Namespace string
	Resources *Resources
	Network   *NetworkConfig
}

type Resources struct {
	CPUs   int
	Memory int64
	Disk   int64
}

type NetworkConfig struct {
	Interface string
	IP        string
	Gateway   string
	DNS       []string
}

type Process struct {
	PID      int
	ExitCode int
	Started  time.Time
	Ended    time.Time
}

func NewVisor(config *Config) *Visor {
	return &Visor{
		ID:      config.ID,
		Config:  config,
		State:   StateCreated,
		created: time.Now(),
	}
}

func (v *Visor) Start(ctx context.Context) error {
	if v.State == StateRunning {
		return ErrAlreadyRunning
	}

	v.State = StateStarting
	v.started = time.Now()

	v.State = StateRunning
	return nil
}

func (v *Visor) Stop() error {
	if v.State != StateRunning {
		return ErrNotRunning
	}

	v.State = StateStopping
	v.State = StateStopped
	return nil
}

func (v *Visor) Stats() (*Stats, error) {
	if v.State != StateRunning {
		return nil, ErrNotRunning
	}

	return &Stats{
		VisorID:  v.ID,
		CPUUsage: 0,
		MemUsage: 0,
	}, nil
}

type Stats struct {
	VisorID    string
	CPUUsage   float64
	MemUsage   int64
	DiskUsage  int64
	NetRxBytes int64
	NetTxBytes int64
	Procs      int
}
