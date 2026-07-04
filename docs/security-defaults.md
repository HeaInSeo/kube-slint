# kube-slint Security Defaults

Date: 2026-07-04
Status: Proposed contract for quality roadmap Sprint 1

## Purpose

This document defines the default security posture kube-slint should preserve
as it matures from post-RC hardening into an externally consumable guardrail.

kube-slint is a shift-left operational SLI guardrail. It measures Kubernetes
operational signals during tests and evaluates the resulting artifacts through
a policy gate. The security defaults below protect that measurement path
without turning kube-slint into a generic Kubernetes linter or a cluster
management tool.

## Source of Truth

- `docs/DECISIONS.md` D-001: kube-slint identity is a shift-left operational
  quality guardrail.
- `docs/DECISIONS.md` D-002: measurement failure is not correctness test
  failure.
- `docs/DECISIONS.md` D-008: `slint-gate` is a separate policy evaluation
  layer.
- `docs/DECISIONS.md` D-014: post-RC hardening prioritizes secret containment,
  namespace-scoped RBAC, conservative gate semantics, and invalid enum
  rejection.
- `docs/project-status.yaml`: machine-readable status source.
- `docs/post-rc-hardening-design.md`: accepted hardening design for token,
  RBAC, lifecycle, and gate semantics.

## Default Security Posture

Default behavior should be conservative:

- Do not send ServiceAccount tokens to non-cluster-local hosts.
- Do not require cluster-wide RBAC for normal measurement.
- Do not create privileged pods.
- Do not mount host paths.
- Do not delete resources unless kube-slint ownership is clear.
- Do not silently PASS malformed summary or policy input.
- Do not let measurement failure look like a trustworthy PASS.
- Do not print raw credentials in logs, summaries, command strings, or errors.

## Default-Deny Patterns

| Pattern | Default policy | Rationale |
|---|---|---|
| External metrics URL | Reject | Prevent ServiceAccount token exfiltration through ServiceURLFormat. |
| Authorization header to external host | Reject | Authorization material belongs inside the cluster-local scrape boundary. |
| `InsecureSkipVerify` | Reject or explicit dangerous opt-in | TLS verification bypass should not be accidental. |
| `ClusterRoleBinding` | Reject in default path | Normal measurement should be namespace-scoped. |
| Privileged pod | Reject | The curl pod should not need elevated container privileges. |
| `hostPath` | Reject | Measurement should not need node filesystem access. |
| `kube-system` target namespace | Reject by default | Avoid accidental execution against cluster control-plane namespaces. |
| Cleanup without owner label | Reject | Cleanup must not delete resources kube-slint did not create or mark. |
| Unknown policy enum | Reject | Unknown policy input must not be treated as safe. |
| Unknown result status | Reject | Gate output must not silently ignore semantic changes. |

## Allowed Exceptions

Exceptions must be explicit, narrow, and auditable.

An exception may be acceptable only when:

- The option name clearly signals danger.
- The default remains safe.
- The exception is documented near the user-facing setting.
- The gate or harness output records that a dangerous option was enabled.
- Tests cover both default rejection and explicit opt-in behavior.

Examples of dangerous option names:

```yaml
dangerouslyAllowExternalMetricsURL: true
dangerouslySkipTLSVerify: true
dangerouslyAllowClusterWideRBAC: true
dangerouslyAllowKubeSystemNamespace: true
dangerouslyAllowUnsafeCleanup: true
```

## Current Versus Target State

Some defaults are already implemented; others are target policy.

Implemented or documented today:

- curl pod token is read inside the pod from the mounted ServiceAccount token.
- command/error redaction covers Bearer tokens and common secret shapes.
- generated RBAC defaults use `Role` and `RoleBinding`.
- `NO_GRADE` is handled conservatively by the GitHub Action default
  `FAIL_OR_NOGRADE`.
- unknown gate/policy enum handling has been hardened.

Target policy:

- ServiceURLFormat validation rejects external hosts before scraping.
- dangerous option names replace ambiguous insecure knobs.
- security bad fixtures cover external URL, insecure TLS, privileged pod,
  hostPath, cluster-wide RBAC, and unsafe cleanup.
- future release docs clearly separate default-safe behavior from advanced
  dangerous opt-ins.

## Acceptance Checklist

- [ ] External metrics URL rejected by default.
- [ ] Token forwarding to external hosts impossible in default mode.
- [ ] Namespace-scoped RBAC remains default.
- [ ] Cluster-wide RBAC requires explicit dangerous opt-in.
- [ ] Privileged pod and hostPath are rejected in generated/default resources.
- [ ] Cleanup requires kube-slint ownership metadata.
- [ ] Invalid summary/policy cannot produce PASS.
- [ ] CI guardrails detect stale docs that contradict accepted hardening.

## Developer Handoff

Implementation tickets should use the template in
`docs/quality-roadmap-ticket-backlog.md`.

Do not implement broad feature changes from this document directly. Convert
each behavior into a ticket with acceptance criteria, tests, docs to update,
and security impact.
