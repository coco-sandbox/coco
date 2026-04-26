# Coco Sandbox – Repository conformance

**Scope:** **Reference** deployment parameters (bind addresses, environment variable names) for the open-source Coco line, and how this document relates to the normative architecture. It helps operators and **spec** readers; it does not replace `02-api.md` or `10-self-hosting-and-operations.md` for behavioral requirements. Implementation in the repository is expected to follow this reference; if it does not, either the tree or this file should be updated under normal project process.

**Status:** Authoritative for the **reference** mapping below.  
**Index:** [Specification index](index.md)

## 1. Precedence

| Layer | Role |
|-------|------|
| `spec/*.md` | Product behavior, architecture, and API *shape* |
| `proto/coco/v1/*.proto` | Machine-generated API contracts where used |
| `go.mod` / `build.zig` | Pinned toolchains (see `09-dependencies.md`) |
| `pkg/config` defaults + env | **coco-gateway** (and other binaries) bootstrap values |

If **reference** defaults or variable names in this file stop matching the mainline open-source tree, that drift is treated as a documentation or product defect to be resolved in the normal change process. Normative *behavior* remains in the numbered specifications (`02`–`10`, `12`); this file names concrete **reference** values for convenience.

## 2. coco-gateway: HTTP listen address and public API

| Item | In-tree default | Override environment variable |
|------|-----------------|------------------------------|
| Bind address (HTTP: API, `/v1` routes, `/health/*`, `/metrics` on same mux) | all interfaces, port 4747 (`:4747`) | `COCO_LISTEN_ADDR` |

The public REST surface under **/v1** is served from that single HTTP server. Scheme (TLS or not) is a deployment choice at the load balancer or reverse proxy.

## 3. coco-gateway: rate limiting

| Item | In-tree default in `config.Default()` | Override environment variable |
|------|----------------------------------------|------------------------------|
| Enabled | true | `COCO_RATE_LIMIT_ENABLED` (`1`/`0`, `true`/`false`, `on`/`off`, `yes`/`no`) |
| Sustained RPS (token bucket) | 100 | `COCO_RATE_LIMIT_RPS` (float) |
| Burst | 200 | `COCO_RATE_LIMIT_BURST` (integer) |

When **disabled**, the rate-limit middleware is not applied; operators accept responsibility for throttling at the edge if needed. Normative *semantics* (token bucket, headers) remain in `02-api.md` and `10-self-hosting-and-operations.md`.

## 4. coco-gateway: other environment variables (subset)

| Variable | Effect |
|----------|--------|
| `COCO_MASTER_ADDR` | gRPC/Connect address for the master (default in tree targets `:4746` style layouts) |
| `COCO_DATA_DIR` | Root for images, store, checkpoints, templates, and related subpaths (see `pkg/config`) |
| `COCO_VISOR_SOCKET` | Path to the Visor Unix socket when overridden |
| `COCO_API_KEYS` | Comma-separated `token:user` pairs for static API auth (optional) |
| `COCO_NODE_ID`, `COCO_POOL_SIZE`, `COCO_SCHEDULER_STRATEGY`, `COCO_ETCD_ENDPOINTS` | As used by node/master paths when shared config loads |

`MetricsEnabled` and `MetricsPort` exist in the `Config` struct; **metrics for gateway are registered on the same HTTP server as the API** via **GET /metrics** in the current implementation. A dedicated listener on `MetricsPort` is reserved for future use if the binary is split; scrapers should use the process’s `ListenAddr` **/metrics** until otherwise documented here.

## 5. Readiness and health (gateway)

| Path | Purpose | Typical HTTP status when dependency missing |
|------|---------|-----------------------------------------------|
| `/health` | Redirects to liveness | 301/302 to `/health/live` |
| `/health/live` | Process up | 200, JSON `status: ok` |
| `/health/ready` | Ready to serve; checks cluster/master reachability in tree | 200; body may show `degraded` if `master` check fails (implementation detail) |

## 6. Change discipline (spec stays current)

- When the **public** HTTP path, error code, or resource field set **changes**, **`02-api.md`** and, when used, **proto** and **generated** code are updated in the same change so the spec and tree **match the present design** (see `14-deployment-topology-and-neutrality.md` section 1).
- New **`COCO_*` variable** names or **reference** defaults in this file are reflected here; toolchains remain in **`09-dependencies.md`** and the repository.
- Announcements to integrators **outside** `spec/` are optional product process; **`spec/` does not** carry compatibility history.

---

Related: [10-self-hosting-and-operations.md](10-self-hosting-and-operations.md) (what operators own vs. what the spec defines).
