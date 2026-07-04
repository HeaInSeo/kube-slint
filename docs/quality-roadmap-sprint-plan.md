# kube-slint Quality Roadmap Sprint Plan

Date: 2026-07-04

## Purpose

This sprint plan turns the 8 -> 9 -> 10 quality roadmap into a focused
non-development workstream.

The goal is not to add product features. The goal is to define the security,
schema, test, release, and documentation contracts that let the implementation
agent build against clear acceptance criteria.

## Confirmed Repo Facts

- `docs/DECISIONS.md` is the highest-priority product contract source.
- `docs/project-status.yaml` is the only machine-readable automation status
  source.
- kube-slint is a shift-left operational SLI guardrail, not a generic
  Kubernetes YAML linter and not a correctness test framework replacement.
- `slint-gate` is a separate policy evaluation layer over measurement outputs.
- Measurement failure is not the same thing as test failure; policy violation
  may fail CI.
- Post-RC hardening already prioritizes secret containment, namespace-scoped
  RBAC, conservative gate semantics, and invalid enum rejection.

## Sprint Scope

This sprint is for a separate quality/docs agent while the main development
agent continues implementation work.

In scope:

- Security threat model and default guardrail policy.
- ServiceURLFormat and ServiceAccount token handling policy.
- Summary and policy contract specification.
- Bad fixture and E2E test matrix planning.
- Semgrep/custom rule planning.
- Release, GitHub Action, documentation IA, and UX backlog planning.
- Implementation tickets for the development agent.

Out of scope:

- Runtime behavior changes.
- Harness API changes.
- Gate output schema changes.
- Controller/operator runtime resurrection.
- Sibling repository edits.
- Treating `test/e2e` as real-cluster operator deployment E2E without code and
  documentation evidence.

## Product Wording Guardrail

Use this framing:

> kube-slint is an operator-first, dataplane-aware shift-left operational SLI
> guardrail for Kubernetes workloads under test.

Avoid wording that makes kube-slint sound like:

- A generic Kubernetes linter.
- A Prometheus replacement.
- A functional test framework replacement.
- A standalone operator repository.
- An MCP-first AI tool.
- A dataplane-only product before the accepted decision log is updated.

## Sprint 0: Alignment and Work Queue Setup

Target: 2026-07-04 to 2026-07-05

Goal: Confirm the planning boundary and create a handoff-ready work queue.

Tasks:

- Read `docs/DECISIONS.md`, `docs/project-status.yaml`,
  `docs/CODEX_OPERATING_RULES.md`, `docs/PROGRESS_LOG.md`, and `README.md`.
- Confirm whether each roadmap item is documentation-only, quasi-development,
  or implementation work.
- Create the ticket template used for all developer handoffs.
- Mark 9-point and 10-point items as backlog unless they unblock 8-point
  security or schema hardening.

Deliverables:

- This sprint plan.
- A prioritized ticket backlog skeleton.
- A list of explicit assumptions and deferred decisions.

Definition of done:

- The quality/docs agent can start without changing runtime behavior.
- The development agent can consume tickets without reinterpreting product
  identity.

## Sprint 1: Security and Guardrail Specification

Target: 2026-07-06 to 2026-07-09

Goal: Make dangerous defaults and unsafe configuration paths explicit before
they become implementation details.

Tasks:

- Define kube-slint security defaults.
- Define forbidden default patterns:
  - external metrics URL
  - Authorization header forwarding to non-cluster-local hosts
  - `InsecureSkipVerify`
  - `ClusterRoleBinding`
  - privileged curl pod
  - `hostPath`
  - default execution against `kube-system`
  - cleanup without kube-slint ownership labels
- Define allowed exception conditions.
- Define dangerous option naming rules.
- Define ServiceURLFormat validation policy.
- Define ServiceAccount token handling policy.
- Define namespace-scoped RBAC default policy.
- Draft custom guardrail/Semgrep rule plan.

Deliverables:

- `docs/security-defaults.md`
- `docs/dangerous-options.md`
- `docs/security/service-url-format.md`
- `docs/security/token-handling.md`
- `docs/security/rbac-model.md`
- `docs/guardrails/security-patterns.md`
- `.semgrep/rules-plan.md`
- Security implementation tickets.

Definition of done:

- External URL token exfiltration is covered by an explicit default-deny policy.
- Dangerous options require obvious opt-in names such as
  `dangerouslyAllowExternalMetricsURL`, `dangerouslySkipTLSVerify`, and
  `dangerouslyAllowClusterWideRBAC`.
- Namespace-scoped RBAC is documented as the default model.
- Cleanup safety requires kube-slint ownership metadata.

## Sprint 2: Summary, Policy, and Bad Fixture Contracts

Target: 2026-07-10 to 2026-07-13

Goal: Prevent invalid summary or policy input from silently becoming PASS.

Tasks:

- Specify summary schema validation requirements.
- Specify policy schema validation requirements.
- Specify gate result semantics and priority.
- Decide and document first-run baseline behavior.
- Decide and document malformed metric value behavior, including NaN/Inf.
- Decide and document unknown field handling.
- Build the bad fixture matrix for summary, policy, and security inputs.
- Convert fixture rows into implementation tickets.

Recommended gate result priority:

```text
FAIL > NO_GRADE > WARN > FIRST_RUN_WARNING > PASS
```

Deliverables:

- `docs/spec/summary-schema.md`
- `docs/spec/policy-schema.md`
- `docs/spec/gate-result-semantics.md`
- `docs/test-matrix/bad-fixtures.md`
- `testdata-plan/bad-fixtures/README.md`
- Summary/policy/security fixture tickets.

Definition of done:

- Missing or unsupported schema versions are reject cases.
- Duplicate result IDs and duplicate threshold names are reject cases.
- Unknown result status and unknown policy operators are reject cases.
- Baseline-required-but-missing behavior is explicitly classified.
- Every bad fixture has an expected result and implementation owner.

## Sprint 3: E2E, Release, Docs IA, and UX Backlog

Target: 2026-07-14 to 2026-07-17

Goal: Prepare 9-point readiness work without competing with the active
implementation stream.

Tasks:

- Write kind E2E scenario matrix.
- Write E2E acceptance criteria.
- Draft release engineering policy.
- Draft GitHub Action stabilization direction.
- Draft documentation information architecture.
- Draft UX failure catalog.
- Keep MCP read-only boundary as backlog unless needed for security framing.

Deliverables:

- `docs/test-matrix/kind-e2e.md`
- `docs/test-matrix/e2e-acceptance.md`
- `docs/release/release-policy.md`
- `docs/integrations/github-action.md`
- `docs/README-structure.md`
- `docs/ux/failure-catalog.md`
- 9-point and 10-point backlog tickets.

Definition of done:

- kind E2E includes happy path, negative path, first-run, regression,
  malformed input, RBAC, cleanup, and parallel artifact scenarios.
- GitHub Action direction prefers release binaries over `go run`.
- Documentation IA keeps kube-slint positioned as a runtime SLI guardrail.
- UX failure messages separate measurement failure, config error, security
  reject, policy violation, and app/test failure.

## Review and Freeze

Target: 2026-07-18

Goal: Freeze the quality roadmap outputs into developer-ready tickets.

Tasks:

- Check all new documents against `docs/DECISIONS.md`.
- Check whether any document implies behavior that is not implemented.
- Mark such behavior as target policy, proposed contract, or TODO.
- Ensure `docs/project-status.yaml` remains the only machine-readable status
  input.
- Produce a final handoff list for the development agent.

Deliverables:

- Final sprint summary.
- Developer ticket list.
- Open decisions list.
- Deferred 9-point and 10-point backlog.

Definition of done:

- No document claims runtime behavior changed unless the implementation agent
  has actually changed it.
- Every implementation request is expressed as a ticket with acceptance
  criteria and test cases.
- Open policy decisions are named explicitly instead of being hidden in prose.

## Developer Ticket Template

Use this format for every implementation handoff:

```text
Ticket:
  Title:
  Background:
  Required behavior:
  Rejected behavior:
  Acceptance criteria:
  Test cases:
  Docs to update:
  Security impact:
```

## Prioritized Backlog

Priority 0:

- ServiceURLFormat default-deny policy for external hosts.
- ServiceAccount token handling and Authorization header containment.
- Namespace-scoped RBAC default contract.
- Invalid summary/policy reject contract.
- NO_GRADE/WARN/FAIL priority and first-run semantics.

Priority 1:

- Bad fixture matrix and fixture generation tickets.
- Semgrep/custom guardrail rule draft.
- kind E2E acceptance matrix.
- UX failure catalog.

Priority 2:

- Release binary and GitHub Action policy.
- Documentation IA and quickstart rewrite plan.
- Supply chain security policy.
- Supported platform matrix.

Priority 3:

- Schema compatibility and migration policy.
- Dogfooding plan.
- kube-slint observability/logging policy.
- MCP read-only boundary.
- External contributor guide.

## Open Decisions

- Whether `.svc` only or both `.svc` and `.svc.cluster.local` are allowed by
  default for ServiceURLFormat.
- Whether plain HTTP is allowed for cluster-local metrics in default mode.
- Whether external metrics URLs are allowed only with Authorization header
  removal, or fully rejected unless a dangerous option is set.
- Whether NaN/Inf metric values are always invalid or classified as
  measurement failure.
- Whether baseline-required-but-missing produces `NO_GRADE` or `FAIL`.
- Which unknown fields may be ignored for append-compatible schema evolution.
- Whether README examples should replace `TLSInsecureSkipVerify: true` with a
  named dangerous option once implementation supports it.

## Status Reporting Format

Quality/docs agent reports should start with repo evidence:

```text
Confirmed facts:
- ...

Current identity:
- ...

Target state:
- ...

Open risks:
- ...

Behavior changed:
- No runtime behavior changed.
```
