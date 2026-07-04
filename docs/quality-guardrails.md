# kube-slint Quality Guardrails

Date: 2026-07-04

## Purpose

This document describes the non-product-code guardrails that enforce the
quality roadmap while the main implementation agent continues development.

These checks do not implement new kube-slint runtime behavior. They prevent
source-of-truth drift, stale security wording, weakened CI gate settings, and
regressions in already-accepted hardening defaults.

## CI Entry Point

Workflow:

- `.github/workflows/quality-guardrails.yml`

Local command:

```sh
bash hack/quality-guardrails.sh
```

## Guarded Contracts

### Source of truth

- `docs/DECISIONS.md` remains the highest-priority product contract source.
- `docs/project-status.yaml` remains the only machine-readable status source.
- `roadmap-status` must continue reading `docs/project-status.yaml`.

### Product identity

- kube-slint remains a shift-left operational SLI guardrail.
- kube-slint must not be described as a generic Kubernetes YAML linter,
  Prometheus replacement, or functional test framework replacement.
- User-facing wording must keep the distinction between tests and measurement.

### Canonical Planning Docs

The quality roadmap planning surface is intentionally small:

- `docs/quality-roadmap.md`
- `docs/quality-roadmap-implementation-handoff.md`
- `docs/security-model.md`
- `docs/gate-contract.md`
- `docs/test-strategy.md`
- `docs/release-devex-plan.md`

### Security

- `SECURITY.md` must describe the current in-pod ServiceAccount token path.
- `SECURITY.md` must not reintroduce stale wording that says the token is
  command-line visible in generated pod specs.
- `docs/security-model.md` must keep ServiceURLFormat external-host handling
  as a Priority 0 default-deny policy.
- Dangerous opt-in options must be visibly named as dangerous.

### RBAC

- `slint-gate init --emit-rbac` default scaffolding must remain
  namespace-scoped.
- Default generated RBAC must not reintroduce `ClusterRoleBinding`.
- The unit test guarding this behavior must remain present.

### Secret redaction

- Redaction must cover Bearer token shapes.
- Redaction must cover common secret key names such as `token`, `password`,
  `passwd`, `secret`, `serviceAccountToken`, and `clientSecret`.
- Tests must cover Authorization bearer header redaction.

### Curl pod security context

- The curl pod must explicitly mount its ServiceAccount token.
- The curl container must disable privilege escalation.
- The curl container must drop Linux capabilities.
- The curl container must run as non-root.
- The curl container must use `RuntimeDefault` seccomp.

### Gate behavior

- The GitHub Action default `fail-on` must include `NO_GRADE`.
- The repo's `slint-gate` workflow must use `FAIL_OR_NOGRADE`.
- The quality roadmap must preserve conservative gate priority:

```text
FAIL > NO_GRADE > WARN > FIRST_RUN_WARNING > PASS
```

## Non-Goals

- This guardrail workflow does not replace Go tests.
- This guardrail workflow does not implement ServiceURLFormat validation.
- This guardrail workflow does not enforce future docs that have not been
  accepted or implemented.
- This guardrail workflow checks Sprint 3 readiness documents as planning
  artifacts, but does not implement release binaries, new action behavior, or
  kind E2E jobs.
- This guardrail workflow does not treat measurement failure as correctness
  test failure.

## How To Extend

Add a check only when one of these is true:

- The behavior is already implemented and accepted.
- The repo source of truth explicitly requires the contract.
- The check prevents stale documentation from contradicting current behavior.

For proposed future behavior, add a task in
`docs/quality-roadmap-implementation-handoff.md` first. Do not fail CI on a
future contract until implementation and documentation are aligned.
