# kube-slint

`kube-slint` is a pure Go framework and observability stack for tracking Operational SLIs (Service Level Indicators) in Kubernetes Operators.

> **IMPORTANT:** This repository has transitioned from a standalone operator to a library/observability framework. The `operator runtime` (e.g., `cmd/main.go`, `controller-runtime` manager loops) has been removed. 

## How to Use

The repository is now divided into two primary concepts:

1. **Deploying the Observability Stack (Kustomize)**
2. **Instrumenting your Operator (Go Library)**

### 1. Deploying the Observability Stack (Kustomize)

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

To test the generated output without downloading, you can run a targeted build with a real SHA:
```sh
kustomize build github.com/HeaInSeo/kube-slint//config/default?ref=ca156d34b0efde18bb54fcf1e9d07727e5e4dce3 | kubectl apply -f -
```

**Local Installation**  
Alternatively, if you have cloned the repository locally:
```sh
kustomize build config/default | kubectl apply -f -
```

### 2. Instrumenting your Operator (Go Library)

Use the `pkg/slo` library in your own Operator code to calculate Churn Rate, Convergence Time, and other SLO metrics. 

> **Note:** `kube-slint` (the Go code) is responsible for *calculating and reporting* SLI JSON output. The Kustomize stack is entirely responsible for *deploying* the observability targets. They serve different purposes.

Ensure your Go modules reference the correct version of this project:
```sh
go get github.com/HeaInSeo/kube-slint@latest
```

> **Note on ServiceMonitors & NetworkPolicies:** Base manifests (like `monitor.yaml` or `metrics_service.yaml`) contain labels specific to individual operators. We have moved the `kube-slint` specific legacy components into `config/samples/`. You MUST copy/adapt these samples to match your target operator's metrics service and labels.

---

## Local Development & Testing

Since this project no longer acts as a running service, standard Go testing tools apply.

### Makefile Targets
We provide standard Makefile targets for development and testing:
- **`make build`**: Compiles the library code (`go build ./...`).
- **`make test`**: Runs unit tests (`go test ./...`).
- **`make fmt`**: Formats the codebase.
- **`make vet`**: Vets the codebase.
- **`make lint`**: Runs `golangci-lint` (recommended before submitting PRs).

> Many legacy deployment commands (`run`, `docker-build`, `deploy`, `install`, etc.) have been stubbed as no-ops with friendly guidance to prevent confusion for returning developers.

### Running End-To-End (E2E) Tests
If you want to validate changes against a live cluster:
```sh
make test-e2e
```
*(Requires `kind` installed locally to spin up a transient test cluster)*

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
