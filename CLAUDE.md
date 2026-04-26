# Coco Sandbox

Coco is a sandbox runtime that provides hardware-level isolation using KVM. Each sandbox runs in its own MicroVM.

## Architecture

Control plane in Go: Gateway (REST), Master (cluster), Node (local resources)
Data plane in Zig: Visor (KVM), Agent (init), Fork (cloning)
Network in Go + C/eBPF: XDP filters at kernel level

## Communication

Client → Gateway (REST) → Master (gRPC) → Node (gRPC) → Visor (Unix socket) → Agent (VSock)

## Spec

All specs in `spec/`. Reference them for architectural decisions.

## Rules

No comments unless requested. No version numbers in specs.
