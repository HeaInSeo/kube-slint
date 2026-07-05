# SLI Gate Onboarding UX

Date: 2026-07-05
Status: Initial technical design note

## Purpose

This document defines the UX problem and target design for helping a new
kube-slint user choose SLIs, generate an initial policy, establish a baseline,
and wire the result into CI.

kube-slint's product center is not generic Kubernetes linting. It is a CI
guardrail for Kubernetes operational SLI regressions observed during tests.

The user-facing onboarding question is:

```text
How does a new project go from "I have an E2E test and /metrics" to
"I have a trustworthy SLI regression gate in CI" without learning every
internal kube-slint concept first?
```

## Current UX Rating

Estimated current score for new-project SLI/gate onboarding:

```text
6.5 / 10
```

The core model is solid, but self-service onboarding still requires too much
manual judgment.

Strong points:

- `pkg/slint` exposes a consumer-facing `Session` API.
- `slint.DefaultSpecs()` gives kubebuilder/operator users a usable starting
  SLI set.
- `.slint/policy.yaml` is the preferred policy path.
- `slint-gate` has a clear `PASS | WARN | FAIL | NO_GRADE` model.
- GitHub Action and kind demo paths exist.
- Namespace-scoped RBAC and token handling are clearer after post-RC hardening.

Friction points:

- Users must decide which SLIs matter for their project.
- Users must hand-tune thresholds before they understand normal metric ranges.
- Baseline creation and approval flow is conceptually heavy.
- `policy.fail_on` and CLI/action `--fail-on` are two separate layers.
- `NO_GRADE` is correct but initially unfamiliar.
- Kubernetes details such as metrics Service, ServiceAccount, RBAC,
  ServiceURLFormat, and TLS settings appear early.
- kube-slint does not yet provide a guided "inspect -> recommend -> baseline
  -> CI" loop.

## Target UX Score

Target onboarding score:

```text
9 / 10
```

10/10 requires long-term maturity such as release binaries, dogfooding,
schema migration policy, full negative E2E coverage, and supply-chain
attestation. The onboarding UX itself should aim for 9/10 first.

## Target User Journey

The ideal first-time workflow:

```text
1. User runs existing E2E test with kube-slint attached.
2. kube-slint writes artifacts/sli-summary.json.
3. User runs an inspect command.
4. kube-slint explains which SLIs were measured and which were missing.
5. User runs a recommend command.
6. kube-slint generates a conservative policy draft.
7. User approves the first healthy run as a baseline.
8. kube-slint prints CI YAML.
9. CI blocks SLI regression or untrustworthy measurement.
```

The user should not need to know the full policy schema before seeing a useful
first gate.

## Proposed CLI Flow

### 1. Initialize

```sh
slint-gate init --profile kubebuilder-operator
```

Expected output:

- `.slint/policy.yaml` draft;
- namespace-scoped RBAC manifest, if requested;
- code snippet using `pkg/slint`;
- next command suggestions.

UX goal:

```text
Give the user a safe starting point, not a blank policy file.
```

### 2. Inspect Summary

```sh
slint-gate inspect --summary artifacts/sli-summary.json
```

Expected output:

```text
Measured SLIs:
  reconcile_total_delta: 14
  reconcile_error_delta: 0
  workqueue_depth_end: 0
  rest_client_429_delta: 0
  rest_client_5xx_delta: 0

Missing or skipped:
  workqueue_retries_total_delta: missing metric

Gate readiness:
  Ready for threshold policy: yes
  Ready for regression baseline: yes
  Measurement confidence: complete
```

UX goal:

```text
Explain what kube-slint saw before asking the user to write policy.
```

### 3. Recommend Policy

```sh
slint-gate recommend-policy \
  --summary artifacts/sli-summary.json \
  --profile kubebuilder-operator \
  --output .slint/policy.yaml
```

Expected behavior:

- generate conservative thresholds for measured SLIs;
- mark missing SLIs as comments, not active rules;
- explain why each rule exists;
- default to `threshold_miss` and `regression_detected` in `fail_on`;
- keep reliability optional unless the user requests strict promotion mode.

Example output:

```yaml
schema_version: "slint.policy.v1"

thresholds:
  - name: "reconcile_error_delta_zero"
    metric: "reconcile_error_delta"
    operator: "=="
    value: 0
  - name: "workqueue_depth_end_max"
    metric: "workqueue_depth_end"
    operator: "<="
    value: 0

regression:
  enabled: true
  tolerance_percent: 10

reliability:
  required: false
  min_level: "partial"

fail_on:
  - "threshold_miss"
  - "regression_detected"
```

UX goal:

```text
Let users edit a reasonable draft instead of inventing a policy from scratch.
```

### 4. Approve Baseline

```sh
slint-gate baseline approve \
  --summary artifacts/sli-summary.json \
  --output docs/baselines/my-service-sli-summary.json
```

Expected behavior:

- validate summary schema;
- reject `NO_GRADE` or incomplete measurement as a baseline unless forced;
- normalize non-deterministic metadata where appropriate;
- print a review summary.

UX goal:

```text
Make baseline creation explicit, reviewable, and hard to do accidentally.
```

### 5. Generate CI Snippet

```sh
slint-gate ci github-actions \
  --summary artifacts/sli-summary.json \
  --policy .slint/policy.yaml \
  --baseline docs/baselines/my-service-sli-summary.json
```

Expected output:

```yaml
- name: Run kube-slint gate
  uses: HeaInSeo/kube-slint/.github/actions/slint-gate@<tag-or-sha>
  with:
    measurement-summary: artifacts/sli-summary.json
    policy: .slint/policy.yaml
    baseline: docs/baselines/my-service-sli-summary.json
    fail-on: FAIL_OR_NOGRADE
```

UX goal:

```text
Turn a successful local gate into CI with minimal translation.
```

## Profiles

Profiles should select default SLI specs and policy recommendations.

Initial profile:

```text
kubebuilder-operator
```

Default SLI candidates:

| SLI | Default recommendation |
|---|---|
| `reconcile_total_delta` | measured, usually threshold `>= 1` for active test scenarios |
| `reconcile_error_delta` | threshold `== 0` |
| `workqueue_depth_end` | threshold `<= 0` or small configured value |
| `workqueue_retries_total_delta` | threshold `== 0` or WARN depending on workload |
| `rest_client_429_delta` | threshold `== 0` |
| `rest_client_5xx_delta` | threshold `== 0` |

Future profiles:

- `dataplane-service`
- `controller-runtime-operator`
- `custom-prometheus`

## UX Concepts To Hide Until Needed

The first-run path should avoid front-loading:

- schema compatibility details;
- every possible `fail-on` mode;
- custom SLI spec authoring;
- ServiceURLFormat override;
- dangerous TLS settings;
- MCP integration;
- supply-chain/release details.

Those topics should remain documented, but the first-time flow should only
surface them when the user's environment requires them.

## Error Message Requirements

Failures should answer four questions:

```text
What happened?
What does it mean for my test/gate?
How do I fix it?
What is the gate result?
```

Example:

```text
kube-slint could not recommend a policy.

Reason:
  The summary has no measured SLI results.

What this means:
  Your E2E test may have passed, but kube-slint did not collect enough
  operational signal to build a gate.

How to fix:
  1. Confirm the metrics Service name.
  2. Confirm the Service exposes /metrics.
  3. Run slint-gate inspect --summary artifacts/sli-summary.json.

Result:
  NO_GRADE
```

## Acceptance Criteria

A new-project onboarding UX is acceptable when:

- a user can generate a starter policy from a valid summary;
- a user can inspect missing SLIs without reading Go structs;
- a user can create a baseline through an explicit approval command;
- CI YAML can be generated from the same paths used locally;
- invalid summary or policy never produces `PASS`;
- `NO_GRADE` is explained as untrustworthy measurement, not app test failure;
- default flow does not require ClusterRoleBinding;
- default flow does not require external metrics URLs;
- dangerous options are not presented as normal quickstart knobs.

## Non-Goals

- Replacing the user's E2E assertions.
- Replacing Prometheus.
- Providing full SLO management.
- Automatically deciding production SLOs.
- Enabling write-capable AI/MCP actions.

## Implementation Handoff

Recommended implementation order:

1. `slint-gate inspect --summary`.
2. `slint-gate recommend-policy --summary --profile`.
3. `slint-gate baseline approve`.
4. `slint-gate ci github-actions`.
5. Docs update for quickstart and troubleshooting.

Each command should be independently useful and testable.

## Open Decisions

- Whether `recommend-policy` should overwrite existing policy files or require
  `--force`.
- Whether recommended thresholds should be strict by default or comment-only
  until the user opts in.
- Whether first-run baseline absence remains `WARN` or becomes configurable
  as `NO_GRADE`.
- Whether profile selection should happen in `.slint.yaml`, `.slint/policy.yaml`,
  or only as CLI input.
- Whether CI snippet generation should target the current local action or only
  future release-binary action usage.
