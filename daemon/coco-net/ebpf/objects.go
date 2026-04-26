// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package ebpf

import (
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
)

type XDPObjects struct {
	Filter *ebpf.Program `ebpf:"xdp_filter"`
	Fwd    *ebpf.Program `ebpf:"xdp_fwd"`
}

type Objects struct {
	XDP XDPObjects
}

func LoadXDPObjects() (*Objects, error) {
	spec := &ebpf.CollectionSpec{
		Programs: map[string]*ebpf.ProgramSpec{
			"xdp_filter": {
				Type:       ebpf.XDP,
				AttachType: ebpf.AttachXDP,
				Instructions: asm.Instructions{
					asm.Mov.Imm(asm.R0, 0),
					asm.Return(),
				},
				License: "Apache-2.0",
			},
			"xdp_fwd": {
				Type:       ebpf.XDP,
				AttachType: ebpf.AttachXDP,
				Instructions: asm.Instructions{
					asm.Mov.Imm(asm.R0, 2),
					asm.Return(),
				},
				License: "Apache-2.0",
			},
		},
		Maps: map[string]*ebpf.MapSpec{
			"flow_table": {
				Type:       ebpf.Hash,
				KeySize:    20,
				ValueSize:  32,
				MaxEntries: 65536,
			},
			"policy_table": {
				Type:       ebpf.Hash,
				KeySize:    8,
				ValueSize:  24,
				MaxEntries: 1024,
			},
		},
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}

	return &Objects{
		XDP: XDPObjects{
			Filter: coll.Programs["xdp_filter"],
			Fwd:    coll.Programs["xdp_fwd"],
		},
	}, nil
}

func (o *Objects) Close() error {
	var errs []error

	if o.XDP.Filter != nil {
		errs = append(errs, o.XDP.Filter.Close())
	}

	if o.XDP.Fwd != nil {
		errs = append(errs, o.XDP.Fwd.Close())
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to close objects: %v", errs)
	}

	return nil
}