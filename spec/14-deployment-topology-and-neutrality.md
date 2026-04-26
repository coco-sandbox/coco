# Coco Sandbox – Deployment topology, neutrality, and how the spec is written

**Scope:** How to read `spec/` as **always-current** truth; **neutrality** of the open-source design; and where **community-supplied worker nodes** (a “semi-decentral” *pattern*) sit relative to the core architecture.  
**Status:** Authoritative for *process and positioning*; it does not add new resource **MUST**s for the data plane (those stay in `02`–`08`).  
**Index:** [Specification index](index.md)

---

## 1. The specification is always the current design

The `spec/` directory is the **single source of truth** for what Coco **is** as a system: components, protocols, isolation model, and public API **shape**.

- The spec is written as **the present product of design**, not as a **history** of how the system used to behave.
- **Do not** read `spec/` for “backward compatibility”, “deprecation timelines”, or “what we supported last quarter.” If the design changes, **the spec is updated** to match. Narratives about **change over time** belong in **project or release documentation outside `spec/`**, not in this tree.
- **Toolchain pins** (for example `go.mod`) live in the repository; **`09-dependencies.md`** points to them. The spec does not **version** the product; it describes **behavior and structure** for the current tree.

This keeps the documentation **fresh by construction**: one reader, one coherent story.

---

## 2. Neutrality of the open-source project

Coco as specified here is **neutral** with respect to:

- **Token** or **blockchain** mechanisms (none are part of the architecture norm).
- **Any particular commercial platform**, marketplace, or “grass”-style community product that might **use** Coco.
- **How** operators monetize, gate, or reputation-score users (product decisions **above** the engine).

The engine defines **isolation**, **execution**, **network policy classes**, and **control-plane roles** (`00`–`08`). Everything that looks like **business logic, community governance, or global scheduling across untrusted operators** is **out of scope** for the core spec unless added as a **separate, explicit** extension with its own normative text.

---

## 3. “Semi-decentral” and community worker nodes (informative pattern)

Many teams want **hardware from a community** (contributed nodes) **without** putting that into the core protocol as a tokenized network.

**How that maps to this spec:**

| Idea | Where it lives in a typical product | In this spec (Coco core) |
|------|-----------------------------------|----------------------------|
| **One operator, many hosts** that are **invited or allowlisted** (community runs `coco-node` into **your** cluster) | Your **onboarding**, credentials, and trust policy | Cluster model in `07-cluster.md`: **Node** registers with **Master** / **etcd** inside **one** trust domain |
| **Several independent Coco installations** (each with its own Master) and a **router** that picks where a job runs | A **separate product** or service (another repository) | Each installation still matches `00`–`08`; the **router** is not defined here |
| **Global peer-to-peer** scheduling with no central coordinator | Not described in core Coco | Would be a **different** system or a **major** future extension, not implied today |

So: **“Semi-decentral” in the sense of *community-supplied workers*** is **architecturally compatible** with Coco by **operating Nodes under a Master you trust**, or by **composing multiple Coco clusters** in a **platform layer**. The spec **does not** require tokens or an on-chain layer; **trust and policy** are **operator-defined** (`10-self-hosting-and-operations.md`).

Coco stays **neutral**: it is an **engine**; **your** platform (separate repo) can implement **community node programs**, reputation, and routing **on top**.

---

## 4. Related reading

| Topic | Document |
|-------|----------|
| Cluster roles, Master, Node | `07-cluster.md` |
| What is fixed by architecture vs operator | `10-self-hosting-and-operations.md` |
| Use-case narratives (including agent stacks) | `13-use-cases-and-consumers.md` |
| Single-source table | `12-glossary-and-conventions.md` section 4 |

---

*This file exists so `spec/` stays **neutral** and **timeless** in tone: current design only, no compatibility history, and a clear line between **Coco** and **products built on Coco**.*
