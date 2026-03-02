# kube-slint

`kube-slint` is a pure Go framework, E2E test harness, and observability stack for tracking Operational SLIs (Service Level Indicators) in Kubernetes Operators.

> **IMPORTANT:** This repository has transitioned from a standalone operator to a library/observability framework. The `operator runtime` (e.g., `cmd/main.go`, `controller-runtime` manager loops) has been removed.

## Features

- **SLI Declarative Specifications (`pkg/slo/spec`)**: Create and enforce metrics definitions like Churn Rate, Convergence Time, etc.
- **Test Harness (`test/e2e/harness`)**: An embeddable E2E testing framework that executes inside Kubernetes clusters, evaluates SLIs over time, and generates strictly formatted JSON reports (`summary.json`) using configurable reliability and strictness scoring.
- **Orphan Sweeper**: Ensures robust cleanup of test infrastructure across runs using `report-only` and `delete` modes.

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
