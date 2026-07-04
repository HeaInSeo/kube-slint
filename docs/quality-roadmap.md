# Quality Roadmap

Date: 2026-07-04
Status: Complete
Progress: 100%

## Purpose

This document is the canonical planning source for the 8 -> 9 -> 10 quality
roadmap. It consolidates the previous sprint plan, sprint summary, and ticket
backlog into one small surface.

The roadmap is a non-runtime planning and guardrail workstream. Runtime
behavior changes belong in implementation tasks.

## Confirmed Facts

- `docs/DECISIONS.md` is the highest-priority product contract source.
- `docs/project-status.yaml` is the only machine-readable automation status
  source.
- kube-slint is a shift-left operational SLI guardrail, not a generic
  Kubernetes YAML linter and not a correctness test framework replacement.
- `slint-gate` is a separate policy evaluation layer over measurement outputs.
- Measurement failure is not correctness test failure; policy violation may
  fail CI.
- D-015 accepts this quality roadmap as CI-guarded planning input.

## Product Wording Guardrail

Use this framing:

```text
kube-slint is an operator-first, dataplane-aware shift-left operational SLI
guardrail for Kubernetes workloads under test.
```

Avoid wording that makes kube-slint sound like:

- a generic Kubernetes linter;
- a Prometheus replacement;
- a functional test framework replacement;
- a standalone operator repository;
- an MCP-first AI tool;
- a dataplane-only product before the accepted decision log is updated.

## Completed Sprint Scope

Completed:

- Security defaults and dangerous option naming.
- ServiceURLFormat and token handling policy.
- Namespace-scoped RBAC model.
- Summary, policy, and gate result contracts.
- Bad fixture matrix and kind E2E scenario matrix.
- Release/GitHub Action direction.
- README IA and UX failure catalog.
- CI-backed quality guardrail workflow.
- Frozen developer handoff.

Runtime behavior changed:

```text
No.
```

Operational repository behavior changed:

```text
Yes. A Quality Guardrails GitHub Actions workflow was added.
```

## CI Guardrail

Workflow:

```text
.github/workflows/quality-guardrails.yml
```

Local command:

```sh
bash hack/quality-guardrails.sh
```

The guardrail checks:

- source-of-truth files exist and retain key accepted decisions;
- `roadmap-status` reads `docs/project-status.yaml`;
- `SECURITY.md` reflects current in-pod ServiceAccount token handling;
- default RBAC does not reintroduce ClusterRoleBinding;
- redaction still covers Bearer and common secret shapes;
- curlpod securityContext remains non-privileged;
- GitHub Action and workflow defaults keep `FAIL_OR_NOGRADE`;
- invalid input must not produce `PASS`;
- ServiceURLFormat external-host rejection remains a Priority 0 policy.

## Developer Handoff

Implementation-owned work is frozen in:

```text
docs/quality-roadmap-implementation-handoff.md
```

Priority 0:

1. Implement ServiceURLFormat default-deny validation.
2. Finalize dangerous option compatibility plan.
3. Convert summary bad fixtures into executable tests.
4. Convert policy bad fixtures into executable tests.
5. Add security bad fixtures for external URL, privileged pod, hostPath, and
   unsafe cleanup.
6. Keep quality guardrails owned and active.

## Open Decisions

- Whether both `.svc` and `.svc.cluster.local` are allowed by default.
- Whether plain HTTP is allowed for cluster-local metrics in default mode.
- Whether external URL support is ever allowed with Authorization stripped, or
  fully rejected unless a dangerous option is set.
- Whether NaN/Inf metric values are always invalid or become measurement
  failure.
- Whether baseline-required-but-missing produces `NO_GRADE` or `FAIL`.
- Which unknown summary/policy fields may be ignored for compatibility.
- How legacy insecure names such as `TLSInsecureSkipVerify` migrate to
  dangerous option naming.
- Whether the published action supports both `summary` and
  `measurement-summary` input names.

## Deferred Backlog

9-point backlog:

- full kind E2E suite;
- release binary and checksums;
- GitHub Action binary download path;
- quickstart and troubleshooting docs;
- UX failure catalog implementation.

10-point backlog:

- schema compatibility policy;
- migration policy;
- supply chain SBOM/provenance;
- supported platform matrix;
- dogfooding plan;
- kube-slint observability/logging policy;
- MCP read-only boundary;
- contributor guide;
- breaking change policy.
