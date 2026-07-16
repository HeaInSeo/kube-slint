# kube-slint

[![Tests](https://github.com/HeaInSeo/kube-slint/actions/workflows/test.yml/badge.svg)](https://github.com/HeaInSeo/kube-slint/actions/workflows/test.yml)
[![Lint](https://github.com/HeaInSeo/kube-slint/actions/workflows/lint.yml/badge.svg)](https://github.com/HeaInSeo/kube-slint/actions/workflows/lint.yml)
[![Semgrep](https://github.com/HeaInSeo/kube-slint/actions/workflows/semgrep.yml/badge.svg)](https://github.com/HeaInSeo/kube-slint/actions/workflows/semgrep.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/HeaInSeo/kube-slint.svg)](https://pkg.go.dev/github.com/HeaInSeo/kube-slint)
[![Go Version](https://img.shields.io/github/go-mod/go-version/HeaInSeo/kube-slint)](go.mod)
[![Release](https://img.shields.io/github/v/release/HeaInSeo/kube-slint)](https://github.com/HeaInSeo/kube-slint/releases)

한국어 문서는 [README(Kor).md](README(Kor).md)를 참조하세요.

**kube-slint does not replace your tests. It measures what happens during them.**

Attach kube-slint to your existing Kubernetes operator E2E session. Its default fetcher reads `/metrics` before and after your workload, but the measurement boundary is source-neutral: point sources (`MetricsFetcher`), snapshot sources (`SnapshotFetcher`), and range/window sources (`WindowFetcher`) can feed the same SLI computation and policy gate when they produce keyed numeric samples. kube-slint computes operational SLI deltas (reconcile rate, workqueue depth, REST errors) and evaluates them against a declarative policy — without modifying your operator code.

**Try it now** (requires kind ≥ v0.22, Docker, and Go 1.22+):

```bash
cd examples/kind-hello-operator
make demo
```

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
import "github.com/HeaInSeo/kube-slint/pkg/slint"

sess := slint.NewSession(slint.SessionConfig{
    Namespace:             "my-operator-system",
    MetricsServiceName:    "my-operator-controller-manager-metrics-service",
    ServiceAccountName:    "kube-slint-scraper",
    ArtifactsDir:          "artifacts",
    Specs:                 slint.DefaultSpecs(),
    CurlImage:             "my-registry/curlimages/curl:8.11.0",
})
sess.Start()
// ... run your E2E scenario ...
sess.End(ctx)
```

**Step 3: Gate the result**

```sh
make slint-gate   # builds bin/slint-gate
./bin/slint-gate --summary artifacts/sli-summary.json \
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

`slint.DefaultSpecs()` (previously `DefaultV3Specs`, `BaselineV3Specs`) returns a preset spec set designed for kubebuilder-generated operators. It covers:

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
specs := slint.DefaultSpecs()
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

Use the source model that matches the data shape:

- point source: implement `fetch.MetricsFetcher` when each fetch returns one
  keyed numeric sample.
- snapshot source: implement `fetch.SnapshotFetcher` when the source should
  cache the start sample at `Session.Start()` before the workload runs.
- range/window source: implement `fetch.WindowFetcher` when the SLI needs many
  samples across the test window.

For non-Prometheus point or snapshot sources, return the same input keys used
by your `SLISpec.Inputs`. Prometheus helpers such as `spec.PromMetric` and
`spec.UnsafePromKey` are conveniences for Prometheus text exposition, not a
requirement of the SLI engine.

For HTTP JSON or Go expvar endpoints, use the built-in JSON endpoint fetcher:

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/fetch/jsonendpoint"

fetcher := jsonendpoint.New("http://127.0.0.1:8080/debug/vars")
sess := slint.NewSession(slint.SessionConfig{
    Specs:   mySpecs, // Inputs use flattened JSON keys, e.g. "memstats.Alloc"
    Fetcher: fetcher,
})
```

For range/window sources, pass a `fetch.WindowFetcher` through
`SessionConfig.WindowFetcher`. For example, Prometheus `query_range`:

```go
import "github.com/HeaInSeo/kube-slint/pkg/slo/fetch/promrange"

windowFetcher := promrange.New("http://prometheus:9090", `rate(http_requests_total[5m])`, 30*time.Second)
sess := slint.NewSession(slint.SessionConfig{
    Specs:         windowSpecs, // e.g. ComputeWindowP95 or ComputeWindowRatio
    WindowFetcher: windowFetcher,
})
```

**Source selection guide**

| Situation | Source type | Go package |
|---|---|---|
| Custom direct scrape or in-process numeric sample | point source | implement `fetch.MetricsFetcher` |
| Kubernetes Service exposes `/metrics` and CI can create a temporary pod | snapshot source | default `pkg/slint` curlpod path |
| Prefer local access through `kubectl port-forward` | snapshot source | `pkg/slo/fetch/portforward` |
| Go expvar or custom status JSON endpoint | snapshot source | `pkg/slo/fetch/jsonendpoint` |
| Prometheus range query for window p95/ratio checks | range/window source | `pkg/slo/fetch/promrange` |

---

### 2. Embed Harness in E2E Tests

```go
import "github.com/HeaInSeo/kube-slint/pkg/slint"

sess := slint.NewSession(slint.SessionConfig{
    // Target namespace where your operator runs
    Namespace: "my-operator-system",

    // Name of the Kubernetes Service exposing /metrics
    MetricsServiceName: "my-operator-controller-manager-metrics-service",

    // ServiceAccount used by the temporary curl pod.
    // The pod reads its own mounted token; the bearer token is not embedded in kubectl args.
    ServiceAccountName: "kube-slint-scraper",

    // Directory where sli-summary.json will be written
    ArtifactsDir: "artifacts",

    // SLI spec set — use preset or custom
    Specs: slint.DefaultSpecs(),

    // Optional real-cluster settings
    CurlImage: "my-registry/curlimages/curl:8.11.0",
})

sess.Start()
// ... run your E2E scenario here ...
sess.End(ctx)
```

**RBAC note:** The harness creates a temporary curl pod to scrape the metrics endpoint. Run `slint-gate init --emit-rbac rbac.yaml` to scaffold the required ServiceAccount, Role, and RoleBinding in the target namespace.

`slint-gate init --profile kubebuilder-operator` is a backward-compatible extension of `init` (the only supported profile today); omitting `--profile` keeps `init`'s existing output unchanged.

**Output:** `sess.End(ctx)` writes two files:
- `artifacts/sli-summary.<runID>.<testcase>.json` — unique audit file
- `artifacts/sli-summary.json` — latest alias; default input for slint-gate

---

### 3. Token and Metrics URL Handling

The default curl pod path reads its own mounted ServiceAccount token inside the
pod. The bearer token is not embedded in kubectl arguments.

For plain HTTP metrics endpoints in development clusters:

```go
sess := slint.NewSession(slint.SessionConfig{
    // ...
    ServiceURLFormat: slint.ServiceURLHTTP, // "http://%s.%s.svc:8080/metrics"
})
```

The default URL format is `slint.ServiceURLHTTPS`
(`"https://%s.%s.svc:8443/metrics"`).

By default, `ServiceURLFormat` must resolve to a cluster-local Service
address (`<service>.<namespace>.svc` or `.svc.cluster.local`, `http`/`https`
only) — anything else is rejected before a curl pod is created, so a
misconfigured or malicious external URL can never receive the scrape's
Authorization token. Set `DangerouslyAllowExternalMetricsURL: true` to
explicitly opt out.

`TLSInsecureSkipVerify` is deprecated in favor of `DangerouslySkipTLSVerify`
(same effect, visibly named) — available for compatibility with self-signed
development clusters, but it weakens TLS verification. Do not enable it in
shared or production-like CI unless you have explicitly accepted that risk.
It is `false` by default.

---

### 4. Gate the Result (slint-gate CLI)

The gate CLI is a Go binary located in `cmd/slint-gate`.

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
| `--summary` | `artifacts/sli-summary.json` | Path to the SLI summary produced by the harness. (`--measurement-summary` still works as a deprecated alias.) |
| `--policy` | `.slint/policy.yaml` | Path to the policy file |
| `--baseline` | `""` (disabled) | Path to a baseline summary for regression comparison; omit to skip |
| `--output` | `slint-gate-summary.json` | Path to write the gate result JSON |
| `--github-step-summary` | false | Write markdown to `$GITHUB_STEP_SUMMARY` for GitHub Actions |
| `--exit-on` | `NEVER` | Exit 1 when gate result meets this level: `NEVER` \| `FAIL` \| `FAIL_OR_WARN` \| `FAIL_OR_NOGRADE` \| `FAIL_WARN_OR_NOGRADE`. (`--fail-on` still works as a deprecated alias — see below.) |

**Exit behavior:** The binary exits 0 by default (`--exit-on NEVER`). Pass `--exit-on FAIL` (or stricter) to exit 1 on a policy violation. Unknown `--exit-on` values are rejected. The GitHub Actions wrapper handles this automatically via its `exit-on` input.

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
coverage:
  required: false
  informational:
    - "reconcile_success_delta"
promote_to_fail:
  - "threshold_miss"
  - "regression_detected"
  # Optional: fail CI on measured-but-not-gated SLIs.
  # - "coverage_gap"
```

**Gate result values**

| Result | Meaning |
|---|---|
| `PASS` | All threshold and regression checks passed |
| `WARN` | A check failed but its category is not listed in `promote_to_fail`, or a non-blocking condition (first run without baseline, reliability below minimum) |
| `FAIL` | Policy violation listed in `promote_to_fail` — threshold miss or regression detected |
| `NO_GRADE` | Evaluation not possible — missing or corrupt inputs |

**`promote_to_fail` semantics**

`policy.promote_to_fail` controls which violation categories promote `gate_result` to `FAIL`. Violations not listed become `WARN` — a failed check can never produce `PASS`. If `promote_to_fail` is omitted or empty, kube-slint applies the default: `threshold_miss` and `regression_detected`. (`fail_on` still works as a deprecated alias; both are unioned, and using `fail_on` adds a non-fatal notice to `slint-gate-summary.json`'s `policy_warnings`.)

`--exit-on` (CLI flag / action input) is a separate layer that controls whether a given `gate_result` exits the process with code 1. The two settings are independent.

**`checks[].observed` field**

`observed` is normally a number. In non-quantifiable edge cases — such as a regression check where the baseline value is zero and the current value is non-zero — `observed` contains a string marker (e.g. `"baseline_zero_current_nonzero"`). Consumers (jq scripts, dashboards) should not assume `observed` is always a number.

---

### 5. Deploy Observability Stack (Kustomize)

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
| `InsideAnnotation` | **Reserved, not yet implemented** — currently behaves identically to `InsideSnapshot` (only the recorded run-mode label differs); do not rely on it for annotation-boundary precision yet |
| `OutsideSnapshot` | External scrape; metrics are collected from an external source rather than inside the session |

---

## Gate Model

Both gate model components are complete.

**Threshold checking** (DONE): Each metric result in `sli-summary.json` is evaluated against the threshold rules in `policy.yaml`. A threshold miss sets `gate_result` to `FAIL` if `threshold_miss` is in `promote_to_fail`; otherwise it sets `gate_result` to `WARN`.

**Regression detection** (DONE): When `--baseline` is provided, each metric result is compared to the stored baseline value. If the change exceeds `tolerance_percent`, the result is flagged as a regression. A detected regression sets `gate_result` to `FAIL` if `regression_detected` is in `promote_to_fail`; otherwise it sets `gate_result` to `WARN`. A regression from a zero baseline to a non-zero current value is always treated as a detected regression.

---

## Security Defaults

kube-slint's default measurement path is namespace-scoped:

- the generated RBAC uses ServiceAccount, Role, and RoleBinding;
- ClusterRoleBinding is not required for the default path;
- the curl pod reads its own mounted ServiceAccount token;
- command and error output are redacted for common token/secret shapes;
- `NO_GRADE` is a first-class gate result for insufficient measurement;
- `ServiceURLFormat` is validated before any curl pod is created — external
  hosts, unsupported schemes, and malformed service/namespace values are
  rejected by default (see `DangerouslyAllowExternalMetricsURL` to opt out);
- `kube-system`/`kube-public`/`kube-node-lease` are rejected as measurement
  target namespaces by default (see `DangerouslyAllowKubeSystemNamespace`).

See `docs/security-model.md` for the full default-deny policy and dangerous
option reference.

---

## CI Integration

### GitHub Composite Action (recommended)

Drop the action into any workflow. The current action runs the Go CLI from
source, so the workflow must set up Go first:

```yaml
jobs:
  e2e:
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      # ... your E2E steps that produce artifacts/sli-summary.json ...

      - name: slint-gate
        uses: HeaInSeo/kube-slint/.github/actions/slint-gate@main
        with:
          summary:              artifacts/sli-summary.json   # default
          policy:               .slint/policy.yaml            # default
          exit-on:              FAIL_OR_NOGRADE               # NEVER | FAIL | FAIL_OR_WARN | FAIL_OR_NOGRADE | FAIL_WARN_OR_NOGRADE
```

**Inputs**

| Input | Default | Description |
|---|---|---|
| `summary` | `` (falls back to `measurement-summary`) | Path to sli-summary.json. Preferred over `measurement-summary` (deprecated, still works — default `artifacts/sli-summary.json`). |
| `measurement-summary` | `artifacts/sli-summary.json` | Deprecated: use `summary` instead. |
| `policy` | `.slint/policy.yaml` | Path to policy YAML |
| `baseline` | `` | Optional baseline summary path |
| `output` | `slint-gate-summary.json` | Output path for gate result |
| `exit-on` | `` (falls back to `fail-on`) | `NEVER`\|`FAIL`\|`FAIL_OR_WARN`\|`FAIL_OR_NOGRADE`\|`FAIL_WARN_OR_NOGRADE`. Preferred over `fail-on` (deprecated, still works — default `FAIL_OR_NOGRADE`). |
| `github-step-summary` | `true` | Append Markdown table to step summary |
| `upload-artifact` | `true` | Upload gate result as artifact |

**Outputs**: `gate-result`, `evaluation-status`, `summary-path`

For long-lived CI, pin the action to a tag or SHA when available instead of
tracking `main`.

### Manual invocation

```yaml
- name: Evaluate slint gate
  run: |
    go run ./cmd/slint-gate \
      --summary artifacts/sli-summary.json \
      --policy .slint/policy.yaml \
      --github-step-summary

- name: Check gate result
  run: |
    result=$(jq -r '.gate_result' slint-gate-summary.json)
    [ "$result" != "FAIL" ] || exit 1
```

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

## Project Docs

- Product and quality roadmap: `docs/quality-roadmap.md`
- Implementation handoff: `docs/quality-roadmap-implementation-handoff.md`
- Security model: `docs/security-model.md`
- Gate contract: `docs/gate-contract.md`
- Test strategy: `docs/test-strategy.md`
- Release and DevEx plan: `docs/release-devex-plan.md`
- Onboarding CLI reference (`init`, `inspect`, `recommend-policy`, `baseline approve/diff/merge`, `ci github-actions`, `quickstart`, `wizard`): `docs/sli-gate-onboarding-ux.md`

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
