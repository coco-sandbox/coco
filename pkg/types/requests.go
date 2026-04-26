package types

import "time"

type CreateSandboxResponse struct {
	Sandbox *Sandbox `json:"sandbox"`
}

type GetSandboxRequest struct {
	ID string `json:"id"`
}

type GetSandboxResponse struct {
	Sandbox *Sandbox `json:"sandbox"`
}

type ListSandboxesResponse struct {
	Sandboxes []*Sandbox `json:"sandboxes"`
}

type DeleteSandboxRequest struct {
	ID string `json:"id"`
}

type PauseSandboxRequest struct {
	ID string `json:"id"`
}

type ResumeSandboxRequest struct {
	ID string `json:"id"`
}

type ForkSandboxRequest struct {
	ParentID string            `json:"parent_id"`
	Name     string            `json:"name"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type ForkSandboxResponse struct {
	Sandbox *Sandbox `json:"sandbox"`
}

type ExecSandboxRequest struct {
	SandboxID  string   `json:"sandbox_id"`
	Cmd        string   `json:"cmd"`
	Args       []string `json:"args"`
	Env        []string `json:"env"`
	WorkingDir string   `json:"working_dir"`
	TimeoutMs  int64    `json:"timeout_ms,omitempty"`
}

type BootSandboxRequest struct {
	SandboxID string `json:"sandbox_id"`
	Template  string `json:"template"`
	MemoryMB  int    `json:"memory_mb"`
	VCPUs     int    `json:"vcpus"`
}

type BootSandboxResponse struct {
	Sandbox *Sandbox `json:"sandbox"`
}

type DestroyVMRequest struct {
	SandboxID string `json:"sandbox_id"`
}

type PauseVMRequest struct {
	SandboxID string `json:"sandbox_id"`
}

type ResumeVMRequest struct {
	SandboxID string `json:"sandbox_id"`
}

type NodeStatus struct {
	NodeID          string    `json:"node_id"`
	ActiveSandboxes int32     `json:"active_sandboxes"`
	PoolFree        int32     `json:"pool_free"`
	MemoryUsedMB    uint64    `json:"memory_used_mb"`
	CPUPercent      int32     `json:"cpu_percent"`
	Healthy         bool      `json:"healthy"`
	LastUpdate      time.Time `json:"last_update"`
}
