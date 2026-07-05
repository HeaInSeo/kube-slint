# Test Strategy

Date: 2026-07-04
Status: Canonical test matrix planning source

## Purpose

This document consolidates bad fixture, kind E2E, and E2E acceptance planning.
It defines what must fail before implementation converts the cases into
executable tests.

## Bad Fixture Matrix

Summary fixtures:

| File | Expected result |
|---|---|
| `summary/missing-schema-version.json` | reject |
| `summary/wrong-schema-version.json` | reject |
| `summary/empty-result-id.json` | reject |
| `summary/duplicate-result-id.json` | reject |
| `summary/unknown-result-status.json` | reject |
| `summary/invalid-generated-at.json` | reject |
| `summary/nan-metric-value.json` | reject recommended; open policy |
| `summary/inf-metric-value.json` | reject recommended; open policy |
| `summary/malformed-json.json` | reject |

Policy fixtures:

| File | Expected result |
|---|---|
| `policy/missing-policy-version.yaml` | reject |
| `policy/wrong-policy-version.yaml` | reject |
| `policy/unknown-operator.yaml` | reject |
| `policy/duplicate-threshold-name.yaml` | reject |
| `policy/missing-metric.yaml` | reject |
| `policy/negative-tolerance.yaml` | reject |
| `policy/nan-threshold-value.yaml` | reject recommended |
| `policy/empty-threshold-name.yaml` | reject |
| `policy/unknown-fail-on.yaml` | reject |
| `policy/baseline-required-but-missing.yaml` | `NO_GRADE` or `FAIL`; open policy |

Security fixtures:

| File | Expected result |
|---|---|
| `security/external-service-url.yaml` | reject |
| `security/external-service-url-template-injection.yaml` | reject |
| `security/ftp-service-url.yaml` | reject |
| `security/insecure-tls-default.yaml` | reject or compatibility warning until migrated |
| `security/clusterrolebinding-default.yaml` | reject |
| `security/privileged-curlpod.yaml` | reject |
| `security/hostpath-curlpod.yaml` | reject |
| `security/kube-system-target.yaml` | reject |
| `security/cleanup-without-owner-label.yaml` | reject |

Fixture rules:

- no real tokens or kubeconfigs;
- filenames describe rejected behavior;
- expected result lives in the test table;
- planned fixtures stay out of executable testdata until validators exist.

## kind E2E Scenario Matrix

| ID | Scenario | Expected result |
|---|---|---|
| E2E-1 | Run with namespace-scoped RBAC only | succeeds |
| E2E-2 | Run without ClusterRoleBinding | succeeds |
| E2E-3 | Metrics Service missing | clear `NO_GRADE` or config error |
| E2E-4 | Metrics endpoint returns HTTP 500 | measurement failure, not app test failure |
| E2E-5 | Metrics endpoint returns malformed Prometheus text | parser failure and `NO_GRADE` |
| E2E-6 | External ServiceURLFormat configured | reject before scraping |
| E2E-7 | Cleanup runs after measurement | only kube-slint-owned resources are deleted |
| E2E-8 | Ten parallel sessions | no artifact or RunID collision |
| E2E-9 | First run with absent baseline | `WARN` or `NO_GRADE` per policy |
| E2E-10 | Regression detected | gate result `FAIL`; strict CI exit code 1 |
| E2E-11 | Invalid policy | reject before trusted evaluation |
| E2E-12 | Invalid summary | reject before trusted evaluation |

## E2E Acceptance Criteria

Every kind E2E scenario should assert:

- command exit code;
- `gate_result`;
- `evaluation_status`;
- `measurement_status`;
- relevant reason codes;
- expected artifact presence or absence;
- cleanup result;
- absence of raw secrets in logs/artifacts.

Do not accept an E2E scenario if:

- invalid input produces `PASS`;
- external ServiceURLFormat creates a curl pod;
- cleanup deletes non-owned resources;
- logs contain raw token or Authorization material;
- missing metrics service is reported as app test assertion failure without a
  kube-slint measurement status;
- ClusterRoleBinding is required for default measurement.

## CI Placement

Recommended phases:

| Phase | Contents | Trigger |
|---|---|---|
| PR smoke | happy path, invalid policy, external URL reject | PR touching gate/harness/security |
| Nightly | full E2E matrix | scheduled |
| Manual | expensive parallel and regression scenarios | workflow_dispatch |

Promotion gates should use:

```yaml
fail-on: FAIL_OR_NOGRADE
```
