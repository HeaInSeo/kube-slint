# Security Pattern Guardrails

Date: 2026-07-04
Status: Proposed contract for quality roadmap Sprint 1

## Purpose

This document lists security-sensitive patterns that kube-slint should catch
through tests, static checks, CI guardrails, or review checklists.

These are not generic Kubernetes best-practice lint rules. They are
kube-slint-specific guardrails for the measurement, artifact, and gate path.

## Pattern Matrix

| Pattern | Default action | Preferred guardrail |
|---|---|---|
| Direct `fmt.Sprintf(ServiceURLFormat, ...)` without validation | Reject | Unit test + Semgrep/custom rule |
| Bearer token in curl args or PodSpec command | Reject | Unit test + Semgrep/custom rule |
| `InsecureSkipVerify: true` in production/default path | Reject or dangerous opt-in | Unit test + static rule |
| Default `ClusterRoleBinding` generation | Reject | Unit test + CI guardrail |
| Cleanup without kube-slint owner label | Reject | Unit test + review checklist |
| `hostPath` in generated curl pod | Reject | Unit test + CI guardrail |
| privileged curl pod | Reject | Unit test + CI guardrail |
| Unknown gate/policy enum | Reject | Unit test |
| Malformed summary/policy silently PASSes | Reject | Bad fixture matrix |

## Enforcement Levels

Use three enforcement levels:

1. `documented`: policy is documented but not yet implemented.
2. `tested`: behavior is implemented and covered by tests.
3. `ci-guarded`: CI blocks drift in the accepted behavior or documentation.

Do not mark future behavior as `ci-guarded` until it is implemented or the
guardrail checks only documentation consistency.

## Current CI-Guarded Items

The `quality-guardrails` workflow currently checks:

- source-of-truth files exist;
- roadmap-status reads `docs/project-status.yaml`;
- README keeps test-vs-measurement positioning;
- `SECURITY.md` reflects the current in-pod token path;
- generated default RBAC does not reintroduce ClusterRoleBinding;
- redaction still covers Bearer and common secret names;
- curlpod securityContext remains non-privileged;
- GitHub Action defaults continue treating `NO_GRADE` as failure.

## Future Guardrail Candidates

- ServiceURLFormat validator fixture test.
- Summary invalid fixture suite.
- Policy invalid fixture suite.
- Security bad fixture suite.
- Semgrep/custom checks for token, RBAC, cleanup, and TLS bypass patterns.

## Review Checklist

For any change touching measurement, curl pod, gate, policy, summary, cleanup,
or CI:

- [ ] Does it expose token or Authorization material?
- [ ] Does it widen RBAC scope?
- [ ] Does it allow external metrics URLs?
- [ ] Does it make `NO_GRADE` look like PASS?
- [ ] Does it silently ignore unknown schema or enum values?
- [ ] Does it delete resources without kube-slint ownership metadata?
- [ ] Does it weaken curl pod securityContext?
- [ ] Does it blur correctness test failure and measurement failure?
