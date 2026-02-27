# Phase 3-prep: E2E Modernization Conceptual Design

**Date:** 2026-02-27
**Purpose:** Draft a plan to rebuild the `e2e` assets, establishing how a library consumer should verify `kube-slint` integration.

## 1. Problem Definition (Why the old legacy_e2e broke)

**Old E2E Model:** Operated under the assumption that `kube-slint` was a running operator container managing cluster resources directly via its own `main.go`. It pulled Docker images, injected CRDs, and generated mock webhooks directly.

**Current Library Model:** `kube-slint` is a pure Go package imported by _other_ Operators. The previous "install the slint manager" code path has completely vanished, leaving the old E2E suite trying to interact with non-existent binaries and ports.

## 2. New Consumer-Centric E2E Goals

### What the New E2E MUST Prove
- That an external operator (e.g. `example-operator`) can import `github.com/HeaInSeo/kube-slint/test/e2e/harness` to orchestrate a test session.
- That the `Session` correctly pulls scraped metrics from a mock HTTP server mimicking `/metrics`.
- That a JSON output (`sli-summary.json`) is correctly generated and scored based on external rules.
- **Bonus:** It must serve as live, compiling Documentation / Tutorial for new developers.

### Out of Scope (What not to prove)
- Internal unit behaviors of the SLI Engine (already covered in `pkg/slo/engine/engine_test.go`).
- End-to-end Kubernetes cluster provisioning logic (e.g. testing `kind` cluster logic). We just want the measurement flow to succeed within Gingko natively.

## 3. Recommended Minimal Structure (Draft)

We shouldn't overcomplicate it. We need a Mock Operator (as a server) and the Harness testing it.

```text
test/
└── e2e/
    ├── example_operator/            <-- A tiny HTTP server serving fixed pseudo-Prometheus metrics
    │   └── mock_server.go
    ├── harness_integration_test.go  <-- The Gingko Suite that brings up the mock server, runs kube-slint harness, and verifies JSON.
    └── Makefile                     <-- (Optional) If it needs a clean isolated `make test-e2e`.
```

## 4. Phasing, Risks, and DoD (Phase 3 Execution)

### Pre-requisites
- Agreement from stakeholders (User + ChatGPT) that the `legacy_e2e` build-tag isolation is to be permanently deleted.

### Step 3-1: Teardown & Mock Setup
1. Remove all `legacy_e2e` marked files fully.
2. Build the `mock_server.go` that simulates an operator exporting standard `controller_runtime` counter formats.

### Step 3-2: Rebuilding the Gingko Orchestration
1. Write a new Gingko suite (`harness_integration_test.go`) that spawns the mock server in the background (or as a separate pod).
2. Use `harness.NewSession(...)` to point the `curlPodFetcher` logic towards the mock.
3. Assert that the `.Attach(...)` block logs properly.

### DoD (Definition of Done)
- [ ] `legacy_e2e` code is 100% removed.
- [ ] New `e2e` tests execute with `$ go test ./test/e2e/...` via standard CI.
- [ ] JSON outputs and `sli-summary.json` paths are verified as accurately produced during integration.

### Risks
- **Flakiness:** If testing against real HTTP servers, port collisions or timeouts could arise. Using `httptest.Server` internally might be safer than full detached mock binaries.
