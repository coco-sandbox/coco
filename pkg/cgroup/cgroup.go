package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	CgroupRoot    = "/sys/fs/cgroup"
	CgroupVersion = 2
)

type Manager struct {
	rootPath    string
	sandboxPath string
	sandboxID   string
}

type Limits struct {
	MemoryLimit  int64
	CPUQuota     int64
	CPUPeriod    int64
	IOBandwidth  int64
	MaxProcesses int
}

func NewManager(sandboxID string) *Manager {
	rootPath := CgroupRoot
	if CgroupVersion == 1 {
		rootPath = filepath.Join(CgroupRoot, "coco")
	}
	return &Manager{
		rootPath:    rootPath,
		sandboxID:   sandboxID,
		sandboxPath: filepath.Join(rootPath, "sandbox-"+sandboxID),
	}
}

func (m *Manager) Create(limits Limits) error {
	if err := os.MkdirAll(m.sandboxPath, 0755); err != nil {
		return fmt.Errorf("failed to create cgroup: %w", err)
	}

	if err := m.SetMemoryLimit(limits.MemoryLimit); err != nil {
		return fmt.Errorf("failed to set memory limit: %w", err)
	}

	if err := m.SetCPUQuota(limits.CPUQuota, limits.CPUPeriod); err != nil {
		return fmt.Errorf("failed to set CPU quota: %w", err)
	}

	if limits.MaxProcesses > 0 {
		if err := m.SetMaxProcesses(limits.MaxProcesses); err != nil {
			return fmt.Errorf("failed to set max processes: %w", err)
		}
	}

	return nil
}

func (m *Manager) SetMemoryLimit(limitBytes int64) error {
	if limitBytes <= 0 {
		return nil
	}
	memPath := filepath.Join(m.sandboxPath, "memory.max")
	return os.WriteFile(memPath, []byte(strconv.FormatInt(limitBytes, 10)), 0644)
}

func (m *Manager) SetCPUQuota(quotaUs int64, periodUs int64) error {
	if quotaUs <= 0 && periodUs <= 0 {
		periodUs = 100000
		quotaUs = periodUs * 1
	}

	if periodUs > 0 {
		periodPath := filepath.Join(m.sandboxPath, "cpu.max")
		if quotaUs > 0 {
			content := fmt.Sprintf("%d %d\n", quotaUs, periodUs)
			return os.WriteFile(periodPath, []byte(content), 0644)
		}
	}

	return nil
}

func (m *Manager) SetMaxProcesses(max int) error {
	if max <= 0 {
		return nil
	}
	procPath := filepath.Join(m.sandboxPath, "pids.max")
	return os.WriteFile(procPath, []byte(strconv.Itoa(max)), 0644)
}

func (m *Manager) AddProcess(pid int) error {
	tasksPath := filepath.Join(m.sandboxPath, "cgroup.procs")
	return os.WriteFile(tasksPath, []byte(strconv.Itoa(pid)), 0644)
}

func (m *Manager) GetMemoryUsage() (int64, error) {
	statPath := filepath.Join(m.sandboxPath, "memory.current")
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
}

func (m *Manager) GetCPUUsage() (int64, error) {
	statPath := filepath.Join(m.sandboxPath, "cpu.stat")
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "usage_usec") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return strconv.ParseInt(parts[1], 10, 64)
			}
		}
	}

	return 0, nil
}

func (m *Manager) Destroy() error {
	pidsPath := filepath.Join(m.sandboxPath, "cgroup.procs")
	data, err := os.ReadFile(pidsPath)
	if err == nil {
		pids := strings.Split(string(data), "\n")
		for _, pidStr := range pids {
			pidStr = strings.TrimSpace(pidStr)
			if pidStr == "" {
				continue
			}
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				continue
			}
			syscall.Kill(pid, syscall.SIGKILL)
		}
	}

	return os.RemoveAll(m.sandboxPath)
}

func (m *Manager) Exists() bool {
	_, err := os.Stat(m.sandboxPath)
	return err == nil
}
