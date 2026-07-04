# Failure Catalog

Date: 2026-07-04
Status: Draft UX contract for quality roadmap Sprint 3

## Purpose

kube-slint failures should tell users what happened, what it means, and how to
fix it. The user should be able to distinguish app failure, measurement
failure, policy violation, security reject, and invalid input.

## Message Shape

Recommended format:

```text
kube-slint could not grade this run.

Reason:
  Metrics service "my-app-metrics" was not found in namespace "default".

What this means:
  Your test may have passed, but kube-slint could not collect SLI data.

How to fix:
  1. Ensure your app exposes /metrics.
  2. Create a Service named my-app-metrics.
  3. Or configure metricsServiceName in .slint.yaml.

Result:
  NO_GRADE
```

## Failure Categories

| Situation | Category | Result |
|---|---|---|
| metrics Service missing | measurement unavailable | `NO_GRADE` or config error |
| RBAC denied | measurement unavailable | `NO_GRADE` with permission hint |
| invalid policy | invalid input | reject or `NO_GRADE` |
| invalid summary | invalid input | reject or `NO_GRADE` |
| regression detected | policy violation | `FAIL` when configured |
| first-run baseline missing | first-run adoption | `WARN` or policy-defined `NO_GRADE` |
| external URL blocked | security reject | reject before scraping |
| cleanup partial failure | cleanup warning | `WARN` or explicit cleanup error |
| collection failed | measurement unavailable | `NO_GRADE` |
| unknown enum | invalid input | reject |

## Required Message Fields

Each user-facing failure should include:

- reason;
- meaning;
- fix;
- result;
- machine-readable reason code where available.

## Secret Handling

Failure messages must not include:

- ServiceAccount token;
- Authorization header;
- kubeconfig credentials;
- CI secrets;
- raw credential material.

## Examples

### External ServiceURLFormat

```text
kube-slint blocked an unsafe metrics URL.

Reason:
  ServiceURLFormat resolved to host "evil.example.com", which is outside the
  cluster-local .svc boundary.

What this means:
  kube-slint will not send ServiceAccount Authorization material to an external
  host by default.

How to fix:
  1. Use https://<service>.<namespace>.svc:<port>/metrics.
  2. Or remove the ServiceURLFormat override.

Result:
  SECURITY_REJECT
```

### Invalid Policy

```text
kube-slint could not evaluate the policy.

Reason:
  policy.yaml is missing schema_version: "slint.policy.v1".

What this means:
  kube-slint cannot safely interpret this policy version.

How to fix:
  Add schema_version: "slint.policy.v1" at the top of the policy file.

Result:
  NO_GRADE
```

## Acceptance Criteria

- [ ] No invalid input error suggests the run passed.
- [ ] Measurement failure copy says the user's test may still have passed.
- [ ] Policy violation copy clearly says CI failed because of the gate.
- [ ] Security reject copy names the trust boundary.
- [ ] Messages contain no secret material.
