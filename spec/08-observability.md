# Coco Sandbox - Observability Specification

This document defines the observability architecture for Coco, including logging, metrics, tracing, and health checks.

## 1. Observability Overview

Observability provides insight into system behavior and performance. It enables operators to understand what is happening, diagnose problems, and optimize performance.

Coco implements three pillars of observability: logs, metrics, and traces. Each provides different insights into system behavior.

## 2. Logging

Logs provide detailed records of system events. They are essential for debugging and forensic analysis.

### 2.1 Log Format

All logs use structured JSON format. This allows easy parsing by log aggregation systems.

Each log entry includes timestamp, level, component, message, and contextual fields. Contextual fields include request ID, sandbox ID, and node ID when relevant.

```json
{
  "timestamp": "2024-01-01T00:00:00.000Z",
  "level": "INFO",
  "component": "coco-gateway",
  "message": "Sandbox created",
  "request_id": "req_abc123",
  "sandbox_id": "sb_xyz789",
  "template_id": "ubuntu-base"
}
```

### 2.2 Log Levels

Four log levels indicate severity.

DEBUG provides detailed information for troubleshooting. It includes high-frequency events and detailed state transitions. Production typically disables DEBUG to reduce volume.

INFO provides normal operational information. It includes significant events like sandbox creation and deletion.

WARN indicates potential issues that require attention. It includes degraded performance or approaching limits.

ERROR indicates failures that require investigation. It includes operation failures and exceptions.

### 2.3 Output Configuration

In development, logs are written to stdout and stderr. This allows viewing logs in the terminal.

In production, logs are written to files or syslog. The destination is configurable. Logs are rotated daily to prevent disk exhaustion.

Structured logging to stdout is recommended for containerized deployments. Container orchestration systems handle log collection.

### 2.4 Audit Logging

Security-relevant events are logged to a separate audit log. These events include authentication attempts, authorization failures, and configuration changes.

Audit logs include the identity of the requester, the action performed, and the outcome. They are retained longer than operational logs for compliance and forensic purposes.

## 3. Metrics

Metrics provide quantitative measurements of system behavior. They enable performance monitoring and alerting.

### 3.1 Metrics Format

Metrics are exported in Prometheus format. This allows integration with Prometheus and compatible systems like Thanos or Cortex.

Each metric has a name, type, labels, and value. Labels provide dimensions for filtering and aggregation.

### 3.2 Core Metrics

Sandbox metrics track sandbox lifecycle and performance.

coco_sandbox_count reports the number of sandboxes in each state. Labels include state and node.

coco_sandbox_create_duration_ms reports sandbox creation duration as a histogram. Labels include template and node.

coco_sandbox_fork_duration_ms reports fork duration as a histogram.

coco_sandbox_exec_duration_ms reports command execution duration as a histogram.

Node metrics track resource usage.

coco_node_memory_used_bytes reports memory used on each node.

coco_node_cpu_percent reports CPU usage percentage on each node.

coco_node_sandbox_count reports the number of sandboxes on each node.

Pool metrics track VM pool status.

coco_pool_available reports the number of available VMs in the pool.

coco_pool_in_use reports the number of VMs in use.

Network metrics track network activity.

coco_network_packets_total reports total packets processed.

coco_network_bytes_total reports total bytes processed.

### 3.3 Metrics Endpoint

Metrics are available at /metrics on port 9090 by default.

```
GET /metrics
```

The response is Prometheus text format. It can be scraped by Prometheus or compatible systems.

### 3.4 Dashboards

Grafana dashboards visualize metrics data. The main dashboard shows system overview with key performance indicators.

Panels include sandbox creation latency percentiles, fork latency percentiles, active sandboxes by node, pool utilization, memory usage, and CPU usage.

## 4. Tracing

Traces provide end-to-end visibility into request processing. They show how requests flow through components.

### 4.1 Trace Format

Traces use the OpenTelemetry standard. This allows integration with Jaeger, Zipkin, or other compatible backends.

Each trace consists of spans. A span represents a single operation. Spans have a name, start time, end time, and attributes.

Spans are connected through parent-child relationships. This shows the full request path.

### 4.2 Trace Context

Trace context is propagated through requests using W3C Trace Context headers. This allows traces to span multiple components and even multiple requests.

When a request arrives at Gateway, a trace is started. The trace ID is included in subsequent requests to Master, Node, and Visor. This connects all operations into a single trace.

### 4.3 Sampling

Full tracing generates significant data volume. Sampling reduces this while maintaining visibility.

Errors are always traced. This ensures problems are always visible.

Normal requests are sampled at a configurable rate. The default is 1%. This provides statistical visibility while limiting data volume.

### 4.4 Export

Traces are exported via OTLP to a backend. The backend can be Jaeger, Zipkin, or another compatible system.

Export configuration includes endpoint URL, authentication, and protocol. TLS is used for secure transmission.

## 5. Health Checks

Health checks indicate whether components are functioning correctly. They are used by orchestrators and load balancers.

### 5.1 Liveness Probe

The liveness probe indicates whether the process is running. It should return success as long as the process is alive.

```
GET /health/live
```

The response is a simple success or failure. No dependencies are checked.

This probe is used by Kubernetes to determine whether to restart a failing pod.

### 5.2 Readiness Probe

The readiness probe indicates whether the component can serve requests. It checks dependencies in addition to the process state.

```
GET /health/ready
```

The response includes the overall status and individual dependency status.

```json
{
  "status": "ready",
  "checks": {
    "visor": "ok",
    "store": "ok",
    "node": "ok"
  }
}
```

This probe is used by Kubernetes to determine whether to route traffic to a pod.

### 5.3 Dependency Health

Components check the health of their dependencies. If a dependency is unhealthy, the component reports itself as not ready.

Gateway checks connectivity to Master. If Master is unreachable, Gateway reports not ready.

Node checks connectivity to Visor and local storage. If either is unhealthy, Node reports not ready.

Master checks connectivity to etcd. If etcd is unavailable, Master reports not ready.

## 6. Alerting

Alerts notify operators of conditions requiring attention.

### 6.1 Alert Definitions

Alerts are defined in Prometheus alert rules. They trigger when conditions are met.

High latency alert fires when P99 creation latency exceeds 100ms for 5 minutes. This indicates performance degradation.

Node down alert fires when a node stops reporting. This indicates a node failure.

Pool exhaustion alert fires when available VMs fall below 10% of capacity. This indicates potential capacity issues.

High memory alert fires when node memory usage exceeds 90%. This indicates resource pressure.

### 6.2 Notification

Alerts are sent to notification systems. The configuration includes webhook URLs for Slack, PagerDuty, or other systems.

Alert severity determines notification urgency. Critical alerts page on-call personnel. Warning alerts are sent during business hours.

### 6.3 Alert Lifecycle

Alerts have states. Pending indicates the condition is met but the duration threshold isn't reached. Firing indicates the alert is active. Resolved indicates the condition is no longer met.

Alert history is retained for analysis. This allows understanding of alert patterns and frequencies.

## 7. Comparison

Coco provides more comprehensive observability than alternatives.

| Feature | Container | VM | Coco |
|---------|-----------|-----|------|
| Logs | Yes | Yes | Structured JSON |
| Metrics | Basic | Basic | Comprehensive |
| Tracing | Optional | No | OpenTelemetry |
| Health Checks | Basic | Basic | Deep |
| Alerting | External | External | Built-in |
