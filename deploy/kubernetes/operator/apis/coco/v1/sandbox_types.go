// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 The Coco Sandbox Authors

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Finalizer = "coco.io/sandbox"

	SandboxStatePending = "Pending"
	SandboxStateRunning = "Running"
	SandboxStateStopped = "Stopped"
	SandboxStateError   = "Error"
)

type SandboxSpec struct {
	Template string            `json:"template,omitempty"`
	Memory   int32             `json:"memory,omitempty"`
	VCPUs    int32             `json:"vcpus,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type SandboxStatus struct {
	State     string `json:"state,omitempty"`
	SandboxID string `json:"sandboxId,omitempty"`
	Node      string `json:"node,omitempty"`
}

type Sandbox struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SandboxSpec   `json:"spec,omitempty"`
	Status SandboxStatus `json:"status,omitempty"`
}

func (s *Sandbox) Hub() {}

type SandboxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Sandbox `json:"items"`
}
