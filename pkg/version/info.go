// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package version

import (
	"fmt"
	"runtime"
	"time"
)

// Info holds version information
type Info struct {
	Version       string    `json:"version"`
	GitCommit     string    `json:"git_commit"`
	GitDescribe   string    `json:"git_describe"`
	BuildDate     time.Time `json:"build_date"`
	GoVersion     string    `json:"go_version"`
	Compiler      string    `json:"compiler"`
	Architecture  string    `json:"architecture"`
	OS            string    `json:"os"`
	KVMVersion    string    `json:"kvm_version,omitempty"`
	KernelVersion string    `json:"kernel_version,omitempty"`
}

// Get returns the version info
func Get() Info {
	return Info{
		Version:      Version,
		GitCommit:    GitCommit,
		GitDescribe:  GitDescribe,
		BuildDate:    BuildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Architecture: runtime.GOARCH,
		OS:           runtime.GOOS,
	}
}

// String returns a human-readable version string
func (v Info) String() string {
	return fmt.Sprintf("coco version %s (go=%s, %s/%s)",
		v.Version, v.GoVersion, v.OS, v.Architecture)
}

// Short returns a short version string
func (v Info) Short() string {
	return v.Version
}

// Full returns a full version string with all details
func (v Info) Full() string {
	return fmt.Sprintf("Version: %s\nGitCommit: %s\nGitDescribe: %s\nBuildDate: %s\nGoVersion: %s\nCompiler: %s\nArchitecture: %s\nOS: %s",
		v.Version, v.GitCommit, v.GitDescribe, v.BuildDate.Format(time.RFC3339),
		v.GoVersion, v.Compiler, v.Architecture, v.OS)
}

// vars are set during build via ldflags
var (
	Version     = "v0.0.0"
	GitCommit   = ""
	GitDescribe = ""
	BuildDate   = time.Now()
)
