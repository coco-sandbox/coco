# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 The Coco Sandbox Authors

CLANG ?= clang
CFLAGS := -O2 -target bpf -I../headers

OBJS := tc_shaper.o

all: $(OBJS)

%.o: %.bpf.c
	$(CLANG) $(CFLAGS) -c $< -o $@

clean:
	rm -f $(OBJS)

.PHONY: all clean
