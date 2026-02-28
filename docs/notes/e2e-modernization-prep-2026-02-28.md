# Phase 3-prep: E2E Modernization Conceptual Design

**Date:** 2026-02-28
**Purpose:** Draft a plan to rebuild the `e2e` assets, establishing how a library consumer should verify `kube-slint` integration.

## 1. Problem Definition & Background

**Why prep only?** We are taking a "policy first, small diff execution" approach to avoid breaking things aggressively before agreeing on the target state.

**Current State (Context):**
- Step 7 / T-2 / T-3 completed (Harness is stable).
- E2E Final Verification completed using fallback mocks.
- Old `legacy_e2e` is quarantined cleanly via build tags.
- The project is now firmly a **library**, not a standalone deployment.

**The Problem:**
The old `legacy_e2e` assumes `kube-slint` runs as its own controller in the cluster. It tries to deploy manifests, wait for pods, and curl its own endpoints. None of this exists anymore. We have a solid harness, but no living, breathing example of a "Consumer Operator" running and being measured by it in a real integration test.

## 2. New Consumer-Centric E2E Goals

### What the New E2E MUST Prove
- **Consumer Integration Flow:** That an external operator (e.g., `example-operator`) can import `github.com/HeaInSeo/kube-slint/test/e2e/harness` to orchestrate a test session.
- **Minimal Success Path:** That the `Session` correctly pulls scraped metrics from a realistic (or mock) target.
- **Documentation Value:** It must serve as live, compiling Documentation / Tutorial for new developers ("How do I use this library?").

### Out of Scope (What not to prove in this phase)
- **Massive Deletions/Moves:** We won't blindly delete `legacy_e2e` until the new suite proves it works.
- **Behavior Changes:** We are not changing *how* the harness evaluates SLIs, just how we test it.
- **CI Policy Overhauls:** We won't break the main CI pipeline immediately.
- **Full Cluster Provisioning (e.g., Kind):** We want a lightweight Mock server first, not a full K8s cluster spin-up if avoidable.

## 3. Test Asset Classification (Draft)

| Asset / Path | Current Role | CI / Runtime Impact | User-facing | Legacy | Candidate Action | Rationale | Immediate Change (Y/N) |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `test/e2e/harness/session_test.go` | Harness logic validation | CI runs this | No | No | Keep | Core fallback test. | N |
| `test/e2e/e2e_test.go` | Old Manager Integration | Ignored (Build tag) | No | Yes | Isolate -> Delete later | Broken library paradigm. | N |
| `test/e2e/manifests/` | Old YAMLs for deployment | Ignored | No | Yes | Isolate -> Delete later | Useless for a library. | N |
| (Proposed) `test/e2e/example_operator/` | Consumer Mock Target | Will run in CI | Yes (Docs) | No | Create | Proves library usage. | N |

## 4. Recommended Minimal Structure (Draft)

We shouldn't overcomplicate it. We need a Mock Operator (as a server) and the Harness testing it.

```text
test/
└── e2e/
    ├── example_operator/            <-- A tiny HTTP server serving fixed pseudo-Prometheus metrics
    │   └── mock_server.go
    ├── harness_integration_test.go  <-- The Gingko Suite that brings up the mock server, runs kube-slint harness, and verifies JSON.
    └── Makefile                     <-- (Optional) If it needs a clean isolated `make test-e2e`.
```

## 5. Phasing, Risks, and DoD (Phase 3 Execution)

### Pre-requisite for Entry (The Gateway)
- Agreement from stakeholders (User + ChatGPT) that the general architecture of the Mock Consumer is sound.

### Step 3-1: Mock Consumer Foundation
1. Create `test/e2e/example_operator/mock_server.go`.
2. Ensure it serves a static `/metrics` payload reflecting standard `controller_runtime` metrics.

### Step 3-2: Rebuilding the Gingko Orchestration
1. Write a new Gingko suite (`harness_integration_test.go`) that spawns the mock server locally.
2. Use `harness.NewSession(...)` to point the fetcher logic towards the mock.
3. Assert that `.Attach(...)` evaluates the metrics and generates `sli-summary.json`.

### Step 3-3: Legacy Teardown
1. Only *after* Step 3-2 passes in CI, remove the `legacy_e2e` build-tagged files and `manifests/`.

### DoD (Definition of Done)
- [ ] New `e2e` tests execute with `$ go test ./test/e2e/...` via standard CI.
- [ ] JSON outputs and `sli-summary.json` paths are verified as accurately produced during integration.
- [ ] `legacy_e2e` code is 100% removed (Condition: Steps 3-1 and 3-2 are merged).

### Risks
- **Flakiness:** Port collisions or mock server shutdown race conditions in Ginkgo. Mitigation: Use `httptest.Server` internally within the test instead of a detached binary if possible.
