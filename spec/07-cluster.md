# Coco Sandbox – Cluster specification

**Scope:** Master–node coordination, scheduling, failover, and Kubernetes integration patterns.  
**Status:** Authoritative.  
**Index:** [Specification index](index.md)

## 1. Cluster architecture overview

Coco operates as a distributed system across multiple machines. The cluster provides high availability, horizontal scalability, and resource efficiency. Components are organized into masters and nodes, with clear separation of concerns.

Masters coordinate cluster operations. They maintain cluster state, make scheduling decisions, and handle failover. Nodes execute sandboxes. They report their capacity to masters and execute sandbox operations.

## 2. Master Components

The master components handle cluster coordination.

### 2.1 Leader Election

One master acts as leader at any time. The leader handles all scheduling decisions and cluster modifications. Other masters remain available as hot standbys.

Leader election uses etcd with the Raft consensus algorithm. When the leader fails, etcd automatically promotes a new leader within seconds.

The promotion process is transparent to clients. Requests continue to be served during the transition. Some requests may fail during the brief transition period, and clients should retry.

### 2.2 Cluster State

Cluster state is stored in etcd. This includes the node registry, sandbox registry, scheduling queue, and configuration.

State changes go through the Raft log, ensuring consistency across replicas. Read operations can be served by any master.

The sandbox registry tracks all sandboxes in the cluster, including their current node, state, and configuration. This allows the scheduler to make informed placement decisions.

### 2.3 Scheduling

When a sandbox creation request arrives, the master selects the optimal node based on current capacity, load, and sandbox requirements.

The scheduler considers several factors. Available memory must exceed the requested sandbox memory. Available vCPUs must meet the request. Node health must be good. Some sandboxes may have affinity requirements.

Multiple scheduling strategies are supported. Least loaded places sandboxes on nodes with the most available resources. Binpack packs sandboxes tightly to minimize resource fragmentation. Random distributes load evenly for better fault tolerance.

### 2.4 Request Routing

The master routes requests to appropriate nodes. When a sandbox is created, the master selects a node and forwards the request. Subsequent operations on that sandbox are routed to the same node.

When a node fails, the master marks the node as unavailable. New sandbox requests are routed to healthy nodes. Existing sandboxes on the failed node may be recovered if checkpoints exist.

## 3. Node Components

Node components handle local resource management.

### 3.1 Node Registration

When a node starts, it registers itself in etcd. The registration includes node ID, address, capacity, and current load.

The node maintains an ephemeral key with a time-to-live. It must periodically renew the key to prove it is still alive. If the key expires, the node is marked as unavailable.

### 3.2 Resource Reporting

Nodes continuously report their resource usage to the master. This includes memory used, CPU used, and sandbox count.

The master uses this information for scheduling decisions. A node reporting high load will receive fewer new sandboxes.

### 3.3 Sandbox Execution

Nodes execute sandbox operations. When a create request arrives, the node creates a VM from a pre-allocated pool. When a delete request arrives, the node destroys the VM and returns resources to the pool.

Each node maintains a VM pool for fast creation. Pre-allocated VMs are ready to use immediately, avoiding the delay of VM creation.

### 3.4 Health Monitoring

Nodes monitor the health of local components. This includes the Visor process, the network agent, and the VM pool.

If a component becomes unhealthy, the node reports this to the master. The master can then take appropriate action, such as rescheduling sandboxes.

## 4. Failover

Failover ensures the cluster continues operating when components fail.

### 4.1 Node Failure

When a node stops sending heartbeats, the master marks it as failed. All sandboxes on that node are marked as failed.

If checkpoints exist, sandboxes can be automatically restored on other nodes. The master selects new nodes based on current capacity and restores from checkpoints.

If no checkpoints exist, the sandbox cannot be recovered. Client applications are notified of the failure and can choose to recreate the sandbox.

### 4.2 Master Failure

When the master leader fails, etcd promotes a new leader. The promotion happens automatically and typically completes within seconds.

During the promotion, some operations may be delayed. Clients should implement retry logic to handle brief unavailability.

### 4.3 Component Failure

Individual components can fail without bringing down the entire node. If Visor fails, the node reports the failure to the master. Sandboxes on that node are rescheduled.

If the network agent fails, network connectivity is lost. The node reports this to the master, and sandboxes are rescheduled to healthy nodes.

## 5. Kubernetes Integration

Coco integrates with Kubernetes for container orchestration.

### 5.1 Operator

The Kubernetes operator manages Sandbox custom resources. When a Sandbox resource is created, the operator creates the sandbox in the Coco cluster. When the resource is deleted, the operator destroys the sandbox.

The operator runs as a Deployment with multiple replicas for high availability. It watches for Sandbox resources and reconciles them with the Coco cluster state.

### 5.2 Custom resource (Sandbox)

The **Sandbox** custom resource is versioned under an API group (for example **coco.io/v1** as implemented). Normative **fields**:

| Section | Field | Description |
|---------|-------|-------------|
| metadata | name | Kubernetes resource name |
| spec | template | Template id or reference |
| spec | memory, vcpus | Integer resource requests (MiB, vCPU count) |
| spec | labels | String key/value map for scheduling or selection |
| status | state | Observed lifecycle state in the cluster |
| status | sandboxId | Coco sandbox id once created |
| status | node | Assigned node id or name when known |

The exact **apiVersion** and **field names** may match the operator implementation; the table above is the logical contract the operator must reconcile with the cluster API in `02-api.md`.

### 5.3 Helm chart

The Helm chart simplifies deployment. It installs the operator, RBAC resources, and required custom resource definitions.

The chart is configurable for different environments. Production deployments might use external etcd, while development can use the embedded etcd.

### 5.4 Service Integration

Services can reference sandboxes through Kubernetes services. The proxy component routes requests to sandbox endpoints.

This allows sandboxes to be integrated into Kubernetes applications seamlessly. Standard service discovery and load balancing work with sandboxes.

## 6. Cluster Sizing

Cluster size depends on capacity requirements and availability needs.

### 6.1 Capacity Planning

Each node can run approximately 2000 sandboxes with default settings. For larger sandboxes, capacity decreases proportionally.

Memory is the primary constraint. A node with 128GB RAM can run 2000 sandboxes at 512MB each, or 30 sandboxes at 4GB each.

CPU is the secondary constraint. A node with 64 cores can run approximately 2000 sandboxes at 1 vCPU each.

### 6.2 Availability Planning

For high availability, multiple masters should be deployed. Three masters provide fault tolerance against one failure. Five masters provide fault tolerance against two failures.

Nodes should be spread across availability zones when possible. This protects against zone-level failures.

### 6.3 Scaling

Clusters can scale horizontally by adding nodes. The master automatically incorporates new nodes into the scheduling pool.

Clusters can also scale vertically by adding resources to existing nodes. More memory and CPU allow higher capacity.

## 7. Cluster Monitoring

Cluster health is continuously monitored.

### 7.1 Node Health

Node health is tracked through heartbeats. If a node misses heartbeats, it is marked as unhealthy.

Healthy nodes report their capacity and load. This information is used for scheduling.

### 7.2 Master Health

Master health is tracked through etcd leader status. If the leader fails, etcd promotes a new leader.

Clients can query any master for cluster information. Masters proxy requests to the leader as needed.

### 7.3 Metrics

Cluster metrics include total sandboxes, running sandboxes, failed sandboxes, node count, healthy node count, and scheduler queue depth.

These metrics are exported to Prometheus for alerting and visualization.
