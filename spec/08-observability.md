# Coco Sandbox – Observability specification

**Scope:** Logging format, metrics, tracing, health endpoints, and alerting concepts.  
**Status:** Authoritative.  
**Index:** [Specification index](index.md)

## 1. Observability overview

Observability provides insight into system behavior and performance. It enables operators to understand what is happening, diagnose problems, and optimize performance.

Coco implements three pillars of observability: logs, metrics, and traces. Each provides different insights into system behavior.

## 2. Logging

Logs provide detailed records of system events. They are essential for debugging and forensic analysis.

### 2.1 Log Format

All logs use structured JSON format. This allows easy parsing by log aggregation systems.

Each log entry includes at least the following **fields** (all structured, typically as JSON on one line per event):

| Field | Type | Description |
|-------|------|-------------|
| timestamp | ISO 8601 | Event time (UTC) |
| level | string | One of DEBUG, INFO, WARN, ERROR |
| component | string | Logical component name (e.g. coco-gateway) |
| message | string | Human-readable summary |
| request_id | string | Correlation id when part of a request (optional) |
| sandbox_id | string | When the event concerns a sandbox (optional) |
| node_id or template_id | string | Additional context as needed (optional) |

### 2.2 Log levels

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

### 3.3 Metrics endpoint

**Path:** HTTP path **/metrics** on the component’s HTTP server. **Port and bind address** are **deployment-defined** (not fixed to 9090; see `10-self-hosting-and-operations.md`).

**Method:** GET. **Response body:** Prometheus text exposition format, scrapeable by Prometheus or compatible systems.

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

### 5.1 Liveness

**Path:** **/health/live** (on the process HTTP server). **Method:** GET. Indicates the process is alive. Does not check upstream dependencies. Used by orchestrators to restart unhealthy instances. Response body is a small successful payload (for example JSON **status: ok**); exact shape is implementation-defined but must be stable for the same healthy state.

### 5.2 Readiness

**Path:** **/health/ready**. **Method:** GET. Indicates the process can accept work; **may** embed checks for critical dependencies (for example master reachability for Gateway). Response body includes an overall status string and a map of named checks to per-dependency status (e.g. ok, degraded, unreachable). Exact key names follow the implementation; the spec requires that a non-ready result use a non-success HTTP status or an explicit **degraded** or **not ready** outcome as documented for that binary.

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

## 7. Design intent (observability)

Coco is specified to use **structured logs**, **Prometheus metrics**, **OpenTelemetry-style tracing** where enabled, and **liveness/readiness** HTTP endpoints suitable for cloud-native operators. **Alerting rules and notification channels** are operator-defined; Prometheus rule thresholds (for example latency or pool depth) are **examples** in §6, not product constants.
