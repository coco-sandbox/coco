# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors
#
# eBPF program build system.
# Compiles C eBPF programs to .o files used by coco-net and coco-visor.
#
# Usage:
#   make              # build all eBPF programs
#   make xdp          # build XDP programs only
#   make clean        # remove build artifacts
#
# Output goes to ebpf-out/ directory.

EBPF_OUT := ebpf-out
CLANG ?= clang
LLC ?= llc
BPF_TARGETS := xdp from_host from_sandbox from_world
ARCH := $(shell uname -m | sed 's/x86_64/x86/;s/aarch64/arm/')

.PHONY: all clean $(BPF_TARGETS)

all: $(BPF_TARGETS)

xdp:
	@mkdir -p $(EBPF_OUT)/xdp
	$(CLANG) -O2 -target bpf -g -Wall -Werror \
		-D__TARGET_ARCH_$(ARCH) \
		-Iheaders \
		-I/usr/include \
		-I/usr/include/bpf \
		-c xdp/xdp_filter.bpf.c -o $(EBPF_OUT)/xdp/xdp_filter.o
	$(CLANG) -O2 -target bpf -g -Wall -Werror \
		-D__TARGET_ARCH_$(ARCH) \
		-Iheaders \
		-I/usr/include \
		-I/usr/include/bpf \
		-c xdp/xdp_fwd.bpf.c -o $(EBPF_OUT)/xdp/xdp_fwd.o
	$(CLANG) -O2 -target bpf -Wall -Werror \
		-D__TARGET_ARCH_$(ARCH) \
		-Iheaders \
		-I/usr/include \
		-I/usr/include/bpf \
		-c xdp/xdp_nat.bpf.c -o $(EBPF_OUT)/xdp/xdp_nat.o
	$(CLANG) -O2 -target bpf -g -Wall -Werror \
		-D__TARGET_ARCH_$(ARCH) \
		-Iheaders \
		-I/usr/include \
		-I/usr/include/bpf \
		-c xdp/xdp_conntrack.bpf.c -o $(EBPF_OUT)/xdp/xdp_conntrack.o
	$(CLANG) -O2 -target bpf -g -Wall -Werror \
		-D__TARGET_ARCH_$(ARCH) \
		-Iheaders \
		-I/usr/include \
		-I/usr/include/bpf \
		-c xdp/xdp_stats.bpf.c -o $(EBPF_OUT)/xdp/xdp_stats.o

from_host:
	@mkdir -p $(EBPF_OUT)/from_host
	$(CLANG) -O2 -target bpf -g -Wall -Werror \
		-D__TARGET_ARCH_$(ARCH) \
		-Iheaders \
		-I/usr/include \
		-I/usr/include/bpf \
		-c from_host/from_host.bpf.c -o $(EBPF_OUT)/from_host/from_host.o

from_sandbox:
	@mkdir -p $(EBPF_OUT)/from_sandbox
	$(CLANG) -O2 -target bpf -g -Wall -Werror \
		-D__TARGET_ARCH_$(ARCH) \
		-Iheaders \
		-I/usr/include \
		-I/usr/include/bpf \
		-c from_sandbox/from_sandbox.bpf.c -o $(EBPF_OUT)/from_sandbox/from_sandbox.o

from_world:
	@mkdir -p $(EBPF_OUT)/from_world
	$(CLANG) -O2 -target bpf -g -Wall -Werror \
		-D__TARGET_ARCH_$(ARCH) \
		-Iheaders \
		-I/usr/include \
		-I/usr/include/bpf \
		-c from_world/from_world.bpf.c -o $(EBPF_OUT)/from_world/from_world.o

clean:
	rm -rf $(EBPF_OUT)
