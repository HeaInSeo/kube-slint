# kube-slint

A shift-left operational SLI guardrail library for Kubernetes Operators.

> **Transition note:** kube-slint is not a standalone operator. It is an embeddable Go library and CLI toolchain. Earlier documentation referred to a standalone operator model; that model has been retired. The current design embeds SLI collection directly into your E2E test session and gates CI via a Go CLI binary (`cmd/slint-gate`).

---

## Identity and Scope

### What kube-slint does

- Collects operational SLI metrics (reconcile rates, workqueue depth, REST client errors) from a running operator during an E2E test session.
- Evaluates collected metrics against a declarative policy (`policy.yaml`) to produce a gate result.
- Detects regressions against a stored baseline.
- Writes structured JSON artifacts (`sli-summary.json`, `slint-gate-summary.json`) for CI consumption and audit.
- Renders a markdown step summary to GitHub Actions via `--github-step-summary`.

### What kube-slint does not do

- kube-slint is not a correctness test framework. It does not assert that your operator behaves correctly.
- kube-slint is not a monitoring or alerting system. It produces point-in-time guardrail results for a test run, not continuous production metrics.
- kube-slint does not fail your E2E test on measurement failure. A failed metric scrape is recorded but does not abort the test session (see Core Contracts).

---

## Core Contracts

1. **Measurement failure is not a test failure.** If kube-slint cannot scrape a metric, the result is recorded as unmeasured. The E2E test continues.
2. **Policy violation may fail CI.** A threshold miss or regression detected against baseline causes the gate step to exit with a non-zero code, failing the CI job.
3. **Guardrail evaluation is separate from correctness testing.** Your existing E2E assertions and kube-slint gate results are independent signals. Both can fail independently.

---

## How It Works

```
E2E Test Process
     |
     |--- sess.Start() --------> begins metric observation
     |
     | (run your E2E scenario)
     |
     |--- sess.End(ctx) -------> collects metrics, evaluates SLI specs
                                  writes artifacts/sli-summary.json
                                         |
                                         v
                              slint-gate CLI
                         (cmd/slint-gate binary)
                                  |
                         reads sli-summary.json
                         reads .slint/policy.yaml
                         reads baseline (optional)
                                  |
                                  v
                       slint-gate-summary.json
                                  |
                         gate_result: PASS
                                     WARN
                                     FAIL      ---> CI fails
                                     NO_GRADE
```

---

## Quick Start

**Step 1: Add the dependency**

```sh
go get github.com/HeaInSeo/kube-slint
```

**Step 2: Embed the harness in your E2E test**

```go
import "github.com/HeaInSeo/kube-slint/test/e2e/harness"

sess := harness.NewSession(harness.SessionConfig{
    Namespace:             "my-operator-system",
    MetricsServiceName:    "my-operator-controller-manager-metrics-service",
    ArtifactsDir:          "artifacts",
    Specs:                 harness.DefaultV3Specs(),
    TLSInsecureSkipVerify: true,
    CurlImage:             "my-registry/curlimages/curl:latest",
})
sess.Start()
// ... run your E2E scenario ...
sess.End(ctx)
```

**Step 3: Gate the result**

```sh
make slint-gate   # builds bin/slint-gate
./bin/slint-gate --measurement-summary artifacts/sli-summary.json \
                 --policy .slint/policy.yaml \
                 --output slint-gate-summary.json
```

Check the gate result:

```sh
jq -r '.gate_result' slint-gate-summary.json
```

---

## Detailed Usage

### 1. Define SLI Specs

**Using preset specs**

`DefaultV3Specs()` (alias: `BaselineV3Specs()`) returns a preset spec set designed for kubebuilder-generated operators. It covers:

| ID | Description |
|---|---|
| `reconcile_total_delta` | Total reconcile invocations during the session |
| `reconcile_success_delta` | Successful reconcile invocations |
| `reconcile_error_delta` | Failed reconcile invocations |
| `workqueue_adds_total_delta` | Items added to the workqueue |
| `workqueue_retries_total_delta` | Workqueue retry count |
| `workqueue_depth_end` | Workqueue depth at session end |
| `rest_client_requests_total_delta` | Total REST client requests |
| `rest_client_429_delta` | Rate-limit (429) responses received |
| `rest_client_5xx_delta` | Server error (5xx) responses received |

```go
specs := harness.DefaultV3Specs()
```

**Defining custom SLI specs**

```go
import (
    "github.com/HeaInSeo/kube-slint/pkg/slo/spec"
)

mySpecs := []spec.SLISpec{
    {
        ID:    "reconcile_error_delta",
        Title: "Reconcile Error Delta",
        Unit:  "count",
        Kind:  "delta_counter",
        Inputs: []spec.MetricRef{
            spec.PromMetric("controller_runtime_reconcile_total", spec.Labels{"result": "error"}),
        },
        Compute: spec.ComputeSpec{Mode: spec.ComputeDelta},
        Judge: &spec.JudgeSpec{Rules: []spec.Rule{
            {Op: spec.OpGT, Target: 0, Level: spec.LevelFail},
        }},
    },
}
```

Pass `mySpecs` as the `Specs` field in `SessionConfig`.

---

### 2. Embed Harness in E2E Tests

```go
import "github.com/HeaInSeo/kube-slint/test/e2e/harness"

sess := harness.NewSession(harness.SessionConfig{
    // Target namespace where your operator runs
    Namespace: "my-operator-system",

    // Name of the Kubernetes Service exposing /metrics
    MetricsServiceName: "my-operator-controller-manager-metrics-service",

    // Directory where sli-summary.json will be written
    ArtifactsDir: "artifacts",

    // SLI spec set — use preset or custom
    Specs: harness.DefaultV3Specs(),

    // Real-cluster settings
    TLSInsecureSkipVerify: true,
    CurlImage:             "my-registry/curlimages/curl:latest",
})

sess.Start()
// ... run your E2E scenario here ...
sess.End(ctx)
```

**RBAC note:** The harness uses a curl-based fetcher that creates a temporary pod to scrape the metrics endpoint. The operator's ServiceAccount must have `pods: create` permission in the target namespace.

**Output:** `artifacts/sli-summary.json` is written by `sess.End(ctx)`. This file is the input to the slint-gate CLI.

---

### 3. Gate the Result (slint-gate CLI)

The gate CLI is a Go binary located in `cmd/slint-gate`. It supersedes the legacy `hack/slint_gate.py` Python script, which is retained for reference only.

**Build**

```sh
make slint-gate
# produces bin/slint-gate

# or run directly without building
go run ./cmd/slint-gate [flags]
```

**Flags**

| Flag | Default | Description |
|---|---|---|
| `--measurement-summary` | `artifacts/sli-summary.json` | Path to the SLI summary produced by the harness |
| `--policy` | `.slint/policy.yaml` | Path to the policy file |
| `--baseline` | `""` (disabled) | Path to a baseline summary for regression comparison; omit to skip |
| `--output` | `slint-gate-summary.json` | Path to write the gate result JSON |
| `--github-step-summary` | false | Write markdown to `$GITHUB_STEP_SUMMARY` for GitHub Actions |

**Exit behavior:** The binary always exits 0. The CI workflow fails by inspecting `gate_result` in the output JSON via `jq`.

**Policy file (`.slint/policy.yaml`)**

```yaml
schema_version: "slint.policy.v1"
thresholds:
  - name: "reconcile_total_delta_min"
    metric: "reconcile_total_delta"   # must match results[].id in sli-summary.json
    operator: ">="
    value: 1
  - name: "workqueue_depth_end_max"
    metric: "workqueue_depth_end"
    operator: "<="
    value: 5
regression:
  enabled: true
  tolerance_percent: 5
reliability:
  required: false
  min_level: "partial"
fail_on:
  - "threshold_miss"
  - "regression_detected"
```

**Gate result values**

| Result | Meaning |
|---|---|
| `PASS` | All threshold and regression checks passed |
| `WARN` | Non-blocking issue (e.g., first run without baseline, reliability below minimum) |
| `FAIL` | Policy violation — threshold miss or regression detected; CI fails |
| `NO_GRADE` | Evaluation not possible — missing or corrupt inputs |

---

### 4. Deploy Observability Stack (Kustomize)

kube-slint ships a Kustomize base that installs the metrics collection infrastructure into your cluster.

**Reference the remote base**

```yaml
# your overlay/kustomization.yaml
resources:
  - github.com/HeaInSeo/kube-slint//config/default?ref=<tag-or-SHA>
```

Rules:
- Always pin to a tag or SHA. Never use `?ref=main`.
- Always declare `namespace:` in your overlay. The base makes no namespace assumptions.

**ServiceMonitor labels**

The ServiceMonitor in the base has hardcoded labels. You must override them with a strategic merge patch to match your Prometheus operator's selector.

```yaml
# overlay/patch-servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kube-slint-controller-manager-metrics-monitor
  namespace: <your-namespace>
spec:
  selector:
    matchLabels:
      # override with labels your Prometheus operator selects
      app: my-operator
```

A full onboarding tutorial is available at `test/consumer-onboarding/kustomize-remote-consumer/`.

---

## Measurement Modes

kube-slint supports three first-class measurement modes, set per-session or per-spec:

| Mode | Description |
|---|---|
| `InsideSnapshot` (default) | Snapshot-based collection at session start and end; delta is computed from the difference |
| `InsideAnnotation` | Precise semantic-boundary collection; measurement aligns with annotated test boundaries |
| `OutsideSnapshot` | External scrape; metrics are collected from an external source rather than inside the session |

---

## Gate Model

Both gate model components are complete.

**Threshold checking** (DONE): Each metric result in `sli-summary.json` is evaluated against the threshold rules in `policy.yaml`. A threshold miss sets `gate_result` to `FAIL` if `threshold_miss` is listed in `fail_on`.

**Regression detection** (DONE): When `--baseline` is provided, each metric result is compared to the stored baseline value. If the change exceeds `tolerance_percent`, the result is flagged as a regression. Regression detection sets `gate_result` to `FAIL` if `regression_detected` is listed in `fail_on`.

---

## CI Integration

The `.github/workflows/slint-gate.yml` workflow evaluates the gate after your E2E tests upload `sli-summary.json` as an artifact.

```yaml
- name: Evaluate slint gate
  run: go run ./cmd/slint-gate --github-step-summary

- name: Upload gate summary
  uses: actions/upload-artifact@v4
  with:
    name: slint-gate-summary
    path: slint-gate-summary.json

- name: Check gate result
  run: |
    result=$(jq -r '.gate_result' slint-gate-summary.json)
    if [ "$result" = "FAIL" ]; then
      echo "Gate result: FAIL"
      exit 1
    fi
    echo "Gate result: $result"
```

The gate step uses default flag values (`--measurement-summary artifacts/sli-summary.json`, `--policy .slint/policy.yaml`) unless overridden.

---

## Baseline Management

A baseline stores a previous gate's metric results for regression comparison.

**Update the baseline**

```sh
make baseline-update-prepare BASELINE_SUMMARY=/path/to/sli-summary.json
```

**Approval requirement:** Baseline changes must go into a dedicated approval PR, not a regular feature PR. This prevents a degraded run from silently resetting the regression anchor.

**First run without baseline:** If `--baseline` is not specified, regression detection is skipped and `gate_result` is set to `WARN` (non-blocking). This is the expected behavior for the first run after onboarding.

---

## Local Development and Testing

```sh
# Run linter
./bin/golangci-lint run --timeout=10m --config=.golangci.yml ./...

# Run unit tests
go test ./...

# Check module consistency
go mod tidy
git diff --exit-code

# Build slint-gate CLI
make slint-gate

# Update baseline from a local summary
make baseline-update-prepare BASELINE_SUMMARY=artifacts/sli-summary.json
```

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
