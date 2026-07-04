# README Structure Plan

Date: 2026-07-04
Status: Draft documentation IA for quality roadmap Sprint 3

## Purpose

The README should help an external user understand kube-slint quickly without
mistaking it for a generic linter, Prometheus replacement, test framework, or
operator repo.

## First-Screen Message

Required message:

```text
kube-slint does not replace your tests.
It measures what happens during them.

It helps catch Kubernetes operational regressions before they reach production.
```

## Recommended README Flow

1. Product one-liner.
2. What kube-slint is not.
3. How it works diagram.
4. Quickstart.
5. Add to E2E test.
6. Run `slint-gate`.
7. Understand gate results.
8. Security defaults.
9. GitHub Action.
10. Docs index.

## Product One-Liner

Recommended:

```text
kube-slint is an operator-first, dataplane-aware shift-left operational SLI
guardrail for Kubernetes workloads under test.
```

## Comparison Table

| Tool | Difference |
|---|---|
| kube-linter | Static manifest best-practice linting. |
| kube-score | Manifest quality scoring. |
| Prometheus | Runtime time-series collection and alerting. |
| Test framework | Functional correctness assertions. |
| kube-slint | Measures SLI changes during tests and gates operational regression in CI. |

## Docs IA

Recommended structure:

```text
README.md
docs/
  concepts.md
  quickstart.md
  policy.md
  summary-schema.md
  gate-semantics.md
  dataplane-preflight.md
  security.md
  rbac.md
  troubleshooting.md
  integrations/
    github-actions.md
    ginkgo.md
    mcp.md
  examples/
    simple-dataplane.md
    regression-gate.md
    first-run-baseline.md
```

## README Guardrails

The README must not:

- imply kube-slint replaces tests;
- imply measurement failure is app correctness failure;
- promote ClusterRoleBinding as the default;
- show dangerous options without explaining the risk;
- position MCP as core functionality;
- describe `test/e2e` as real-cluster operator deployment E2E unless that is
  true in code and docs.

## Handoff

Before rewriting README, confirm:

- current public API examples;
- current GitHub Action interface;
- current release artifact strategy;
- whether dangerous option names have been implemented.
