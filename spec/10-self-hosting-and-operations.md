# Coco Sandbox – Self-hosting, configuration, and open source operations

**Scope:** Normative architecture versus **deployment-tunable** parameters, operator expectations, and fork naming.  
**Status:** Authoritative.  
**Index:** [Specification index](index.md)

This document does not restate the API or network design; it scopes what other specification files require versus what each installation must choose.

---

## 1. Normative versus deployment-defined

- **Normative in `spec/`**: component boundaries, protocols (REST, gRPC, Unix socket, VSock), isolation model, security layers, error shapes, and observable behavior classes (e.g. default-deny network, token-bucket style limits where applicable).
- **Not normative in `spec/`**: concrete numeric limits for request rates, per-sandbox network rates, default pool sizes, retention days, or listen ports, except where a document explicitly states *example* or *illustrative*.

Operators and distributions choose limits based on capacity, policy, and threat model. **Illustrative YAML or header examples** in other spec files (for example, rate limit headers) show **shape and semantics**, not product defaults for every build.

---

## 2. Configuration and observability of limits

**Single source in deployments**: All tunables (listen addresses, TLS, etcd endpoints, rate limits, auth material, data directories, feature flags) are supplied by **operator configuration** as implemented in the repository (for example, environment variables, config files, or both). The specification does not require a single configuration file format; it requires that every installation can override limits without recompiling for policy reasons.

**Documentation rule**: `configs/` examples in the repository are **templates**. They are not a guarantee that a given binary loads every file by default. Operators should follow the runtime documentation for each binary.

**When limits apply**:

| Layer            | What is configurable (conceptually)     | Spec reference        |
|-----------------|----------------------------------------|------------------------|
| Gateway HTTP     | Per-key or global request rate, enable/disable, burst | `02-api.md` (semantics) |
| Coco Net / eBPF  | Packets/s, bytes/s, burst per policy or sandbox | `03-network.md`       |
| API surface      | Authentication, scopes, optional MTLS  | `04-security.md`      |

---

## 3. Gateway rate limiting (open source expectations)

- **Algorithm (normative)**: Token bucket (or equivalent) with optional identification per API key or client identity, with standard `X-RateLimit-*` response headers where rate limiting is active.
- **Parameters (deployment-defined)**: Sustained rate, burst size, and whether limiters are on at all. **No fixed requests-per-second value is part of the Coco architecture norm**; a value chosen for a public SaaS, a private cluster, or a developer laptop are all valid.
- **Operator documentation**: Distributors and deployers must document the effective limits for their build or helm chart, not only copy numbers from an API example in `02-api.md`.

---

## 4. License and third-party expectations

- **Licensing**: The project license is defined in the repository root (for example, `LICENSE`). The specification does not restate full license text.
- **Third-party SDKs and compatibility**: E2B-oriented client examples describe **compatibility goals** for migration; exact coverage depends on the implementation version. Client code should target documented endpoints and error codes, not a vendor name in isolation.

---

## 5. Threat model: operator responsibility

- **Provider boundary**: The architecture assumes a **trust boundary** between the operator (host, control plane, configuration) and sandbox workloads. The operator secures the API, secrets storage, network egress policy, and updates.
- **Not implied**: That any default number in an example file alone satisfies compliance or abuse prevention. Operators with public APIs combine rate limits with authentication, quotas, billing, and WAF or edge controls as needed.

---

## 6. Forks and product naming

- Forks that ship incompatible APIs, ports, or defaults should **use a distinct name or prefix** in user-facing clients and documentation to avoid confusion with upstream Coco behavior. The spec remains the contract for a **Coco**-shaped system; derived products should document their deltas.

---

## 7. API stability and evolution

- Breaking HTTP path or JSON field changes to documented resources should be treated as a **versioned** or **compatibly announced** change in project policy (outside this document). The spec describes the current intended surface; the repository defines release practice.

---

## 8. Related reading

| Topic              | Document              |
|--------------------|-----------------------|
| API shapes         | `02-api.md`           |
| Network policy     | `03-network.md`       |
| Isolation         | `04-security.md`      |
| SLO-style targets  | `06-performance.md`   (targets, not universal defaults) |
| Build requirements | `09-dependencies.md`  (toolchain and environment)     |
