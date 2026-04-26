package filters

import (
	"coco/pkg/scheduler"
)

type CapacityFilter struct{}

func NewCapacityFilter() *CapacityFilter {
	return &CapacityFilter{}
}

func (f *CapacityFilter) Filter(node *scheduler.NodeEntry, req *scheduler.ScheduleRequest) bool {
	if req.MemoryMB > 0 && node.MemMB < req.MemoryMB {
		return false
	}

	if req.VCPUs > 0 && node.CPUs < req.VCPUs {
		return false
	}

	return true
}

type ScheduleRequest struct {
	MemoryMB uint64
	VCPUs    int
	Labels   map[string]string
}

func (f *CapacityFilter) Name() string {
	return "capacity"
}
