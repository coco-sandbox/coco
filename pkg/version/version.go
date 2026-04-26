// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package version

import (
	"fmt"
	"strings"
)

// Protocol is the gRPC/API protocol version
const Protocol = "1.0"

// APIVersion returns the API version string
func APIVersion() string {
	return Protocol
}

// IsCompatible checks if a client version is compatible with this server
func IsCompatible(clientVersion string) bool {
	return clientVersion == Protocol
}

// ParseVersion parses a version string into components
func ParseVersion(v string) (major, minor int, ok bool) {
	parts := strings.Split(v, ".")
	if len(parts) != 2 {
		return 0, 0, false
	}
	fmt.Sscanf(parts[0], "%d", &major)
	fmt.Sscanf(parts[1], "%d", &minor)
	return major, minor, true
}

// Compare compares two version strings
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func Compare(a, b string) int {
	am, an, _ := ParseVersion(a)
	bm, bn, _ := ParseVersion(b)
	if am < bm {
		return -1
	}
	if am > bm {
		return 1
	}
	if an < bn {
		return -1
	}
	if an > bn {
		return 1
	}
	return 0
}