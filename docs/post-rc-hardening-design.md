# Post-RC Hardening Design

Date: 2026-07-02

## Confirmed Facts

- `docs/DECISIONS.md` defines kube-slint as a shift-left operational SLI guardrail, not an operator correctness framework.
- `docs/DECISIONS.md` D-002 says measurement failure is not automatically test failure.
- `docs/DECISIONS.md` D-008 says `slint-gate` is a separate policy evaluation layer over measurement outputs.
- `docs/project-status.yaml` lists post-RC hardening as the next milestone.
- `pkg/slo/fetch/curlpod/client.go` currently embeds the bearer token in the curl pod shell command.
- `pkg/kubeutil/runner.go` logs the full command line before execution.
- The external review found that `cmd/slint-gate/init.go` emitted `ClusterRole` and `ClusterRoleBinding`.
- The external review found that `internal/gate/gate.go` allowed unknown `fail_on` policy values and let `WARN` outrank `NO_GRADE`.

## Current Identity

kube-slint remains a library/harness/gate toolchain for applying operational SLI guardrails during Kubernetes Operator development. It measures what happens during an existing test session and evaluates the resulting artifacts through a separate gate.

This hardening work does not turn kube-slint into a standalone operator, a generic correctness test framework, or an always-fail-on-measurement-error test runner.

## Intended Target State

1. Secret-bearing values are not embedded in Kubernetes Pod command args or command logs.
2. Generated RBAC defaults are namespace-scoped unless a future task explicitly introduces cluster-wide behavior.
3. Port-forward lifecycle is tied to the measurement session, not to a short scrape timeout context.
4. Start snapshot collection failures are represented in machine-readable reliability/gate output.
5. `NO_GRADE` outranks `WARN` for aggregate gate results, except for explicitly documented first-run baseline behavior.
6. CLI and policy enums reject unknown values instead of silently falling back.
7. Run IDs used in Kubernetes label selectors are generated and sanitized for label-value safety.
8. Regression checks use metric direction before treating a baseline difference as a regression.

## Non-Goals

- No controller/operator runtime resurrection.
- No harness API rewrite unless required to close the lifecycle bug.
- No gate output schema replacement.
- No real-cluster E2E reinterpretation of `test/e2e`.
- No feature expansion beyond security, lifecycle, and gate correctness hardening.

## Design Decisions

### Secret Handling

The curl pod should read its bearer token from the mounted ServiceAccount token file inside the pod:

```sh
TOKEN="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
curl -H "Authorization: Bearer ${TOKEN}" ...
```

`SessionConfig.Token` remains available for compatibility until a dedicated deprecation decision exists, but the default curlpod path should prefer the in-pod token when `ServiceAccountName` is configured.

Command execution logs must pass through redaction before logging or returning command-bearing errors. Existing `pkg/slo/evidence.RedactString` is the preferred helper.

### RBAC Scope

`slint-gate init --emit-rbac` should emit:

- `ServiceAccount`
- namespaced `Role`
- namespaced `RoleBinding`

Default rules should avoid cluster-wide binding and remove unused verbs where current code evidence does not require them.

### Measurement Failure Semantics

Start snapshot failure should not be hidden as a stderr-only warning. It should be visible in summary reliability data and should become `NO_GRADE` in gate evaluation when it prevents a trustworthy policy decision.

This preserves D-002 because the E2E test can continue, while the gate can still fail CI when the caller uses `FAIL_OR_NOGRADE` or `FAIL_WARN_OR_NOGRADE`.

### Gate Severity

Aggregate severity should be:

```text
FAIL > NO_GRADE > WARN > PASS
```

First-run baseline absence may remain `WARN` when regression is optional/first-run friendly, but corrupt or unavailable required inputs should remain `NO_GRADE`.

### Validation

The following values should be validated:

- CLI `--fail-on`
- action `fail-on`
- policy `schema_version`
- policy `fail_on`
- policy reliability levels
- future regression direction values

Unknown values should produce `policy_status=invalid`, `gate_result=NO_GRADE`, or a CLI usage error depending on where they are found.

## Open Risks

- Moving token sourcing to in-pod ServiceAccount token changes behavior for users who rely on an externally supplied token that differs from the scraper ServiceAccount token.
- Role/RoleBinding defaults can break users who expected a single generated ClusterRoleBinding to work across namespaces.
- Direction-aware regression requires a policy schema extension; the first patch should validate existing behavior before changing the schema.
- Start snapshot failure propagation may require a small internal interface extension so `Session.Start()` can pass collection status into `Session.End()`.

## Verification Plan

- Unit tests for command redaction and curlpod command construction.
- Unit tests for RBAC template output.
- Unit tests for `fail-on` validation and `NO_GRADE` severity priority.
- `go test ./pkg/kubeutil ./pkg/slo/fetch/curlpod ./cmd/slint-gate ./internal/gate`.
- If toolchain permits, `go test ./...`.
