# kube-slint

`kube-slint` is a shift-left operational quality guardrail for Kubernetes Operators.

> **IMPORTANT:** This repository has transitioned from a standalone operator to a library/observability framework. The `operator runtime` (e.g., `cmd/main.go`, `controller-runtime` manager loops) has been removed.

## Identity and Scope

`kube-slint` is **not** an operator correctness test framework.

It provides a guardrail layer that applies operational SLIs during operator development so teams can detect reliability/performance regressions earlier.

### What kube-slint does

- Defines/evaluates operational SLI specs (`pkg/slo/spec`, `pkg/slo/engine`)
- Produces structured summary output (`sli-summary.json`) with reliability signals
- Supports multiple measurement modes as first-class options
- Provides policy-oriented gating direction for CI (absolute threshold + regression)

### What kube-slint does not do

- It does not replace correctness tests (`go test`, lint, unit/integration tests)
- It does not require production reconcile-path instrumentation (non-invasive principle)
- It does not treat every measurement failure as a test failure by default

## Core Contracts

1. Measurement failure is not equivalent to test failure.
2. Policy violation (absolute threshold miss or regression vs baseline) may fail CI.
3. Guardrail evaluation is separate from correctness testing.

## Measurement Modes (First-class)

- `InsideSnapshot` (default)
- `InsideAnnotation` (precise / semantic-boundary)
- `OutsideSnapshot` (environment-specific)

## Gate Model

- Absolute threshold gate: supported through current SLI judgment rules.
- Regression comparison gate: **in progress** (`Phase 6-c Regression Gate Model`).
- CI visibility for guardrail stages/gates: **in progress** (`Phase 6-d GitHub Actions visibility`).

## Relationship to Tests and CI

- Correctness path: lint/unit/mock-e2e validate implementation behavior.
- Guardrail path: `slint-gate` (planned) evaluates policy outcomes and may fail CI on policy violation.
- This separation keeps measurement reliability issues distinct from correctness failures.

## Canonical Consumer DX Validation

- `hello-operator` is the canonical consumer DX validation repository for kube-slint adoption flows.
- ko+tilt inner-loop validation for this path is **planned/in progress** (`Phase 7-a`).

## How to Use

The repository is now divided into two primary concepts:

1. **Instrumenting your Operator (Go Library)**
2. **Deploying the Observability Stack (Kustomize)**

### 1. Instrumenting your Operator (Go Library)

Use the `pkg/slo` library in your own Operator code to calculate Churn Rate, Convergence Time, and other SLO metrics. You can also embed the harness into your E2E tests:

```sh
go get github.com/HeaInSeo/kube-slint@latest
```

**Real-cluster Integration Options:**
When embedding `kube-slint` in a real-cluster environment (outside of local tests), you may need to bypass self-signed metrics TLS certificates or use a private registry for the `curl` image. You can configure these overrides via the `SessionConfig`:

```go
sess := harness.NewSession(harness.SessionConfig{
    Namespace: "my-operator-system",
    MetricsServiceName: "my-operator-metrics",
    Specs: mySpecs,
    
    // -- Real-cluster Integration Knobs --
    // Bypass x509: certificate signed by unknown authority
    TLSInsecureSkipVerify: true, 
    // Proxy/Private registry pull rate-limits
    CurlImage: "my-private-registry.com/curlimages/curl:latest",
})
```

> **Note on RBAC:** The `kube-slint` curl fetcher runs a temporary pod to scrape metrics. Make sure your controller's `ServiceAccount` has RBAC permissions to `create pods` in the target namespace.

> **Note:** `kube-slint` (the Go code) is responsible for *calculating, evaluating, and reporting* SLI JSON outputs inside the cluster harness. The Kustomize stack is entirely responsible for *deploying* the observability targets.

### 2. Deploying the Observability Stack (Kustomize)

The Kustomize manifests here provide the Prometheus tags, recording rules, and dashboards needed for monitoring `kube-slint` metrics.

**Remote Resource Installation (Recommended)**  
You can embed the observability stack directly into your project's Kustomize overlays. 

> **CRITICAL:** Do NOT use branches like `?ref=main`. You must pin the remote resource to a specific tag or commit SHA to ensure reproducible, immutable builds.

Create a `kustomization.yaml` in your consumer repository. Because the base stack intentionally does **not** hardcode a namespace (Strategy A: Zero-Assumption Base), you must declare which namespace the stack should be deployed to using the `namespace` field in your overlay:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# (Required) Inject the destination namespace into the remote stack
namespace: your-target-namespace

resources:
  # Pin to a specific tag or commit SHA
  - github.com/HeaInSeo/kube-slint//config/default?ref=<tag or commitSHA>
```

> **Note on ServiceMonitors (Explicit Local Override Strategy):** Remote Kustomize fetch (`github.com/...`) works technically, but the `config/samples/prometheus` resources still contain hardcoded labels (e.g., `app.kubernetes.io/name: kube-slint`). If used as a direct drop-in, it will cause a **silent failure** because Prometheus will not scrape your operator's pods.
> 
> As a short-term recommended mitigation, you must use an **Explicit Local Override (Kustomize Strategic Merge Patch)** in your repository to inject your target operator's name into `spec.selector.matchLabels`. 
> *(For a full tutorial and patch examples, see [`test/consumer-onboarding/kustomize-remote-consumer/`](test/consumer-onboarding/kustomize-remote-consumer/README.md))*

---

## Local Development & Testing

Since this project no longer acts as a running service, standard Go testing tools apply.

### Development Commands

We provide standard targets for development and testing. **Always ensure `go mod tidy` passes clean diffs before pushing.**

- `bin/golangci-lint run --timeout=10m --config=.golangci.yml ./...` : Run static analysis.
- `go test ./...` : Run unit tests (including E2E harness simulation).
- `go mod tidy` : Clean missing dependencies.
- `git diff --exit-code` : Validate dependency integrity.

> Many legacy deployment commands (`run`, `docker-build`, `deploy`, `install`, etc.) have been stubbed as no-ops with friendly guidance to prevent confusion for returning developers.

---

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
